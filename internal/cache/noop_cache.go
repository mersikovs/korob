package cache

type noopCache struct{}

func (noopCache) Get(string) ([]string, bool) { return nil, false }
func (noopCache) Set(string, []string)        {}

func NewNoopCache() MetricsCache {
	return noopCache{}
}
