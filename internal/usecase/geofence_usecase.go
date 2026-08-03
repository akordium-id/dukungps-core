package usecase

import (
	"context"

	"github.com/akordium-id/dukungps/internal/domain/geofence"
	"github.com/akordium-id/dukungps/internal/repository"
)

type GeofenceUseCase interface {
	ListGeofences(ctx context.Context) ([]*geofence.Geofence, error)
	GetGeofence(ctx context.Context, id int64) (*geofence.Geofence, error)
	CreateGeofence(ctx context.Context, g *geofence.Geofence) error
	UpdateGeofence(ctx context.Context, g *geofence.Geofence) error
	DeleteGeofence(ctx context.Context, id int64) error
}

type GeofenceService struct {
	geofenceRepo repository.GeofenceRepository
}

func NewGeofenceService(geofenceRepo repository.GeofenceRepository) *GeofenceService {
	return &GeofenceService{geofenceRepo: geofenceRepo}
}

func (s *GeofenceService) ListGeofences(ctx context.Context) ([]*geofence.Geofence, error) {
	return s.geofenceRepo.FindAll(ctx)
}

func (s *GeofenceService) GetGeofence(ctx context.Context, id int64) (*geofence.Geofence, error) {
	return s.geofenceRepo.FindByID(ctx, id)
}

func (s *GeofenceService) CreateGeofence(ctx context.Context, g *geofence.Geofence) error {
	return s.geofenceRepo.Create(ctx, g)
}

func (s *GeofenceService) UpdateGeofence(ctx context.Context, g *geofence.Geofence) error {
	return s.geofenceRepo.Update(ctx, g)
}

func (s *GeofenceService) DeleteGeofence(ctx context.Context, id int64) error {
	return s.geofenceRepo.Delete(ctx, id)
}

var _ GeofenceUseCase = (*GeofenceService)(nil)