package v0

import (
	"github.com/skyguide-ansp/cis-examples/api/common"
)

type TrafficSurveilledArea struct {
	// Unique identifier for this Traffic Surveilled Area.
	Id string `json:"id,omitempty"`
	// The base URL of a USS implementation that implements the parts of the USS-USS API necessary for providing the
	// details of this operational intent, and telemetry during non-conformance or contingency, if applicable.
	UssBaseUrl string `json:"uss_base_url,omitempty"`
	// Assigned by the DSS based on creating client’s ID (via access token).
	// Used for restricting mutation and deletion operations to owner.
	Owner string `json:"owner,omitempty"`
	// A version string used to reference an object at a particular point in time.
	// Any updates to an object must contain the corresponding version to maintain idempotent updates.
	Version string `json:"version,omitempty"`
	// Beginning time of surveillance.
	TimeStart *common.Time `json:"time_start,omitempty"`
	// End time of surveillance.
	TimeEnd *common.Time `json:"time_end,omitempty"`
}

type SearchTrafficSurveilledAreasResponse struct {
	// Traffic Surveilled Area in the area of interest.
	SurveilledAreas []*TrafficSurveilledArea `json:"surveilled_areas,omitempty"`
}

type AircraftId struct {
	// Icao The ICAO hex code broadcast by the transponder. This is a six-digit hexadecimal [0-9,a-f] code.
	Icao string `json:"icao,omitempty"`
	// IcaoType The standard aircraft type code as published by the ICAO. For example, a Cessna 172 is “C172” and a Boeing 777-300 is “B773”
	IcaoType string `json:"icao_type,omitempty"`
	// Registration Aircraft Registration or “Tail Number”. These should be entered in all caps.
	Registration string `json:"registration,omitempty"`
}

type Position struct {
	// altitude. This value is provided in meters and must have a minimum resolution of 1 meter.  Invalid, No Value or Unknown is -1000 m.
	Alt *float64 `json:"alt,omitempty"`
	// latitude
	Lat *float64 `json:"lat,omitempty"`
	// longitude
	Lng *float64 `json:"lng,omitempty"`
	// PressureAltitude The uncorrected altitude (based on reference standard 29.92 inHg, 1013.25 mb) provides a reference for algorithms that utilize "altitude deltas" between aircraft.  This value is provided in meters and must have a minimum resolution of 1 meter.  Invalid, No Value or Unknown is -1000 m.
	PressureAltitude *float64 `json:"pressure_altitude,omitempty"`
}

type AircraftState struct {
	// aircraft location
	Position Position `json:"position"`
	// Ground speed of flight in meters per second.  Invalid, No Value, or Unknown is 255 m/s, if speed is >254.25 m/s then report 254.25 m/s.
	Speed *float64 `json:"speed,omitempty"`
	// Timestamp Time at which this state was valid.  This may be the time coming from the source, such as a GPS, or the time when the system computes the values using an algorithm such as an Extended Kalman Filter (EKF).  Timestamp must be expressed with a minimum resolution of 1/10th of a second.
	Timestamp common.Time `json:"timestamp"`
	// VerticalSpeed Speed up (vertically) WGS84-HAE, m/s.  Invalid, No Value, or Unknown is 63 m/s, if speed is >62 m/s then report 62 m/s.
	VerticalSpeed *float64 `json:"vertical_speed,omitempty"`
	// Track Direction of flight expressed as a "True North-based" ground track angle.  This value is provided in clockwise degrees with a minimum resolution of 1 degree.  If aircraft is not moving horizontally, use the "Unknown" value.  A value of 361 indicates invalid, no value, or unknown.
	Track *float64 `json:"track,omitempty"`
}

type Source struct {
	// Data Raw data
	Data []byte `json:"data,omitempty"`
	// Format Data source data format
	Format string `json:"format,omitempty"`
	// Id Data source identifier
	Id string `json:"id,omitempty"`
}

type Flight struct {
	// AircraftId Identification of the aircraft. At least one field of this object must be specified.
	AircraftId AircraftId `json:"aircraft_id,omitempty"`
	// kind of aircraft
	AircraftType string `json:"aircraft_type,omitempty"`
	// Aircraft location, speed and heading summary
	CurrentState *AircraftState `json:"current_state,omitempty"`
	// Raw Data
	Source *Source `json:"source,omitempty"`
}
