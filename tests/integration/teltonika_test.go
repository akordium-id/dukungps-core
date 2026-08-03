package integration

import (
	"context"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/infrastructure/cache"
	"github.com/akordium-id/dukungps/internal/infrastructure/database"
	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
	"github.com/akordium-id/dukungps/internal/protocol"
	"github.com/akordium-id/dukungps/internal/protocol/teltonika"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type e2eEnv struct {
	positionRepo repository.PositionRepository
	deviceRepo   repository.DeviceRepository
	hub          *websocket.Hub
	cacheImpl    cache.Cache
	positionSvc  *usecase.PositionService
}

func TestTeltonikaEndToEnd(t *testing.T) {
	env := startTestEnv(t)
	ctx := context.Background()

	// TCP server on random port, using the PRODUCTION handler with auto-register
	registry := protocol.NewRegistry()
	registry.Register(&teltonika.Protocol{})

	tcpServer := protocol.NewServer(registry)
	tcpServer.RegisterPort("teltonika", 0)
	tcpServer.SetHandler("teltonika", usecase.NewDevicePositionHandler(env.positionSvc, env.deviceRepo, true))
	require.NoError(t, tcpServer.Start())
	defer tcpServer.Stop()

	addr := tcpServer.Addr("teltonika")
	require.NotNil(t, addr)

	// subscribe websocket client before device sends data
	wsClient := websocket.NewClient(env.hub)
	env.hub.Register(wsClient)

	// mock device connects
	conn, err := net.Dial("tcp", addr.String())
	require.NoError(t, err)
	defer conn.Close()

	// 1. identification (device NOT pre-registered -> exercises auto-register)
	imei := []byte("123456789012345")
	login := append([]byte{0x00, 0x0F}, imei...)
	_, err = conn.Write(login)
	require.NoError(t, err)

	ack := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadFull(conn, ack)
	require.NoError(t, err)
	assert.Equal(t, byte(0x01), ack[0])

	// 2. location data (codec 8, 1 record)
	data := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x20,
		0x08,
		0x01,
		0x00, 0x00, 0x01, 0x5F, 0x93, 0x03, 0x7C, 0x00,
		0x00,
		0x0E, 0x50, 0xCC, 0x48,
		0x20, 0xBF, 0xFE, 0xBC,
		0x00, 0x83,
		0x01, 0x0E,
		0x08,
		0x00, 0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
	}
	_, err = conn.Write(data)
	require.NoError(t, err)

	count := make([]byte, 4)
	_, err = io.ReadFull(conn, count)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x01}, count)

	// 3. device auto-registered on first contact
	var devID int64
	require.Eventually(t, func() bool {
		dev, err := env.deviceRepo.FindByUniqueID(ctx, "123456789012345")
		if err != nil {
			return false
		}
		devID = dev.ID
		return true
	}, 5*time.Second, 100*time.Millisecond)
	assert.True(t, devID > 0)

	// 4. position persisted
	var positions []position.Position
	require.Eventually(t, func() bool {
		var err error
		positions, err = env.positionRepo.FindByDeviceID(ctx, devID, 10)
		return err == nil && len(positions) == 1
	}, 5*time.Second, 100*time.Millisecond)
	require.Len(t, positions, 1)
	assert.InDelta(t, 24.01762, positions[0].Longitude, 0.0001)
	assert.InDelta(t, 54.94535, positions[0].Latitude, 0.0001)

	// 5. websocket broadcast
	select {
	case msg := <-wsClient.SendChannel():
		assert.Contains(t, string(msg), "deviceId")
	case <-time.After(2 * time.Second):
		t.Fatal("no websocket broadcast received")
	}
}

func TestTeltonikaRejectsUnknownDeviceWhenAutoRegisterDisabled(t *testing.T) {
	env := startTestEnv(t)
	ctx := context.Background()

	registry := protocol.NewRegistry()
	registry.Register(&teltonika.Protocol{})

	tcpServer := protocol.NewServer(registry)
	tcpServer.RegisterPort("teltonika", 0)
	// auto-register OFF: unknown device must be dropped
	tcpServer.SetHandler("teltonika", usecase.NewDevicePositionHandler(env.positionSvc, env.deviceRepo, false))
	require.NoError(t, tcpServer.Start())
	defer tcpServer.Stop()

	addr := tcpServer.Addr("teltonika")
	require.NotNil(t, addr)

	conn, err := net.Dial("tcp", addr.String())
	require.NoError(t, err)
	defer conn.Close()

	imei := []byte("987654321098765")
	_, err = conn.Write(append([]byte{0x00, 0x0F}, imei...))
	require.NoError(t, err)

	ack := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadFull(conn, ack)
	require.NoError(t, err)
	assert.Equal(t, byte(0x01), ack[0])

	// device frame accepted by decoder, but position dropped -> no DB row, no device row
	data := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x20,
		0x08,
		0x01,
		0x00, 0x00, 0x01, 0x5F, 0x93, 0x03, 0x7C, 0x00,
		0x00,
		0x0E, 0x50, 0xCC, 0x48,
		0x20, 0xBF, 0xFE, 0xBC,
		0x00, 0x83,
		0x01, 0x0E,
		0x08,
		0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	_, err = conn.Write(data)
	require.NoError(t, err)

	_, err = io.ReadFull(conn, make([]byte, 4))
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	_, err = env.deviceRepo.FindByUniqueID(ctx, "987654321098765")
	assert.Error(t, err, "unknown device must not be auto-created when auto-register is disabled")
}

func startTestEnv(t *testing.T) *e2eEnv {
	t.Helper()

	ctx := context.Background()

	// postgres
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "dukungps_test",
			"POSTGRES_USER":     "dukungps",
			"POSTGRES_PASSWORD": "dukungps",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}
	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	host, err := pg.Host(ctx)
	require.NoError(t, err)
	port, err := pg.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := "postgres://dukungps:dukungps@" + host + ":" + port.Port() + "/dukungps_test?sslmode=disable"
	pool, err := database.NewPool(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	schema, err := os.ReadFile("../../migrations/001_initial_schema.sql")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(schema))
	require.NoError(t, err)

	// redis (in-memory)
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})

	hub := websocket.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	positionRepo := repository.NewPositionRepo(pool)
	deviceRepo := repository.NewDeviceRepo(pool)
	cacheImpl := cache.NewCache(redisClient)
	positionSvc := usecase.NewPositionService(positionRepo, deviceRepo, cacheImpl, hub)

	return &e2eEnv{
		positionRepo: positionRepo,
		deviceRepo:   deviceRepo,
		hub:          hub,
		cacheImpl:    cacheImpl,
		positionSvc:  positionSvc,
	}
}
