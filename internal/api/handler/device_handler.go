package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type DeviceHandler struct {
	deviceUC usecase.DeviceUseCase
}

func NewDeviceHandler(deviceUC usecase.DeviceUseCase) *DeviceHandler {
	return &DeviceHandler{deviceUC: deviceUC}
}

func (h *DeviceHandler) List(c fiber.Ctx) error {
	devices, err := h.deviceUC.ListDevices(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(devices)
}

func (h *DeviceHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid device id"})
	}

	d, err := h.deviceUC.GetDevice(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "device not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(d)
}

func (h *DeviceHandler) Create(c fiber.Ctx) error {
	var d device.Device
	if err := c.Bind().Body(&d); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.deviceUC.CreateDevice(c.Context(), &d); err != nil {
		if errors.Is(err, usecase.ErrDeviceValidation) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "unique_id and protocol are required"})
		}
		if errors.Is(err, repository.ErrDeviceExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "device already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(d)
}
