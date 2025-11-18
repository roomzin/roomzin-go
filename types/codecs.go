package types

import (
	"strings"
)

type Status string

type Codecs struct {
	Amenities   []string `yaml:"amenities"`
	RateCancels []string `yaml:"rate_cancels"`
}

func ValidateAmenities(codecs *Codecs, input []string) (bool, string) {
	var invalid []string
	for _, amenity := range input {
		found := false
		for _, valid := range codecs.Amenities {
			if amenity == valid {
				found = true
				break
			}
		}
		if !found {
			invalid = append(invalid, amenity)
		}
	}
	if len(invalid) > 0 {
		return false, "Invalid amenities: " + strings.Join(invalid, ", ")
	}
	return true, ""
}

func ValidateRateCancels(codecs *Codecs, input []string) (bool, string) {
	var invalid []string
	for _, rate := range input {
		found := false
		for _, valid := range codecs.RateCancels {
			if rate == valid {
				found = true
				break
			}
		}
		if !found {
			invalid = append(invalid, rate)
		}
	}
	if len(invalid) > 0 {
		return false, "Invalid rate cancels: " + strings.Join(invalid, ", ")
	}
	return true, ""
}
