package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"time"

	provider "github.com/skyguide-ansp/cis-examples/display/surveillance/v0"
	httpUtil "github.com/skyguide-ansp/cis-examples/http"
	"github.com/skyguide-ansp/cis-examples/util"
)

// starts a client that is in charge to retrieve the data
func main() {
	// flags
	dssUrl := flag.String("dss-url", "", "base url of the dss, expect protocol to be part of it")
	dssBasePath := flag.String("dss-base-path", "/surveillance/v0", "base path for the dss")
	oidcTokenUrl := flag.String("oidc-token-url", "", "url of the authentication server, token endpoint expected, protocol expected")
	oidcClientId := flag.String("oidc-client-id", "", "oidc client id")
	oidcClientSecret := flag.String("oidc-client-secret", "", "oidc client secret")
	oidcDssScopes := flag.String("dss-scopes", "surveillance.display_provider", "dss scopes to pass to oidc, default to surveillance.display_provider, optional")
	oidcUssScopes := flag.String("uss-scopes", "surveillance.display_provider", "uss scopes to pass to oidc, default to surveillance.display_provider, optional")
	view := flag.String("view", "", "lat1,lng1,lat2,lng2 each as float")

	flag.Parse()

	dssHost, err := url.Parse(*dssUrl)
	if err != nil {
		panic("unparsable dss url")
	}

	// retrieve oidc credential
	DssCredentials := httpUtil.Credential{
		Scopes:       util.StringToList(*oidcDssScopes),
		Audiences:    []string{dssHost.Host},
		ClientID:     *oidcClientId,
		ClientSecret: *oidcClientSecret,
		TokenURL:     *oidcTokenUrl,
	}

	UssCredentials := httpUtil.Credential{
		Scopes:       util.StringToList(*oidcUssScopes),
		ClientID:     *oidcClientId,
		ClientSecret: *oidcClientSecret,
		TokenURL:     *oidcTokenUrl,
	}

	// create client
	displayProvider, err := provider.NewConsumer(
		DssCredentials,
		UssCredentials,
		*dssUrl,
		*dssBasePath,
	)
	if err != nil {
		panic(err)
	}

	// get providers
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if !util.ValidateView(*view) {
		panic(errors.New("view must be: lat1,lng1,lat2,lng2 with: -90 <= lat <= 90 and long -180 <= lng <= 180"))
	}
	stream, err := displayProvider.GetCurrentTrafficFromView(ctx, *view)
	if err != nil {
		panic(err)
	}

	// print
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-stream:
			if !ok {
				return
			}

			jsonFlightInfo, convertErr := json.Marshal(data.Data)
			if convertErr != nil {
				fmt.Printf("unexpected err : %v", convertErr)
			}
			fmt.Printf("area owner: %s , value: %s", data.Provider, string(jsonFlightInfo))
		}
	}

}
