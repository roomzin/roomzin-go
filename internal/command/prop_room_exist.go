package command

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/roomzin/roomzin-go/types"

	"github.com/roomzin/roomzin-go/internal/protocol"
)

func BuildPropRoomExistPayload(p types.PropRoomExistPayload) ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "PROPROOMEXIST"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(2)) // two fields

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

func ParsePropRoomExistResp(status string, fields []protocol.Field) (bool, error) {
	if status == "SUCCESS" {
		return fields[0].Data[0] == 1, nil
	}
	if len(fields) > 0 && fields[0].FieldType == 0x01 {
		return false, fmt.Errorf("%s", string(fields[0].Data))
	}
	return false, fmt.Errorf("RESPONSE_ERROR")
}
