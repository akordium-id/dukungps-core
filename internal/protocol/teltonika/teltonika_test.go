package teltonika

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/command"
)

func TestEncodeEngineStop(t *testing.T) {
	e := NewEncoder()
	out, err := e.Encode(command.Command{Type: command.TypeEngineStop})
	require.NoError(t, err)
	assert.Equal(t, byte(Codec12), out[8])
	assert.True(t, bytes.Contains(out, []byte("setdigout 1")))
}

func TestEncodeEngineResume(t *testing.T) {
	e := NewEncoder()
	out, err := e.Encode(command.Command{Type: command.TypeEngineResume})
	require.NoError(t, err)
	assert.True(t, bytes.Contains(out, []byte("setdigout 0")))
}

func TestEncodeCustom(t *testing.T) {
	e := NewEncoder()
	out, err := e.Encode(command.Command{Type: command.TypeCustom, Data: "hello"})
	require.NoError(t, err)
	assert.True(t, bytes.Contains(out, []byte("hello")))
}

func TestEncodeUnsupported(t *testing.T) {
	e := NewEncoder()
	_, err := e.Encode(command.Command{Type: "rebootDevice"})
	assert.Error(t, err)
}

func TestDecodeIdentification(t *testing.T) {
	// IMEI 15 digit ASCII, prefixed by u16 length
	data := []byte{
		0x00, 0x0F,
		'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '1', '2', '3', '4', '5',
	}

	d := NewDecoder()
	res, err := d.Decode(data)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "123456789012345", res.DeviceID)
	assert.Equal(t, []byte{0x01}, res.Response)
	assert.Empty(t, res.Positions)
}

func TestDecodeCodec8(t *testing.T) {
	// Codec 8, 1 record, captured from Traccar test fixture:
	// timestamp=1510000000000ms priority=0 lon=240176200 lat=549453500
	// alt=131 course=270 sat=8 speed=0 event=0 ioCount=0 (empty groups)
	data := []byte{
		0x00, 0x00, 0x00, 0x00, // prefix
		0x00, 0x00, 0x00, 0x20, // data length = 32
		0x08,                                           // codec 8
		0x01,                                           // count = 1
		0x00, 0x00, 0x01, 0x5F, 0x93, 0x03, 0x7C, 0x00, // timestamp 1510000000000
		0x00,                   // priority
		0x0E, 0x50, 0xCC, 0x48, // longitude 240176200
		0x20, 0xBF, 0xFE, 0xBC, // latitude 549453500
		0x00, 0x83, // altitude 131
		0x01, 0x0E, // course 270
		0x08,       // satellites 8
		0x00, 0x00, // speed 0
		0x00, // event id
		0x00, // total io records
		0x00, // 1-byte params count
		0x00, // 2-byte params count
		0x00, // 4-byte params count
		0x00, // 8-byte params count
	}

	d := NewDecoder()
	res, err := d.Decode(data)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Positions, 1)

	pos := res.Positions[0]
	assert.Equal(t, time.UnixMilli(1510000000000).UTC(), pos.Time.UTC())
	assert.Equal(t, 24.01762, pos.Longitude)
	assert.Equal(t, 54.94535, pos.Latitude)
	assert.Equal(t, float64(131), pos.Altitude)
	assert.Equal(t, float64(270), pos.Course)
	assert.True(t, pos.Valid)
	assert.Equal(t, 0.0, pos.Speed)
	assert.Equal(t, uint8(8), pos.Attributes["satellites"])
	// response = count echo (4 bytes)
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x01}, res.Response)
}

func TestDecodeCodec8WithIO(t *testing.T) {
	// Codec 8 with IO: 1-byte param 239 (ignition) = 1, 2-byte param 67 (battery) = 4100
	data := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x25,
		0x08,
		0x01,
		0x00, 0x00, 0x01, 0x5F, 0x93, 0x03, 0x7C, 0x00,
		0x00,
		0x0E, 0x50, 0xCC, 0x48,
		0x20, 0xBF, 0xFE, 0xBC,
		0x00, 0x83,
		0x01, 0x0E,
		0x08,
		0x00, 0x00,
		0x00,       // event id
		0x01,       // total io records
		0x01,       // 1-byte params count
		0xEF, 0x01, // id 239, value 1
		0x01,             // 2-byte params count
		0x43, 0x10, 0x04, // id 67, value 4100
		0x00, // 4-byte params count
		0x00, // 8-byte params count
	}

	d := NewDecoder()
	res, err := d.Decode(data)
	require.NoError(t, err)
	require.Len(t, res.Positions, 1)

	pos := res.Positions[0]
	assert.Equal(t, true, pos.Attributes["ignition"])
	assert.Equal(t, float64(4.1), pos.Attributes["battery"])
}
