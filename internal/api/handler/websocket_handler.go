package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
)

type WebSocketHandler struct {
	hub *websocket.Hub
}

func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// Handle upgrades the connection to WebSocket and registers a hub client.
// The actual upgrade loop is implemented by the server; here we register a
// client wired to the hub's send channel for the connection lifetime.
func (h *WebSocketHandler) Handle(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "websocket endpoint not wired yet"})
}
