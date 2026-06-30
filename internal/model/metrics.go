package models

import "slices"

const (
	Counter = "counter"
	Gauge   = "gauge"
)

type MemoryStorage struct {
	Metrics map[string]map[string]Metrics
}

type Storage interface {
	Get(mType, name string) (*Metrics, error)
	GetNamesList() []string
	Save(mType, name string, m Metrics) error
	Delete(mType, name string) error
}

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

func ValidateMetricType(metricType string) bool {
	_, ok := ValidMetricTypes[metricType]
	return ok
}

func NewStorage() *MemoryStorage {
	return &MemoryStorage{
		Metrics: make(map[string]map[string]Metrics),
	}
}

func (s *MemoryStorage) GetNamesList() []string {
	list := make([]string, 0)
	for _, mNames := range s.Metrics {
		for name := range mNames {
			list = append(list, name)
		}
	}
	slices.Sort(list)
	return list
}

func (s *MemoryStorage) Get(mType, name string) (*Metrics, error) {
	if _, ok := s.Metrics[mType]; !ok {
		return nil, nil
	}

	metric, ok := s.Metrics[mType][name]
	if !ok {
		return nil, nil
	}
	return &metric, nil
}

func (s *MemoryStorage) Save(mType, name string, m Metrics) error {
	if _, ok := s.Metrics[mType]; !ok {
		s.Metrics[mType] = make(map[string]Metrics)
	}

	s.Metrics[mType][name] = m

	return nil
}

func (s *MemoryStorage) Delete(mType, name string) error {
	if _, ok := s.Metrics[mType]; !ok {
		return nil
	}

	_, ok := s.Metrics[mType][name]
	if !ok {
		return nil
	}

	delete(s.Metrics[mType], name)

	return nil
}
