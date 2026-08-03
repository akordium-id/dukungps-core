package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
	"github.com/akordium-id/dukungps/internal/repository"
)

type PositionHandler struct {
	positionRepo repository.PositionRepository
	hub          *websocket.Hub
}

func NewPositionHandler(positionRepo repository.PositionRepository, hub *websocket.Hub) *PositionHandler {
	return &PositionHandler{positionRepo: positionRepo, hub: hub}
}

func (h *PositionHandler) ListByDevice(c fiber.Ctx) error {
	deviceID, err := strconv.ParseInt(c.Params("deviceId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid device id"})
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	positions, err := h.positionRepo.FindByDeviceID(c.Context(), deviceID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(positions)
}
