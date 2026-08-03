package position

import (
	"time"
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
