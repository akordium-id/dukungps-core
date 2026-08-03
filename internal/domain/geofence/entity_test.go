package geofence

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeofence_ContainsPolygon(t *testing.T) {
	g := Geofence{
		Type: TypePolygon,
		Area: []Coordinate{
			{Latitude: -7.3, Longitude: 112.7},
			{Latitude: -7.3, Longitude: 112.8},
			{Latitude: -7.2, Longitude: 112.8},
			{Latitude: -7.2, Longitude: 112.7},
		},
	}

	tests := []struct {
		name string
		lat  float64
		lon  float64
		want bool
	}{
		{name: "inside", lat: -7.25, lon: 112.75, want: true},
		{name: "outside east", lat: -7.25, lon: 112.85, want: false},
		{name: "outside north", lat: -7.15, lon: 112.75, want: false},
		{name: "on edge", lat: -7.3, lon: 112.75, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, g.Contains(tt.lat, tt.lon))
		})
	}
}

func TestGeofence_ContainsCircle(t *testing.T) {
	g := Geofence{
		Type:   TypeCircle,
		Center: Coordinate{Latitude: -7.25, Longitude: 112.75},
		Radius: 1000,
	}

	tests := []struct {
		name string
		lat  float64
		lon  float64
		want bool
	}{
		{name: "center", lat: -7.25, lon: 112.75, want: true},
		{name: "near center", lat: -7.2505, lon: 112.7505, want: true},
		{name: "far away", lat: -7.30, lon: 112.85, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, g.Contains(tt.lat, tt.lon))
		})
	}
}

func TestGeofence_ContainsPolygonTooFewPoints(t *testing.T) {
	g := Geofence{
		Type: TypePolygon,
		Area: []Coordinate{
			{Latitude: -7.3, Longitude: 112.7},
			{Latitude: -7.2, Longitude: 112.8},
		},
	}

	assert.False(t, g.Contains(-7.25, 112.75))
}
