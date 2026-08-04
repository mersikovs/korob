package models

const (
	Counter = "counter"
	Gauge   = "gauge"
)

var ValidMetricTypes = map[string]struct{}{
	Counter: {},
	Gauge:   {},
}

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

type ServerConfig struct {
	Address         string
	DatabaseDSN     string
	FileStoragePath string
	Restore         bool
	StoreInterval   int
}

func ValidateMetricType(metricType string) bool {
	_, ok := ValidMetricTypes[metricType]
	return ok
}
