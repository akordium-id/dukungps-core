package huabao

import (
	"encoding/binary"
	"errors"
	"strconv"
	"time"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/protocol"
)

const (
	msgTerminalAuth     = 0x0102
	msgTerminalRegister = 0x0100
	msgHeartbeat        = 0x0002
	msgLocationReport   = 0x0200
	msgLocationReport2  = 0x5501
	msgLocationBlind    = 0x5502
	msgTimeSyncRequest  = 0x0109
	msgLocationBatch    = 0x0704
	msgLocationBatch2   = 0x0210
	msgTransparent      = 0x0900
	msgAcceleration     = 0x2070
	msgReportText       = 0x6006
	msgCommandResponse  = 0x0701
	msgDriverIdentity   = 0x0702
	msgPhoto            = 0x8888
)

var (
	errNoFrame   = errors.New("huabao: no complete frame")
	errBadCRC    = errors.New("huabao: checksum mismatch")
	errShortData = errors.New("huabao: data too short")
)

type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d *Decoder) Decode(data []byte) (*protocol.DecodeResult, error) {
	if len(data) < 4 || data[0] != 0x7E {
		return nil, errNoFrame
	}
	// find closing 0x7E, unescape 0x7D 0x01 -> 0x7D, 0x7D 0x02 -> 0x7E
	frame, err := unescapeFrame(data)
	if err != nil {
		return nil, err
	}

	// header: delimiter(1) type(2) attribute(2) id(6)
	if len(frame) < 11 {
		return nil, errShortData
	}

	msgType := binary.BigEndian.Uint16(frame[1:3])
	// attribute = data length (not used for framing)
	deviceID, err := decodeDeviceID(frame[5:11])
	if err != nil {
		return nil, err
	}

	// index: 1 byte for 0x5501/0x5502, else 2 bytes
	indexSize := 2
	if msgType == msgLocationReport2 || msgType == msgLocationBlind {
		indexSize = 1
	}
	bodyOffset := 5 + 6 + indexSize
	if len(frame) <= bodyOffset+1 {
		return nil, errShortData
	}
	// last byte before closing 0x7E is checksum
	if !verifyChecksum(frame, bodyOffset) {
		return nil, errBadCRC
	}
	body := frame[bodyOffset : len(frame)-2] // exclude checksum + closing 0x7E

	switch msgType {
	case msgTerminalAuth, msgTerminalRegister, msgHeartbeat, msgPhoto, msgTimeSyncRequest:
		return &protocol.DecodeResult{
			DeviceID: deviceID,
			Response: buildGeneralResponse([]byte(deviceID), indexBytes(frame, msgType, indexSize, bodyOffset), msgType),
		}, nil
	case msgLocationReport:
		pos, err := decodeLocation(body)
		if err != nil {
			return nil, err
		}
		return &protocol.DecodeResult{
			DeviceID:  deviceID,
			Positions: []*position.Position{pos},
			Response:  buildGeneralResponse([]byte(deviceID), indexBytes(frame, msgType, indexSize, bodyOffset), msgType),
		}, nil
	default:
		return &protocol.DecodeResult{
			DeviceID: deviceID,
			Response: buildGeneralResponse([]byte(deviceID), indexBytes(frame, msgType, indexSize, bodyOffset), msgType),
		}, nil
	}
}

func indexBytes(frame []byte, msgType uint16, indexSize, bodyOffset int) []byte {
	if msgType == msgLocationReport2 || msgType == msgLocationBlind {
		return frame[bodyOffset-1 : bodyOffset]
	}
	return frame[bodyOffset-2 : bodyOffset]
}

// FrameLen reports the length of a complete Huabao frame by scanning for the
// closing 0x7E, skipping 0x7D-escaped bytes (0x7D 0x01 -> 0x7D, 0x7D 0x02 -> 0x7E).
func (d *Decoder) FrameLen(data []byte) (int, error) {
	if len(data) == 0 || data[0] != 0x7E {
		return 0, errNoFrame
	}
	for i := 1; i < len(data); i++ {
		if data[i] == 0x7D && i+1 < len(data) {
			i++
			continue
		}
		if data[i] == 0x7E {
			return i + 1, nil
		}
	}
	return 0, errNoFrame
}

func unescapeFrame(data []byte) ([]byte, error) {
	frame := make([]byte, 0, len(data))
	frame = append(frame, data[0])
	for i := 1; i < len(data); i++ {
		switch data[i] {
		case 0x7E:
			frame = append(frame, 0x7E)
			return frame, nil
		case 0x7D:
			if i+1 < len(data) {
				switch data[i+1] {
				case 0x01:
					frame = append(frame, 0x7D)
				case 0x02:
					frame = append(frame, 0x7E)
				}
				i++
			}
		default:
			frame = append(frame, data[i])
		}
	}
	return nil, errNoFrame
}

func verifyChecksum(frame []byte, bodyOffset int) bool {
	var x byte
	for _, b := range frame[1 : len(frame)-2] {
		x ^= b
	}
	return x == frame[len(frame)-2]
}

func decodeDeviceID(id []byte) (string, error) {
	allDigits := true
	for _, b := range id {
		if b < '0' || b > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return string(id), nil
	}

	// binary id: (u16 BE << 32) | u32 BE -> 14-digit number + Luhn check digit
	num := uint64(binary.BigEndian.Uint16(id[:2]))<<32 | uint64(binary.BigEndian.Uint32(id[2:6]))
	digits := strconv.FormatUint(num, 10)
	check := luhnCheckDigit(digits)
	return digits + check, nil
}

func luhnCheckDigit(digits string) string {
	sum := 0
	double := true
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return strconv.Itoa((10 - sum%10) % 10)
}

func decodeLocation(body []byte) (*position.Position, error) {
	if len(body) < 26 {
		return nil, errShortData
	}

	alarm := binary.BigEndian.Uint32(body[0:4])
	status := binary.BigEndian.Uint32(body[4:8])
	latRaw := binary.BigEndian.Uint32(body[8:12])
	lonRaw := binary.BigEndian.Uint32(body[12:16])

	pos := &position.Position{
		Latitude:   float64(latRaw) * 0.000001,
		Longitude:  float64(lonRaw) * 0.000001,
		Altitude:   float64(int16(binary.BigEndian.Uint16(body[16:18]))),
		Speed:      float64(binary.BigEndian.Uint16(body[18:20])) * 0.1 / 1.852,
		Course:     float64(binary.BigEndian.Uint16(body[20:22])),
		Valid:      status&0b10 != 0,
		Attributes: make(map[string]any),
	}
	if status&0b100 != 0 {
		pos.Latitude = -pos.Latitude
	}
	if status&0b1000 != 0 {
		pos.Longitude = -pos.Longitude
	}

	// Status Bitmask (JT/T 808 standard)
	// Bit 0: ACC (0 = OFF, 1 = ON)
	pos.Attributes[position.AttrIgnition] = (status & 0b1) != 0
	// Bit 10: Oil / Electricity circuit status (1 = disconnected / cut-engine active)
	if (status & (1 << 10)) != 0 {
		pos.Attributes[position.AttrBlocked] = true
	}
	if pos.Speed > 0 {
		pos.Attributes[position.AttrMotion] = true
	}

	// Alarm Bitmask (JT/T 808 standard)
	// Bit 0: Emergency SOS alarm
	if (alarm & 0b1) != 0 {
		pos.Attributes[position.AttrAlarm] = position.AlarmSOS
	} else if (alarm & 0b10) != 0 { // Bit 1: Overspeed alarm
		pos.Attributes[position.AttrAlarm] = position.AlarmOverspeed
	} else if (alarm & (1 << 8)) != 0 { // Bit 8: Main power cutoff
		pos.Attributes[position.AttrAlarm] = position.AlarmPowerCut
	}

	pos.Time = decodeBCDTime(body[22:28])
	return pos, nil
}

func decodeBCDTime(b []byte) time.Time {
	year := int(bcd(b[0])) + 2000
	return time.Date(
		year, time.Month(bcd(b[1])), int(bcd(b[2])),
		int(bcd(b[3])), int(bcd(b[4])), int(bcd(b[5])), 0, time.UTC)
}

func bcd(b byte) int {
	return int(b>>4)*10 + int(b&0x0F)
}

func buildGeneralResponse(deviceID, index []byte, msgType uint16) []byte {
	body := make([]byte, 0, 2+2+6+2+2+2+1+1)
	body = binary.BigEndian.AppendUint16(body, 0x8001)
	body = binary.BigEndian.AppendUint16(body, 5) // data length: index2 + type2 + result1
	body = append(body, deviceID...)
	body = binary.BigEndian.AppendUint16(body, 1) // index
	body = binary.BigEndian.AppendUint16(body, indexToU16(index))
	body = binary.BigEndian.AppendUint16(body, msgType)
	body = append(body, 0x00) // result
	cs := xorChecksum(body)
	body = append(body, cs)

	frame := []byte{0x7E}
	frame = append(frame, body...)
	frame = append(frame, 0x7E)
	return frame
}

func indexToU16(b []byte) uint16 {
	if len(b) == 1 {
		return uint16(b[0])
	}
	return binary.BigEndian.Uint16(b)
}

func xorChecksum(body []byte) byte {
	var x byte
	for _, b := range body {
		x ^= b
	}
	return x
}
