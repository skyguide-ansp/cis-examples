package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type Authorizer struct {
	Credential Credential
	Token      *Token
	Lock       sync.Mutex
}

// set Authorization header calling Oidc tools
func (client *Authorizer) SetAuthorizationHeader(ctx context.Context, req *http.Request) (*http.Request, error) {
	client.Lock.Lock()
	defer client.Lock.Unlock()

	if IsTokenExpired(client.Token) {
		token, refreshErr := AuthenticateToDSSWithClientCredentials(ctx, client.Credential)
		if refreshErr != nil {
			return nil, refreshErr
		}
		client.Token = token
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token.AccessToken))
	return req, nil
}

// control error code, and decode response as type T
func DecodeHttpRequest[T any](resp *http.Response) (*T, error) {
	if resp == nil {
		return nil, nil
	}

	defer func() {
		if errC := resp.Body.Close(); errC != nil {
			fmt.Printf("failed to close response body %+v", errC)
		}
	}()

	// errors handling
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("authentication forbidden")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, errors.New("authentication failed")
	}
	if resp.StatusCode == http.StatusRequestEntityTooLarge {
		return nil, errors.New("requested area is too large")
	}

	// if no response return nothing
	if resp.StatusCode == http.StatusNoContent || resp.ContentLength == 0 {
		return nil, nil
	}

	// if response is json -> decode it as generic given type
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		// decode the data as the expected format
		var data T
		err := json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			return nil, errors.New("unparsable token")
		}
		return &data, nil
	}

	return nil, nil
}
