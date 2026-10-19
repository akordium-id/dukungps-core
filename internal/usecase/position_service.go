package usecase

import (
	"context"
	"time"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/infrastructure/cache"
	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
	"github.com/akordium-id/dukungps/internal/repository"
)

type PositionSearchParams struct {
	DeviceID int64
	Limit    int
	From     *time.Time
	To       *time.Time
	MinSpeed float64
	MaxSpeed float64
}

// PrePositionFilter determines whether a position should continue downstream to DB/broadcast,
// and can enrich attributes (e.g. distance, odometer, speed clamp).
// Returning keep = false drops the position without saving to DB.
type PrePositionFilter func(ctx context.Context, p *position.Position, last *position.Position) (keep bool, err error)

type PositionHook func(ctx context.Context, p *position.Position) error

type PositionUseCase interface {
	ProcessPosition(ctx context.Context, p *position.Position) error
	GetPositionsByDevice(ctx context.Context, deviceID int64, limit int) ([]position.Position, error)
	SearchPositions(ctx context.Context, params PositionSearchParams) ([]position.Position, error)
	RegisterPositionHook(hook PositionHook)
	RegisterPreFilter(filter PrePositionFilter)
}

type PositionService struct {
	positionRepo repository.PositionRepository
	deviceRepo   repository.DeviceRepository
	cache        cache.Cache
	hub          *websocket.Hub
	preFilters   []PrePositionFilter
	hooks        []PositionHook
}

func NewPositionService(
	positionRepo repository.PositionRepository,
	deviceRepo repository.DeviceRepository,
	cache cache.Cache,
	hub *websocket.Hub,
) *PositionService {
	return &PositionService{
		positionRepo: positionRepo,
		deviceRepo:   deviceRepo,
		cache:        cache,
		hub:          hub,
		preFilters:   make([]PrePositionFilter, 0),
		hooks:        make([]PositionHook, 0),
	}
}

func (s *PositionService) RegisterPositionHook(hook PositionHook) {
	if hook != nil {
		s.hooks = append(s.hooks, hook)
	}
}

func (s *PositionService) RegisterPreFilter(filter PrePositionFilter) {
	if filter != nil {
		s.preFilters = append(s.preFilters, filter)
	}
}

func (s *PositionService) ProcessPosition(ctx context.Context, p *position.Position) error {
	if !p.IsValidFix() {
		return nil
	}

	// Fetch last known position from cache for pre-processing / enrichment / filtering
	var lastPos *position.Position
	if s.cache != nil {
		if cached, err := s.cache.GetPosition(ctx, p.DeviceID); err == nil {
			lastPos = cached
		}
	}

	// Run pre-commit filters (drift filter, speed clamp, distance/odometer enricher, etc.)
	for _, filter := range s.preFilters {
		keep, err := filter(ctx, p, lastPos)
		if err != nil {
			continue
		}
		if !keep {
			// Position dropped before DB / broadcast
			return nil
		}
	}

	if err := s.positionRepo.Create(ctx, p); err != nil {
		return err
	}

	if err := s.cache.SetPosition(ctx, p.DeviceID, p); err != nil {
		return err
	}

	s.hub.Broadcast(&websocket.PositionUpdate{
		DeviceID: p.DeviceID,
		Position: p,
	})

	for _, hook := range s.hooks {
		if err := hook(ctx, p); err != nil {
			// Hook errors are logged/ignored from aborting the pipeline,
			// ensuring one failing plugin does not crash position processing
			continue
		}
	}

	return nil
}

func (s *PositionService) GetPositionsByDevice(ctx context.Context, deviceID int64, limit int) ([]position.Position, error) {
	return s.positionRepo.FindByDeviceID(ctx, deviceID, limit)
}

func (s *PositionService) SearchPositions(ctx context.Context, params PositionSearchParams) ([]position.Position, error) {
	return s.positionRepo.FindByDeviceID(ctx, params.DeviceID, params.Limit)
}

var _ PositionUseCase = (*PositionService)(nil)
