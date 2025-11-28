package command

import (
	"bytes"
	"encoding/binary"

	"github.com/roomzin/roomzin-go/types"

	"github.com/roomzin/roomzin-go/internal/protocol"

	"errors"
)

func BuildDelPropRoomPayload(p types.DelPropRoomPayload) ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "DELPROPROOM"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))

	fields := []struct {
		id   uint16
		typ  byte
		data []byte
	}{
		{0x01, 0x01, []byte(p.PropertyID)},
		{0x02, 0x01, []byte(p.RoomType)},
	}
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

func ParseDelPropRoomResp(status string, fields []protocol.Field) error {
	if status == "SUCCESS" {
		return nil
	}
	if len(fields) > 0 && fields[0].FieldType == 0x01 {
		return errors.New(string(fields[0].Data))
	}
	return errors.New("RESPONSE_ERROR")
}
