package protocol

import (
	"bytes"
	"encoding/binary"
)

func BuildLoginPayload(token string) ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "LOGIN"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(1)) // one field

	// Write uint16 field ID (2 bytes) instead of byte (1 byte)
	idBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(idBytes, 0x01) // field id ← 2 bytes
	buf.Write(idBytes)

	buf.WriteByte(0x01) // type string
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(token)))
	buf.WriteString(token)

	return buf.Bytes(), nil
}
