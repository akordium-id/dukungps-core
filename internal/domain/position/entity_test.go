package position

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPosition_IsValidFix(t *testing.T) {
	tests := []struct {
		name     string
		position Position
		want     bool
	}{
		{
			name: "valid fix",
			position: Position{
				DeviceID:  1,
				Latitude:  -7.2456,
				Longitude: 112.7389,
				Valid:     true,
			},
			want: true,
		},
		{
			name: "invalid fix flag",
			position: Position{
				DeviceID:  1,
				Latitude:  -7.2456,
				Longitude: 112.7389,
				Valid:     false,
			},
			want: false,
		},
		{
			name: "latitude out of range",
			position: Position{
				DeviceID:  1,
				Latitude:  95.0,
				Longitude: 112.7389,
				Valid:     true,
			},
			want: false,
		},
		{
			name: "longitude out of range",
			position: Position{
				DeviceID:  1,
				Latitude:  -7.2456,
				Longitude: 200.0,
				Valid:     true,
			},
			want: false,
		},
		{
			name: "zero device id",
			position: Position{
				DeviceID:  0,
				Latitude:  -7.2456,
				Longitude: 112.7389,
				Valid:     true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.position.IsValidFix())
		})
	}
}
