package command

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/roomzin/roomzin-go/internal/protocol"
	"github.com/roomzin/roomzin-go/types"
)

// BuildGetCodecsPayload builds the payload for GETCODECS command
func BuildGetCodecsPayload() ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "GETCODECS"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(0)) // field count = 0

	return buf.Bytes(), nil
}

// ParseGetCodecsResp parses the response for GETCODECS command
func ParseGetCodecsResp(status string, fields []protocol.Field) (*types.Codecs, error) {
	if status != "SUCCESS" {
		if len(fields) > 0 && fields[0].FieldType == 0x01 {
			return nil, fmt.Errorf("%s", string(fields[0].Data))
		}
		return nil, fmt.Errorf("unknown error")
	}

	// GETCODECS response should have exactly 1 field with type 0x09 (YAML/raw bytes)
	if len(fields) != 1 {
		return nil, fmt.Errorf("invalid field count: expected 1 field, got %d", len(fields))
	}

	field := fields[0]
	if field.FieldType != 0x09 {
		return nil, fmt.Errorf("expected YAML field type 0x09, got type %d", field.FieldType)
	}

	// Parse the YAML data into Codecs struct
	codecs, err := parseCodecsFromDelimited(field.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML data: %w", err)
	}

	return codecs, nil
}

func parseCodecsFromDelimited(data []byte) (*types.Codecs, error) {
	parts := strings.Split(string(data), "|")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid codecs format: expected 2 parts, got %d", len(parts))
	}

	amenities := strings.Split(parts[0], ",")
	rateCancels := strings.Split(parts[1], ",")

	// Filter out empty strings from empty lists
	amenities = filterEmpty(amenities)
	rateCancels = filterEmpty(rateCancels)

	return &types.Codecs{
		Amenities:   amenities,
		RateCancels: rateCancels,
	}, nil
}

func filterEmpty(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
