package integration

import (
	"context"
	"encoding/binary"
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

	"github.com/akordium-id/dukungps/internal/domain/device"
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
}

func TestTeltonikaEndToEnd(t *testing.T) {
	env := startTestEnv(t)
	ctx := context.Background()

	// pre-register device
	d := &device.Device{UniqueID: "123456789012345", Name: "Truk 01", Protocol: "teltonika"}
	require.NoError(t, env.deviceRepo.Create(ctx, d))

	// TCP server on random port
	registry := protocol.NewRegistry()
	registry.Register(&teltonika.Protocol{})

	tcpServer := protocol.NewServer(registry)
	tcpServer.RegisterPort("teltonika", 0)
	tcpServer.SetHandler("teltonika", &testHandler{env: env, deviceRepo: env.deviceRepo})
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

	// 1. identification
	imei := []byte("123456789012345")
	login := append([]byte{0x00, 0x0F}, imei...)
	_, err = conn.Write(login)
	require.NoError(t, err)

	ack := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadFull(conn, ack)
	require.NoError(t, err)
	assert.Equal(t, byte(0x01), ack[0])

	// 2. location data (codec 8, 1 record, lat/lon Surabaya)
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

	// 3. verify position persisted (handler runs after ack in server goroutine)
	var positions []position.Position
	require.Eventually(t, func() bool {
		var err error
		positions, err = env.positionRepo.FindByDeviceID(ctx, d.ID, 10)
		return err == nil && len(positions) == 1
	}, 5*time.Second, 100*time.Millisecond)
	require.Len(t, positions, 1)
	assert.InDelta(t, 24.01762, positions[0].Longitude, 0.0001)
	assert.InDelta(t, 54.94535, positions[0].Latitude, 0.0001)

	// 4. verify websocket broadcast
	select {
	case msg := <-wsClient.SendChannel():
		assert.Contains(t, string(msg), "deviceId")
	case <-time.After(2 * time.Second):
		t.Fatal("no websocket broadcast received")
	}
}

type testHandler struct {
	env        *e2eEnv
	deviceRepo repository.DeviceRepository
}

func (h *testHandler) HandlePositions(deviceID string, res *protocol.DecodeResult) {
	for _, p := range res.Positions {
		if deviceID != "" {
			dev, err := h.deviceRepo.FindByUniqueID(context.Background(), deviceID)
			if err != nil {
				continue
			}
			p.DeviceID = dev.ID
		}
		_ = h.env.cacheImpl.SetPosition(context.Background(), p.DeviceID, p)
		_ = h.env.positionRepo.Create(context.Background(), p)
		h.env.hub.Broadcast(&websocket.PositionUpdate{DeviceID: p.DeviceID, Position: p})
	}
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

	return &e2eEnv{
		positionRepo: repository.NewPositionRepo(pool),
		deviceRepo:   repository.NewDeviceRepo(pool),
		hub:          hub,
		cacheImpl:    cache.NewCache(redisClient),
	}
}

var _ = binary.BigEndian
var _ = usecase.NewPositionService
