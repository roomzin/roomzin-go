package command

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"

	"github.com/roomzin/roomzin-go/types"

	"github.com/roomzin/roomzin-go/internal/protocol"
)

func BuildSetRoomPkgPayload(p types.SetRoomPkgPayload) ([]byte, error) {
	if p.PropertyID == "" || p.RoomType == "" || p.Date == "" {
		return nil, errors.New("missing required fields")
	}

	var buf bytes.Buffer

	// command name
	cmdName := "SETROOMPKG"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	// fields
	type fld struct {
		id   uint16
		typ  byte
		data []byte
	}
	fields := []fld{
		{0x01, 0x01, []byte(p.PropertyID)},
		{0x02, 0x01, []byte(p.RoomType)},
		{0x03, 0x01, []byte(p.Date)},
	}
	if p.Availability != nil {
		fields = append(fields, fld{0x04, 0x02, []byte{byte(*p.Availability)}})
	}
	if p.FinalPrice != nil {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(*p.FinalPrice))
		fields = append(fields, fld{0x05, 0x03, b})
	}
	if p.RateCancel != nil {
		fields = append(fields, fld{0x06, 0x01, []byte(strings.Join(p.RateCancel, ","))})
	}

	// field count
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(fields)))

	// fields
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

func ParseSetRoomPkgResp(status string, fields []protocol.Field) error {
	if status == "SUCCESS" {
		return nil
	}
	if len(fields) > 0 && fields[0].FieldType == 0x01 {
		return errors.New(string(fields[0].Data))
	}
	return errors.New("RESPONSE_ERROR")
}
