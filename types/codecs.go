package types

import (
	"errors"
	"strings"
)

type Status string

type Codecs struct {
	Amenities   []string `yaml:"amenities"`
	RateCancels []string `yaml:"rate_cancels"`
}

func ValidateAmenities(codecs *Codecs, input []string) error {
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
		return errors.New("Invalid amenities: " + strings.Join(invalid, ", "))
	}
	return nil
}

func ValidateRateCancels(codecs *Codecs, input []string) error {
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
		return errors.New("Invalid rate cancels: " + strings.Join(invalid, ", "))
	}
	return nil
}
