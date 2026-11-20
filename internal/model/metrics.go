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

// ValidateMetricType reports whether metricType is a known metric type
// according to [ValidMetricTypes].
func ValidateMetricType(metricType string) bool {
	_, ok := ValidMetricTypes[metricType]
	return ok
}
