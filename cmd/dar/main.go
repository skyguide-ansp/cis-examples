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

	"github.com/skyguide-ansp/cis-examples/api/common"
	dar "github.com/skyguide-ansp/cis-examples/api/dar/v1"
	api "github.com/skyguide-ansp/cis-examples/api/utm/v1"
	httpUtil "github.com/skyguide-ansp/cis-examples/http"
	"github.com/skyguide-ansp/cis-examples/util"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
}

func main() {
	dssURL := flag.String("dss-url", "", "base url of the dss, expect protocol to be part of it")
	view := flag.String("view", "", "lat1,lng1,lat2,lng2 each as float")
	darCoordinatorURL := flag.String("dar-coordinator-url", "", "base url of the dar coordinator, expect protocol to be part of it")
	darID := flag.String("dar-id", "", "dar identifier")
	startTimeValue := flag.String("start-time", "", "dar start time in rfc3339 format")
	endTimeValue := flag.String("end-time", "", "dar end time in rfc3339 format")
	oidcTokenURL := flag.String("oidc-token-url", "", "url of the authentication server, token endpoint expected, protocol expected")
	oidcClientID := flag.String("oidc-client-id", "", "oidc client id")
	oidcClientSecret := flag.String("oidc-client-secret", "", "oidc client secret")
	oidcScopes := flag.String("oidc-scopes", "utm.constraint_processing", "scopes to pass to oidc, default to utm.constraint_processing, optional")
	darVacatedOIDCScope := flag.String("dar-vacated-oidc-scope", "utm.dar_vacated", "scope to pass to oidc for the DAR VACATED callback")
	flag.Parse()

	var missing []string
	if *dssURL == "" {
		missing = append(missing, "dss-url")
	}
	if *view == "" {
		missing = append(missing, "view")
	}
	if *darCoordinatorURL == "" {
		missing = append(missing, "dar-coordinator-url")
	}
	if *darID == "" {
		missing = append(missing, "dar-id")
	}
	if *startTimeValue == "" {
		missing = append(missing, "start-time")
	}
	if *endTimeValue == "" {
		missing = append(missing, "end-time")
	}
	if *oidcTokenURL == "" {
		missing = append(missing, "oidc-token-url")
	}
	if *oidcClientID == "" {
		missing = append(missing, "oidc-client-id")
	}
	if *oidcClientSecret == "" {
		missing = append(missing, "oidc-client-secret")
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Missing required flags: %s\n\n", strings.Join(missing, ", "))
		flag.Usage()
		os.Exit(1)
	}

	dssBaseURL, err := parseBaseURL(*dssURL, "DSS")
	if err != nil {
		log.Fatal(err)
	}
	coordinatorURL, err := url.Parse(*darCoordinatorURL)
	if err != nil || coordinatorURL.Scheme == "" || coordinatorURL.Hostname() == "" {
		log.Fatalf("Invalid DAR Coordinator URL: %q", *darCoordinatorURL)
	}

	startTime, err := time.Parse(time.RFC3339, *startTimeValue)
	if err != nil {
		log.Fatalf("Invalid DAR start time: %q", *startTimeValue)
	}
	endTime, err := time.Parse(time.RFC3339, *endTimeValue)
	if err != nil {
		log.Fatalf("Invalid DAR end time: %q", *endTimeValue)
	}
	if !endTime.After(startTime) {
		log.Fatal("DAR end time must be after start time")
	}
	min, max, err := util.ParseView(*view)
	if err != nil {
		log.Fatalf("Invalid DAR area: %v", err)
	}
	area := darArea(min, max, startTime, endTime)
	scopes := util.StringToList(*oidcScopes)
	credentialsFor := func(audience string, requestScopes []string) httpUtil.Credential {
		return httpUtil.Credential{
			Audiences:    []string{audience},
			ClientID:     *oidcClientID,
			ClientSecret: *oidcClientSecret,
			TokenURL:     *oidcTokenURL,
			Scopes:       requestScopes,
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. retrieve token for DSS interaction
	dssToken, err := authenticate(ctx, credentialsFor(dssBaseURL.Hostname(), scopes))
	if err != nil {
		log.Fatalf("Failed to authenticate with DSS: %v", err)
	}
	log.Print("DSS token fetched")

	// 2. retrieve token for DAR Coordinator interaction
	darVacatedScopes := util.StringToList(*darVacatedOIDCScope)
	coordinatorToken, err := authenticate(ctx, credentialsFor(coordinatorURL.Hostname(), darVacatedScopes))
	if err != nil {
		log.Fatalf("Failed to authenticate with DAR Coordinator: %v", err)
	}
	log.Print("DAR Coordinator token fetched")

	// 3. create DAR through the sandbox
	if err := createDAR(ctx, coordinatorURL, *darID, startTime, endTime, coordinatorToken); err != nil {
		log.Fatalf("Failed to create DAR: %v", err)
	}
	log.Printf("DAR %q created", *darID)

	// 4. retrieve initial DAR status
	status, err := getDARStatus(ctx, coordinatorURL, *darID, coordinatorToken)
	if err != nil {
		log.Fatalf("Failed to retrieve DAR status: %v", err)
	}
	logDARStatus(status)

	// 5. query the published constraint in DSS
	constraints, err := queryConstraintReferences(ctx, dssBaseURL, area, dssToken)
	if err != nil {
		log.Fatalf("Failed to query DSS constraint references: %v", err)
	}
	log.Printf("Constraints discovered: %d", len(constraints))
	if len(constraints) == 0 {
		log.Print("No constraints discovered for the DAR area and time window")
	}

	// 6. retrieve constraint details from each USS manager, looking for our DAR's VACATED callback URL
	var vacatedCallbackURL *url.URL
	for _, constraint := range constraints {
		if constraint.UssBaseUrl == "" {
			log.Printf("constraint %s: no USS base URL, skipping", constraint.Id)
			continue
		}

		managerURL, err := parseBaseURL(constraint.UssBaseUrl, "USS manager")
		if err != nil {
			log.Printf("constraint %s: %v", constraint.Id, err)
			continue
		}
		managerToken, err := authenticate(ctx, credentialsFor(managerURL.Hostname(), scopes))
		if err != nil {
			log.Printf("constraint %s: failed to authenticate with USS manager %s: %v", constraint.Id, managerURL.Hostname(), err)
			continue
		}
		log.Printf("constraint %s: token fetched for USS manager %s", constraint.Id, managerURL.Hostname())
		constraintDetails, err := getConstraintDetails(ctx, managerURL, constraint.Id, managerToken)
		if err != nil {
			log.Printf("constraint %s: failed to retrieve details: %v", constraint.Id, err)
			continue
		}
		if constraintDetails == nil || constraintDetails.Details == nil {
			log.Printf("constraint %s: details did not include an ED-318 geozone", constraint.Id)
			continue
		}
		log.Printf("constraint %s: ED-318 geozone retrieved", constraint.Id)

		var feature dar.ED318Feature
		if err := json.Unmarshal(constraintDetails.Details.GeozoneEd318, &feature); err != nil {
			log.Printf("constraint %s: decode ED-318 geozone: %v", constraint.Id, err)
			continue
		}
		callbackURL, err := darVacatedCallbackURL(&feature, *darID)
		if err != nil {
			log.Printf("constraint %s: %v", constraint.Id, err)
			continue
		}
		log.Printf("constraint %s: VACATED callback URL discovered: %s", constraint.Id, callbackURL)
		vacatedCallbackURL = callbackURL
	}
	if vacatedCallbackURL == nil {
		log.Fatalf("Could not discover a VACATED callback URL for DAR %q", *darID)
	}

	// 7. report the USSP as vacated, using the callback URL discovered in the ED-318 geozone
	vacatedToken, err := authenticate(ctx, credentialsFor(vacatedCallbackURL.Hostname(), darVacatedScopes))
	if err != nil {
		log.Fatalf("Failed to authenticate for DAR VACATED callback: %v", err)
	}
	if err := markDARVacated(ctx, vacatedCallbackURL, vacatedToken); err != nil {
		log.Fatalf("Failed to report DAR VACATED: %v", err)
	}
	log.Printf("DAR %q marked as vacated", *darID)

	// 8. retrieve DAR status after the VACATED callback
	status, err = getDARStatus(ctx, coordinatorURL, *darID, coordinatorToken)
	if err != nil {
		log.Fatalf("Failed to retrieve DAR status after VACATED callback: %v", err)
	}
	logDARStatus(status)

	// 9. deactivate DAR through the sandbox
	if err := deactivateDAR(ctx, coordinatorURL, *darID, coordinatorToken); err != nil {
		log.Fatalf("Failed to deactivate DAR: %v", err)
	}
	log.Printf("DAR %q deactivated", *darID)
}

func authenticate(ctx context.Context, credentials httpUtil.Credential) (*httpUtil.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	token, err := httpUtil.AuthenticateWithClientCredentials(ctx, credentials)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	return token, nil
}

func parseBaseURL(value, name string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("invalid %s URL: %q", name, value)
	}
	return parsed, nil
}

func darArea(min, max util.Point, startTime, endTime time.Time) *api.Volume4D {
	return &api.Volume4D{
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
		TimeStart: (*common.Time)(&startTime),
		TimeEnd:   (*common.Time)(&endTime),
	}
}

func darVacatedCallbackURL(feature *dar.ED318Feature, darID string) (*url.URL, error) {
	darProperties := feature.Properties.ExtendedProperties.DAR
	if darProperties.ID != darID {
		return nil, fmt.Errorf("geozone does not belong to DAR %q", darID)
	}
	if darProperties.VacatedCallbackURL == "" {
		return nil, fmt.Errorf("geozone for DAR %q has no VACATED callback URL", darID)
	}
	return parseBaseURL(darProperties.VacatedCallbackURL, "VACATED callback")
}

func logDARStatus(status *dar.Status) {
	statusJSON, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		log.Printf("Failed to encode DAR status: %v", err)
		return
	}
	log.Print(string(statusJSON))
}

func createDAR(ctx context.Context, coordinatorURL *url.URL, darID string, startTime, endTime time.Time, token *httpUtil.Token) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	payload, err := json.Marshal(dar.CreateRequest{
		ID:        darID,
		StartTime: startTime,
		EndTime:   endTime,
	})
	if err != nil {
		return fmt.Errorf("encode DAR creation request: %w", err)
	}

	createDARURL := coordinatorURL.JoinPath("/dar_sandbox/v1/dars")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createDARURL.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create DAR request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send DAR creation request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected DAR creation response status: %s", resp.Status)
	}
	return nil
}

func getDARStatus(ctx context.Context, coordinatorURL *url.URL, darID string, token *httpUtil.Token) (*dar.Status, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	getDARStatusURL := coordinatorURL.JoinPath("/dar_sandbox/v1/dars", darID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getDARStatusURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create DAR status request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send DAR status request: %w", err)
	}
	defer resp.Body.Close()

	status, err := httpUtil.DecodeJson[dar.Status](resp)
	if err != nil {
		return nil, fmt.Errorf("decode DAR status response: %w", err)
	}
	return status, nil
}

func queryConstraintReferences(ctx context.Context, dssURL *url.URL, area *api.Volume4D, token *httpUtil.Token) ([]*api.ConstraintReference, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	payload, err := json.Marshal(&api.QueryConstraintReferenceParameters{AreaOfInterest: area})
	if err != nil {
		return nil, fmt.Errorf("encode constraint reference query: %w", err)
	}

	queryURL := dssURL.JoinPath("/dss/v1/constraint_references/query")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, queryURL.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create constraint reference query: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send constraint reference query: %w", err)
	}
	defer resp.Body.Close()

	result, err := httpUtil.DecodeJson[api.QueryConstraintReferencesResponse](resp)
	if err != nil {
		return nil, fmt.Errorf("decode constraint reference query: %w", err)
	}
	return result.ConstraintReferences, nil
}

func getConstraintDetails(ctx context.Context, ussURL *url.URL, constraintID string, token *httpUtil.Token) (*api.Constraint, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	detailsURL := ussURL.JoinPath("/uss/v1/constraints", constraintID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailsURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create constraint details request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send constraint details request: %w", err)
	}
	defer resp.Body.Close()

	result, err := httpUtil.DecodeJson[api.GetConstraintDetailsResponse](resp)
	if err != nil {
		return nil, fmt.Errorf("decode constraint details response: %w", err)
	}
	return result.Constraint, nil
}

func markDARVacated(ctx context.Context, callbackURL *url.URL, token *httpUtil.Token) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create DAR VACATED request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send DAR VACATED request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected DAR VACATED response status: %s", resp.Status)
	}
	return nil
}

func deactivateDAR(ctx context.Context, coordinatorURL *url.URL, darID string, token *httpUtil.Token) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	deactivationURL := coordinatorURL.JoinPath("/dar_sandbox/v1/dars", darID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deactivationURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create DAR deactivation request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send DAR deactivation request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected DAR deactivation response status: %s", resp.Status)
	}
	return nil
}
