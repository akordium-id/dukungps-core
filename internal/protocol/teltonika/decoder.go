package teltonika

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/protocol"
)

const (
	CodecGH3000 = 0x07
	Codec8      = 0x08
	Codec8Ext   = 0x8E
	Codec12     = 0x0C
	Codec13     = 0x0D
	Codec16     = 0x10
)

var errShortFrame = errors.New("teltonika: frame too short")

type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d *Decoder) Decode(data []byte) (*protocol.DecodeResult, error) {
	if len(data) < 2 {
		return nil, errShortFrame
	}
	if binary.BigEndian.Uint16(data[:2]) > 0 {
		return d.decodeIdentification(data)
	}
	return d.decodeData(data)
}

// FrameLen reports the length of a complete Teltonika frame.
// Identification: u16 imei length + imei. Data: u32 zero prefix + u32 length + payload.
func (d *Decoder) FrameLen(data []byte) (int, error) {
	if len(data) < 2 {
		return 0, errShortFrame
	}
	if binary.BigEndian.Uint16(data[:2]) > 0 {
		length := int(binary.BigEndian.Uint16(data[:2]))
		if len(data) < 2+length {
			return 0, errShortFrame
		}
		return 2 + length, nil
	}
	if len(data) < 8 {
		return 0, errShortFrame
	}
	dataLen := int(binary.BigEndian.Uint32(data[4:8]))
	total := 8 + dataLen
	if len(data) < total {
		return 0, errShortFrame
	}
	return total, nil
}

func (d *Decoder) decodeIdentification(data []byte) (*protocol.DecodeResult, error) {
	length := int(binary.BigEndian.Uint16(data[:2]))
	if len(data) < 2+length {
		return nil, errors.New("teltonika: identification frame truncated")
	}
	imei := string(data[2 : 2+length])

	return &protocol.DecodeResult{
		DeviceID: imei,
		Response: []byte{0x01},
	}, nil
}

func (d *Decoder) decodeData(data []byte) (*protocol.DecodeResult, error) {
	if len(data) < 8 {
		return nil, errors.New("teltonika: data frame too short")
	}
	dataLen := binary.BigEndian.Uint32(data[4:8])
	if len(data) < 8+int(dataLen) {
		return nil, errors.New("teltonika: data frame truncated")
	}

	payload := data[8 : 8+dataLen]
	codec := int(payload[0])
	count := int(payload[1])
	offset := 2

	positions := make([]*position.Position, 0, count)
	for range count {
		pos, next, err := d.decodeLocation(payload, offset, codec)
		if err != nil {
			return nil, err
		}
		offset = next
		positions = append(positions, pos)
	}

	response := make([]byte, 4)
	binary.BigEndian.PutUint32(response, uint32(count))

	return &protocol.DecodeResult{
		Positions: positions,
		Response:  response,
	}, nil
}

func (d *Decoder) decodeLocation(data []byte, offset, codec int) (*position.Position, int, error) {
	pos := &position.Position{
		Time:       time.UnixMilli(int64(binary.BigEndian.Uint64(data[offset:]))),
		Attributes: make(map[string]any),
	}
	offset += 8

	pos.Attributes["priority"] = data[offset]
	offset++

	pos.Longitude = float64(int32(binary.BigEndian.Uint32(data[offset:]))) / 1e7
	offset += 4
	pos.Latitude = float64(int32(binary.BigEndian.Uint32(data[offset:]))) / 1e7
	offset += 4

	pos.Altitude = float64(int16(binary.BigEndian.Uint16(data[offset:])))
	offset += 2
	pos.Course = float64(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	satellites := data[offset]
	offset++
	pos.Attributes["satellites"] = satellites
	pos.Valid = satellites != 0

	pos.Speed = float64(binary.BigEndian.Uint16(data[offset:])) / 1.852
	offset += 2

	eventID, next, err := readExtByte(data, offset, codec, Codec8Ext, Codec16)
	if err != nil {
		return nil, offset, err
	}
	offset = next
	pos.Attributes["event"] = eventID

	if codec == Codec16 {
		offset++
	}

	totalIO, next, err := readExtByte(data, offset, codec, Codec8Ext)
	if err != nil {
		return nil, offset, err
	}
	offset = next
	pos.Attributes["totalIO"] = totalIO

	// 1-byte params
	offset, err = decodeParams(data, offset, codec, pos, 1)
	if err != nil {
		return nil, offset, err
	}
	// 2-byte params
	offset, err = decodeParams(data, offset, codec, pos, 2)
	if err != nil {
		return nil, offset, err
	}
	// 4-byte params
	offset, err = decodeParams(data, offset, codec, pos, 4)
	if err != nil {
		return nil, offset, err
	}
	// 8-byte params (codec 8, 8E, 16)
	if codec == Codec8 || codec == Codec8Ext || codec == Codec16 {
		offset, err = decodeParams(data, offset, codec, pos, 8)
		if err != nil {
			return nil, offset, err
		}
	}

	return pos, offset, nil
}

func decodeParams(data []byte, offset, codec int, pos *position.Position, size int) (int, error) {
	cnt, next, err := readExtByte(data, offset, codec, Codec8Ext)
	if err != nil {
		return offset, err
	}
	offset = next

	for i := 0; i < int(cnt); i++ {
		id, next, err := readExtByte(data, offset, codec, Codec8Ext, Codec16)
		if err != nil {
			return offset, err
		}
		offset = next

		value, next, err := readValue(data, offset, size)
		if err != nil {
			return offset, err
		}
		offset = next

		applyParameter(pos, id, value)
	}

	return offset, nil
}

func readExtByte(data []byte, offset, codec int, extCodecs ...int) (int, int, error) {
	ext := slices.Contains(extCodecs, codec)
	if ext {
		if offset+2 > len(data) {
			return 0, offset, errors.New("teltonika: truncated ext byte")
		}
		return int(binary.BigEndian.Uint16(data[offset : offset+2])), offset + 2, nil
	}
	if offset+1 > len(data) {
		return 0, offset, errors.New("teltonika: truncated byte")
	}
	return int(data[offset]), offset + 1, nil
}

func readValue(data []byte, offset, size int) (uint64, int, error) {
	if offset+size > len(data) {
		return 0, offset, fmt.Errorf("teltonika: truncated value (size %d)", size)
	}
	switch size {
	case 1:
		return uint64(data[offset]), offset + 1, nil
	case 2:
		return uint64(binary.BigEndian.Uint16(data[offset : offset+2])), offset + 2, nil
	case 4:
		return uint64(binary.BigEndian.Uint32(data[offset : offset+4])), offset + 4, nil
	case 8:
		return binary.BigEndian.Uint64(data[offset : offset+8]), offset + 8, nil
	default:
		return 0, offset, fmt.Errorf("teltonika: unsupported value size %d", size)
	}
}

func applyParameter(pos *position.Position, id int, value uint64) {
	switch id {
	case 1: // Digital Output 1 (DOUT1) - commonly wired to Engine cut-off relay
		pos.Attributes[position.AttrBlocked] = value > 0
	case 66:
		pos.Attributes[position.AttrPower] = float64(value) * 0.001
	case 67:
		pos.Attributes[position.AttrBattery] = float64(value) * 0.001
	case 113:
		pos.Attributes["batteryLevel"] = value
	case 181:
		pos.Attributes["pdop"] = float64(value) * 0.1
	case 182:
		pos.Attributes["hdop"] = float64(value) * 0.1
	case 236: // Panic / SOS alarm trigger
		if value > 0 {
			pos.Attributes[position.AttrAlarm] = position.AlarmSOS
		}
	case 239:
		pos.Attributes[position.AttrIgnition] = value > 0
	case 240:
		pos.Attributes[position.AttrMotion] = value > 0
	case 16:
		pos.Attributes["odometer"] = value
	case 21:
		pos.Attributes["rssi"] = value
	default:
		pos.Attributes[fmt.Sprintf("io%d", id)] = value
	}
}
