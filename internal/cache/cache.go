package cache

// MetricsCache defines the contract for caching metric names by key,
// supporting multiple storage backends (in-memory, no-op, etc.).
type MetricsCache interface {
	Get(key string) ([]string, bool)
	Set(key string, value []string)
}
