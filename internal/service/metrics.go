// Package service storage, caching, and audit publishing
package service

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"

	"github.com/mersikovs/korob.git/internal/audit"
	"github.com/mersikovs/korob.git/internal/cache"
	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/repository"
)

// Pinger is an optional interface for storage health checks.
type Pinger interface {
	Ping(ctx context.Context) error
}

// MetricService orchestrates metric operations with storage, caching, and audit publishing.
type MetricService struct {
	cache cache.MetricsCache
	// Publisher dispatches audit events for metric changes.
	Publisher *audit.Dispatcher
	repo      repository.Storage
	// Logger records service operations and errors.
	Logger logger.Logger
}

// NewMetricService creates a MetricService wiring storage, audit dispatcher, cache, and logger.
func NewMetricService(repo repository.Storage, dispatcher *audit.Dispatcher, cache cache.MetricsCache, log logger.Logger) *MetricService {
	return &MetricService{
		cache:     cache,
		Publisher: dispatcher,
		repo:      repo,
		Logger:    log,
	}
}

// GetMetric returns the string value of a metric identified by type and name.
func (m *MetricService) GetMetric(ctx context.Context, metricType, metricName string) (string, error) {
	metric, err := m.repo.Get(ctx, metricType, metricName)
	if err != nil {
		return "", fmt.Errorf("неизвестная метрика %s", metricName)
	}

	switch metricType {
	case models.Counter:
		if metric.Delta == nil {
			return "", fmt.Errorf("не установлено значение %s %s", metricType, metricName)
		}

		return fmt.Sprintf("%d", *metric.Delta), nil
	case models.Gauge:
		if metric.Value == nil {
			return "", fmt.Errorf("не установлено значение %s %s", metricType, metricName)
		}

		return strconv.FormatFloat(*metric.Value, 'f', -1, 64), nil
	}

	return "", fmt.Errorf("неизвестный тип метрики %s", metricType)
}

// UpdateMetric updates a single metric by type, name, and string value.
func (m *MetricService) UpdateMetric(ctx context.Context, metricType, metricName, metricValue string) error {

	switch metricType {
	case models.Counter:
		counter, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении метрики %s: %v", metricName, err)
		}

		metric, err := m.repo.Get(ctx, metricType, metricName)
		if err != nil {
			return fmt.Errorf("ошибка при получении метрики %s: %v", metricName, err)
		}

		if metric.Delta != nil {
			counter += *metric.Delta
		}

		err = m.repo.Save(ctx, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Delta: &counter,
			Value: nil,
		})

		if err != nil {
			return fmt.Errorf("ошибка при сохранении метрики %s: %v", metricName, err)
		}

		return nil
	case models.Gauge:
		gauge, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении метрики %s: %v", metricName, err)
		}

		err = m.repo.Save(ctx, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Delta: nil,
			Value: &gauge,
		})

		if err != nil {
			return fmt.Errorf("ошибка при сохранении метрики %s: %v", metricName, err)
		}

		return nil
	}

	return fmt.Errorf("неизвестный тип метрики %s", metricType)
}

// ListMetrics returns the names of all metrics in storage, using cache when available.
func (m *MetricService) ListMetrics(ctx context.Context) ([]string, error) {
	if cached, ok := m.cache.Get("ListAllMetric"); ok {
		return cached, nil
	}
	metrics, err := m.repo.GetNamesList(ctx)
	if err != nil {
		return nil, err
	}
	m.cache.Set("ListAllMetric", metrics)
	return metrics, nil
}

// UpdateMetricFromStruct updates a single metric identified by type and name.
func (m *MetricService) UpdateMetricFromStruct(ctx context.Context, metricType, metricName string, metricValue models.Metrics) error {

	switch metricType {
	case models.Counter:

		if metricValue.Delta == nil {
			return fmt.Errorf("ошибка при обновлении метрики %s", metricName)
		}

		metric, err := m.repo.Get(ctx, metricType, metricName)
		if err != nil {
			return fmt.Errorf("ошибка при получении метрики %s: %v", metricName, err)
		}

		var counter int64
		if metric.Delta != nil {
			counter = counter + *metric.Delta
		}
		if metricValue.Delta != nil {
			counter = counter + *metricValue.Delta
		}

		err = m.repo.Save(ctx, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Delta: &counter,
		})

		if err != nil {
			return fmt.Errorf("ошибка при сохранении метрики %s: %v", metricName, err)
		}

		return nil
	case models.Gauge:

		err := m.repo.Save(ctx, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Value: metricValue.Value,
		})

		if err != nil {
			return fmt.Errorf("ошибка при сохранении метрики %s: %v", metricName, err)
		}

		return nil
	}

	return fmt.Errorf("неизвестный тип метрики %s", metricType)
}

// UpdateMetricsFromStruct performs a batch update of multiple metrics in storage.
func (m *MetricService) UpdateMetricsFromStruct(ctx context.Context, metrics []models.Metrics) error {
	metricsForSave := make([]models.Metrics, 0)
	allReadyAdd := make(map[string]struct{})
	counterSum := make(map[string]int64)
	slices.Reverse(metrics)
	for _, metric := range metrics {
		metricType := metric.MType
		switch metricType {
		case models.Counter:

			if metric.Delta == nil {
				m.Logger.Info("не валидная метрика %s", metric.ID)
				continue
			}

			counterSum[metric.ID] += *metric.Delta
		case models.Gauge:
			if _, find := allReadyAdd[models.Gauge+metric.ID]; find {
				continue
			}
			if metric.Value == nil {
				m.Logger.Info("не валидная метрика %s", metric.ID)
				continue
			}
			metricsForSave = append(metricsForSave, metric)
			allReadyAdd[models.Gauge+metric.ID] = struct{}{}
		}
	}

	for id, value := range counterSum {
		m := models.Metrics{
			ID:    id,
			Delta: &value,
			MType: models.Counter,
			Value: nil,
		}
		metricsForSave = append(metricsForSave, m)
	}

	sort.Slice(metricsForSave, func(i, j int) bool {
		if metricsForSave[i].ID != metricsForSave[j].ID {
			return metricsForSave[i].ID < metricsForSave[j].ID
		}
		return metricsForSave[i].MType < metricsForSave[j].MType
	})

	return m.repo.BatchSave(ctx, metricsForSave)
}

// Ping checks the health of the underlying storage.
func (m *MetricService) Ping(ctx context.Context) error {
	pinger, ok := m.repo.(Pinger)
	if !ok {
		return nil
	}

	return pinger.Ping(ctx)
}
