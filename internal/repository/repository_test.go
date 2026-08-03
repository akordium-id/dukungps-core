package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/infrastructure/database"
)

func TestDeviceRepo(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	repo := NewDeviceRepo(pool)

	d := &device.Device{
		UniqueID: "123456789012345",
		Name:     "Truk 01",
		Model:    "FMB920",
		Protocol: "teltonika",
		Status:   device.StatusOnline,
	}
	require.NoError(t, repo.Create(ctx, d))
	assert.True(t, d.ID > 0)

	got, err := repo.FindByUniqueID(ctx, "123456789012345")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Truk 01", got.Name)
	assert.Equal(t, "teltonika", got.Protocol)

	// duplicate unique_id rejected
	dup := &device.Device{UniqueID: "123456789012345", Name: "Dup", Protocol: "teltonika"}
	assert.Error(t, repo.Create(ctx, dup))

	// update
	d.Name = "Truk 01 Updated"
	require.NoError(t, repo.Update(ctx, d))
	got, err = repo.FindByUniqueID(ctx, "123456789012345")
	require.NoError(t, err)
	assert.Equal(t, "Truk 01 Updated", got.Name)
}

func TestPositionRepo(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	deviceRepo := NewDeviceRepo(pool)
	d := &device.Device{UniqueID: "987654321098765", Name: "Mobil", Protocol: "gt06"}
	require.NoError(t, deviceRepo.Create(ctx, d))

	posRepo := NewPositionRepo(pool)

	p := &position.Position{
		DeviceID:   d.ID,
		Latitude:   -7.2456,
		Longitude:  112.7389,
		Speed:      42.0,
		Course:     180.0,
		Valid:      true,
		Time:       time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
		Attributes: map[string]any{"ignition": true, "battery": 12.5},
	}
	require.NoError(t, posRepo.Create(ctx, p))
	assert.True(t, p.ID > 0)

	positions, err := posRepo.FindByDeviceID(ctx, d.ID, 10)
	require.NoError(t, err)
	require.Len(t, positions, 1)
	assert.Equal(t, -7.2456, positions[0].Latitude)
	assert.Equal(t, true, positions[0].Attributes["ignition"])
	assert.InDelta(t, 12.5, positions[0].Attributes["battery"], 0.001)
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	t.Parallel()

	ctx := context.Background()

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

	t.Cleanup(func() {
		_ = pg.Terminate(ctx)
	})

	host, err := pg.Host(ctx)
	require.NoError(t, err)
	port, err := pg.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := "postgres://dukungps:dukungps@" + host + ":" + port.Port() + "/dukungps_test?sslmode=disable"

	pool, err := database.NewPool(ctx, dsn)
	require.NoError(t, err)

	schema, err := os.ReadFile("../../migrations/001_initial_schema.sql")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(schema))
	require.NoError(t, err)

	return pool
}
