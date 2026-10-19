package alert

import (
	"time"
)

const (
	AlertTypeGeofenceEnter = "geofence.enter"
	AlertTypeGeofenceExit  = "geofence.exit"
	AlertTypeSpeed         = "speed"
	AlertTypeDeviceOffline = "device.offline"
	AlertTypeCustom        = "custom"
)

const (
	AlertStatusPending = "pending"
	AlertStatusSent    = "sent"
	AlertStatusFailed  = "failed"
)

type Alert struct {
	ID          int64
	DeviceID    int64
	Type        string
	Title       string
	Description string
	Status      string
	Metadata    map[string]any
	SentAt      time.Time
	CreatedAt   time.Time
}

func (a *Alert) IsValid() bool {
	return a.DeviceID > 0 && a.Type != ""
}
