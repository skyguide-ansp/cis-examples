package http

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// SSE Event structure of response Body:
// event: <eventName>
// data: <data, ex: first line of json>
// data: <data ex: second line of json>
// data: <data ex: third line of json>
// ...
// data: <data ex: last line of json>
//
//	<- this space here indicates the end of the event here (there is only one \n despite this comment)
//
// data: <data ex: first line of second json>
// ...
// data: <data ex: last line of second json>
//
// data: <data ex: first line of third json>
//
// And this as long as the context of the request is alive, the server will continuously write into it

// ReadOneSSEEvent gather all lines between the \n space that separate the data
func ReadOneSSEEvent(ctx context.Context, r *bufio.Reader) ([]string, error) {
	var lines []string

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, err
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			return lines, nil // end of event
		}
		lines = append(lines, line)
	}
}

// ParseEventFromSSE convert lines into json and decode according to the given type
func ParseEventFromSSE[T any](lines []string) (*T, error) {
	var dataLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "data:") {
			// Per SSE spec: trim exactly one leading space if present
			payload := strings.TrimPrefix(line, "data:")
			payload = strings.TrimPrefix(payload, " ")
			dataLines = append(dataLines, payload)
		}
	}

	if len(dataLines) == 0 {
		return nil, errors.New("SSE event contains no data field")
	}

	// Concatenate multiline data payloads
	jsonPayload := strings.Join(dataLines, "\n")

	var event T
	if err := json.Unmarshal([]byte(jsonPayload), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FlightEvent data: %w", err)
	}

	return &event, nil
}
