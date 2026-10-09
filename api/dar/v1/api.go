package v1

import "time"

// CreateRequest creates a DAR through the Coordinator sandbox.
type CreateRequest struct {
	// ID identifies the DAR.
	ID string `json:"id"`
	// StartTime is the start of the DAR activation window.
	StartTime time.Time `json:"start_time"`
	// EndTime is the end of the DAR activation window.
	EndTime time.Time `json:"end_time"`
}

type Status struct {
	// ID identifies the DAR.
	ID string `json:"id"`
	// Status is the DAR lifecycle state.
	Status string `json:"status"`
	// NotifiedUSSURLs lists USS callback URLs notified when the DAR was activated.
	NotifiedUSSURLs []string `json:"notified_uss_urls"`
	// VacatedSubjects lists authenticated USS subjects that reported the DAR vacated.
	VacatedSubjects []string `json:"vacated_subjects"`
}

// ED318Feature models the DAR extension of an ED-318 geozone.
type ED318Feature struct {
	Properties struct {
		ExtendedProperties struct {
			DAR struct {
				ID                 string `json:"id"`
				VacatedCallbackURL string `json:"vacatedCallbackUrl"`
			} `json:"dar"`
		} `json:"extendedProperties"`
	} `json:"properties"`
}
