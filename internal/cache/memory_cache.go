package cache

import (
	"time"

	gomemcache "github.com/patrickmn/go-cache"
)

type goCacheMetricsCache struct {
	c *gomemcache.Cache
}

// Get retrieves the cached metric names for the given key, returning nil if not found or expired.
func (g *goCacheMetricsCache) Get(key string) ([]string, bool) {
	if v, ok := g.c.Get(key); ok {
		return v.([]string), true
	}
	return nil, false
}

// Set stores the metric names in the cache with the default TTL.
func (g *goCacheMetricsCache) Set(key string, value []string) {
	g.c.Set(key, value, gomemcache.DefaultExpiration)
}

// NewGoCache creates an in-memory cache with the specified default TTL
// and background cleanup interval for expired entries.
func NewGoCache(defaultTTL, cleanupInterval time.Duration) MetricsCache {
	return &goCacheMetricsCache{
		c: gomemcache.New(defaultTTL, cleanupInterval),
	}
}
