package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// control error code, and decode response as type T if the application/json header is present
func DecodeJson[T any](resp *http.Response) (*T, error) {
	// errors handling
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("decode json: response status %s", http.StatusText(resp.StatusCode))
	}

	// if no response return nothing
	if resp.ContentLength == 0 {
		return nil, errors.New("decode json: response is empty")
	}

	// if response is json -> decode it as generic given type
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		return nil, errors.New("decode json: response is not application/json")
	}

	// decode the data as the expected format
	var data T
	err := json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	return &data, nil
}
