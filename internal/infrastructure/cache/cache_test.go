package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/position"
)

func TestCacheSetGetPosition(t *testing.T) {
	client := newTestRedis(t)
	defer client.Close()

	c := NewCache(client)

	p := &position.Position{
		ID:        1,
		DeviceID:  42,
		Latitude:  -7.2456,
		Longitude: 112.7389,
		Speed:     42.0,
		Valid:     true,
		Time:      time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
		Attributes: map[string]any{
			"ignition": true,
			"battery":  12.5,
		},
	}
	require.NoError(t, c.SetPosition(context.Background(), 42, p))

	got, err := c.GetPosition(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(42), got.DeviceID)
	assert.Equal(t, -7.2456, got.Latitude)
	assert.Equal(t, true, got.Attributes["ignition"])
	assert.InDelta(t, 12.5, got.Attributes["battery"], 0.001)
}

func TestCachePositionNotFound(t *testing.T) {
	client := newTestRedis(t)
	defer client.Close()

	c := NewCache(client)

	_, err := c.GetPosition(context.Background(), 999)
	assert.Error(t, err)
}

func TestCacheDeviceOnline(t *testing.T) {
	client := newTestRedis(t)
	defer client.Close()

	c := NewCache(client)

	online, err := c.IsDeviceOnline(context.Background(), 7)
	require.NoError(t, err)
	assert.False(t, online)

	require.NoError(t, c.SetDeviceOnline(context.Background(), 7))

	online, err = c.IsDeviceOnline(context.Background(), 7)
	require.NoError(t, err)
	assert.True(t, online)
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	s := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: s.Addr()})
}
