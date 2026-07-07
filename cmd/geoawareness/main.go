package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	api "github.com/skyguide-ansp/cis-examples/api/utm/v1"
	httpUtil "github.com/skyguide-ansp/cis-examples/http"
	"github.com/skyguide-ansp/cis-examples/util"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
}

// starts a utm constraint-processing client
func main() {
	// flags
	dssUrl := flag.String("dss-url", "", "base url of the dss, expect protocol to be part of it")
	dssBasePath := flag.String("dss-base-path", "/", "surveillance service base path for the dss")
	oidcTokenUrl := flag.String("oidc-token-url", "", "url of the authentication server, token endpoint expected, protocol expected")
	oidcClientId := flag.String("oidc-client-id", "", "oidc client id")
	oidcClientSecret := flag.String("oidc-client-secret", "", "oidc client secret")
	oidcScopes := flag.String("oidc-scopes", "utm.constraint_processing", "scopes to pass to oidc, default to utm.constraint_processing, optional")
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
	area := &api.Volume4D{
		Volume: &api.Volume3D{
			OutlinePolygon: &api.Polygon{
				Vertices: []*api.LatLngPoint{
					{Lat: min.Lat, Lng: min.Lng},
					{Lat: max.Lat, Lng: min.Lng},
					{Lat: max.Lat, Lng: max.Lng},
					{Lat: min.Lat, Lng: max.Lng},
				},
			},
		},
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
	log.Printf("Query constraint references %q...\n", *view)
	refs, err := queryConstraintReferences(ctx, dssBaseUrl, area, dssToken)
	if err != nil {
		log.Panicf("Failed to query constraint references: %v", err)
	}

	type uss struct {
		manager string
		baseUrl string
	}

	refsByUss := make(map[uss][]*api.ConstraintReference)
	for _, ref := range refs {
		uss := uss{
			manager: ref.Manager,
			baseUrl: ref.UssBaseUrl,
		}
		ussRefs := refsByUss[uss]
		ussRefs = append(ussRefs, ref)
		refsByUss[uss] = ussRefs
	}

	log.Printf("Constraints discovered: %d\n", len(refs))
	for uss, refs := range refsByUss {
		ussBaseUrl, err := url.Parse(uss.baseUrl)
		if err != nil {
			log.Printf("%s: Failed to parse uss base url: %v\n", uss.manager, err)
			continue
		}

		// 3. retrieve token for USS interaction
		ussCredentials := httpUtil.Credential{
			Scopes:       util.StringToList(*oidcScopes),
			Audiences:    []string{ussBaseUrl.Hostname()},
			ClientID:     *oidcClientId,
			ClientSecret: *oidcClientSecret,
			TokenURL:     *oidcTokenUrl,
		}

		log.Printf("%s: Get token for audience %q...\n", uss.manager, ussBaseUrl.Hostname())
		ussToken, err := authenticate(ctx, ussCredentials)
		if err != nil {
			log.Printf("%s: Failed to authenticate: %v\n", uss.manager, err)
			continue
		}
		log.Printf("%s: Token fetched\n", uss.manager)

		for _, ref := range refs {
			cstr, err := getConstraintDetails(ctx, ussBaseUrl, ref.Id, ussToken)
			if err != nil {
				log.Printf("%s: Failed to get constraint details: %v\n", uss.manager, err)
				continue
			}

			log.Printf("%s: geozone: %v\n", uss.manager, string(cstr.Details.GeozoneEd318))
		}
	}
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

func queryConstraintReferences(ctx context.Context, dssBaseUrl *url.URL, area *api.Volume4D, token *httpUtil.Token) ([]*api.ConstraintReference, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	queryConstraintReferencesUrl := dssBaseUrl.JoinPath("/dss/v1/constraint_references/query")

	params := &api.QueryConstraintReferenceParameters{AreaOfInterest: area}
	data, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("create query constraint references request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, queryConstraintReferencesUrl.String(), bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("create query constraint references request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query constraint references: %w", err)
	}
	defer resp.Body.Close()

	queryConstraintResp, err := httpUtil.DecodeJson[api.QueryConstraintReferencesResponse](resp)
	if err != nil {
		return nil, fmt.Errorf("decode query constraint references response: %w", err)
	}

	return queryConstraintResp.ConstraintReferences, nil
}

func getConstraintDetails(ctx context.Context, ussBaseUrl *url.URL, id string, token *httpUtil.Token) (*api.Constraint, error) {
	getConstraintDetailsUrl := ussBaseUrl.JoinPath("/uss/v1/constraints", id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getConstraintDetailsUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create get constraint details request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get constraint details: %w", err)
	}
	defer resp.Body.Close()

	getConstraintResp, err := httpUtil.DecodeJson[api.GetConstraintDetailsResponse](resp)
	if err != nil {
		return nil, fmt.Errorf("decode get constraint details response: %w", err)
	}

	return getConstraintResp.Constraint, nil
}
