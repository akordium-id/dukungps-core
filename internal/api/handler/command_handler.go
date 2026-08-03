package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/domain/command"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type CommandHandler struct {
	commandUC usecase.CommandUseCase
}

func NewCommandHandler(commandUC usecase.CommandUseCase) *CommandHandler {
	return &CommandHandler{commandUC: commandUC}
}

func (h *CommandHandler) List(c fiber.Ctx) error {
	commands, err := h.commandUC.ListCommands(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(commands)
}

func (h *CommandHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid command id"})
	}

	cmd, err := h.commandUC.GetCommand(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrCommandNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "command not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cmd)
}

func (h *CommandHandler) ListByDevice(c fiber.Ctx) error {
	deviceID, err := strconv.ParseInt(c.Params("deviceId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid device id"})
	}

	commands, err := h.commandUC.GetCommandsByDevice(c.Context(), deviceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(commands)
}

func (h *CommandHandler) Create(c fiber.Ctx) error {
	var cmd command.Command
	if err := c.Bind().Body(&cmd); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.commandUC.CreateCommand(c.Context(), &cmd); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(cmd)
}

func (h *CommandHandler) Dispatch(c fiber.Ctx) error {
	var cmd command.Command
	if err := c.Bind().Body(&cmd); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.commandUC.DispatchCommand(c.Context(), &cmd); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(cmd)
}

func (h *CommandHandler) Update(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid command id"})
	}

	var cmd command.Command
	if err := c.Bind().Body(&cmd); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	cmd.ID = id

	if err := h.commandUC.UpdateCommand(c.Context(), &cmd); err != nil {
		if errors.Is(err, repository.ErrCommandNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "command not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cmd)
}

func (h *CommandHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid command id"})
	}

	if err := h.commandUC.DeleteCommand(c.Context(), id); err != nil {
		if errors.Is(err, repository.ErrCommandNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "command not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}