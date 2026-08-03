package device

import (
	"strings"
	"time"
)

const (
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusUnknown = "unknown"
)

type Device struct {
	ID        int64
	UniqueID  string
	Name      string
	Model     string
	Protocol  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (d *Device) IsValid() bool {
	return strings.TrimSpace(d.UniqueID) != "" && d.Protocol != ""
}

type DeviceGroup struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (g *DeviceGroup) IsValid() bool {
	return strings.TrimSpace(g.Name) != ""
}
