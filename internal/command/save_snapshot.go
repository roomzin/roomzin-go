package command

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/roomzin/roomzin-go/internal/protocol"
)

func BuildSaveSnapshotPayload() ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "SAVESNAPSHOT"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(0)) // field count = 0
	return buf.Bytes(), nil
}

func ParseSaveSnapshotResp(status string, fields []protocol.Field) error {
	if status == "SUCCESS" {
		return nil
	}
	if len(fields) > 0 && fields[0].FieldType == 0x01 {
		return fmt.Errorf("%s", string(fields[0].Data))
	}
	return fmt.Errorf("")
}
