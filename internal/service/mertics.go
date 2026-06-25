package service

import (
	"fmt"
	"strconv"

	models "github.com/mersikovs/korob.git/internal/model"
)

type MetricService struct {
	repo models.Storage
}

func NewMetricService(repo models.Storage) *MetricService {
	return &MetricService{repo: repo}
}

func (m *MetricService) UpdateMetric(metricType, metricName, metricValue string) error {

	switch metricType {
	case models.Counter:
		counter, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении метрики %s: %v", metricName, err)
		}

		metric, err := m.repo.Get(metricType, metricName)
		if err != nil {
			fmt.Printf("error")
		}

		if err == nil && metric != nil && metric.Delta != nil {
			counter += *metric.Delta
		}

		m.repo.Save(metricType, metricName, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Delta: &counter,
			Value: nil,
		})

		return nil
	case models.Gauge:
		gauge, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении метрики %s: %v", metricName, err)
		}

		m.repo.Save(metricType, metricName, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Delta: nil,
			Value: &gauge,
		})
		return nil
	}

	return fmt.Errorf("неизвестный тип метрики %s", metricType)
}
