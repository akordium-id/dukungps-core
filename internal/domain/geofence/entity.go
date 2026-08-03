package geofence

import (
	"math"
	"time"
)

const (
	TypePolygon = "polygon"
	TypeCircle  = "circle"
)

type Coordinate struct {
	Latitude  float64
	Longitude float64
}

type Geofence struct {
	ID          int64
	Name        string
	Description string
	Type        string
	Area        []Coordinate
	Center      Coordinate
	Radius      float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (g *Geofence) Contains(lat, lon float64) bool {
	switch g.Type {
	case TypeCircle:
		return g.containsCircle(lat, lon)
	default:
		return g.containsPolygon(lat, lon)
	}
}

func (g *Geofence) containsPolygon(lat, lon float64) bool {
	if len(g.Area) < 3 {
		return false
	}

	inside := false
	j := len(g.Area) - 1
	for i := 0; i < len(g.Area); i++ {
		vi := g.Area[i]
		vj := g.Area[j]
		if (vi.Latitude > lat) != (vj.Latitude > lat) &&
			lon < (vj.Longitude-vi.Longitude)*(lat-vi.Latitude)/(vj.Latitude-vi.Latitude)+vi.Longitude {
			inside = !inside
		}
		j = i
	}
	return inside
}

func (g *Geofence) containsCircle(lat, lon float64) bool {
	if g.Radius <= 0 {
		return false
	}

	const earthRadiusM = 6371000.0

	lat1 := g.Center.Latitude * math.Pi / 180
	lat2 := lat * math.Pi / 180
	dLat := (lat - g.Center.Latitude) * math.Pi / 180
	dLon := (lon - g.Center.Longitude) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusM*c <= g.Radius
}
