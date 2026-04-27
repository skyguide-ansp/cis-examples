package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"time"

	provider "github.com/skyguide-ansp/cis-examples/display/surveillance/v0"
	httpUtil "github.com/skyguide-ansp/cis-examples/http"
	"github.com/skyguide-ansp/cis-examples/util"
)

// starts a client that is in charge to retrieve the data
func main() {
	// flags
	ussUrl := flag.String("uss-url", "", "base url of the uss")
	ussBasePath := flag.String("uss-base-path", "/surveillance/v0", "base path of the uss")
	dssUrl := flag.String("dss-url", "", "base url of the dss")
	dssBasePath := flag.String("dss-base-path", "/surveillance/v0", "base path for the dss")
	oidcTokenUrl := flag.String("oidc-url", "", "url of the authentication server, token endpoint exected")
	oidcClientId := flag.String("client-id", "", "oidc client id")
	oidcClientSecret := flag.String("client-secret", "", "oidc client secret")
	oidcDssScopes := flag.String("dss-scopes", "surveillance.display_provider", "dss scopes to pass to oidc")
	oidcUssScopes := flag.String("uss-scopes", "surveillance.display_provider", "uss scopes to pass to oidc")
	oidcDssAudiences := flag.String("dss-audiences", "", "dss audience to pass to oidc")
	oidcUssAudiences := flag.String("uss-audiences", "", "uss audience to pass to oidc")
	view := flag.String("view", "", "lat1,lng1,lat2,lng2 each as float")

	flag.Parse()

	// retrieve oidc credential
	DssCredentials := httpUtil.Credential{
		Scopes:       httpUtil.StringToList(*oidcDssScopes),
		Audiences:    httpUtil.StringToList(*oidcDssAudiences),
		ClientID:     *oidcClientId,
		ClientSecret: *oidcClientSecret,
		TokenURL:     *oidcTokenUrl,
	}

	UssCredentials := httpUtil.Credential{
		Scopes:       httpUtil.StringToList(*oidcUssScopes),
		Audiences:    httpUtil.StringToList(*oidcUssAudiences),
		ClientID:     *oidcClientId,
		ClientSecret: *oidcClientSecret,
		TokenURL:     *oidcTokenUrl,
	}

	// create client
	displayProvider, err := provider.NewClient(
		DssCredentials,
		UssCredentials,
		*dssUrl,
		*dssBasePath,
		*ussUrl,
		*ussBasePath,
	)
	if err != nil {
		panic(err)
	}

	// get providers
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
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
		case data := <-stream:
			jsonFlightInfo, convertErr := json.Marshal(data.Data)
			if convertErr != nil {
				fmt.Printf("unexpected err : %v", convertErr)
			}
			fmt.Printf("area owner: %s , value: %s", data.Provider, string(jsonFlightInfo))
		}
	}

}
