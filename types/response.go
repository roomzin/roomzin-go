package types

// GetRoomDayResult defines the result for retrieving room details for a specific date (GETPROPROOMDAY command).
type GetRoomDayResult struct {
	PropertyID   string
	Date         string
	Availability uint8
	FinalPrice   uint32
	RateCancel   []string
}

// DayAvail one day inside a property.
type DayAvail struct {
	Date         string
	Availability uint8
	FinalPrice   uint32
	RateCancel   []string
}

// PropertyAvail one property + all its days.
type PropertyAvail struct {
	PropertyID string
	Days       []DayAvail
}

type SegmentInfo struct {
	Segment   string
	PropCount uint32
}
