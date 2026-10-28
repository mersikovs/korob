package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"runtime/metrics"
	"time"

	models "github.com/mersikovs/korob.git/internal/model"
)

// Sender defines the contract for executing HTTP requests.
type Sender interface {
	Do(req *http.Request) (*http.Response, error)
}

// DirectMapping преобразует стандартные пути метрик из пакета runtime/metrics
// во внутренние имена метрик, ожидаемые приложением.
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

// GetRuntimeMetrics collects and formats Go runtime statistics
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

// Float64Ptr returns a pointer to the given float64 value
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

// PostMetricsBatch serializes a map of metrics into a JSON array and sends it
func PostMetricsBatch(sender Sender, baseURL string, metrics map[string]models.Metrics) error {
	var err error

	metricsSlice := make([]models.Metrics, 0)
	for _, v := range metrics {
		metricsSlice = append(metricsSlice, v)
	}

	var jsonData []byte
	if jsonData, err = json.Marshal(metricsSlice); err != nil {
		return fmt.Errorf("create json error: %w", err)
	}

	buf := bytes.NewBuffer(jsonData)

	var req *http.Request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if req, err = http.NewRequestWithContext(ctx, "POST", baseURL, buf); err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	var resp *http.Response
	if resp, err = sender.Do(req); err != nil {
		return fmt.Errorf("sent metrics error: %w", err)
	}

	if resp != nil && resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			return fmt.Errorf("body close error: %w", err)
		}
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("sent metrics error status: %s", resp.Status)
	}

	return nil
}
