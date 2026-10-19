package positionfilter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/position"
)

func TestHaversineDistance(t *testing.T) {
	// Jakarta (-6.2088, 106.8456) to Surabaya (-7.2575, 112.7521) ~ 660 km
	dist := HaversineDistance(-6.2088, 106.8456, -7.2575, 112.7521)
	assert.Greater(t, dist, 650000.0)
	assert.Less(t, dist, 680000.0)

	// Same point -> 0 meters
	distZero := HaversineDistance(-7.25, 112.75, -7.25, 112.75)
	assert.Equal(t, 0.0, distZero)
}

func TestDistanceAndOdometerEnricher(t *testing.T) {
	enricher := DistanceAndOdometerEnricher()
	ctx := context.Background()

	// 1. First fix (no last position)
	p1 := &position.Position{
		DeviceID:  1,
		Latitude:  -7.250000,
		Longitude: 112.750000,
	}
	keep, err := enricher(ctx, p1, nil)
	require.NoError(t, err)
	assert.True(t, keep)
	assert.Equal(t, 0.0, p1.Attributes["distance"])
	assert.Nil(t, p1.Attributes["total_distance"])

	// 2. Second fix (~111 meters North)
	p2 := &position.Position{
		DeviceID:  1,
		Latitude:  -7.249000,
		Longitude: 112.750000,
	}
	keep, err = enricher(ctx, p2, p1)
	require.NoError(t, err)
	assert.True(t, keep)
	dist, ok := p2.Attributes["distance"].(float64)
	assert.True(t, ok)
	assert.InDelta(t, 111.0, dist, 10.0)
	assert.InDelta(t, 111.0, p2.Attributes["total_distance"].(float64), 10.0)
}

func TestDriftFilter(t *testing.T) {
	filter := DriftFilter(25.0) // 25 meters threshold
	ctx := context.Background()

	last := &position.Position{
		DeviceID:  1,
		Latitude:  -7.250000,
		Longitude: 112.750000,
		Speed:     0,
	}

	// Case 1: Speed is 0, coordinate moved 5 meters (jitter/drift) -> should drop (keep == false)
	jitterPos := &position.Position{
		DeviceID:  1,
		Latitude:  -7.250040, // ~4.4 meters
		Longitude: 112.750000,
		Speed:     0,
	}
	keep, err := filter(ctx, jitterPos, last)
	require.NoError(t, err)
	assert.False(t, keep, "should drop small drift when speed is 0")

	// Case 2: Speed is 0, but distance moved is 50 meters -> keep (real movement or moved while off)
	validMoveWhileStopped := &position.Position{
		DeviceID:  1,
		Latitude:  -7.250500, // ~55 meters
		Longitude: 112.750000,
		Speed:     0,
	}
	keep, err = filter(ctx, validMoveWhileStopped, last)
	require.NoError(t, err)
	assert.True(t, keep)

	// Case 3: Speed > 0 (vehicle moving), even small distance is kept
	movingPos := &position.Position{
		DeviceID:  1,
		Latitude:  -7.250040,
		Longitude: 112.750000,
		Speed:     15.0,
	}
	keep, err = filter(ctx, movingPos, last)
	require.NoError(t, err)
	assert.True(t, keep)
}

func TestMaxSpeedFilter(t *testing.T) {
	filter := MaxSpeedFilter(180.0)
	ctx := context.Background()

	// Normal speed
	keep, err := filter(ctx, &position.Position{Speed: 60.0}, nil)
	require.NoError(t, err)
	assert.True(t, keep)

	// High speed spike (> 180 km/h)
	keep, err = filter(ctx, &position.Position{Speed: 350.0}, nil)
	require.NoError(t, err)
	assert.False(t, keep, "should drop unrealistic speed spike")
}
