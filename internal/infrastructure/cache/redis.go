package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/akordium-id/dukungps/internal/domain/position"
)

var ErrPositionNotFound = errors.New("position not found in cache")

const (
	onlineTTL = 90 * time.Second
)

type Cache interface {
	SetPosition(ctx context.Context, deviceID int64, p *position.Position) error
	GetPosition(ctx context.Context, deviceID int64) (*position.Position, error)
	SetDeviceOnline(ctx context.Context, deviceID int64) error
	IsDeviceOnline(ctx context.Context, deviceID int64) (bool, error)
}

type RedisCache struct {
	client *redis.Client
}

func NewCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) SetPosition(ctx context.Context, deviceID int64, p *position.Position) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, positionKey(deviceID), data, 0).Err()
}

func (c *RedisCache) GetPosition(ctx context.Context, deviceID int64) (*position.Position, error) {
	data, err := c.client.Get(ctx, positionKey(deviceID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrPositionNotFound
	}
	if err != nil {
		return nil, err
	}
	p := &position.Position{}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *RedisCache) SetDeviceOnline(ctx context.Context, deviceID int64) error {
	return c.client.Set(ctx, onlineKey(deviceID), "1", onlineTTL).Err()
}

func (c *RedisCache) IsDeviceOnline(ctx context.Context, deviceID int64) (bool, error) {
	val, err := c.client.Get(ctx, onlineKey(deviceID)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

func positionKey(deviceID int64) string {
	return fmt.Sprintf("device:%d:position", deviceID)
}

func onlineKey(deviceID int64) string {
	return fmt.Sprintf("device:%d:online", deviceID)
}
