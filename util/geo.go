package util

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateLatitude(lat float64) error {
	if lat < -90.0 || 90.0 < lat {
		return errors.New("latitude outside range")
	}
	return nil
}

func ValidateLongitude(lng float64) error {
	if lng < -180.0 || 180.0 < lng {
		return errors.New("longitude outside range")
	}
	return nil
}

func ValidateLatitudes(latitudes ...float64) error {
	for _, lat := range latitudes {
		if err := ValidateLatitude(lat); err != nil {
			return err
		}
	}
	return nil
}

func ValidateLongitudes(longitudes ...float64) error {
	for _, lng := range longitudes {
		if err := ValidateLongitude(lng); err != nil {
			return err
		}
	}
	return nil
}

type Point struct {
	Lat float64
	Lng float64
}

func ParseView(view string) (Point, Point, error) {
	var min, max Point
	_, err := fmt.Sscanf(view, "%f,%f,%f,%f", &min.Lat, &min.Lng, &max.Lat, &max.Lng)
	if err != nil {
		return min, max, errors.New("invalid format")
	}
	if err := ValidateLongitudes(min.Lat, max.Lat); err != nil {
		return min, max, err
	}
	if err := ValidateLatitudes(min.Lng, max.Lng); err != nil {
		return min, max, err
	}
	return min, max, nil
}

type Polygon []Point

func (p Polygon) String() string {
	coords := make([]string, len(p))
	for i, pt := range p {
		coords[i] = fmt.Sprintf("%f,%f", pt.Lat, pt.Lng)
	}
	return strings.Join(coords, ",")
}
