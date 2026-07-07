package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	api "github.com/skyguide-ansp/cis-examples/api/surveillance/v0"
	httpUtil "github.com/skyguide-ansp/cis-examples/http"
	"github.com/skyguide-ansp/cis-examples/util"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
}

// starts a surveillance display-provider client
func main() {
	// flags
	dssUrl := flag.String("dss-url", "", "base url of the dss, expect protocol to be part of it")
	dssBasePath := flag.String("dss-base-path", "/surveillance/v0", "surveillance service base path for the dss")
	oidcTokenUrl := flag.String("oidc-token-url", "", "url of the authentication server, token endpoint expected, protocol expected")
	oidcClientId := flag.String("oidc-client-id", "", "oidc client id")
	oidcClientSecret := flag.String("oidc-client-secret", "", "oidc client secret")
	oidcScopes := flag.String("oidc-scopes", "surveillance.display_provider", "scopes to pass to oidc, default to surveillance.display_provider, optional")
	view := flag.String("view", "", "lat1,lng1,lat2,lng2 each as float")

	flag.Parse()

	dssBaseUrl, err := url.Parse(*dssUrl + *dssBasePath)
	if err != nil {
		log.Panicf("Failed to parse dss url: %v", err)
	}

	var missing []string
	if *dssUrl == "" {
		missing = append(missing, "dss-url")
	}
	if *oidcTokenUrl == "" {
		missing = append(missing, "oidc-token-url")
	}
	if *oidcClientId == "" {
		missing = append(missing, "oidc-client-id")
	}
	if *oidcClientSecret == "" {
		missing = append(missing, "oidc-client-secret")
	}
	if *view == "" {
		missing = append(missing, "view")
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Missing required flags: %s\n\n", strings.Join(missing, ", "))
		flag.Usage()
		os.Exit(1)
	}

	min, max, err := util.ParseView(*view)
	if err != nil {
		log.Panicf("Failed to parse view: %v", err)
	}
	area := util.Polygon{
		min,
		util.Point{Lat: max.Lat, Lng: min.Lng},
		max,
		util.Point{Lat: min.Lat, Lng: max.Lng},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. retrieve token for DSS interaction
	log.Printf("Get token for audience %q...\n", dssBaseUrl.Hostname())
	dssCredentials := httpUtil.Credential{
		Scopes:       util.StringToList(*oidcScopes),
		Audiences:    []string{dssBaseUrl.Hostname()},
		ClientID:     *oidcClientId,
		ClientSecret: *oidcClientSecret,
		TokenURL:     *oidcTokenUrl,
	}
	dssToken, err := authenticate(ctx, dssCredentials)
	if err != nil {
		log.Panicf("Failed to authenticate: %v", err)
	}
	log.Printf("Token fetched\n")

	// 2. search/subscribe traffic surveilled area in DSS
	log.Printf("Search traffic surveilled areas %q...\n", area)
	tsas, err := searchTrafficSurveilledAreas(ctx, dssBaseUrl, area, dssToken)
	if err != nil {
		log.Panicf("Failed to search traffic surveilled areas: %v", err)
	}
	log.Printf("Surveilled areas discovered: %d\n", len(tsas))

	type uss struct {
		owner   string
		baseUrl string
	}

	now := time.Now()
	providers := make(map[uss]struct{})
	for _, tsa := range tsas {
		if tsa.TimeStart != nil && now.Before((time.Time)(*tsa.TimeStart)) ||
			tsa.TimeEnd != nil && now.After((time.Time)(*tsa.TimeEnd)) {
			log.Printf("%s: Skip inactive surveilled area: %q\n", tsa.Owner, tsa.Id)
			return
		}

		uss := uss{
			owner:   tsa.Owner,
			baseUrl: tsa.UssBaseUrl,
		}
		providers[uss] = struct{}{}
	}

	var wg sync.WaitGroup
	for uss, _ := range providers {
		wg.Go(func() {
			defer func() { log.Printf("%s: Done", uss.owner) }()

			ussBaseUrl, err := url.Parse(uss.baseUrl)
			if err != nil {
				log.Printf("%s: Failed to parse uss base url: %v\n", uss.owner, err)
				return
			}

			// 3. retrieve token for USS interaction
			ussCredentials := httpUtil.Credential{
				Scopes:       util.StringToList(*oidcScopes),
				Audiences:    []string{ussBaseUrl.Hostname()},
				ClientID:     *oidcClientId,
				ClientSecret: *oidcClientSecret,
				TokenURL:     *oidcTokenUrl,
			}

			log.Printf("%s: Get token for audience %q...\n", uss.owner, ussBaseUrl.Hostname())
			ussToken, err := authenticate(ctx, ussCredentials)
			if err != nil {
				log.Printf("%s: Failed to authenticate: %v\n", uss.owner, err)
				return
			}
			log.Printf("%s: Token fetched\n", uss.owner)

			log.Printf("%s: Stream flights...\n", uss.owner)
			err = streamFlights(ctx, ussBaseUrl, *view, ussToken, uss.owner)
			if err != nil {
				log.Printf("%s: %v\n", uss.owner, err)
			}
		})
	}
	wg.Wait()
}

func authenticate(ctx context.Context, creds httpUtil.Credential) (*httpUtil.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	token, err := httpUtil.AuthenticateWithClientCredentials(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	return token, nil
}

func searchTrafficSurveilledAreas(ctx context.Context, dssBaseUrl *url.URL, area util.Polygon, token *httpUtil.Token) ([]*api.TrafficSurveilledArea, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	searchTrafficSurveilledAreasUrl := dssBaseUrl.JoinPath("/dss/traffic_surveilled_areas")
	params := url.Values{"area": {area.String()}}
	searchTrafficSurveilledAreasUrl.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchTrafficSurveilledAreasUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create search traffic surveilled areas request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search traffic surveilled areas: %w", err)
	}

	searchTSAResp, err := httpUtil.DecodeJson[api.SearchTrafficSurveilledAreasResponse](resp)
	if err != nil {
		return nil, fmt.Errorf("decode search traffic surveilled area response: %w", err)
	}

	return searchTSAResp.SurveilledAreas, nil
}

func streamFlights(ctx context.Context, ussBaseUrl *url.URL, view string, token *httpUtil.Token, owner string) error {
	streamFlightsUrl := ussBaseUrl.JoinPath("/uss/flights/stream")
	params := url.Values{"view": {view}}
	streamFlightsUrl.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamFlightsUrl.String(), nil)
	if err != nil {
		return fmt.Errorf("create stream flight request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("stream flights: %w", err)
	}
	defer resp.Body.Close()

	// errors handling
	if resp.StatusCode != http.StatusOK {
		return errors.New(http.StatusText(resp.StatusCode))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			lines, err := httpUtil.ReadOneSSEEvent(reader)
			if err != nil {
				return fmt.Errorf("read SSE event: %w", err)
			}

			flight, err := httpUtil.ParseEventFromSSE[api.Flight](lines)
			if err != nil {
				log.Printf("%s: Failed decoding %q\n", owner, lines)
				continue
			}

			data, err := json.Marshal(flight)
			if err != nil {
				log.Printf("%s: Failed marshalling %v\n", owner, flight)
				continue
			}

			log.Printf("%s: %s\n", owner, data)
		}
	}
}
