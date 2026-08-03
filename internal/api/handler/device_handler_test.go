package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/repository"
	"github.com/akordium-id/dukungps/internal/usecase"
)

type mockDeviceUseCase struct {
	mock.Mock
}

func (m *mockDeviceUseCase) ListDevices(ctx context.Context) ([]*device.Device, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*device.Device), args.Error(1)
}

func (m *mockDeviceUseCase) GetDevice(ctx context.Context, id int64) (*device.Device, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*device.Device), args.Error(1)
}

func (m *mockDeviceUseCase) CreateDevice(ctx context.Context, d *device.Device) error {
	args := m.Called(ctx, d)
	return args.Error(0)
}

func (m *mockDeviceUseCase) UpdateDevice(ctx context.Context, d *device.Device) error {
	args := m.Called(ctx, d)
	return args.Error(0)
}

var _ usecase.DeviceUseCase = (*mockDeviceUseCase)(nil)

type mockPositionUseCase struct {
	mock.Mock
}

func (m *mockPositionUseCase) GetPositionsByDevice(ctx context.Context, deviceID int64, limit int) ([]position.Position, error) {
	args := m.Called(ctx, deviceID, limit)
	return args.Get(0).([]position.Position), args.Error(1)
}

func (m *mockPositionUseCase) ProcessPosition(ctx context.Context, p *position.Position) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

var _ usecase.PositionUseCase = (*mockPositionUseCase)(nil)

func TestDeviceHandlerList(t *testing.T) {
	uc := &mockDeviceUseCase{}
	uc.On("ListDevices", mock.Anything).Return([]*device.Device{
		{ID: 1, UniqueID: "123", Name: "Truk 01", Protocol: "teltonika"},
	}, nil)

	app := fiber.New()
	h := NewDeviceHandler(uc)
	app.Get("/api/v1/devices", h.List)

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
	uc := &mockDeviceUseCase{}
	uc.On("GetDevice", mock.Anything, int64(99)).Return(nil, repository.ErrDeviceNotFound)

	app := fiber.New()
	h := NewDeviceHandler(uc)
	app.Get("/api/v1/devices/:id", h.Get)

	req := httptest.NewRequest("GET", "/api/v1/devices/99", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestDeviceHandlerCreateValidation(t *testing.T) {
	uc := &mockDeviceUseCase{}
	uc.On("CreateDevice", mock.Anything, mock.AnythingOfType("*device.Device")).
		Return(usecase.ErrDeviceValidation)

	app := fiber.New()
	h := NewDeviceHandler(uc)
	app.Post("/api/v1/devices", h.Create)

	body := `{"uniqueId":"","protocol":""}`
	req := httptest.NewRequest("POST", "/api/v1/devices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 422, resp.StatusCode)
}

func TestDeviceHandlerCreateConflict(t *testing.T) {
	uc := &mockDeviceUseCase{}
	uc.On("CreateDevice", mock.Anything, mock.AnythingOfType("*device.Device")).
		Return(repository.ErrDeviceExists)

	app := fiber.New()
	h := NewDeviceHandler(uc)
	app.Post("/api/v1/devices", h.Create)

	body := `{"uniqueId":"123","name":"Test","protocol":"teltonika"}`
	req := httptest.NewRequest("POST", "/api/v1/devices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode)
}

func TestDeviceHandlerCreateSuccess(t *testing.T) {
	uc := &mockDeviceUseCase{}
	uc.On("CreateDevice", mock.Anything, mock.AnythingOfType("*device.Device")).Return(nil)

	app := fiber.New()
	h := NewDeviceHandler(uc)
	app.Post("/api/v1/devices", h.Create)

	body := `{"uniqueId":"123","name":"Truk 01","protocol":"teltonika"}`
	req := httptest.NewRequest("POST", "/api/v1/devices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	uc.AssertCalled(t, "CreateDevice", mock.Anything, mock.Anything)
}

func TestPositionHandlerListByDevice(t *testing.T) {
	uc := &mockPositionUseCase{}
	uc.On("GetPositionsByDevice", mock.Anything, int64(1), 100).
		Return([]position.Position{{ID: 10, DeviceID: 1}}, nil)

	app := fiber.New()
	h := NewPositionHandler(uc, nil)
	app.Get("/api/v1/devices/:deviceId/positions", h.ListByDevice)

	req := httptest.NewRequest("GET", "/api/v1/devices/1/positions", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPositionHandlerListByDeviceError(t *testing.T) {
	uc := &mockPositionUseCase{}
	uc.On("GetPositionsByDevice", mock.Anything, int64(1), 100).
		Return([]position.Position(nil), errors.New("db error"))

	app := fiber.New()
	h := NewPositionHandler(uc, nil)
	app.Get("/api/v1/devices/:deviceId/positions", h.ListByDevice)

	req := httptest.NewRequest("GET", "/api/v1/devices/1/positions", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}
