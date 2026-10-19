package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/pkg/plugin"
)

// Plugin forwards position updates to external HTTP webhook endpoints.
type Plugin struct {
	mu         sync.RWMutex
	client     *http.Client
	webhookURL string
	events     []WebhookEvent
}

type WebhookEvent struct {
	DeviceID   int64     `json:"device_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Speed      float64   `json:"speed"`
	SentAt     time.Time `json:"sent_at"`
	StatusCode int       `json:"status_code"`
}

func New() *Plugin {
	return &Plugin{
		client: &http.Client{Timeout: 5 * time.Second},
		events: make([]WebhookEvent, 0),
	}
}

func (p *Plugin) Name() string {
	return "webhook"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Init(core plugin.Core) error {
	cfg := plugin.NewPluginConfig(p.Name(), core)
	p.webhookURL = cfg.GetString("URL", "")

	// Hook into position pipeline
	core.OnPosition(p.handlePosition)

	// Register plugin endpoints under /api/v1/plugins/webhook
	router := core.Router(p.Name())
	if router != nil {
		router.Get("/status", p.handleStatus)
		router.Get("/history", p.handleHistory)
	}

	log.Printf("[plugin:webhook] Started (target URL: %q)", p.webhookURL)
	return nil
}

func (p *Plugin) handlePosition(ctx context.Context, pos *position.Position) error {
	if p.webhookURL == "" {
		return nil
	}

	payload, err := json.Marshal(pos)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	statusCode := 0
	if err == nil {
		statusCode = resp.StatusCode
		_ = resp.Body.Close()
	}

	p.mu.Lock()
	if len(p.events) >= 50 {
		p.events = p.events[1:]
	}
	p.events = append(p.events, WebhookEvent{
		DeviceID:   pos.DeviceID,
		Latitude:   pos.Latitude,
		Longitude:  pos.Longitude,
		Speed:      pos.Speed,
		SentAt:     time.Now(),
		StatusCode: statusCode,
	})
	p.mu.Unlock()

	return nil
}

func (p *Plugin) handleStatus(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"plugin":     p.Name(),
		"version":    p.Version(),
		"target_url": p.webhookURL,
		"enabled":    p.webhookURL != "",
	})
}

func (p *Plugin) handleHistory(c fiber.Ctx) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return c.JSON(fiber.Map{
		"total":  len(p.events),
		"events": p.events,
	})
}

func (p *Plugin) Close() error {
	log.Printf("[plugin:webhook] Shutdown complete")
	return nil
}
