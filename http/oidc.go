package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Credential struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
	Audiences    []string
}

type Token struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token,omitempty"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
	ExpiresIn        int    `json:"expires_in"` // Seconds
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	RequestTime      time.Time
}

// AuthenticateWithClientCredentials create auth request as client_credentials with scope and audience and run it
func AuthenticateWithClientCredentials(ctx context.Context, credential Credential) (*Token, error) {
	values := url.Values{}
	values.Add("client_id", credential.ClientID)
	values.Add("client_secret", credential.ClientSecret)
	values.Add("grant_type", "client_credentials")
	if len(credential.Scopes) > 0 {
		values.Add("scope", strings.Join(credential.Scopes, " "))
	}
	if len(credential.Audiences) > 0 {
		values.Add("audience", strings.Join(credential.Audiences, " "))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, credential.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request, %+w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// perform request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create token: %+w", err)
	}

	// decode token and set the request time
	token, err := DecodeJson[Token](resp)
	if err != nil {
		return nil, fmt.Errorf("decode token: %+w", err)
	}
	if token == nil {
		return nil, errors.New("could not retrieve token")
	}
	token.RequestTime = time.Now()

	return token, nil
}
