package command

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/roomzin/roomzin-go/types"

	"github.com/roomzin/roomzin-go/internal/protocol"
)

func BuildIncRoomAvlPayload(p types.UpdRoomAvlPayload) ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "INCROOMAVL"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(4)) // four fields

	fields := []struct {
		id   uint16
		typ  byte
		data []byte
	}{
		{0x01, 0x01, []byte(p.PropertyID)},
		{0x02, 0x01, []byte(p.RoomType)},
		{0x03, 0x01, []byte(p.Date)},
		{0x04, 0x02, []byte{p.Amount}},
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

func ParseIncRoomAvlResp(status string, fields []protocol.Field) (uint8, error) {
	if status == "SUCCESS" {
		b := fields[0].Data
		if len(b) != 1 {
			return 0, errors.New("missing or invalid scalar value")
		}
		return b[0], nil
	}
	msgB := fields[0].Data
	return 0, fmt.Errorf("inc room avl error: %s", string(msgB))
}
