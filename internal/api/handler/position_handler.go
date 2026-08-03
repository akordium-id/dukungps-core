package handler

import (
	"strconv"
	"time"

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

func (h *PositionHandler) Search(c fiber.Ctx) error {
	deviceID, err := strconv.ParseInt(c.Query("device_id"), 10, 64)
	if err != nil || deviceID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "valid device_id query param required"})
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	var from, to *time.Time
	if f := c.Query("from"); f != "" {
		if t, err := time.Parse(time.RFC3339, f); err == nil {
			from = &t
		}
	}
	if t := c.Query("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = &parsed
		}
	}

	minSpeed := -1.0
	if s := c.Query("min_speed"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v >= 0 {
			minSpeed = v
		}
	}

	maxSpeed := -1.0
	if s := c.Query("max_speed"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v >= 0 {
			maxSpeed = v
		}
	}

	positions, err := h.positionUC.SearchPositions(c.Context(), usecase.PositionSearchParams{
		DeviceID: deviceID,
		Limit:    limit,
		From:     from,
		To:       to,
		MinSpeed: minSpeed,
		MaxSpeed: maxSpeed,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(positions)
}
