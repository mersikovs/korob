package repository

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
)

// MemoryStorage is an in-memory implementation of Storage with optional
// periodic persistence to disk.
type MemoryStorage struct {
	mu sync.RWMutex
	// Metrics is a two-level map keyed by metric type and name.
	Metrics       map[string]map[string]models.Metrics
	filePath      string
	storeInterval atomic.Int64
}

// NewMemoryStorage creates an in-memory storage
func NewMemoryStorage(ctx context.Context, cnf *models.ServerConfig, logger logger.Logger) (*MemoryStorage, error) {
	ms := MemoryStorage{
		Metrics:  make(map[string]map[string]models.Metrics),
		filePath: cnf.FileStoragePath,
	}

	ms.StartPeriodicSave(ctx, cnf.StoreInterval, logger)
	if cnf.Restore {
		err := ms.Restore()
		if err != nil {
			logger.Info("произошла ошибка восстановления данных %w", err)
		}
	}

	return &ms, nil
}

// GetNamesList returns all metric names from memory.
func (s *MemoryStorage) GetNamesList(_ context.Context) ([]string, error) {
	s.mu.RLock()

	list := make([]string, 0)
	for _, mNames := range s.Metrics {
		for name := range mNames {
			list = append(list, name)
		}
	}

	s.mu.RUnlock()

	slices.Sort(list)
	return list, nil
}

// Get retrieves a single metric from memory by type and name.
func (s *MemoryStorage) Get(_ context.Context, mType, name string) (models.Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	typeMap, ok := s.Metrics[mType]
	if !ok {
		return models.Metrics{}, nil
	}

	metric, ok := typeMap[name]
	if !ok {
		return models.Metrics{}, nil
	}
	return metric, nil
}

// Save stores a single metric in memory by type and name.
func (s *MemoryStorage) Save(_ context.Context, m models.Metrics) error {
	s.mu.Lock()
	s.upsertLocked(m)
	s.mu.Unlock()

	return s.maybeStore()
}

// BatchSave persists multiple metrics in memory at once.
func (s *MemoryStorage) BatchSave(_ context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	s.mu.Lock()
	for _, m := range metrics {
		s.upsertLocked(m)
	}
	s.mu.Unlock()

	return s.maybeStore()
}

func (s *MemoryStorage) upsertLocked(m models.Metrics) {
	byID, ok := s.Metrics[m.MType]
	if !ok {
		byID = make(map[string]models.Metrics)
		s.Metrics[m.MType] = byID
	}

	if prev, found := byID[m.ID]; found &&
		m.MType == models.Counter &&
		m.Delta != nil && prev.Delta != nil {
		sum := *m.Delta + *prev.Delta
		m.Delta = &sum
	}

	byID[m.ID] = m
}

func (s *MemoryStorage) maybeStore() error {
	if s.storeInterval.Load() == 0 {
		return s.Store()
	}
	return nil
}

// Store persists the current in-memory metrics to disk.
func (s *MemoryStorage) Store() error {
	s.mu.RLock()
	metrics := s.Metrics
	s.mu.RUnlock()
	data := make([]models.Metrics, 0)

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

// Restore loads persisted metrics from disk into memory at startup.
func (s *MemoryStorage) Restore() error {

	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var data []models.Metrics

	err = json.Unmarshal(b, &data)
	if err != nil {
		return err
	}

	metrics := make(map[string]map[string]models.Metrics)

	for _, m := range data {
		_, find := metrics[m.MType]
		if !find {
			metrics[m.MType] = make(map[string]models.Metrics)
		}

		metrics[m.MType][m.ID] = m
	}

	s.mu.Lock()
	s.Metrics = metrics
	defer s.mu.Unlock()

	return nil
}

// StartPeriodicSave launches a background goroutine to periodically
// persist in-memory metrics to disk.
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

// Ping is a no-op for MemoryStorage and always returns nil,
// as in-memory storage requires no connection check.
func (s *MemoryStorage) Ping(_ context.Context) error {
	return nil
}
