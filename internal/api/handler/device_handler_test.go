package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/repository"
)

type mockDeviceRepo struct {
	mock.Mock
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *device.Device) error {
	args := m.Called(ctx, d)
	return args.Error(0)
}

func (m *mockDeviceRepo) FindByUniqueID(ctx context.Context, uniqueID string) (*device.Device, error) {
	args := m.Called(ctx, uniqueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*device.Device), args.Error(1)
}

func (m *mockDeviceRepo) FindByID(ctx context.Context, id int64) (*device.Device, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*device.Device), args.Error(1)
}

func (m *mockDeviceRepo) FindAll(ctx context.Context) ([]*device.Device, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*device.Device), args.Error(1)
}

func (m *mockDeviceRepo) Update(ctx context.Context, d *device.Device) error {
	args := m.Called(ctx, d)
	return args.Error(0)
}

type mockPositionRepo struct {
	mock.Mock
}

func (m *mockPositionRepo) Create(ctx context.Context, p *position.Position) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *mockPositionRepo) FindByDeviceID(ctx context.Context, deviceID int64, limit int) ([]position.Position, error) {
	args := m.Called(ctx, deviceID, limit)
	return args.Get(0).([]position.Position), args.Error(1)
}

var _ repository.DeviceRepository = (*mockDeviceRepo)(nil)
var _ repository.PositionRepository = (*mockPositionRepo)(nil)

func TestDeviceHandlerList(t *testing.T) {
	repo := &mockDeviceRepo{}
	repo.On("FindAll", mock.Anything).Return([]*device.Device{
		{ID: 1, UniqueID: "123", Name: "Truk 01", Protocol: "teltonika"},
	}, nil)

	app := fiber.New()
	handler := NewDeviceHandler(repo)
	app.Get("/api/v1/devices", handler.List)

	req := httptest.NewRequest("GET", "/api/v1/devices", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var devices []*device.Device
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&devices))
	require.Len(t, devices, 1)
	assert.Equal(t, "Truk 01", devices[0].Name)
}

func TestDeviceHandlerGetNotFound(t *testing.T) {
	repo := &mockDeviceRepo{}
	repo.On("FindByID", mock.Anything, int64(99)).Return(nil, repository.ErrDeviceNotFound)

	app := fiber.New()
	handler := NewDeviceHandler(repo)
	app.Get("/api/v1/devices/:id", handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/devices/99", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}
