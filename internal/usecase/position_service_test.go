package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/device"
	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/internal/infrastructure/cache"
	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
	"github.com/akordium-id/dukungps/internal/repository"
)

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

func (m *mockPositionRepo) Search(ctx context.Context, params repository.PositionSearchParams) ([]position.Position, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]position.Position), args.Error(1)
}

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

func TestPositionService_ProcessPosition(t *testing.T) {
	posRepo := &mockPositionRepo{}
	posRepo.On("Create", mock.Anything, mock.AnythingOfType("*position.Position")).Return(nil)

	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	cacheImpl := cache.NewCache(newMiniredis(t))

	svc := NewPositionService(posRepo, nil, cacheImpl, hub)

	p := &position.Position{
		DeviceID:  1,
		Latitude:  -7.2456,
		Longitude: 112.7389,
		Valid:     true,
		Time:      time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, svc.ProcessPosition(context.Background(), p))

	posRepo.AssertCalled(t, "Create", mock.Anything, p)

	got, err := cacheImpl.GetPosition(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, -7.2456, got.Latitude)
}

func TestPositionService_PositionHook(t *testing.T) {
	posRepo := &mockPositionRepo{}
	posRepo.On("Create", mock.Anything, mock.AnythingOfType("*position.Position")).Return(nil)

	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	cacheImpl := cache.NewCache(newMiniredis(t))
	svc := NewPositionService(posRepo, nil, cacheImpl, hub)

	hookCalled := false
	var capturedID int64
	svc.RegisterPositionHook(func(ctx context.Context, p *position.Position) error {
		hookCalled = true
		capturedID = p.DeviceID
		return nil
	})

	p := &position.Position{
		DeviceID:  42,
		Latitude:  -7.25,
		Longitude: 112.75,
		Valid:     true,
	}

	err := svc.ProcessPosition(context.Background(), p)
	require.NoError(t, err)
	assert.True(t, hookCalled)
	assert.Equal(t, int64(42), capturedID)
}

func TestPositionService_ProcessPositionBroadcast(t *testing.T) {
	posRepo := &mockPositionRepo{}
	posRepo.On("Create", mock.Anything, mock.AnythingOfType("*position.Position")).Return(nil)

	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	client := websocket.NewClient(hub)
	hub.Register(client)

	svc := NewPositionService(posRepo, nil, cache.NewCache(newMiniredis(t)), hub)

	p := &position.Position{DeviceID: 2, Valid: true, Latitude: -7.1, Longitude: 112.7}
	require.NoError(t, svc.ProcessPosition(context.Background(), p))

	select {
	case data := <-client.SendChannel():
		assert.Contains(t, string(data), "deviceId")
	case <-time.After(1 * time.Second):
		t.Fatal("hub did not broadcast position")
	}
}

func TestPositionService_PreFilter(t *testing.T) {
	posRepo := &mockPositionRepo{}
	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	cacheImpl := cache.NewCache(newMiniredis(t))
	svc := NewPositionService(posRepo, nil, cacheImpl, hub)

	// Filter that drops any position with speed > 100
	svc.RegisterPreFilter(func(ctx context.Context, p *position.Position, last *position.Position) (bool, error) {
		if p.Speed > 100 {
			return false, nil // drop
		}
		if p.Attributes == nil {
			p.Attributes = make(map[string]any)
		}
		p.Attributes["enriched"] = true
		return true, nil
	})

	// Case 1: Drop position
	dropPos := &position.Position{DeviceID: 99, Valid: true, Speed: 150}
	err := svc.ProcessPosition(context.Background(), dropPos)
	require.NoError(t, err)
	posRepo.AssertNotCalled(t, "Create", mock.Anything, dropPos)

	// Case 2: Keep & Enrich position
	posRepo.On("Create", mock.Anything, mock.AnythingOfType("*position.Position")).Return(nil)
	keepPos := &position.Position{DeviceID: 99, Valid: true, Speed: 60}
	err = svc.ProcessPosition(context.Background(), keepPos)
	require.NoError(t, err)
	posRepo.AssertCalled(t, "Create", mock.Anything, keepPos)
	assert.Equal(t, true, keepPos.Attributes["enriched"])
}

func newMiniredis(t *testing.T) *redis.Client {
	t.Helper()
	s := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: s.Addr()})
}

var _ repository.PositionRepository = (*mockPositionRepo)(nil)
var _ repository.DeviceRepository = (*mockDeviceRepo)(nil)
