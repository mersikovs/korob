package cache

type noopCache struct{}

func (noopCache) Get(string) ([]string, bool) { return nil, false }
func (noopCache) Set(string, []string)        {}

// NewNoopCache creates a no-op cache instance that always returns empty results.
func NewNoopCache() MetricsCache {
	return noopCache{}
}
