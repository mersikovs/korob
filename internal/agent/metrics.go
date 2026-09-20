package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"runtime/metrics"
	"time"

	models "github.com/mersikovs/korob.git/internal/model"
)

type Sender interface {
	Do(req *http.Request) (*http.Response, error)
}

// AI генерация кода DirectMapping:
var DirectMapping = map[string]string{
	"/gc/heap/allocs:bytes":                       "TotalAlloc",
	"/gc/heap/allocs:objects":                     "Mallocs",
	"/gc/heap/frees:objects":                      "Frees",
	"/gc/heap/objects:objects":                    "HeapObjects",
	"/gc/heap/goals:bytes":                        "NextGC",
	"/gc/cycles/total:gc-cycles":                  "NumGC",
	"/memory/classes/heap/released:bytes":         "HeapReleased",
	"/memory/classes/heap/stacks:bytes":           "StackInuse",
	"/memory/classes/metadata/mcache/inuse:bytes": "MCacheInuse",
	"/memory/classes/metadata/mspan/inuse:bytes":  "MSpanInuse",
	"/memory/classes/profiling/buckets:bytes":     "BuckHashSys",
	"/memory/classes/other:bytes":                 "OtherSys",
}

func GetRuntimeMetrics() map[string]models.Metrics {
	result := make(map[string]models.Metrics)
	descriptions := metrics.All()

	samples := make([]metrics.Sample, len(descriptions))

	i := 0
	for key := range DirectMapping {
		samples[i].Name = key
		i++
	}
	metrics.Read(samples)

	for _, sample := range samples {
		name, ok := DirectMapping[sample.Name]
		if !ok {
			continue
		}
		m := models.Metrics{
			ID:    name,
			MType: models.Gauge,
		}
		switch sample.Value.Kind() {
		case metrics.KindUint64:
			v := float64(sample.Value.Uint64())
			m.Value = &v
		case metrics.KindFloat64:
			v := sample.Value.Float64()
			m.Value = &v
		default:
			// Игнорируем гистограммы пока
		}

		result[name] = m
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	gaugeMetrics := map[string]uint64{
		"Alloc":        m.Alloc,
		"Frees":        m.Frees,
		"GCSys":        m.GCSys,
		"HeapAlloc":    m.HeapAlloc,
		"HeapIdle":     m.HeapIdle,
		"HeapInuse":    m.HeapInuse,
		"HeapSys":      m.HeapSys,
		"HeapReleased": m.HeapReleased,
		"HeapObjects":  m.HeapObjects,
		"Sys":          m.Sys,
		"StackSys":     m.StackSys,
		"LastGC":       m.LastGC,
		"Lookups":      m.Lookups,
		"PauseTotalNs": m.PauseTotalNs,
		"MCacheSys":    m.MCacheSys,
		"MSpanSys":     m.MSpanSys,
		"NextGC":       m.NextGC,
	}

	for name, value := range gaugeMetrics {
		result[name] = models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: new(float64(value)),
		}
	}

	result["NumForcedGC"] = models.Metrics{
		ID:    "NumForcedGC",
		MType: models.Gauge,
		Value: new(float64(m.NumForcedGC)),
	}

	result["GCCPUFraction"] = models.Metrics{
		ID:    "GCCPUFraction",
		MType: models.Gauge,
		Value: new(float64(m.GCCPUFraction)),
	}

	return result
}

func Float64Ptr(v float64) *float64 {
	return &v
}

func float64PtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func int64PtrEqual(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func PostMetrics(baseURL string, metrics map[string]string, metricType string) error {
	var errs []error
	for k, v := range metrics {
		url := fmt.Sprintf("%s/%s/%s/%s", baseURL, metricType, k, v)
		resp, err := http.Post(url, "text/plain", nil)
		if err != nil {
			errs = append(errs, fmt.Errorf("ошибка отправки метрики %s: %v", k, err))
			time.Sleep(time.Duration(10) * time.Millisecond)
			continue
		}
		if resp.StatusCode != 200 {
			errs = append(errs, fmt.Errorf("ошибка отправки метрики %s: %v", k, resp.Status))
		}
		if resp.Body != nil {
			err := resp.Body.Close()
			if err != nil {
				errs = append(errs, fmt.Errorf("ошибка закрытия тела ответа для метрики %s: %v", k, err))
			}
		}
		time.Sleep(time.Duration(10) * time.Millisecond)
	}

	if len(errs) > 0 {
		return fmt.Errorf("ошибки отправки метрик: %v", errs)
	}
	return nil
}

func PostMetricsJSON(baseURL string, metrics map[string]models.Metrics) error {
	var errs []error
	for _, v := range metrics {
		jsonData, err := json.Marshal(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("ошибка создания json %s: %v", v.ID, err))
			time.Sleep(time.Duration(10) * time.Millisecond)
			continue
		}

		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err = gz.Write(jsonData); err != nil {
			return fmt.Errorf("ошибка записи gzip: %w", err)
		}

		if err = gz.Close(); err != nil {
			return fmt.Errorf("ошибка закрытия gz: %w", err)
		}

		req, err := http.NewRequest("POST", baseURL, &buf)
		if err != nil {
			return fmt.Errorf("ошибка создания запроса: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errs = append(errs, fmt.Errorf("ошибка отправки метрики %s: %v", v.ID, err))
			time.Sleep(time.Duration(10) * time.Millisecond)
			continue
		}
		if resp.StatusCode != 200 {
			errs = append(errs, fmt.Errorf("ошибка отправки метрики %s: %v", v.ID, resp.Status))
		}
		if resp.Body != nil {
			err := resp.Body.Close()
			if err != nil {
				errs = append(errs, fmt.Errorf("ошибка закрытия тела ответа для метрики %s: %v", v.ID, err))
			}
		}

		time.Sleep(time.Duration(10) * time.Millisecond)
	}

	if len(errs) > 0 {
		return fmt.Errorf("ошибки отправки метрик: %v", errs)
	}
	return nil
}

func PostMetricsBatch(sender Sender, baseURL string, metrics map[string]models.Metrics) error {
	metricsSlice := make([]models.Metrics, 0)
	for _, v := range metrics {
		metricsSlice = append(metricsSlice, v)
	}

	jsonData, err := json.Marshal(metricsSlice)

	if err != nil {
		return fmt.Errorf("ошибка создания json: %v", err)
	}

	var resp *http.Response

	buf := bytes.NewBuffer(jsonData)

	req, err := http.NewRequest("POST", baseURL, buf)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req = req.WithContext(ctx)

	resp, err = sender.Do(req)
	if resp != nil && resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			return fmt.Errorf("ошибка закрытия тела запроса %w", err)
		}
	}

	if err != nil {
		return fmt.Errorf("ошибка отправки метрик: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("ошибка отправки метрик: %v", resp.Status)
	}

	return nil
}
