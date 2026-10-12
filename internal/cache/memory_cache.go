package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
	gocache "github.com/patrickmn/go-cache"
)

type goCacheMetricsCache struct {
	c *gocache.Cache
}

func (g *goCacheMetricsCache) Get(key string) ([]string, bool) {
	if v, ok := g.c.Get(key); ok {
		return v.([]string), true
	}
	return nil, false
}

func (g *goCacheMetricsCache) Set(key string, value []string) {
	g.c.Set(key, value, cache.DefaultExpiration)
}

func NewGoCache(defaultTTL, cleanupInterval time.Duration) MetricsCache {
	return &goCacheMetricsCache{
		c: gocache.New(defaultTTL, cleanupInterval),
	}
}
