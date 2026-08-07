package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"runtime/metrics"
	"strings"
	"time"

	models "github.com/mersikovs/korob.git/internal/model"
)

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

	//TODO Старые метрики заменить на актуальные из runtime/metrics
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
			MType: "gauge",
			Value: Float64Ptr(float64(value)),
		}
	}

	result["NumForcedGC"] = models.Metrics{
		ID:    "NumForcedGC",
		MType: "gauge",
		Value: Float64Ptr(float64(m.NumForcedGC)),
	}

	result["GCCPUFraction"] = models.Metrics{
		ID:    "GCCPUFraction",
		MType: "gauge",
		Value: Float64Ptr(float64(m.GCCPUFraction)),
	}

	return result
}

func Float64Ptr(v float64) *float64 {
	return &v
}

func GetCountDiff(oldMetrics, newMetrics map[string]models.Metrics) int {
	countDiff := 0

	for key, newMetric := range newMetrics {
		oldMetric, exists := oldMetrics[key]
		if !exists {
			countDiff++
			continue
		}

		if !float64PtrEqual(newMetric.Value, oldMetric.Value) ||
			!int64PtrEqual(newMetric.Delta, oldMetric.Delta) {
			countDiff++
		}
	}

	return countDiff
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
		gz.Write(jsonData)
		gz.Close()

		req, err := http.NewRequest("POST", baseURL, &buf)
		if err != nil {
			return err
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

func PostMetricsBatch(baseURL string, metrics map[string]models.Metrics) error {
	metricsSlice := make([]models.Metrics, 0)
	for _, v := range metrics {
		metricsSlice = append(metricsSlice, v)
	}

	jsonData, err := json.Marshal(metricsSlice)

	if err != nil {
		return fmt.Errorf("ошибка создания json: %v", err)
	}

	delays := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	retry := 3
	var lastError error
	var resp *http.Response

	for i := range retry {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write(jsonData)
		gz.Close()
		req, err := http.NewRequest("POST", baseURL, &buf)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
		req.WithContext(ctx)
		resp, lastError = http.DefaultClient.Do(req)
		if lastError != nil {
			if isRetryable(lastError) && i < retry {
				delay := delays[i]
				time.Sleep(delay)
				cancel()
				continue
			}
			cancel()
			return lastError
		}

		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		cancel()
	}

	if lastError != nil {
		return fmt.Errorf("ошибка отправки метрик: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("ошибка отправки метрик: %v", resp.Status)
	}

	return nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	var uerr *url.Error
	if errors.As(err, &uerr) {
		if uerr.Err != nil {
			err = uerr.Err
		}
	}

	//  no such host
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// сonnection refused
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		msg := strings.ToLower(opErr.Error())
		if strings.Contains(msg, "connection refused") {
			return true
		}
	}

	// timeout при попытке подключения (dial)
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "dial") {
				return true
			}
		}
	}

	return false
}
