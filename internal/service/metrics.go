package service

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/repository"
)

type MetricService struct {
	Repo   repository.Storage
	Logger logger.Logger
}

func NewMetricService(repo repository.Storage, log logger.Logger) *MetricService {
	return &MetricService{
		Repo:   repo,
		Logger: log,
	}
}

func (m *MetricService) GetMetric(metricType, metricName string) (string, error) {
	metric, err := m.Repo.Get(metricType, metricName)
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

func (m *MetricService) UpdateMetric(metricType, metricName, metricValue string) error {

	switch metricType {
	case models.Counter:
		counter, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении метрики %s: %v", metricName, err)
		}

		metric, err := m.Repo.Get(metricType, metricName)
		if err != nil {
			return fmt.Errorf("ошибка при получении метрики %s: %v", metricName, err)
		}

		if metric.Delta != nil {
			counter += *metric.Delta
		}

		err = m.Repo.Save(metricType, metricName, models.Metrics{
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

		err = m.Repo.Save(metricType, metricName, models.Metrics{
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

func (m *MetricService) ListMetrics() []string {
	return m.Repo.GetNamesList()
}

func (m *MetricService) UpdateMetricFromStruct(metricType, metricName string, metricValue models.Metrics) error {

	switch metricType {
	case models.Counter:

		if metricValue.Delta == nil {
			return fmt.Errorf("ошибка при обновлении метрики %s", metricName)
		}

		metric, err := m.Repo.Get(metricType, metricName)
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

		err = m.Repo.Save(metricType, metricName, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Delta: &counter,
		})

		if err != nil {
			return fmt.Errorf("ошибка при сохранении метрики %s: %v", metricName, err)
		}

		return nil
	case models.Gauge:

		err := m.Repo.Save(metricType, metricName, models.Metrics{
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

func (m *MetricService) UpdateMetricsFromStruct(metrics []models.Metrics) error {
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

	return m.Repo.BatchSave(metricsForSave)
}
