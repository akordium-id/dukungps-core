package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/api/handler"
	"github.com/akordium-id/dukungps/internal/infrastructure/cache"
	"github.com/akordium-id/dukungps/internal/infrastructure/database"
	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
	"github.com/akordium-id/dukungps/internal/protocol"
	"github.com/akordium-id/dukungps/internal/protocol/gt06"
	"github.com/akordium-id/dukungps/internal/protocol/huabao"
	"github.com/akordium-id/dukungps/internal/protocol/teltonika"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
	"github.com/akordium-id/dukungps/internal/usecase/positionfilter"
	"github.com/akordium-id/dukungps/pkg/plugin"
	"github.com/akordium-id/dukungps/plugins/geofencealert"
	"github.com/akordium-id/dukungps/plugins/webhook"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	// databases
	pool, err := database.NewPool(ctx, envOr("DATABASE_URL", "postgres://dukungps:dukungps@localhost:5432/dukungps"))
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: envOr("REDIS_URL", "localhost:6379")})
	defer func() { _ = redisClient.Close() }()

	l1, err := cache.NewRistrettoCache()
	if err != nil {
		log.Fatalf("ristretto: %v", err)
	}
	defer l1.Close()

	// domain services
	hub := websocket.NewHub()
	go hub.Run()

	deviceRepo := repository.NewDeviceRepo(pool)
	positionRepo := repository.NewPositionRepo(pool)
	geofenceRepo := repository.NewGeofenceRepo(pool)
	userRepo := repository.NewUserRepo(pool)
	commandRepo := repository.NewCommandRepo(pool)
	deviceGroupRepo := repository.NewDeviceGroupRepo(pool)
	alertRepo := repository.NewAlertRepo(pool)
	cacheImpl := cache.NewMultilevelCache(l1, cache.NewCache(redisClient))

	deviceService := usecase.NewDeviceService(deviceRepo)
	positionService := usecase.NewPositionService(positionRepo, deviceRepo, cacheImpl, hub)
	// Register default pre-commit filters (Traccar-style GPS drift, speed filter, telemetry normalizer, distance/odometer enricher, and trip/idle detector)
	positionService.RegisterPreFilter(positionfilter.DriftFilter(20.0))
	positionService.RegisterPreFilter(positionfilter.MaxSpeedFilter(180.0))
	positionService.RegisterPreFilter(positionfilter.TelemetryNormalizer())
	positionService.RegisterPreFilter(positionfilter.DistanceAndOdometerEnricher())
	positionService.RegisterPreFilter(positionfilter.TripAndStateDetector())

	geofenceService := usecase.NewGeofenceService(geofenceRepo)
	userService := usecase.NewUserService(userRepo)
	commandService := usecase.NewCommandService(commandRepo)
	deviceGroupService := usecase.NewDeviceGroupService(deviceGroupRepo)
	alertService := usecase.NewAlertService(alertRepo)

	// protocol registry + TCP server
	registry := protocol.NewRegistry()
	registry.Register(&teltonika.Protocol{})
	registry.Register(&gt06.Protocol{})
	registry.Register(&huabao.Protocol{})

	tcpServer := protocol.NewServer(registry)
	autoRegister := envBool("AUTO_REGISTER_DEVICES")
	handlerAdapter := usecase.NewDevicePositionHandler(positionService, deviceRepo, autoRegister)
	for _, name := range registry.Names() {
		tcpServer.SetHandler(name, handlerAdapter)
	}

	// protocol ports (overridable via env), default mengikuti Traccar
	tcpServer.RegisterPort("teltonika", portOr("TELTONIKA_PORT", teltonika.DefaultPort))
	tcpServer.RegisterPort("gt06", portOr("GT06_PORT", gt06.DefaultPort))
	tcpServer.RegisterPort("huabao", portOr("HUABAO_PORT", huabao.DefaultPort))

	if err := tcpServer.Start(); err != nil {
		log.Fatalf("protocol server: %v", err)
	}

	// HTTP API
	app := fiber.New()
	apiGroup := handler.SetupRoutes(app, &handler.Services{
		DeviceUseCase:      deviceService,
		PositionUseCase:    positionService,
		GeofenceUseCase:    geofenceService,
		UserUseCase:        userService,
		CommandUseCase:     commandService,
		DeviceGroupUseCase: deviceGroupService,
		AlertUseCase:       alertService,
		Hub:                hub,
	})

	// Plugin subsystem (open-core extensible architecture)
	coreAdapter := plugin.NewCoreAdapter(pool, positionService, deviceRepo, geofenceService, apiGroup)
	pluginRegistry := plugin.NewManagedRegistry(coreAdapter)

	// Register built-in / optional plugins
	_ = pluginRegistry.Register(webhook.New())
	_ = pluginRegistry.Register(geofencealert.New())

	if err := pluginRegistry.InitAll(); err != nil {
		log.Fatalf("plugin init: %v", err)
	}
	defer pluginRegistry.CloseAll()

	if err := app.Listen(envOr("HTTP_ADDR", ":8080")); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func portOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
