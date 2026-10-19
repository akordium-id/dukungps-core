package position

import (
	"time"
)

// Standardized Telemetry Attributes (Traccar & Fleet Management compatible)
const (
	AttrIgnition     = "ignition"       // bool: true = ACC ON, false = ACC OFF
	AttrBlocked      = "blocked"        // bool: true = cut-engine / relay active
	AttrMotion       = "motion"         // bool: true = device is moving
	AttrCharge       = "charge"         // bool: true = connected to external power / alternator
	AttrPower        = "power"          // float64: main external voltage (V)
	AttrBattery      = "battery"        // float64: backup internal battery voltage (V)
	AttrAlarm        = "alarm"          // string: alarm type (e.g., AlarmSOS, AlarmPowerCut)
	AttrDistance     = "distance"       // float64: trip delta distance in meters
	AttrOdometer     = "total_distance" // float64: cumulative odometer in meters
	AttrStatus       = "status"         // string: operational state ("moving", "idling", "stopped")
	AttrTripDistance = "trip_distance"  // float64: distance of current ongoing trip in meters
	AttrTripDuration = "trip_duration"  // int64: duration of current ongoing trip in seconds
	AttrIdleDuration = "idle_duration"  // int64: duration of continuous idling in seconds
)

// Standard Operational Statuses
const (
	StatusMoving  = "moving"  // ACC ON + Speed > 0 / Motion
	StatusIdling  = "idling"  // ACC ON + Speed == 0 (Engine running while stopped)
	StatusStopped = "stopped" // ACC OFF (Parked)
)

// Standard Alarm Types
const (
	AlarmSOS       = "sos"
	AlarmPowerCut  = "powerCut"
	AlarmOverspeed = "overspeed"
	AlarmLowPower  = "lowPower"
	AlarmVibration = "vibration"
)

type Position struct {
	ID         int64
	DeviceID   int64
	Latitude   float64
	Longitude  float64
	Altitude   float64
	Speed      float64
	Course     float64
	Valid      bool
	Time       time.Time
	DeviceTime time.Time
	Attributes map[string]any
	CreatedAt  time.Time
}

func (p *Position) IsValidFix() bool {
	return p.DeviceID > 0 &&
		p.Valid &&
		p.Latitude >= -90 && p.Latitude <= 90 &&
		p.Longitude >= -180 && p.Longitude <= 180
}

func (p *Position) EnsureAttributes() {
	if p.Attributes == nil {
		p.Attributes = make(map[string]any)
	}
}

func (p *Position) SetAttribute(key string, val any) {
	p.EnsureAttributes()
	p.Attributes[key] = val
}

func (p *Position) GetBool(key string) bool {
	if p.Attributes == nil {
		return false
	}
	if b, ok := p.Attributes[key].(bool); ok {
		return b
	}
	return false
}
