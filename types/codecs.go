package types

import (
	"errors"
	"slices"
	"strings"
)

type Status string

type Codecs struct {
	RateCancels []string `yaml:"rate_cancels"`
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
