package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type PositionHandler struct {
	positionUC usecase.PositionUseCase
	hub        *websocket.Hub
}

func NewPositionHandler(positionUC usecase.PositionUseCase, hub *websocket.Hub) *PositionHandler {
	return &PositionHandler{positionUC: positionUC, hub: hub}
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

	positions, err := h.positionUC.GetPositionsByDevice(c.Context(), deviceID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(positions)
}
