package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dgraph-io/ristretto"

	"github.com/akordium-id/dukungps/internal/domain/position"
)

type MultilevelCache struct {
	l1 *ristretto.Cache[string, any]
	l2 Cache
}

func NewMultilevelCache(l1 *ristretto.Cache[string, any], l2 Cache) *MultilevelCache {
	return &MultilevelCache{l1: l1, l2: l2}
}

func (c *MultilevelCache) SetPosition(ctx context.Context, deviceID int64, p *position.Position) error {
	if err := c.l2.SetPosition(ctx, deviceID, p); err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	c.l1.SetWithTTL(positionKey(deviceID), data, 1, 5*time.Minute)
	return nil
}

func (c *MultilevelCache) GetPosition(ctx context.Context, deviceID int64) (*position.Position, error) {
	if val, ok := c.l1.Get(positionKey(deviceID)); ok {
		p := &position.Position{}
		if err := json.Unmarshal(val.([]byte), p); err != nil {
			return nil, err
		}
		return p, nil
	}

	p, err := c.l2.GetPosition(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(p)
	c.l1.SetWithTTL(positionKey(deviceID), data, 1, 5*time.Minute)
	return p, nil
}

func (c *MultilevelCache) SetDeviceOnline(ctx context.Context, deviceID int64) error {
	if err := c.l2.SetDeviceOnline(ctx, deviceID); err != nil {
		return err
	}
	c.l1.SetWithTTL(onlineKey(deviceID), "1", 1, 60*time.Second)
	return nil
}

func (c *MultilevelCache) IsDeviceOnline(ctx context.Context, deviceID int64) (bool, error) {
	if val, ok := c.l1.Get(onlineKey(deviceID)); ok {
		return val.(string) == "1", nil
	}

	online, err := c.l2.IsDeviceOnline(ctx, deviceID)
	if err != nil {
		return false, err
	}

	c.l1.SetWithTTL(onlineKey(deviceID), map[bool]string{true: "1", false: "0"}[online], 1, 60*time.Second)
	return online, nil
}
