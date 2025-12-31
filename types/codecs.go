package types

import (
	"errors"
	"slices"
	"strings"
)

type Status string

type Codecs struct {
	RateFeatures []string `yaml:"rate_features"`
}

func ValidateRateFeatures(codecs *Codecs, input []string) error {
	var invalid []string
	for _, rate := range input {
		if !slices.Contains(codecs.RateFeatures, rate) {
			invalid = append(invalid, rate)
		}
	}
	if len(invalid) > 0 {
		return errors.New("Invalid rate features: " + strings.Join(invalid, ", "))
	}
	return nil
}
