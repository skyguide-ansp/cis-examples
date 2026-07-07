package common

import (
	"encoding/json"
	"time"
)

type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	// We can create a "shadow" type to avoid infinite recursion
	// or manually construct a string/map.
	return json.Marshal(&struct {
		// RFC3339-formatted time/date string.  The time zone must be 'Z'.
		Value  string `json:"value,omitempty"`
		Format string `json:"format,omitempty"`
	}{
		Value:  (time.Time)(t).UTC().Format(time.RFC3339),
		Format: "RFC3339",
	})
}

func (t *Time) UnmarshalJSON(data []byte) error {
	aux := struct {
		Value string `json:"value,omitempty"`
	}{}
	err := json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}
	goTime, err := time.Parse(time.RFC3339, aux.Value)
	if err != nil {
		return err
	}
	*t = (Time)(goTime)
	return nil
}
