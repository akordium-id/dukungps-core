package geofencealert_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/geofence"
	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/pkg/plugin"
	"github.com/akordium-id/dukungps/plugins/geofencealert"
)

type mockGeofenceReader struct {
	geofences []*geofence.Geofence
}

func (m *mockGeofenceReader) ListGeofences(ctx context.Context) ([]*geofence.Geofence, error) {
	return m.geofences, nil
}

func (m *mockGeofenceReader) GetGeofence(ctx context.Context, id int64) (*geofence.Geofence, error) {
	for _, g := range m.geofences {
		if g.ID == id {
			return g, nil
		}
	}
	return nil, nil
}

type mockAlertCore struct {
	plugin.Core
	hook      plugin.PositionHook
	router    fiber.Router
	geofences plugin.GeofenceReader
}

func (m *mockAlertCore) OnPosition(hook plugin.PositionHook) {
	m.hook = hook
}

func (m *mockAlertCore) Router(name string) fiber.Router {
	return m.router
}

func (m *mockAlertCore) Geofences() plugin.GeofenceReader {
	return m.geofences
}

func TestGeofenceAlertPlugin_EntryAndExit(t *testing.T) {
	app := fiber.New()
	subRouter := app.Group("/api/v1/plugins/geofence-alert")

	// Geofence circle centered at (-7.25, 112.75) with radius 1000m
	testFence := &geofence.Geofence{
		ID:     1,
		Name:   "Depot A",
		Type:   geofence.TypeCircle,
		Center: geofence.Coordinate{Latitude: -7.25, Longitude: 112.75},
		Radius: 1000,
	}

	core := &mockAlertCore{
		router:    subRouter,
		geofences: &mockGeofenceReader{geofences: []*geofence.Geofence{testFence}},
	}

	p := geofencealert.New()
	require.NoError(t, p.Init(core))

	// Step 1: Device is outside
	posOutside := &position.Position{
		DeviceID:  10,
		Latitude:  -7.35, // Far away
		Longitude: 112.75,
		Time:      time.Now(),
	}
	require.NoError(t, core.hook(context.Background(), posOutside))

	// Step 2: Device enters Depot A
	posInside := &position.Position{
		DeviceID:  10,
		Latitude:  -7.25, // Exactly at center
		Longitude: 112.75,
		Time:      time.Now(),
	}
	require.NoError(t, core.hook(context.Background(), posInside))

	// Step 3: Check API events
	req := httptest.NewRequest("GET", "/api/v1/plugins/geofence-alert/events", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Step 4: Device exits Depot A
	require.NoError(t, core.hook(context.Background(), posOutside))

	// Verify status endpoint
	reqStatus := httptest.NewRequest("GET", "/api/v1/plugins/geofence-alert/status", nil)
	respStatus, err := app.Test(reqStatus)
	require.NoError(t, err)
	assert.Equal(t, 200, respStatus.StatusCode)

	require.NoError(t, p.Close())
}
