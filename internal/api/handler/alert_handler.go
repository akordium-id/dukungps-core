package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/domain/alert"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type AlertHandler struct {
	alertUC usecase.AlertUseCase
}

func NewAlertHandler(alertUC usecase.AlertUseCase) *AlertHandler {
	return &AlertHandler{alertUC: alertUC}
}

func (h *AlertHandler) List(c fiber.Ctx) error {
	alerts, err := h.alertUC.ListAlerts(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(alerts)
}

func (h *AlertHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid alert id"})
	}

	a, err := h.alertUC.GetAlert(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrAlertNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "alert not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(a)
}

func (h *AlertHandler) ListByDevice(c fiber.Ctx) error {
	deviceID, err := strconv.ParseInt(c.Params("deviceId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid device id"})
	}

	alerts, err := h.alertUC.GetAlertsByDevice(c.Context(), deviceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(alerts)
}

func (h *AlertHandler) Create(c fiber.Ctx) error {
	var a alert.Alert
	if err := c.Bind().Body(&a); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.alertUC.CreateAlert(c.Context(), &a); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(a)
}

func (h *AlertHandler) Dispatch(c fiber.Ctx) error {
	var a alert.Alert
	if err := c.Bind().Body(&a); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.alertUC.DispatchAlert(c.Context(), &a); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(a)
}

func (h *AlertHandler) Update(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid alert id"})
	}

	var a alert.Alert
	if err := c.Bind().Body(&a); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	a.ID = id

	if err := h.alertUC.UpdateAlert(c.Context(), &a); err != nil {
		if errors.Is(err, repository.ErrAlertNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "alert not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(a)
}

func (h *AlertHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid alert id"})
	}

	if err := h.alertUC.DeleteAlert(c.Context(), id); err != nil {
		if errors.Is(err, repository.ErrAlertNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "alert not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}