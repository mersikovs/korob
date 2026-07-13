package agent

import (
	"fmt"
	"net/http"
	"runtime/metrics"
	"time"
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

func GetRuntimeMetrics() map[string]string {
	result := make(map[string]string)
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
		switch sample.Value.Kind() {
		case metrics.KindUint64:
			result[name] = fmt.Sprintf("%v", sample.Value.Uint64())
		case metrics.KindFloat64:
			result[name] = fmt.Sprintf("%v", sample.Value.Float64())
		default:
			// Игнорируем гистограммы пока
		}
	}
	return result
}

func GetCountDiff(oldMetrics, newMetrics map[string]string) int {
	countDiff := 0
	for key, newValue := range newMetrics {
		if oldValue, ok := oldMetrics[key]; ok {
			if newValue != oldValue {
				countDiff++
			}
		} else {
			countDiff++
		}
	}
	return countDiff
}

func PostMetrics(baseURL string, metrics map[string]string, metricType string) error {
	var errs []error
	for k, v := range metrics {
		url := fmt.Sprintf("%s/%s/%s/%s", baseURL, metricType, k, v)
		resp, err := http.Post(url, "text/plain", nil)
		if err != nil {
			errs = append(errs, fmt.Errorf("ошибка отправки метрики %s: %v", k, err))
			time.Sleep(time.Duration(1) * time.Second)
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
		time.Sleep(time.Duration(1) * time.Second)
	}

	if len(errs) > 0 {
		return fmt.Errorf("ошибки отправки метрик: %v", errs)
	}
	return nil
}
