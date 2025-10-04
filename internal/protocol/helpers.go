package protocol

import (
	"encoding/binary"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/roomzin/roomzin-go/types"
)

// bitmaskToRateCancelStrings converts 8-bit mask → []string (same logic as Rust)
func BitmaskToRateCancelStrings(mask uint8) []string {
	out := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		if mask&(1<<i) != 0 {
			out = append(out, types.RateCancels[i])
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

func BytesToPropertyID(data []byte) string {
	// 1. Too short → return empty
	if len(data) < 7 {
		return ""
	}

	// 2. Short string marker
	if data[6] == 0xF0 {
		// Left segment: 0..5
		leftLen := 0
		for i := 0; i < 6; i++ {
			if i >= len(data) || data[i] == 0 {
				break
			}
			leftLen++
		}

		// Right segment: 7..15
		rightLen := 0
		for i := 7; i < len(data); i++ {
			if data[i] == 0 {
				break
			}
			rightLen++
		}

		result := make([]byte, leftLen+rightLen)
		copy(result[:leftLen], data[:leftLen])
		copy(result[leftLen:], data[7:7+rightLen])
		return string(result)
	}

	// 3. UUID detection (valid version)
	version := (data[6] & 0xF0) >> 4
	switch version {
	case 1, 2, 3, 4, 5, 7:
		var uuidBytes [16]byte
		if len(data) >= 16 {
			copy(uuidBytes[:], data[:16])
		} else {
			copy(uuidBytes[:], data) // pad remaining with zeros
		}

		if u, err := uuid.FromBytes(uuidBytes[:]); err == nil {
			return u.String()
		}
	}

	// This should never happen with proper server data
	return ""
}
