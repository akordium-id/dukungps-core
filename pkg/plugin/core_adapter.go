package plugin

import (
	"context"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/domain/geofence"
	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
)

// coreAdapter implements plugin.Core for the host application.
type coreAdapter struct {
	pool        *pgxpool.Pool
	posUC       usecase.PositionUseCase
	deviceRepo  repository.DeviceRepository
	geofenceUC  usecase.GeofenceUseCase
	pluginGroup fiber.Router
}

// NewCoreAdapter constructs a Core implementation given host services.
func NewCoreAdapter(
	pool *pgxpool.Pool,
	posUC usecase.PositionUseCase,
	deviceRepo repository.DeviceRepository,
	geofenceUC usecase.GeofenceUseCase,
	apiGroup fiber.Router,
) Core {
	var pluginGroup fiber.Router
	if apiGroup != nil {
		pluginGroup = apiGroup.Group("/plugins")
	}
	return &coreAdapter{
		pool:        pool,
		posUC:       posUC,
		deviceRepo:  deviceRepo,
		geofenceUC:  geofenceUC,
		pluginGroup: pluginGroup,
	}
}

func (c *coreAdapter) OnPosition(hook PositionHook) {
	if c.posUC != nil && hook != nil {
		c.posUC.RegisterPositionHook(func(ctx context.Context, p *position.Position) error {
			return hook(ctx, p)
		})
	}
}

func (c *coreAdapter) Router(pluginName string) fiber.Router {
	if c.pluginGroup == nil {
		return nil
	}
	return c.pluginGroup.Group("/" + pluginName)
}

func (c *coreAdapter) Devices() DeviceReader {
	return &deviceReaderAdapter{repo: c.deviceRepo}
}

func (c *coreAdapter) Positions() PositionReader {
	return &positionReaderAdapter{posUC: c.posUC}
}

func (c *coreAdapter) Geofences() GeofenceReader {
	return &geofenceReaderAdapter{geofenceUC: c.geofenceUC}
}

func (c *coreAdapter) DB() *pgxpool.Pool {
	return c.pool
}

func (c *coreAdapter) Config(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// deviceReaderAdapter provides read-only device queries.
type deviceReaderAdapter struct {
	repo repository.DeviceRepository
}

func (r *deviceReaderAdapter) FindByID(ctx context.Context, id int64) (*device.Device, error) {
	if r.repo == nil {
		return nil, nil
	}
	return r.repo.FindByID(ctx, id)
}

func (r *deviceReaderAdapter) FindByUniqueID(ctx context.Context, uniqueID string) (*device.Device, error) {
	if r.repo == nil {
		return nil, nil
	}
	return r.repo.FindByUniqueID(ctx, uniqueID)
}

func (r *deviceReaderAdapter) FindAll(ctx context.Context) ([]*device.Device, error) {
	if r.repo == nil {
		return nil, nil
	}
	return r.repo.FindAll(ctx)
}

// positionReaderAdapter provides read-only position queries.
type positionReaderAdapter struct {
	posUC usecase.PositionUseCase
}

func (r *positionReaderAdapter) GetPositionsByDevice(ctx context.Context, deviceID int64, limit int) ([]position.Position, error) {
	if r.posUC == nil {
		return nil, nil
	}
	return r.posUC.GetPositionsByDevice(ctx, deviceID, limit)
}

// geofenceReaderAdapter provides read-only geofence queries.
type geofenceReaderAdapter struct {
	geofenceUC usecase.GeofenceUseCase
}

func (r *geofenceReaderAdapter) ListGeofences(ctx context.Context) ([]*geofence.Geofence, error) {
	if r.geofenceUC == nil {
		return nil, nil
	}
	return r.geofenceUC.ListGeofences(ctx)
}

func (r *geofenceReaderAdapter) GetGeofence(ctx context.Context, id int64) (*geofence.Geofence, error) {
	if r.geofenceUC == nil {
		return nil, nil
	}
	return r.geofenceUC.GetGeofence(ctx, id)
}
