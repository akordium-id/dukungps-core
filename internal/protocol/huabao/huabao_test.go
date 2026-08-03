package huabao

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checksum(body []byte) byte {
	var x byte
	for _, b := range body {
		x ^= b
	}
	return x
}

func buildFrame(t *testing.T, msgType uint16, deviceID []byte, data []byte) []byte {
	t.Helper()
	// index: 2 bytes u16 for non 0x5501/0x5502 types
	body := make([]byte, 0, 2+2+len(deviceID)+2+len(data)+1)
	body = binary.BigEndian.AppendUint16(body, msgType)
	body = binary.BigEndian.AppendUint16(body, uint16(len(data))) // attribute = data length
	body = append(body, deviceID...)
	body = binary.BigEndian.AppendUint16(body, 1) // index
	body = append(body, data...)
	cs := checksum(body)
	body = append(body, cs)

	frame := []byte{0x7E}
	frame = append(frame, body...)
	frame = append(frame, 0x7E)
	return frame
}

func TestDecodeTerminalAuth(t *testing.T) {
	// ASCII digit device ID
	deviceID := []byte("123456")
	frame := buildFrame(t, 0x0102, deviceID, nil)

	d := NewDecoder()
	res, err := d.Decode(frame)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "123456", res.DeviceID)
	assert.Empty(t, res.Positions)
	// ack 0x8001: [7E][00 80 01][attr][id][index 2][...][crc][7E]
	require.NotEmpty(t, res.Response)
	assert.Equal(t, byte(0x7E), res.Response[0])
	assert.Equal(t, byte(0x7E), res.Response[len(res.Response)-1])
	assert.Equal(t, []byte{0x80, 0x01}, res.Response[1:3])
}

func TestDecodeLocation(t *testing.T) {
	deviceID := []byte("123456")

	// 0x0200 body: alarm u32, status i32, lat u32*1e-6, lon u32*1e-6,
	// alt i16, speed u16*0.1, course u16, time BCD YYMMDDHHMMSS
	data := make([]byte, 0, 28)
	data = binary.BigEndian.AppendUint32(data, 0)                 // alarm
	data = binary.BigEndian.AppendUint32(data, 0b1110)            // status: bit1 valid, bit2 lat neg, bit3 lon neg
	data = binary.BigEndian.AppendUint32(data, 7245600)           // lat raw 7.245600
	data = binary.BigEndian.AppendUint32(data, 112738900)         // lon raw 112.738900
	data = binary.BigEndian.AppendUint16(data, 0x0000)            // altitude placeholder (i16)
	data = binary.BigEndian.AppendUint16(data, 0)                 // speed 0
	data = binary.BigEndian.AppendUint16(data, 180)               // course 180
	data = append(data, 0x26, 0x08, 0x03, 0x12, 0x34, 0x56)       // time 2026-08-03 12:34:56 BCD

	frame := buildFrame(t, 0x0200, deviceID, data)

	d := NewDecoder()
	res, err := d.Decode(frame)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Positions, 1)

	pos := res.Positions[0]
	assert.True(t, pos.Valid)
	assert.InDelta(t, -7.2456, pos.Latitude, 0.000001)
	assert.InDelta(t, -112.7389, pos.Longitude, 0.000001)
	assert.Equal(t, float64(180), pos.Course)
	// ack present
	assert.NotEmpty(t, res.Response)
}
