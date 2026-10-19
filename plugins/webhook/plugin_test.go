package webhook_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/pkg/plugin"
	"github.com/akordium-id/dukungps/plugins/webhook"
)

type testCore struct {
	hook       plugin.PositionHook
	router     fiber.Router
	webhookURL string
}

func (c *testCore) OnPosition(hook plugin.PositionHook) {
	c.hook = hook
}

func (c *testCore) Router(pluginName string) fiber.Router {
	return c.router
}

func (c *testCore) Devices() plugin.DeviceReader {
	return nil
}

func (c *testCore) Positions() plugin.PositionReader {
	return nil
}

func (c *testCore) DB() any {
	return nil
}

func (c *testCore) Config(key string, fallback string) string {
	if key == "PLUGIN_WEBHOOK_URL" {
		return c.webhookURL
	}
	return fallback
}

func TestWebhookPluginLifecycleAndHook(t *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"Latitude":-7.25`)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := fiber.New()
	subRouter := app.Group("/api/v1/plugins/webhook")

	core := &mockCoreWrapper{
		webhookURL: server.URL,
		subRouter:  subRouter,
	}

	p := webhook.New()
	err := p.Init(core)
	require.NoError(t, err)

	// Trigger hook
	pos := &position.Position{
		DeviceID:  123,
		Latitude:  -7.25,
		Longitude: 112.75,
		Speed:     45.5,
		Time:      time.Now(),
	}
	err = core.hook(context.Background(), pos)
	require.NoError(t, err)

	assert.Equal(t, int32(1), atomic.LoadInt32(&count))

	// Test endpoint /api/v1/plugins/webhook/status
	req := httptest.NewRequest("GET", "/api/v1/plugins/webhook/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	require.NoError(t, p.Close())
}

type mockCoreWrapper struct {
	plugin.Core
	hook       plugin.PositionHook
	subRouter  fiber.Router
	webhookURL string
}

func (m *mockCoreWrapper) OnPosition(hook plugin.PositionHook) {
	m.hook = hook
}

func (m *mockCoreWrapper) Router(name string) fiber.Router {
	return m.subRouter
}

func (m *mockCoreWrapper) Config(key string, fallback string) string {
	if key == "PLUGIN_WEBHOOK_URL" {
		return m.webhookURL
	}
	return fallback
}
