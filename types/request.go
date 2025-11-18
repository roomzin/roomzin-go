package types

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func ValidateDate(date string) (bool, string) {
	var errors []string

	// Check format YYYY-MM-DD
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, date); !matched {
		errors = append(errors, fmt.Sprintf("invalid date format: %s, expected YYYY-MM-DD", date))
	} else {
		// Parse to ensure valid date
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			errors = append(errors, fmt.Sprintf("invalid date: %s", date))
		} else {
			// Check if date is in the past
			today := time.Now().Truncate(24 * time.Hour)
			if parsedDate.Before(today) {
				errors = append(errors, fmt.Sprintf("date %s is in the past", date))
			}
			// Check if date is beyond 365 days
			oneYearFromNow := today.AddDate(1, 0, 0)
			if parsedDate.After(oneYearFromNow) {
				errors = append(errors, fmt.Sprintf("date %s is beyond 365 days from today", date))
			}
		}
	}

	if len(errors) > 0 {
		return false, strings.Join(errors, "; ")
	}
	return true, ""
}

func ValidateDates(dates []string) (bool, string) {
	var errors []string
	for _, date := range dates {
		valid, err := ValidateDate(date)
		if !valid {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return false, "Date errors: " + strings.Join(errors, "; ")
	}
	return true, ""
}

// LoginPayload defines the payload for the AUTH command.
type LoginPayload struct {
	Token string // Static token for authentication (optional)
}

// SetPropPayload defines the payload for adding a new property (ADDPROP command).
type SetPropPayload struct {
	Segment      string
	Area         string
	PropertyID   string
	PropertyType string
	Category     string
	Stars        uint8
	Latitude     float64
	Longitude    float64
	Amenities    []string
}

// PropRoomExistPayload defines the payload for checking if a property has a specific room type (PROPROOMEXIST command).
type PropRoomExistPayload struct {
	PropertyID string
	RoomType   string
}

// Verify validates the PropRoomExistPayload.
func (p PropRoomExistPayload) Verify() (bool, string) {
	return true, ""
}

// DelPropRoomPayload defines the payload for deleting a room type from a property (DELPROPROOM command).
type DelPropRoomPayload struct {
	PropertyID string
	RoomType   string
}

// Verify validates the DelPropRoomPayload.
func (p DelPropRoomPayload) Verify() (bool, string) {
	return true, ""
}

// PropRoomDateListPayload defines the payload for listing dates with availability for a room type (PROPROOMDATELIST command).
type PropRoomDateListPayload struct {
	PropertyID string
	RoomType   string
}

// Verify validates the PropRoomDateListPayload.
func (p PropRoomDateListPayload) Verify() (bool, string) {
	return true, ""
}

// DelRoomDayRequest defines the payload for deleting a room’s data for a specific date (DELROOMDAY command).
type DelRoomDayRequest struct {
	PropertyID string
	RoomType   string
	Date       string // YYYY-MM-DD
}

// Verify validates the DelRoomDayRequest.
func (p DelRoomDayRequest) Verify() (bool, string) {
	return ValidateDate(p.Date)
}

// UpdRoomAvlPayload defines the payload for updating room availability (INCROOMAVL, DECROOMAVL, SETROOMAVL commands).
type UpdRoomAvlPayload struct {
	PropertyID string
	RoomType   string
	Date       string // YYYY-MM-DD
	Amount     uint8
}

// Verify validates the UpdRoomAvlPayload.
func (p UpdRoomAvlPayload) Verify() (bool, string) {
	return ValidateDate(p.Date)
}

// SetRoomPkgPayload defines the payload for setting room availability, pricing, and cancellation policy (SETROOMPKG command).
type SetRoomPkgPayload struct {
	PropertyID   string
	RoomType     string
	Date         string // YYYY-MM-DD
	Availability *uint8
	FinalPrice   *uint32
	RateCancel   []string // Optional; empty slice if not provided
}

// Verify validates the SetRoomPkgPayload.
func (p SetRoomPkgPayload) Verify(codecs *Codecs) (bool, string) {
	validDate, dateErr := ValidateDate(p.Date)
	validRateCancel, rateCancelErr := ValidateRateCancels(codecs, p.RateCancel)

	var errors []string
	if !validDate {
		errors = append(errors, dateErr)
	}
	if !validRateCancel {
		errors = append(errors, rateCancelErr)
	}

	if len(errors) > 0 {
		return false, strings.Join(errors, "; ")
	}
	return true, ""
}

// GetRoomDayRequest defines the payload for retrieving room details for a specific date (GETPROPROOMDAY command).
type GetRoomDayRequest struct {
	PropertyID string
	RoomType   string
	Date       string // YYYY-MM-DD
}

// Verify validates the GetRoomDayRequest.
func (p GetRoomDayRequest) Verify() (bool, string) {
	return ValidateDate(p.Date)
}

type SearchPropPayload struct {
	Segment   string
	Area      *string
	Type      *string
	Stars     *uint8
	Category  *string
	Amenities *[]string
	Longitude *float64
	Latitude  *float64
	Limit     *uint64
}

// Verify validates the SearchPropPayload.
func (p SearchPropPayload) Verify(codecs *Codecs) (bool, string) {
	if p.Amenities != nil {
		return ValidateAmenities(codecs, *p.Amenities)
	}
	return true, ""
}

type SearchAvailPayload struct {
	Segment      string
	RoomType     string
	Area         *string
	PropertyID   *string
	Type         *string
	Stars        *uint8
	Category     *string
	Amenities    []string
	Longitude    *float64
	Latitude     *float64
	Date         []string
	Availability *uint8
	FinalPrice   *uint32
	RateCancel   []string
	Limit        *uint64
}

// DelPropDayRequest defines the payload for deleting all room data for a property on a specific date (DELPROPDAY command).
type DelPropDayRequest struct {
	PropertyID string
	Date       string // YYYY-MM-DD
}

// Verify validates the DelPropDayRequest.
func (p DelPropDayRequest) Verify() (bool, string) {
	return ValidateDate(p.Date)
}
