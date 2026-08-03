package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type Services struct {
	DeviceUseCase   usecase.DeviceUseCase
	PositionUseCase usecase.PositionUseCase
	Hub             *websocket.Hub
}

func SetupRoutes(app *fiber.App, services *Services) {
	deviceHandler := NewDeviceHandler(services.DeviceUseCase)
	positionHandler := NewPositionHandler(services.PositionUseCase, services.Hub)
	wsHandler := NewWebSocketHandler(services.Hub)

	api := app.Group("/api/v1")
	api.Get("/devices", deviceHandler.List)
	api.Get("/devices/:id", deviceHandler.Get)
	api.Post("/devices", deviceHandler.Create)
	api.Get("/devices/:deviceId/positions", positionHandler.ListByDevice)
	api.Get("/ws", wsHandler.Handle)
}
