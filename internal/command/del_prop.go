package command

import (
	"bytes"
	"encoding/binary"

	"github.com/roomzin/roomzin-go/internal/protocol"

	"errors"
)

func BuildDelPropPayload(propertyID string) ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "DELPROP"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))

	idBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(idBytes, 0x01)
	buf.Write(idBytes)
	buf.WriteByte(0x01)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(propertyID)))
	buf.WriteString(propertyID)

	return buf.Bytes(), nil
}

func ParseDelPropResp(status string, fields []protocol.Field) error {
	if status == "SUCCESS" {
		return nil
	}
	if len(fields) > 0 && fields[0].FieldType == 0x01 {
		return errors.New(string(fields[0].Data))
	}
	return errors.New("")
}
