package protocol

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/position"
)

var errShortFrame = errors.New("mock: frame too short")

type mockHandler struct {
	deviceID  string
	positions int
}

func (m *mockHandler) HandlePositions(deviceID string, res *DecodeResult) {
	m.deviceID = deviceID
	m.positions += len(res.Positions)
}

// frameProtocol: text protocol, frame = [u16 length][payload], payload starting
// with "IMEI:<id>" is identification, "POS" is a position, ack = [0xAA].
type frameProtocol struct{}

func (p *frameProtocol) Name() string     { return "mock" }
func (p *frameProtocol) Decoder() Decoder { return &frameDecoder{} }
func (p *frameProtocol) Encoder() Encoder { return nil }

type frameDecoder struct{}

func (d *frameDecoder) Decode(data []byte) (*DecodeResult, error) {
	payload := data[2:]
	switch {
	case string(payload[:5]) == "IMEI:":
		return &DecodeResult{DeviceID: string(payload[5:]), Response: []byte{0xAA}}, nil
	case string(payload) == "POS":
		return &DecodeResult{
			Positions: []*position.Position{{DeviceID: 7, Latitude: -7.1, Valid: true}},
			Response:  []byte{0xAA},
		}, nil
	default:
		return &DecodeResult{}, nil
	}
}

func (d *frameDecoder) FrameLen(data []byte) (int, error) {
	if len(data) < 2 {
		return 0, errShortFrame
	}
	length := int(binary.BigEndian.Uint16(data[:2]))
	if len(data) < 2+length {
		return 0, errShortFrame
	}
	return 2 + length, nil
}

var (
	_ Protocol     = (*frameProtocol)(nil)
	_ Decoder      = (*frameDecoder)(nil)
	_ FrameDecoder = (*frameDecoder)(nil)
)

func buildMockFrame(payload []byte) []byte {
	frame := make([]byte, 2+len(payload))
	binary.BigEndian.PutUint16(frame[:2], uint16(len(payload)))
	copy(frame[2:], payload)
	return frame
}

func TestServerHandlesProtocolSession(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	handler := &mockHandler{}
	srv := NewServer(nil)
	srv.SetHandler("mock", handler)

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.handleConnection(server, "mock", &frameProtocol{})
	}()

	// identification
	_, err := client.Write(buildMockFrame([]byte("IMEI:123456789012345")))
	require.NoError(t, err)

	ack := make([]byte, 1)
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = io.ReadFull(client, ack)
	require.NoError(t, err)
	assert.Equal(t, byte(0xAA), ack[0])

	// two frames in one write (fragmented stream handling)
	combined := append(buildMockFrame([]byte("POS")), buildMockFrame([]byte("POS"))...)
	_, err = client.Write(combined)
	require.NoError(t, err)

	_, err = io.ReadFull(client, ack)
	require.NoError(t, err)
	_, err = io.ReadFull(client, ack)
	require.NoError(t, err)

	// close client so server read loop terminates
	client.Close()

	<-done
	assert.Equal(t, 2, handler.positions)
	assert.Equal(t, "123456789012345", handler.deviceID)
}
