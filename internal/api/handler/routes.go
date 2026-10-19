package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type Services struct {
	DeviceUseCase      usecase.DeviceUseCase
	PositionUseCase    usecase.PositionUseCase
	GeofenceUseCase    usecase.GeofenceUseCase
	UserUseCase        usecase.UserUseCase
	CommandUseCase     usecase.CommandUseCase
	DeviceGroupUseCase usecase.DeviceGroupUseCase
	AlertUseCase       usecase.AlertUseCase
	Hub                *websocket.Hub
}

func SetupRoutes(app *fiber.App, services *Services) fiber.Router {
	deviceHandler := NewDeviceHandler(services.DeviceUseCase)
	positionHandler := NewPositionHandler(services.PositionUseCase, services.Hub)
	wsHandler := NewWebSocketHandler(services.Hub)
	geofenceHandler := NewGeofenceHandler(services.GeofenceUseCase)
	userHandler := NewUserHandler(services.UserUseCase)
	commandHandler := NewCommandHandler(services.CommandUseCase)
	deviceGroupHandler := NewDeviceGroupHandler(services.DeviceGroupUseCase)
	alertHandler := NewAlertHandler(services.AlertUseCase)

	api := app.Group("/api/v1")
	api.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	api.Get("/devices", deviceHandler.List)
	api.Get("/devices/:id", deviceHandler.Get)
	api.Post("/devices", deviceHandler.Create)
	api.Get("/devices/:deviceId/positions", positionHandler.ListByDevice)
	api.Get("/devices/:deviceId/positions/search", positionHandler.Search)
	api.Get("/geofences", geofenceHandler.List)
	api.Get("/geofences/:id", geofenceHandler.Get)
	api.Post("/geofences", geofenceHandler.Create)
	api.Put("/geofences/:id", geofenceHandler.Update)
	api.Delete("/geofences/:id", geofenceHandler.Delete)
	api.Get("/users", userHandler.List)
	api.Get("/users/:id", userHandler.Get)
	api.Post("/users", userHandler.Create)
	api.Put("/users/:id", userHandler.Update)
	api.Delete("/users/:id", userHandler.Delete)
	api.Get("/commands", commandHandler.List)
	api.Get("/commands/:id", commandHandler.Get)
	api.Get("/devices/:deviceId/commands", commandHandler.ListByDevice)
	api.Post("/commands", commandHandler.Create)
	api.Post("/commands/dispatch", commandHandler.Dispatch)
	api.Put("/commands/:id", commandHandler.Update)
	api.Delete("/commands/:id", commandHandler.Delete)
	api.Get("/device-groups", deviceGroupHandler.List)
	api.Get("/device-groups/:id", deviceGroupHandler.Get)
	api.Post("/device-groups", deviceGroupHandler.Create)
	api.Put("/device-groups/:id", deviceGroupHandler.Update)
	api.Delete("/device-groups/:id", deviceGroupHandler.Delete)
	api.Post("/device-groups/:groupId/devices", deviceGroupHandler.AddDevice)
	api.Delete("/device-groups/:groupId/devices/:deviceId", deviceGroupHandler.RemoveDevice)
	api.Get("/device-groups/:groupId/devices", deviceGroupHandler.GetDevices)
	api.Get("/alerts", alertHandler.List)
	api.Get("/alerts/:id", alertHandler.Get)
	api.Get("/devices/:deviceId/alerts", alertHandler.ListByDevice)
	api.Post("/alerts", alertHandler.Create)
	api.Post("/alerts/dispatch", alertHandler.Dispatch)
	api.Put("/alerts/:id", alertHandler.Update)
	api.Delete("/alerts/:id", alertHandler.Delete)
	api.Get("/ws", wsHandler.Handle)

	return api
}
