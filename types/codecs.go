package types

import (
	"errors"
	"slices"
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
		if !slices.Contains(codecs.Amenities, amenity) {
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
		if !slices.Contains(codecs.RateCancels, rate) {
			invalid = append(invalid, rate)
		}
	}
	if len(invalid) > 0 {
		return errors.New("Invalid rate cancels: " + strings.Join(invalid, ", "))
	}
	return nil
}
