package models

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mersikovs/korob.git/internal/logger"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

type MemoryStorage struct {
	mu            sync.RWMutex
	Metrics       map[string]map[string]Metrics
	filePath      string
	storeInterval atomic.Int64
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

func NewStorage(filePath string) *MemoryStorage {
	return &MemoryStorage{
		Metrics:  make(map[string]map[string]Metrics),
		filePath: filePath,
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

	if _, ok := s.Metrics[mType]; !ok {
		s.Metrics[mType] = make(map[string]Metrics)
	}

	s.Metrics[mType][name] = m
	s.mu.Unlock()
	currentInterval := int(s.storeInterval.Load())
	if currentInterval == 0 {
		if err := s.Store(); err != nil {
			return err
		}
	}

	return nil
}

func (s *MemoryStorage) Store() error {
	s.mu.RLock()
	metrics := s.Metrics
	s.mu.RUnlock()
	data := make([]Metrics, 0)

	for _, m := range metrics {
		for _, v := range m {
			data = append(data, v)
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, jsonData, 0644)
}

func (s *MemoryStorage) Restore() error {

	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var data []Metrics

	err = json.Unmarshal(b, &data)
	if err != nil {
		return err
	}

	metrics := make(map[string]map[string]Metrics)

	for _, m := range data {
		_, find := metrics[m.MType]
		if !find {
			metrics[m.MType] = make(map[string]Metrics)
		}

		metrics[m.MType][m.ID] = m
	}

	s.mu.Lock()
	s.Metrics = metrics
	defer s.mu.Unlock()

	return nil
}

func (s *MemoryStorage) StartPeriodicSave(ctx context.Context, interval int, l logger.Logger) {
	if interval == 0 {
		return
	}

	s.storeInterval.Store(int64(interval))

	go func() {
		for {
			currentInterval := int(s.storeInterval.Load())

			if currentInterval <= 0 {
				l.Info("Остановим переодическое сохранение, вдруг в программе изменю интервал")
				return
			}

			time.Sleep(time.Duration(currentInterval) * time.Second)

			select {
			case <-ctx.Done():
				l.Info("остановка по контексту")
				return
			default:
			}

			if err := s.Store(); err != nil {
				l.Info("метрика не сохранена", "error", err)
			}
		}
	}()
}
