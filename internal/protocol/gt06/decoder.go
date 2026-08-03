package gt06

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/protocol"
)

const (
	msgLogin     = 0x01
	msgGPS       = 0x10
	msgGPSLBS1   = 0x12
	msgStatus    = 0x13
	msgAlarm     = 0x16
	msgHeartbeat = 0x23

	flagLatPositive = 0x0400
	flagLonNegative = 0x0800
	flagValid       = 0x1000
)

var (
	errShortFrame = errors.New("gt06: frame too short")
	errBadCRC     = errors.New("gt06: checksum mismatch")
	errBadFooter  = errors.New("gt06: missing frame footer")
)

type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d *Decoder) Decode(data []byte) (*protocol.DecodeResult, error) {
	if len(data) < 10 {
		return nil, errShortFrame
	}

	extended := false
	var lenField int
	switch {
	case data[0] == 0x78 && data[1] == 0x78:
		lenField = int(data[2])
	case data[0] == 0x79 && data[1] == 0x79:
		extended = true
		lenField = int(binary.BigEndian.Uint16(data[2:4]))
	default:
		return nil, errors.New("gt06: invalid start marker")
	}

	// LEN counts type + payload + serial + crc (everything after LEN field, excluding 0D 0A)
	total := lenField + 5
	if extended {
		total = lenField + 6
	}
	if len(data) < total {
		return nil, errShortFrame
	}
	if data[total-2] != 0x0D || data[total-1] != 0x0A {
		return nil, errBadFooter
	}

	if !verifyCRC(data[:total]) {
		return nil, errBadCRC
	}

	msgTypeOffset := 3
	if extended {
		msgTypeOffset = 4
	}
	msgType := data[msgTypeOffset]
	payloadEnd := total - 6 // exclude serial + crc
	payload := data[msgTypeOffset+1 : payloadEnd]
	serial := binary.BigEndian.Uint16(data[total-6 : total-4])

	switch msgType {
	case msgLogin:
		return d.decodeLogin(payload, serial)
	case msgGPS, msgGPSLBS1, msgAlarm:
		return d.decodeLocation(payload, serial)
	case msgStatus, msgHeartbeat:
		return &protocol.DecodeResult{}, nil
	default:
		// command and other types: no positions, no ack
		return &protocol.DecodeResult{}, nil
	}
}

func verifyCRC(frame []byte) bool {
	return binary.BigEndian.Uint16(frame[len(frame)-4:len(frame)-2]) == crcX25(frame[2:len(frame)-4])
}

func crcX25(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0x8408
			} else {
				crc >>= 1
			}
		}
	}
	return crc ^ 0xFFFF
}

func (d *Decoder) decodeLogin(payload []byte, serial uint16) (*protocol.DecodeResult, error) {
	if len(payload) < 8 {
		return nil, errors.New("gt06: login payload too short")
	}
	// 8 bytes -> 16 hex chars -> drop first char -> 15-digit IMEI
	hexStr := hex.EncodeToString(payload[:8])
	imei := hexStr[1:]

	response := buildAck(msgLogin, nil, serial)
	return &protocol.DecodeResult{
		DeviceID: imei,
		Response: response,
	}, nil
}

func (d *Decoder) decodeLocation(payload []byte, serial uint16) (*protocol.DecodeResult, error) {
	pos := &position.Position{
		Time:       time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Valid:      false,
		Attributes: make(map[string]any),
	}

	// GPS section: datetime(6) sat(1) lat(4) lon(4) speed(1) flags(2)
	if len(payload) >= 18 {
		year := int(payload[0]) + 2000
		pos.Time = time.Date(
			year, time.Month(payload[1]), int(payload[2]),
			int(payload[3]), int(payload[4]), int(payload[5]), 0, time.UTC)

		pos.Attributes["satellites"] = payload[6] & 0x0F

		latRaw := binary.BigEndian.Uint32(payload[7:11])
		lonRaw := binary.BigEndian.Uint32(payload[11:15])
		pos.Latitude = float64(latRaw) / 1800000
		pos.Longitude = float64(lonRaw) / 1800000

		pos.Speed = float64(payload[15]) / 1.852

		flags := binary.BigEndian.Uint16(payload[16:18])
		pos.Course = float64(flags & 0x03FF)
		pos.Valid = flags&flagValid != 0
		if flags&flagLatPositive == 0 {
			pos.Latitude = -pos.Latitude
		}
		if flags&flagLonNegative != 0 {
			pos.Longitude = -pos.Longitude
		}
	}

	response := buildAck(msgGPSLBS1, nil, serial)
	return &protocol.DecodeResult{
		Positions: []*position.Position{pos},
		Response:  response,
	}, nil
}

func buildAck(msgType byte, content []byte, serial uint16) []byte {
	lenField := byte(5 + len(content)) // type + content + serial + crc
	frame := []byte{0x78, 0x78, lenField, msgType}
	frame = append(frame, content...)
	frame = binary.BigEndian.AppendUint16(frame, serial)
	crc := crcX25(frame[2:])
	frame = binary.BigEndian.AppendUint16(frame, crc)
	return append(frame, 0x0D, 0x0A)
}
