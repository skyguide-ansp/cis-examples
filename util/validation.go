package util

import "fmt"

func ValidateLatitude(lat float64) bool {
	return lat >= -90.0 && lat <= 90.0
}

func ValidateLongitude(lng float64) bool {
	return lng >= -180.0 && lng <= 180.0
}

func ValidateLatitudes(latitudes ...float64) bool {
	for _, lat := range latitudes {
		if !ValidateLatitude(lat) {
			return false
		}
	}
	return true
}

func ValidateLongitudes(longitudes ...float64) bool {
	for _, lng := range longitudes {
		if !ValidateLongitude(lng) {
			return false
		}
	}
	return true
}

func ValidateView(view string) bool {
	var lng1, lng2, lat1, lat2 float64
	_, err := fmt.Sscanf(view, "%f,%f,%f,%f", &lat1, &lng1, &lat2, &lng2)
	if err != nil {
		return false
	}
	if !ValidateLongitudes(lng1, lng2) || !ValidateLatitudes(lat1, lat2) {
		return false
	}
	return true
}
