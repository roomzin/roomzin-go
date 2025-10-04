package command

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/roomzin/roomzin-go/types"

	"github.com/roomzin/roomzin-go/internal/protocol"
)

func BuildSearchAvailPayload(p types.SearchAvailPayload) ([]byte, error) {
	var buf bytes.Buffer

	// command name
	cmdName := "SEARCHAVAIL"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	type fld struct {
		id   uint16
		typ  byte
		data []byte
	}
	var fields []fld

	// required first
	fields = append(fields,
		fld{0x01, 0x01, []byte(p.Segment)},
		fld{0x02, 0x01, []byte(p.RoomType)},
	)

	// optional helpers
	if v := p.Area; v != nil {
		fields = append(fields, fld{0x03, 0x01, []byte(*v)})
	}
	if v := p.PropertyID; v != nil {
		fields = append(fields, fld{0x04, 0x01, []byte(*v)})
	}
	if v := p.Type; v != nil {
		fields = append(fields, fld{0x05, 0x01, []byte(*v)})
	}
	if v := p.Stars; v != nil {
		fields = append(fields, fld{0x06, 0x02, []byte{*v}})
	}
	if v := p.Category; v != nil {
		fields = append(fields, fld{0x07, 0x01, []byte(*v)})
	}
	if len(p.Amenities) > 0 {
		fields = append(fields, fld{0x08, 0x01, []byte(strings.Join(p.Amenities, ","))})
	}
	if v := p.Longitude; v != nil {
		fields = append(fields, fld{0x09, 0x03, protocol.MakeF64(*v)})
	}
	if v := p.Latitude; v != nil {
		fields = append(fields, fld{0x0A, 0x03, protocol.MakeF64(*v)})
	}
	if len(p.Date) > 0 {
		fields = append(fields, fld{0x0B, 0x01, []byte(strings.Join(p.Date, ","))})
	}
	if v := p.Availability; v != nil {
		fields = append(fields, fld{0x0C, 0x02, []byte{*v}})
	}
	if v := p.FinalPrice; v != nil {
		fields = append(fields, fld{0x0D, 0x03, protocol.MakeU32(*v)})
	}
	if len(p.RateCancel) > 0 {
		fields = append(fields, fld{0x0E, 0x01, []byte(strings.Join(p.RateCancel, ","))})
	}
	if v := p.Limit; v != nil {
		fields = append(fields, fld{0x0F, 0x03, protocol.MakeU64(*v)})
	}

	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(fields)))
	for _, f := range fields {
		idBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(idBytes, f.id) // ← Write 2 bytes for ID
		buf.Write(idBytes)
		buf.WriteByte(f.typ)
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(f.data)))
		buf.Write(f.data)
	}
	return buf.Bytes(), nil
}

func ParseSearchAvailResp(status string, fields []protocol.Field) ([]types.PropertyAvail, error) {
	if status != "SUCCESS" {
		if len(fields) > 0 && fields[0].FieldType == 0x01 {
			return nil, fmt.Errorf("%s", string(fields[0].Data))
		}
		return nil, fmt.Errorf("search failed with status=%s", status)
	}
	// ---- num_days field (must be id=1, type=0x02, len=2) ----
	numDaysField := fields[0]
	if numDaysField.ID != 1 || numDaysField.FieldType != 0x02 || len(numDaysField.Data) != 2 {
		return nil, fmt.Errorf("expected num_days field (id=1, type=0x02, len=2)")
	}
	numDays := binary.LittleEndian.Uint16(numDaysField.Data)

	out := make([]types.PropertyAvail, 0)
	idx := 1 // start after num_days

	// ---- parse properties ----
	for idx < len(fields) {
		f := fields[idx]

		// Property field: type=0x01 (string)
		if f.FieldType != 0x01 {
			return nil, fmt.Errorf("expected property field at index=%d, got type=0x%02x id=%d",
				idx, f.FieldType, f.ID)
		}

		propID := protocol.BytesToPropertyID(f.Data) // no nested length prefix
		idx++

		// Each property must have numDays * 4 fields (date, avail, price, rate)
		expectedDayFields := int(numDays) * 4
		if idx+expectedDayFields > len(fields) {
			return nil, fmt.Errorf("property %q truncated at index=%d: need %d fields, got %d",
				propID, idx, expectedDayFields, len(fields)-idx)
		}

		days := make([]types.DayAvail, 0, numDays)
		for d := 0; d < int(numDays); d++ {
			chunk := fields[idx : idx+4]

			// Validate types
			if chunk[0].FieldType != 0x02 ||
				chunk[1].FieldType != 0x02 ||
				chunk[2].FieldType != 0x03 ||
				chunk[3].FieldType != 0x02 {
				return nil, fmt.Errorf("bad day field types for property=%q at day=%d", propID, d)
			}
			// Validate lengths
			if len(chunk[0].Data) != 2 || len(chunk[1].Data) != 1 ||
				len(chunk[2].Data) != 4 || len(chunk[3].Data) != 1 {
				return nil, fmt.Errorf("bad day field lengths for property=%q at day=%d", propID, d)
			}

			// Parse values
			datePacked := binary.LittleEndian.Uint16(chunk[0].Data)
			dateStr, err := protocol.U16ToDate(datePacked)
			if err != nil {
				return nil, fmt.Errorf("invalid date for property=%q: %w", propID, err)
			}

			days = append(days, types.DayAvail{
				Date:         dateStr,
				Availability: chunk[1].Data[0],
				FinalPrice:   binary.LittleEndian.Uint32(chunk[2].Data),
				RateCancel:   protocol.BitmaskToRateCancelStrings(chunk[3].Data[0]),
			})

			idx += 4
		}

		out = append(out, types.PropertyAvail{
			PropertyID: propID,
			Days:       days,
		})
	}

	// ---- ensure no trailing fields ----
	if idx != len(fields) {
		return nil, fmt.Errorf("extra fields after parsing: consumed=%d total=%d",
			idx, len(fields))
	}

	return out, nil
}
