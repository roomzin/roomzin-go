package types

import "strings"

type Status string

const (
	StatusSuccess Status = "SUCCESS"
	StatusError   Status = "ERROR"
)

var acceptableAmenities = [...]string{
	"wifi",
	"pool",
	"gym",
	"parking",
	"breakfast",
	"spa",
	"pet_friendly",
	"bar",
	"restaurant",
	"air_conditioner",
	"kitchen",
	"laundry",
	"shuttle",
	"family_rooms",
	"ev_charging",
	"beach_access",
}

func ValidateAmenities(input []string) (bool, string) {
	var invalid []string
	for _, amenity := range input {
		found := false
		for _, valid := range acceptableAmenities {
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

func GetAcceptableAmenities() []string {
	return acceptableAmenities[:]
}


var RateCancels = []string{
	"free_cancellation",
	"non_refundable",
	"pay_at_property",
	"includes_breakfast",
	"free_wifi",
	"no_prepayment",
	"partial_refund",
	"instant_confirmation",
}


func ValidateRateCancels(input []string) (bool, string) {
	var invalid []string
	for _, rate := range input {
		found := false
		for _, valid := range RateCancels {
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

func GetRateCancels() []string {
	return RateCancels[:]
}
