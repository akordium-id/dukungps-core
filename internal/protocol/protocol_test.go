package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProtocol struct{}

func (m mockProtocol) Name() string { return "mock" }
func (m mockProtocol) Decoder() Decoder {
	return nil
}
func (m mockProtocol) Encoder() Encoder {
	return nil
}

var _ Protocol = (*mockProtocol)(nil)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(mockProtocol{})

	p, err := reg.Get("mock")
	require.NoError(t, err)
	assert.Equal(t, "mock", p.Name())
}

func TestRegistry_GetUnknown(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("tidak-ada")
	assert.Error(t, err)
}

func TestRegistry_Names(t *testing.T) {
	reg := NewRegistry()
	reg.Register(mockProtocol{})

	names := reg.Names()
	assert.Contains(t, names, "mock")
}

func TestProtocolNameConstants(t *testing.T) {
	assert.Equal(t, "teltonika", Teltonika)
	assert.Equal(t, "gt06", GT06)
	assert.Equal(t, "huabao", Huabao)
}
