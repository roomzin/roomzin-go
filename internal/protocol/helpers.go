package protocol

import (
	"encoding/binary"
	"errors"
	"math"
	"time"
)

var rateCancels = []string{
	"free_cancellation",
	"non_refundable",
	"pay_at_property",
	"includes_breakfast",
	"free_wifi",
	"no_prepayment",
	"partial_refund",
	"instant_confirmation",
}

// bitmaskToRateCancelStrings converts 8-bit mask → []string (same logic as Rust)
func BitmaskToRateCancelStrings(mask uint8) []string {
	out := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		if mask&(1<<i) != 0 {
			out = append(out, rateCancels[i])
		}
	}
	return out
}

// u16ToDate unpacks the 16-bit packed date (same bit layout as Rust)
func U16ToDate(packed uint16) (string, error) {
	yearOffset := int((packed >> 9) & 0b111)
	month := int((packed>>5)&0b1111) + 1
	day := int(packed&0b11111) + 1

	// use current year as base (like Rust)
	baseYear := time.Now().Year()
	t := time.Date(baseYear+yearOffset, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Month() != time.Month(month) || t.Day() != day {
		return "", errors.New("invalid packed date")
	}
	return t.Format("2006-01-02"), nil
}

func MakeF64(v float64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, math.Float64bits(v))
	return b
}
func MakeU64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func MakeU32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}
