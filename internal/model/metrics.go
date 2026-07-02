package models

import (
	"slices"
	"sync"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

type MemoryStorage struct {
	mu      sync.RWMutex
	Metrics map[string]map[string]Metrics
}

type Storage interface {
	Get(mType, name string) (Metrics, error)
	GetNamesList() []string
	Save(mType, name string, m Metrics) error
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
	s.mu.RLock()

	list := make([]string, 0)
	for _, mNames := range s.Metrics {
		for name := range mNames {
			list = append(list, name)
		}
	}

	s.mu.RUnlock()

	slices.Sort(list)
	return list
}

func (s *MemoryStorage) Get(mType, name string) (Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	typeMap, ok := s.Metrics[mType]
	if !ok {
		return Metrics{}, nil
	}

	metric, ok := typeMap[name]
	if !ok {
		return Metrics{}, nil
	}
	return metric, nil
}

func (s *MemoryStorage) Save(mType, name string, m Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Metrics[mType]; !ok {
		s.Metrics[mType] = make(map[string]Metrics)
	}

	s.Metrics[mType][name] = m

	return nil
}
