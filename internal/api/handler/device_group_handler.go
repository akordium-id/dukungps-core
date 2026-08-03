package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type DeviceGroupHandler struct {
	groupUC usecase.DeviceGroupUseCase
}

func NewDeviceGroupHandler(groupUC usecase.DeviceGroupUseCase) *DeviceGroupHandler {
	return &DeviceGroupHandler{groupUC: groupUC}
}

func (h *DeviceGroupHandler) List(c fiber.Ctx) error {
	groups, err := h.groupUC.ListGroups(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(groups)
}

func (h *DeviceGroupHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid group id"})
	}

	g, err := h.groupUC.GetGroup(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrDeviceGroupNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "group not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(g)
}

func (h *DeviceGroupHandler) Create(c fiber.Ctx) error {
	var g device.DeviceGroup
	if err := c.Bind().Body(&g); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.groupUC.CreateGroup(c.Context(), &g); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(g)
}

func (h *DeviceGroupHandler) Update(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid group id"})
	}

	var g device.DeviceGroup
	if err := c.Bind().Body(&g); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	g.ID = id

	if err := h.groupUC.UpdateGroup(c.Context(), &g); err != nil {
		if errors.Is(err, repository.ErrDeviceGroupNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "group not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(g)
}

func (h *DeviceGroupHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid group id"})
	}

	if err := h.groupUC.DeleteGroup(c.Context(), id); err != nil {
		if errors.Is(err, repository.ErrDeviceGroupNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "group not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *DeviceGroupHandler) AddDevice(c fiber.Ctx) error {
	groupID, err := strconv.ParseInt(c.Params("groupId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid group id"})
	}

	var req struct {
		DeviceID int64 `json:"device_id"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.groupUC.AddDevice(c.Context(), groupID, req.DeviceID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "device added to group"})
}

func (h *DeviceGroupHandler) RemoveDevice(c fiber.Ctx) error {
	groupID, err := strconv.ParseInt(c.Params("groupId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid group id"})
	}

	deviceID, err := strconv.ParseInt(c.Params("deviceId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid device id"})
	}

	if err := h.groupUC.RemoveDevice(c.Context(), groupID, deviceID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *DeviceGroupHandler) GetDevices(c fiber.Ctx) error {
	groupID, err := strconv.ParseInt(c.Params("groupId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid group id"})
	}

	devices, err := h.groupUC.GetDevicesByGroup(c.Context(), groupID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(devices)
}