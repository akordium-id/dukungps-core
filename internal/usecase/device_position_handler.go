package usecase

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/protocol"
	"github.com/akordium-id/dukungps/internal/repository"
)

// DevicePositionHandler implements protocol.PositionHandler. It resolves the
// IMEI from device frames to a persisted device, optionally auto-registering
// unknown devices when AutoRegister is enabled, then forwards positions to the
// PositionService pipeline.
type DevicePositionHandler struct {
	positionSvc  *PositionService
	deviceRepo   repository.DeviceRepository
	AutoRegister bool
}

func NewDevicePositionHandler(
	positionSvc *PositionService,
	deviceRepo repository.DeviceRepository,
	autoRegister bool,
) *DevicePositionHandler {
	return &DevicePositionHandler{
		positionSvc:  positionSvc,
		deviceRepo:   deviceRepo,
		AutoRegister: autoRegister,
	}
}

func (h *DevicePositionHandler) HandlePositions(deviceID string, res *protocol.DecodeResult) {
	deviceID = strings.TrimSpace(deviceID)
	for _, p := range res.Positions {
		if deviceID != "" {
			dev, err := h.resolveDevice(deviceID)
			if err != nil {
				log.Printf("resolve device %s: %v", deviceID, err)
				continue
			}
			p.DeviceID = dev.ID
		}
		if err := h.positionSvc.ProcessPosition(context.Background(), p); err != nil {
			log.Printf("process position: %v", err)
		}
	}
}

func (h *DevicePositionHandler) resolveDevice(deviceID string) (*device.Device, error) {
	dev, err := h.deviceRepo.FindByUniqueID(context.Background(), deviceID)
	if err == nil {
		return dev, nil
	}
	if !errors.Is(err, repository.ErrDeviceNotFound) {
		return nil, err
	}
	if !h.AutoRegister {
		return nil, errors.New("device not registered")
	}

	// auto-register on first contact (gated by AutoRegister flag)
	newDev := &device.Device{
		UniqueID: deviceID,
		Name:     deviceID,
		Status:   device.StatusOnline,
	}
	if err := h.deviceRepo.Create(context.Background(), newDev); err != nil && !errors.Is(err, repository.ErrDeviceExists) {
		return nil, err
	}

	dev, err = h.deviceRepo.FindByUniqueID(context.Background(), deviceID)
	if err != nil {
		return nil, err
	}
	return dev, nil
}
