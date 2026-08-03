package usecase

import (
	"context"
	"time"

	"github.com/akordium-id/dukungps/internal/domain/alert"
	"github.com/akordium-id/dukungps/internal/repository"
)

type AlertUseCase interface {
	ListAlerts(ctx context.Context) ([]alert.Alert, error)
	GetAlert(ctx context.Context, id int64) (*alert.Alert, error)
	GetAlertsByDevice(ctx context.Context, deviceID int64) ([]alert.Alert, error)
	CreateAlert(ctx context.Context, a *alert.Alert) error
	DispatchAlert(ctx context.Context, a *alert.Alert) error
	UpdateAlert(ctx context.Context, a *alert.Alert) error
	DeleteAlert(ctx context.Context, id int64) error
}

type AlertService struct {
	alertRepo repository.AlertRepository
}

func NewAlertService(alertRepo repository.AlertRepository) *AlertService {
	return &AlertService{alertRepo: alertRepo}
}

func (s *AlertService) ListAlerts(ctx context.Context) ([]alert.Alert, error) {
	return s.alertRepo.FindAll(ctx)
}

func (s *AlertService) GetAlert(ctx context.Context, id int64) (*alert.Alert, error) {
	return s.alertRepo.FindByID(ctx, id)
}

func (s *AlertService) GetAlertsByDevice(ctx context.Context, deviceID int64) ([]alert.Alert, error) {
	return s.alertRepo.FindByDeviceID(ctx, deviceID)
}

func (s *AlertService) CreateAlert(ctx context.Context, a *alert.Alert) error {
	return s.alertRepo.Create(ctx, a)
}

func (s *AlertService) DispatchAlert(ctx context.Context, a *alert.Alert) error {
	if err := s.alertRepo.Create(ctx, a); err != nil {
		return err
	}
	a.Status = alert.AlertStatusSent
	a.SentAt = time.Now()
	return s.alertRepo.Update(ctx, a)
}

func (s *AlertService) UpdateAlert(ctx context.Context, a *alert.Alert) error {
	return s.alertRepo.Update(ctx, a)
}

func (s *AlertService) DeleteAlert(ctx context.Context, id int64) error {
	return s.alertRepo.Delete(ctx, id)
}

var _ AlertUseCase = (*AlertService)(nil)