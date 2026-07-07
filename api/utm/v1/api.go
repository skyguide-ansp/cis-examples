package v1

import (
	"encoding/json"

	"github.com/skyguide-ansp/cis-examples/api/common"
)

type Constraint struct {
	Reference *ConstraintReference `json:"reference,omitempty"`
	Details   *ConstraintDetails   `json:"details,omitempty"`
}

type ConstraintReference struct {
	Id              string       `json:"id,omitempty"`
	Manager         string       `json:"manager,omitempty"`
	UssAvailability string       `json:"uss_availability,omitempty"`
	Version         int32        `json:"version,omitempty"`
	Ovn             string       `json:"ovn,omitempty"`
	TimeStart       *common.Time `json:"time_start,omitempty"`
	TimeEnd         *common.Time `json:"time_end,omitempty"`
	UssBaseUrl      string       `json:"uss_base_url,omitempty"`
}

type ConstraintDetails struct {
	Volumes      []*Volume4D     `json:"volumes,omitempty"`
	Type         string          `json:"type,omitempty"`
	GeozoneEd318 json.RawMessage `json:"geozone_ed318,omitempty"`
}

type Volume4D struct {
	Volume    *Volume3D    `json:"volume,omitempty"`
	TimeStart *common.Time `json:"time_start,omitempty"`
	TimeEnd   *common.Time `json:"time_end,omitempty"`
}

type Volume3D struct {
	OutlineCircle  *Circle   `json:"outline_circle,omitempty"`
	OutlinePolygon *Polygon  `json:"outline_polygon,omitempty"`
	AltitudeLower  *Altitude `json:"altitude_lower,omitempty"`
	AltitudeUpper  *Altitude `json:"altitude_upper,omitempty"`
}

type Circle struct {
	Center *LatLngPoint `json:"center,omitempty"`
	Radius *Radius      `json:"radius,omitempty"`
}

type Radius struct {
	Value float64 `json:"value,omitempty"`
	Units string  `json:"units,omitempty"`
}

type Polygon struct {
	Vertices []*LatLngPoint `json:"vertices,omitempty"`
}

type LatLngPoint struct {
	Lat float64 `json:"lat,omitempty"`
	Lng float64 `json:"lng,omitempty"`
}

type Altitude struct {
	Value     float64 `json:"value,omitempty"`
	Reference string  `json:"reference,omitempty"`
	Units     string  `json:"units,omitempty"`
}

type QueryConstraintReferenceParameters struct {
	AreaOfInterest *Volume4D `json:"area_of_interest,omitempty"`
}

type QueryConstraintReferencesResponse struct {
	ConstraintReferences []*ConstraintReference `json:"constraint_references,omitempty"`
}

type GetConstraintDetailsResponse struct {
	Constraint *Constraint `json:"constraint,omitempty"`
}
