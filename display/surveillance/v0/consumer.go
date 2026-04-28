package provider

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	surveillance_dss_v0 "github.com/skyguide-ansp/cis-examples/api/surveillance/v0/dss"
	surveillance_uss_v0 "github.com/skyguide-ansp/cis-examples/api/surveillance/v0/uss"
	httpUtil "github.com/skyguide-ansp/cis-examples/http"
)

type SurveillanceClientV0 struct {
	DssConfig *httpUtil.Authorizer
	UssConfig *httpUtil.Authorizer
	dss       surveillance_dss_v0.ClientInterface
	uss       surveillance_uss_v0.ClientInterface
}

type Provider struct {
	name string
	url  string
}

type TrafficDataAndProvider struct {
	Provider string
	Data     *surveillance_uss_v0.Flight
}

// pass configuration to create a new client for display provider
// as for uss and dss there might be the same server url, but the scopes and audiences may changes
func NewClient(dssCredential httpUtil.Credential, ussCredential httpUtil.Credential, dssBaseUrl, dssBasePath, ussBaseUrl, ussBasePath string) (*SurveillanceClientV0, error) {
	dssAuthorizer := &httpUtil.Authorizer{
		Credential: dssCredential,
		Token:      nil,
	}

	ussAuthorizer := &httpUtil.Authorizer{
		Credential: ussCredential,
		Token:      nil,
	}

	// use generated openApi client but reuse the same token
	dssOpenApiClient, err := surveillance_dss_v0.NewClientWithResponses(
		dssBaseUrl+"/"+strings.TrimPrefix(dssBasePath, "/"),
		surveillance_dss_v0.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			_, tokenErr := dssAuthorizer.SetAuthorizationHeader(ctx, req)
			return tokenErr
		}),
	)

	ussOpenApiClient, err := surveillance_uss_v0.NewClientWithResponses(
		ussBaseUrl+"/"+strings.TrimPrefix(ussBasePath, "/"),
		surveillance_uss_v0.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			_, tokenErr := ussAuthorizer.SetAuthorizationHeader(ctx, req)
			return tokenErr
		}),
	)

	if err != nil {
		return nil, err
	}

	return &SurveillanceClientV0{
		DssConfig: dssAuthorizer,
		UssConfig: ussAuthorizer,
		dss:       dssOpenApiClient,
		uss:       ussOpenApiClient,
	}, nil
}

// list the providers in the given area

// Call dss over the views and retrieve all the traffic area providers
// Then call each of them to retrieve the original data stream
func (client *SurveillanceClientV0) GetCurrentTrafficFromView(ctx context.Context, view string) (chan *TrafficDataAndProvider, error) {
	// get the list of uss from the DSS client
	surveilledAreas, err := client.listTrafficSurveilledArea(ctx, &surveillance_dss_v0.SearchTrafficSurveilledAreasParams{
		Area:         view,
		LatestTime:   time.Now().Add(time.Hour),
		EarliestTime: time.Now().Add(time.Duration(-1) * time.Hour),
	})
	if err != nil {
		return nil, err
	}
	if surveilledAreas == nil {
		return nil, errors.New("response is empty")
	}
	if surveilledAreas.ServiceAreas == nil {
		return nil, errors.New("no provider found")
	}

	return client.StreamFlightInAreas(ctx, surveilledAreas, view)
}

func (client *SurveillanceClientV0) StreamFlightInAreas(ctx context.Context, surveilledAreas *surveillance_dss_v0.SearchTrafficSurveilledAreasResponse, view string) (chan *TrafficDataAndProvider, error) {
	// prepare a chanel for the stream of flights
	flightEventStream := make(chan *TrafficDataAndProvider, 100)

	// start streaming to the channel
	for _, area := range *surveilledAreas.ServiceAreas {
		go func() {
			listenErr := client.listenTrafficFromSource(ctx, &area, flightEventStream, view)
			if listenErr != nil {
				fmt.Printf("closing stream to %s, because of error: %v", area.Owner, listenErr)
			}
		}()
	}

	return flightEventStream, nil
}

// Call DSS to get all the Providers Area concerned by the view
// performs GET /uss/traffic_surveilled_areas
func (client *SurveillanceClientV0) listTrafficSurveilledArea(ctx context.Context, param *surveillance_dss_v0.SearchTrafficSurveilledAreasParams) (*surveillance_dss_v0.SearchTrafficSurveilledAreasResponse, error) {
	resp, err := client.dss.SearchTrafficSurveilledAreas(ctx, param)
	if err != nil {
		return nil, err
	}

	decoded, err := httpUtil.DecodeHttpRequest[surveillance_dss_v0.SearchTrafficSurveilledAreasHttpResponse](resp)
	if err != nil {
		return nil, err
	}
	return decoded.JSON200, nil
}

// Call the USS to listen the traffic and stream it into a channel
// For each Traffic Surveilled Area -> call GET /uss/flights/stream
func (client *SurveillanceClientV0) listenTrafficFromSource(ctx context.Context, area *surveillance_dss_v0.TrafficSurveilledArea, output chan *TrafficDataAndProvider, view string) error {

	provider := area.Owner

	resp, err := client.uss.StreamFlights(ctx, &surveillance_uss_v0.StreamFlightsParams{
		View: view,
	})
	if err != nil {
		return err
	}

	reader := bufio.NewReader(resp.Body)

	for {
		select {
		case <-ctx.Done():
			_ = resp.Body.Close()
			return nil
		default:
			lines, readErr := httpUtil.ReadOneSSEEvent(ctx, reader)
			if readErr != nil {
				continue
			}

			event, readErr := httpUtil.ParseEventFromSSE[surveillance_uss_v0.Flight](lines)
			if readErr != nil {
				continue
			}

			output <- &TrafficDataAndProvider{
				Provider: provider,
				Data:     event,
			}
		}

	}

}
