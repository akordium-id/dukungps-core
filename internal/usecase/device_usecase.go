package usecase

import (
	"context"
	"errors"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/repository"
)

// DeviceUseCase defines the business operations for device management.
// The HTTP handler layer depends on this interface, not on the repository
// directly, keeping the dependency rule: handler -> usecase -> repository.
type DeviceUseCase interface {
	ListDevices(ctx context.Context) ([]*device.Device, error)
	GetDevice(ctx context.Context, id int64) (*device.Device, error)
	CreateDevice(ctx context.Context, d *device.Device) error
	UpdateDevice(ctx context.Context, d *device.Device) error
}

type DeviceService struct {
	deviceRepo repository.DeviceRepository
}

func NewDeviceService(deviceRepo repository.DeviceRepository) *DeviceService {
	return &DeviceService{deviceRepo: deviceRepo}
}

func (s *DeviceService) ListDevices(ctx context.Context) ([]*device.Device, error) {
	return s.deviceRepo.FindAll(ctx)
}

func (s *DeviceService) GetDevice(ctx context.Context, id int64) (*device.Device, error) {
	return s.deviceRepo.FindByID(ctx, id)
}

func (s *DeviceService) CreateDevice(ctx context.Context, d *device.Device) error {
	if !d.IsValid() {
		return ErrDeviceValidation
	}
	return s.deviceRepo.Create(ctx, d)
}

func (s *DeviceService) UpdateDevice(ctx context.Context, d *device.Device) error {
	if !d.IsValid() {
		return ErrDeviceValidation
	}
	return s.deviceRepo.Update(ctx, d)
}

var ErrDeviceValidation = errors.New("unique_id and protocol are required")

var _ DeviceUseCase = (*DeviceService)(nil)
