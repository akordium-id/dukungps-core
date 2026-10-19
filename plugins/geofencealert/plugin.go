package geofencealert

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/pkg/plugin"
)

// GeofenceState tracks device presence relative to geofences.
type DeviceGeofenceState struct {
	DeviceID   int64     `json:"device_id"`
	GeofenceID int64     `json:"geofence_id"`
	Geofence   string    `json:"geofence_name"`
	IsInside   bool      `json:"is_inside"`
	LastEvent  string    `json:"last_event"` // "ENTER" or "EXIT"
	EventTime  time.Time `json:"event_time"`
}

// Plugin is a commercial/premium-tier plugin example that performs real-time geofence entry/exit detection.
type Plugin struct {
	mu           sync.RWMutex
	core         plugin.Core
	deviceStates map[string]*DeviceGeofenceState // key: "deviceID:geofenceID"
	recentEvents []DeviceGeofenceState
}

func New() *Plugin {
	return &Plugin{
		deviceStates: make(map[string]*DeviceGeofenceState),
		recentEvents: make([]DeviceGeofenceState, 0),
	}
}

func (p *Plugin) Name() string {
	return "geofence-alert"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Init(core plugin.Core) error {
	p.core = core

	// Listen to position updates
	core.OnPosition(p.handlePosition)

	// Register premium API endpoints under /api/v1/plugins/geofence-alert
	router := core.Router(p.Name())
	if router != nil {
		router.Get("/events", p.handleListEvents)
		router.Get("/status", p.handleStatus)
	}

	log.Printf("[plugin:geofence-alert] Premium geofence alert plugin initialized")
	return nil
}

func (p *Plugin) handlePosition(ctx context.Context, pos *position.Position) error {
	if p.core == nil || p.core.Geofences() == nil {
		return nil
	}

	geofences, err := p.core.Geofences().ListGeofences(ctx)
	if err != nil || len(geofences) == 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, g := range geofences {
		if g == nil {
			continue
		}

		insideNow := g.Contains(pos.Latitude, pos.Longitude)
		key := stateKey(pos.DeviceID, g.ID)

		prev, exists := p.deviceStates[key]
		if !exists {
			// Initial state
			event := "ENTER"
			if !insideNow {
				event = "OUTSIDE"
			}
			p.deviceStates[key] = &DeviceGeofenceState{
				DeviceID:   pos.DeviceID,
				GeofenceID: g.ID,
				Geofence:   g.Name,
				IsInside:   insideNow,
				LastEvent:  event,
				EventTime:  time.Now(),
			}
			continue
		}

		// State transition
		if !prev.IsInside && insideNow {
			// ENTER event
			event := DeviceGeofenceState{
				DeviceID:   pos.DeviceID,
				GeofenceID: g.ID,
				Geofence:   g.Name,
				IsInside:   true,
				LastEvent:  "ENTER",
				EventTime:  time.Now(),
			}
			p.recordEvent(event)
			*prev = event
			log.Printf("[plugin:geofence-alert] ALERT: Device %d ENTERED geofence %q", pos.DeviceID, g.Name)
		} else if prev.IsInside && !insideNow {
			// EXIT event
			event := DeviceGeofenceState{
				DeviceID:   pos.DeviceID,
				GeofenceID: g.ID,
				Geofence:   g.Name,
				IsInside:   false,
				LastEvent:  "EXIT",
				EventTime:  time.Now(),
			}
			p.recordEvent(event)
			*prev = event
			log.Printf("[plugin:geofence-alert] ALERT: Device %d EXITED geofence %q", pos.DeviceID, g.Name)
		}
	}

	return nil
}

func (p *Plugin) recordEvent(ev DeviceGeofenceState) {
	if len(p.recentEvents) >= 100 {
		p.recentEvents = p.recentEvents[1:]
	}
	p.recentEvents = append(p.recentEvents, ev)
}

func (p *Plugin) handleListEvents(c fiber.Ctx) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return c.JSON(fiber.Map{
		"total":  len(p.recentEvents),
		"events": p.recentEvents,
	})
}

func (p *Plugin) handleStatus(c fiber.Ctx) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return c.JSON(fiber.Map{
		"plugin":         p.Name(),
		"version":        p.Version(),
		"tracked_states": len(p.deviceStates),
	})
}

func (p *Plugin) Close() error {
	log.Printf("[plugin:geofence-alert] Shutdown complete")
	return nil
}

func stateKey(deviceID, geofenceID int64) string {
	return fmt.Sprintf("%d:%d", deviceID, geofenceID)
}
