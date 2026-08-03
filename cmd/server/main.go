package main

import (
	"context"
	"log"
	"os"
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
	defer redisClient.Close()

	// domain services
	hub := websocket.NewHub()
	go hub.Run()

	deviceRepo := repository.NewDeviceRepo(pool)
	positionRepo := repository.NewPositionRepo(pool)
	cacheImpl := cache.NewCache(redisClient)

	positionService := usecase.NewPositionService(positionRepo, deviceRepo, cacheImpl, hub)

	// protocol registry + TCP server
	registry := protocol.NewRegistry()
	registry.Register(&teltonika.Protocol{})
	registry.Register(&gt06.Protocol{})
	registry.Register(&huabao.Protocol{})

	tcpServer := protocol.NewServer(registry)
	handlerAdapter := &positionHandlerAdapter{
		svc:        positionService,
		deviceRepo: deviceRepo,
	}
	for _, name := range registry.Names() {
		tcpServer.SetHandler(name, handlerAdapter)
	}

	if err := tcpServer.Start(); err != nil {
		log.Fatalf("protocol server: %v", err)
	}

	// HTTP API
	app := fiber.New()
	handler.SetupRoutes(app, &handler.Services{
		DeviceRepo:   deviceRepo,
		PositionRepo: positionRepo,
		Hub:          hub,
	})

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

type positionHandlerAdapter struct {
	svc        *usecase.PositionService
	deviceRepo repository.DeviceRepository
}

func (a *positionHandlerAdapter) HandlePositions(deviceID string, res *protocol.DecodeResult) {
	deviceID = strings.TrimSpace(deviceID)
	for _, p := range res.Positions {
		if deviceID != "" {
			dev, err := a.deviceRepo.FindByUniqueID(context.Background(), deviceID)
			if err != nil {
				log.Printf("resolve device %s: %v", deviceID, err)
				continue
			}
			p.DeviceID = dev.ID
		}
		if err := a.svc.ProcessPosition(context.Background(), p); err != nil {
			log.Printf("process position: %v", err)
		}
	}
}
