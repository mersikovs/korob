package models

const (
	// Counter is the metric type for counters.
	Counter = "counter"
	// Gauge is the metric type for gauges.
	Gauge = "gauge"
)

// ValidMetricTypes contains all supported metric types.
// Use [ValidateMetricType] to check if a metric type is valid
var ValidMetricTypes = map[string]struct{}{
	Counter: {},
	Gauge:   {},
}

// Metrics represents a single metric data point.
type Metrics struct {
	// ID is the unique name of the metric
	ID string `json:"id"`
	// MType is the metric type
	MType string `json:"type"`
	// Delta is the value for "counter" metrics. Nil for "gauge".
	Delta *int64 `json:"delta,omitempty"`
	// Value is the value for "gauge" metrics. Nil for "counter".
	Value *float64 `json:"value,omitempty"`
	// Hash not use
	Hash string `json:"hash,omitempty"`
}

// ServerConfig holds the runtime configuration for the metrics server.
type ServerConfig struct {
	// Address is the network address to listen on
	Address string
	// AuditFile is the local file path for audit logging.
	AuditFile string
	// AuditURL is the remote HTTP endpoint for audit logging.
	AuditURL string
	// CacheTTLSec is the cache TTL in seconds (0 disables caching).
	CacheTTLSec int
	// Key is the secret key used for HMAC-SHA256 payload signing.
	Key []byte
	// DatabaseDSN is the PostgreSQL connection string.
	// If empty, the server falls back to in-memory storage.
	DatabaseDSN string
	// FileStoragePath is the path for periodic file-based persistence.
	FileStoragePath string
	// Restore indicates whether to load metrics from FileStoragePath on startup.
	Restore bool
	// StoreInterval is the period in seconds for saving metrics to FileStoragePath.
	StoreInterval int
}

// ValidateMetricType reports whether metricType is a known metric type
// according to [ValidMetricTypes].
func ValidateMetricType(metricType string) bool {
	_, ok := ValidMetricTypes[metricType]
	return ok
}
