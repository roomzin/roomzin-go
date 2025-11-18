package command

import (
	"bytes"
	"encoding/binary"

	"github.com/roomzin/roomzin-go/types"

	"github.com/roomzin/roomzin-go/internal/protocol"

	"errors"
)

func BuildGetPropRoomDayPayload(p types.GetRoomDayRequest) ([]byte, error) {
	var buf bytes.Buffer

	cmdName := "GETPROPROOMDAY"
	buf.WriteByte(byte(len(cmdName)))
	buf.WriteString(cmdName)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(3))

	fields := []struct {
		id   uint16
		typ  byte
		data []byte
	}{
		{0x01, 0x01, []byte(p.PropertyID)},
		{0x02, 0x01, []byte(p.RoomType)},
		{0x03, 0x01, []byte(p.Date)},
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

func ParseGetPropRoomDayResp(codecs *types.Codecs, status string, fields []protocol.Field) (types.GetRoomDayResult, error) {
	var res types.GetRoomDayResult
	if status == "SUCCESS" {
		chunk := fields[:5]
		return types.GetRoomDayResult{
			PropertyID:   string(chunk[0].Data),
			Date:         string(chunk[1].Data),
			Availability: chunk[2].Data[0],
			FinalPrice:   binary.LittleEndian.Uint32(chunk[3].Data),
			RateCancel:   protocol.BitmaskToRateCancelStrings(codecs, chunk[4].Data[0]),
		}, nil
	}
	if len(fields) > 0 && fields[0].FieldType == 0x01 {
		return res, errors.New(string(fields[0].Data))
	}
	return res, errors.New("")
}
