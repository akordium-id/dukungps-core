package cache

import (
	"log"
	"os"
	"strconv"

	"github.com/dgraph-io/ristretto/v2"
)

const (
	defaultRistrettoMaxCostMB = 50
)

func NewRistrettoCache() (*ristretto.Cache[string, any], error) {
	maxCost := defaultRistrettoMaxCostMB << 20
	if v := os.Getenv("CACHE_MAX_COST_MB"); v != "" {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			maxCost = mb << 20
		} else {
			log.Printf("cache: invalid CACHE_MAX_COST_MB %q, fallback to %d MB", v, defaultRistrettoMaxCostMB)
		}
	}

	return ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1e7,
		MaxCost:     int64(maxCost),
		BufferItems: 64,
	})
}
