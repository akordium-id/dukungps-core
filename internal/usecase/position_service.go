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

type PositionUseCase interface {
	ProcessPosition(ctx context.Context, p *position.Position) error
	GetPositionsByDevice(ctx context.Context, deviceID int64, limit int) ([]position.Position, error)
	SearchPositions(ctx context.Context, params PositionSearchParams) ([]position.Position, error)
}

type PositionService struct {
	positionRepo repository.PositionRepository
	deviceRepo   repository.DeviceRepository
	cache        cache.Cache
	hub          *websocket.Hub
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
	}
}

func (s *PositionService) ProcessPosition(ctx context.Context, p *position.Position) error {
	if !p.IsValidFix() {
		return nil
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

	return nil
}

func (s *PositionService) GetPositionsByDevice(ctx context.Context, deviceID int64, limit int) ([]position.Position, error) {
	return s.positionRepo.FindByDeviceID(ctx, deviceID, limit)
}

func (s *PositionService) SearchPositions(ctx context.Context, params PositionSearchParams) ([]position.Position, error) {
	return s.positionRepo.FindByDeviceID(ctx, params.DeviceID, params.Limit)
}

var _ PositionUseCase = (*PositionService)(nil)
