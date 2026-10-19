package gt06

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CRC-16/X25 over all bytes after the 2-byte start marker
// (called before the CRC field is appended)
func testCRC(t *testing.T, frame []byte) uint16 {
	t.Helper()
	crc := uint16(0xFFFF)
	for _, b := range frame[2:] {
		crc ^= uint16(b)
		for range 8 {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0x8408
			} else {
				crc >>= 1
			}
		}
	}
	return crc ^ 0xFFFF
}

func buildShortFrame(t *testing.T, msgType byte, payload []byte, serial uint16) []byte {
	t.Helper()
	content := append([]byte{msgType}, payload...)
	lenField := byte(len(content) + 2 + 2) // type + payload + serial + crc
	frame := []byte{0x78, 0x78, lenField}
	frame = append(frame, content...)
	frame = binary.BigEndian.AppendUint16(frame, serial)
	crc := testCRC(t, frame)
	frame = binary.BigEndian.AppendUint16(frame, crc)
	return append(frame, 0x0D, 0x0A)
}

func TestDecodeLogin(t *testing.T) {
	// IMEI field: 8 bytes, hex-encoded 16 chars, first char dropped
	imeiField := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45}
	frame := buildShortFrame(t, 0x01, imeiField, 0x0001)

	d := NewDecoder()
	res, err := d.Decode(frame)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "123456789012345", res.DeviceID)
	assert.Empty(t, res.Positions)
	// login ack: 78 78 05 01 [serial 00 01] [crc] 0D 0A
	require.Len(t, res.Response, 10)
	assert.Equal(t, []byte{0x78, 0x78, 0x05, 0x01}, res.Response[:4])
	assert.Equal(t, []byte{0x00, 0x01}, res.Response[4:6])
	assert.Equal(t, []byte{0x0D, 0x0A}, res.Response[8:10])
}

func TestDecodeLocation(t *testing.T) {
	// lat 54.94535, lon 24.01762 (Surabaya-like positive quadrant)
	latRaw := uint32(54.94535 * 1800000) // 0x05E51E7E
	lonRaw := uint32(24.01762 * 1800000) // 0x0293A9E4
	course := 270
	// flags: course(bits0-9) | lat positive(bit10=1) | valid(bit12=1)
	flags := uint16(course | 0x0400 | 0x1000)

	payload := make([]byte, 0, 18)
	payload = append(payload, 0x0F, 0x0A, 0x14, 0x0E, 0x15, 0x0B) // 2015-10-20 14:21:11
	payload = append(payload, 0x05)                               // satellites
	payload = binary.BigEndian.AppendUint32(payload, latRaw)
	payload = binary.BigEndian.AppendUint32(payload, lonRaw)
	payload = append(payload, 84) // speed km/h
	payload = binary.BigEndian.AppendUint16(payload, flags)

	frame := buildShortFrame(t, 0x12, payload, 0x0002)

	d := NewDecoder()
	res, err := d.Decode(frame)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Positions, 1)

	pos := res.Positions[0]
	assert.True(t, pos.Valid)
	assert.InDelta(t, 54.94535, pos.Latitude, 0.0001)
	assert.InDelta(t, 24.01762, pos.Longitude, 0.0001)
	assert.InDelta(t, 84/1.852, pos.Speed, 0.01)
	assert.Equal(t, float64(270), pos.Course)

	// location ack echoes serial
	require.Len(t, res.Response, 10)
	assert.Equal(t, []byte{0x78, 0x78, 0x05, 0x12}, res.Response[:4])
	assert.Equal(t, []byte{0x00, 0x02}, res.Response[4:6])
}

func TestDecodeLocationWithTerminalStatus(t *testing.T) {
	latRaw := uint32(54.94535 * 1800000)
	lonRaw := uint32(24.01762 * 1800000)
	flags := uint16(270 | 0x0400 | 0x1000)

	payload := make([]byte, 0, 25)
	payload = append(payload, 0x0F, 0x0A, 0x14, 0x0E, 0x15, 0x0B)
	payload = append(payload, 0x05)
	payload = binary.BigEndian.AppendUint32(payload, latRaw)
	payload = binary.BigEndian.AppendUint32(payload, lonRaw)
	payload = append(payload, 84)
	payload = binary.BigEndian.AppendUint16(payload, flags)
	// Append 6 bytes LBS (MCC 2, MNC 1, LAC 2, CellID 3 approx)
	payload = append(payload, 0x01, 0xCC, 0x00, 0x28, 0x5D, 0x00)
	// Append Terminal Information byte:
	// Bit 7: Oil cut (0x80)
	// Bit 2: Charging (0x04)
	// Bit 1: ACC ON (0x02)
	// Bits 5-3: 100 (SOS alarm = 0x20)
	// statusByte = 0x80 | 0x20 | 0x04 | 0x02 = 0xA6
	statusByte := byte(0xA6)
	payload = append(payload, statusByte)

	frame := buildShortFrame(t, 0x12, payload, 0x0005)

	d := NewDecoder()
	res, err := d.Decode(frame)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Positions, 1)

	pos := res.Positions[0]
	assert.True(t, pos.GetBool("ignition"), "ignition should be true")
	assert.True(t, pos.GetBool("blocked"), "cut-engine blocked should be true")
	assert.True(t, pos.GetBool("charge"), "charge should be true")
	assert.True(t, pos.GetBool("motion"), "motion should be true")
	assert.Equal(t, "sos", pos.Attributes["alarm"])
}

func TestDecodeUnknownType(t *testing.T) {
	// command types 0x80-0x82 -> no positions, no response
	frame := buildShortFrame(t, 0x80, nil, 0x0003)

	d := NewDecoder()
	res, err := d.Decode(frame)
	require.NoError(t, err)
	assert.Empty(t, res.Positions)
	assert.Empty(t, res.Response)
}
