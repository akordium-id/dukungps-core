package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/domain/geofence"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type GeofenceHandler struct {
	geofenceUC usecase.GeofenceUseCase
}

func NewGeofenceHandler(geofenceUC usecase.GeofenceUseCase) *GeofenceHandler {
	return &GeofenceHandler{geofenceUC: geofenceUC}
}

func (h *GeofenceHandler) List(c fiber.Ctx) error {
	geofences, err := h.geofenceUC.ListGeofences(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(geofences)
}

func (h *GeofenceHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid geofence id"})
	}

	g, err := h.geofenceUC.GetGeofence(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrGeofenceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "geofence not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(g)
}

func (h *GeofenceHandler) Create(c fiber.Ctx) error {
	var g geofence.Geofence
	if err := c.Bind().Body(&g); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.geofenceUC.CreateGeofence(c.Context(), &g); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(g)
}

func (h *GeofenceHandler) Update(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid geofence id"})
	}

	var g geofence.Geofence
	if err := c.Bind().Body(&g); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	g.ID = id

	if err := h.geofenceUC.UpdateGeofence(c.Context(), &g); err != nil {
		if errors.Is(err, repository.ErrGeofenceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "geofence not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(g)
}

func (h *GeofenceHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid geofence id"})
	}

	if err := h.geofenceUC.DeleteGeofence(c.Context(), id); err != nil {
		if errors.Is(err, repository.ErrGeofenceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "geofence not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
