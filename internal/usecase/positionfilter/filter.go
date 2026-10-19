package positionfilter

import (
	"context"
	"math"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/usecase"
)

// PrePositionFilter aliases usecase.PrePositionFilter for convenience.
type PrePositionFilter = usecase.PrePositionFilter

const (
	// EarthRadiusMeters is the approximate mean radius of the Earth in meters.
	EarthRadiusMeters = 6371000.0
	// DefaultMaxSpeedKmh is the realistic speed ceiling for road vehicles in km/h.
	DefaultMaxSpeedKmh = 180.0
	// DefaultDriftDistanceMeters is the minimum movement required when speed is 0 to be considered valid movement.
	DefaultDriftDistanceMeters = 20.0
)

// HaversineDistance calculates the great-circle distance between two coordinates in meters.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	radLat1 := lat1 * math.Pi / 180.0
	radLat2 := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(radLat1)*math.Cos(radLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusMeters * c
}

// DistanceAndOdometerEnricher calculates distance from the previous fix in meters and updates total odometer.
func DistanceAndOdometerEnricher() PrePositionFilter {
	return func(ctx context.Context, p *position.Position, last *position.Position) (bool, error) {
		if p.Attributes == nil {
			p.Attributes = make(map[string]any)
		}

		if last == nil {
			p.Attributes["distance"] = 0.0
			return true, nil
		}

		dist := HaversineDistance(last.Latitude, last.Longitude, p.Latitude, p.Longitude)
		p.Attributes["distance"] = math.Round(dist*100) / 100 // 2 decimal precision

		// Accumulate total odometer if present in last position
		if lastTotal, ok := last.Attributes["total_distance"].(float64); ok {
			p.Attributes["total_distance"] = math.Round((lastTotal+dist)*100) / 100
		} else {
			p.Attributes["total_distance"] = math.Round(dist*100) / 100
		}

		return true, nil
	}
}

// DriftFilter drops small random GPS coordinate hops when the vehicle is stationary (speed == 0).
func DriftFilter(minDistanceThreshold float64) PrePositionFilter {
	if minDistanceThreshold <= 0 {
		minDistanceThreshold = DefaultDriftDistanceMeters
	}

	return func(ctx context.Context, p *position.Position, last *position.Position) (bool, error) {
		if last == nil {
			return true, nil
		}

		// When vehicle speed is zero or negligible
		if p.Speed == 0 {
			dist := HaversineDistance(last.Latitude, last.Longitude, p.Latitude, p.Longitude)
			// If distance moved is within jitter radius while stationary, drop coordinate jitter
			if dist < minDistanceThreshold {
				return false, nil
			}
		}

		return true, nil
	}
}

// MaxSpeedFilter drops absurd teleportation spikes where speed exceeds max realistic speed.
func MaxSpeedFilter(maxSpeedKmh float64) PrePositionFilter {
	if maxSpeedKmh <= 0 {
		maxSpeedKmh = DefaultMaxSpeedKmh
	}

	return func(ctx context.Context, p *position.Position, last *position.Position) (bool, error) {
		if p.Speed > maxSpeedKmh {
			return false, nil
		}
		return true, nil
	}
}
