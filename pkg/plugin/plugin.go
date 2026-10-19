package plugin

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/domain/position"
)

// PositionHook is a callback invoked after a position is validated, stored, and cached.
type PositionHook func(ctx context.Context, p *position.Position) error

// DeviceReader provides read-only access to device queries for plugins.
type DeviceReader interface {
	FindByID(ctx context.Context, id int64) (*device.Device, error)
	FindByUniqueID(ctx context.Context, uniqueID string) (*device.Device, error)
	FindAll(ctx context.Context) ([]*device.Device, error)
}

// PositionReader provides read-only access to position queries for plugins.
type PositionReader interface {
	GetPositionsByDevice(ctx context.Context, deviceID int64, limit int) ([]position.Position, error)
}

// Core is the primary capability host exposed to plugins (WordPress-like Core API).
type Core interface {
	// Hooks
	OnPosition(hook PositionHook)

	// Routing (sub-router scoped under /api/v1/plugins/<plugin-name>)
	Router(pluginName string) fiber.Router

	// Data Readers
	Devices() DeviceReader
	Positions() PositionReader

	// Infra & Config
	DB() *pgxpool.Pool
	Config(key string, fallback string) string
}

// Plugin defines the lifecycle contract every extension module must implement.
type Plugin interface {
	Name() string
	Version() string
	Init(core Core) error
	Close() error
}
