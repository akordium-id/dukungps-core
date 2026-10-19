package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDevice_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   bool
	}{
		{
			name:   "valid device",
			device: Device{UniqueID: "123456789012345", Name: "Truk 01", Protocol: "teltonika"},
			want:   true,
		},
		{
			name:   "missing unique id",
			device: Device{Name: "Truk 01", Protocol: "teltonika"},
			want:   false,
		},
		{
			name:   "missing protocol",
			device: Device{UniqueID: "123456789012345", Name: "Truk 01"},
			want:   false,
		},
		{
			name:   "blank unique id",
			device: Device{UniqueID: "   ", Name: "Truk 01", Protocol: "teltonika"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.device.IsValid())
		})
	}
}

func TestDeviceStatusConstants(t *testing.T) {
	assert.Equal(t, "online", StatusOnline)
	assert.Equal(t, "offline", StatusOffline)
	assert.Equal(t, "unknown", StatusUnknown)
}
