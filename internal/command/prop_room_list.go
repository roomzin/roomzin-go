package command

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/roomzin/roomzin-go/internal/protocol"
)

func BuildPropRoomListPayload(propertyID string) ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "PROPROOMLIST"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(1)) // one field

	// single field: id 0x01, type 0x01, value = propertyID
	idBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(idBytes, 0x01)
	buf.Write(idBytes)
	buf.WriteByte(0x01)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(propertyID)))
	buf.WriteString(propertyID)

	return buf.Bytes(), nil
}

func ParsePropRoomListResp(status string, fields []protocol.Field) ([]string, error) {
	if status != "SUCCESS" {
		if len(fields) > 0 && fields[0].FieldType == 0x01 {
			return nil, fmt.Errorf("%s", string(fields[0].Data))
		}
		return nil, fmt.Errorf("")
	}
	list := make([]string, 0, len(fields))
	for i := range fields {
		list = append(list, string(fields[i].Data))
	}
	return list, nil
}
