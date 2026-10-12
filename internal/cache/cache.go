package cache

type MetricsCache interface {
	Get(key string) ([]string, bool)
	Set(key string, value []string)
}
