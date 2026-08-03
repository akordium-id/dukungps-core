package usecase

import (
	"context"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/repository"
)

type DeviceGroupUseCase interface {
	ListGroups(ctx context.Context) ([]*device.DeviceGroup, error)
	GetGroup(ctx context.Context, id int64) (*device.DeviceGroup, error)
	CreateGroup(ctx context.Context, g *device.DeviceGroup) error
	UpdateGroup(ctx context.Context, g *device.DeviceGroup) error
	DeleteGroup(ctx context.Context, id int64) error
	AddDevice(ctx context.Context, groupID int64, deviceID int64) error
	RemoveDevice(ctx context.Context, groupID int64, deviceID int64) error
	GetDevicesByGroup(ctx context.Context, groupID int64) ([]*device.Device, error)
}

type DeviceGroupService struct {
	groupRepo repository.DeviceGroupRepository
}

func NewDeviceGroupService(groupRepo repository.DeviceGroupRepository) *DeviceGroupService {
	return &DeviceGroupService{groupRepo: groupRepo}
}

func (s *DeviceGroupService) ListGroups(ctx context.Context) ([]*device.DeviceGroup, error) {
	return s.groupRepo.FindAll(ctx)
}

func (s *DeviceGroupService) GetGroup(ctx context.Context, id int64) (*device.DeviceGroup, error) {
	return s.groupRepo.FindByID(ctx, id)
}

func (s *DeviceGroupService) CreateGroup(ctx context.Context, g *device.DeviceGroup) error {
	return s.groupRepo.Create(ctx, g)
}

func (s *DeviceGroupService) UpdateGroup(ctx context.Context, g *device.DeviceGroup) error {
	return s.groupRepo.Update(ctx, g)
}

func (s *DeviceGroupService) DeleteGroup(ctx context.Context, id int64) error {
	return s.groupRepo.Delete(ctx, id)
}

func (s *DeviceGroupService) AddDevice(ctx context.Context, groupID int64, deviceID int64) error {
	return s.groupRepo.AddDevice(ctx, groupID, deviceID)
}

func (s *DeviceGroupService) RemoveDevice(ctx context.Context, groupID int64, deviceID int64) error {
	return s.groupRepo.RemoveDevice(ctx, groupID, deviceID)
}

func (s *DeviceGroupService) GetDevicesByGroup(ctx context.Context, groupID int64) ([]*device.Device, error) {
	return s.groupRepo.FindDevicesByGroup(ctx, groupID)
}

var _ DeviceGroupUseCase = (*DeviceGroupService)(nil)