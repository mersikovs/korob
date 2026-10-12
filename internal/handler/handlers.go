// Package handler provides HTTP endpoints for metric operations,
// handling request parsing, validation, and response formatting.
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/service"
)

// MetricHandler processes HTTP requests for metric operations,
// delegating business logic to the MetricService.
type MetricHandler struct {
	metricService *service.MetricService
}

// NewMetricHandler creates a MetricHandler with the provided MetricService dependency.
func NewMetricHandler(metricService *service.MetricService) *MetricHandler {
	return &MetricHandler{metricService: metricService}
}

// PingDB is an HTTP handler that checks the database connectivity.
//
// It returns 200 OK with "text/plain" content type if the database is reachable,
// or 500 Internal Server Error if the ping fails
func (h *MetricHandler) PingDB(w http.ResponseWriter, r *http.Request) {
	err := h.metricService.Ping(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// ListMetrics renders an HTML page displaying all available metric names.
//
//   - Returns 200 OK with "text/html; charset=utf-8" on success.
//   - Returns 500 Internal Server Error if metric retrieval fails.
func (h *MetricHandler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	metricsNames, err := h.metricService.ListMetrics(r.Context())
	if err != nil {
		h.metricService.Logger.Info("ошибка получения метрик:", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	estimatedSize := 200
	for _, name := range metricsNames {
		estimatedSize += len(name) + 10
	}
	estimatedSize = int(float64(estimatedSize) * 1.2)

	var sb strings.Builder
	sb.Grow(estimatedSize)

	sb.WriteString(`<!DOCTYPE html>
<html>
<head><title>Список метрик</title></head>
<body>
    <h1>Список метрик:</h1>
    <ul>
`)

	if len(metricsNames) == 0 {
		sb.WriteString("        <li>Список метрик пуст</li>\n")
	} else {
		for _, name := range metricsNames {
			sb.WriteString("        <li>")
			sb.WriteString(template.HTMLEscapeString(name))
			sb.WriteString("</li>\n")
		}
	}

	sb.WriteString(`    </ul>
</body>
</html>`)

	if _, err := io.WriteString(w, sb.String()); err != nil {
		h.metricService.Logger.Info("ошибка записи ответа: ", "error", err)
	}

}

// GetMetricFromPath is an HTTP handler that retrieves a single metric's value
// based on the "type" and "name" parameters extracted from the URL path.
//
//   - Returns 200 OK with "text/plain; charset=utf-8" and the metric value on success.
//   - Returns 404 Not Found if the metric name is missing in the path or the metric does not exist in storage.
func (h *MetricHandler) GetMetricFromPath(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	value, serviceErr := h.metricService.GetMetric(r.Context(), metricType, metricName)
	if serviceErr != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(value))
	if err != nil {
		h.metricService.Logger.Info("ошибка записи ответа: %v", err)
	}
}

// FetchMetricFromBody retrieves a metric's value based on a JSON payload in the request body.
//
//   - Returns 200 OK with "application/json" and the strongly-typed metric
//     (Delta as int64 for Counter, Value as float64 for Gauge).
//   - Returns 415 Unsupported Media Type if Content-Type is not "application/json".
//   - Returns 400 Bad Request on invalid JSON, unknown fields, or type parsing mismatches.
//   - Returns 404 Not Found if the metric ID is missing or the metric does not exist in storage.
func (h *MetricHandler) FetchMetricFromBody(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var req models.Metrics
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	value, serviceErr := h.metricService.GetMetric(r.Context(), req.MType, req.ID)
	if serviceErr != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	response := models.Metrics{
		ID:    req.ID,
		MType: req.MType,
	}

	switch req.MType {
	case models.Counter:
		delta, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		response.Delta = &delta
	case models.Gauge:
		gaugeValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		response.Value = &gaugeValue
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		h.metricService.Logger.Info("ошибка записи ответа: %v", err)
	}
}

// UpdateMetricFromPath updates a single metric using parameters extracted from the URL path.
//
//   - Returns 200 OK with "text/plain; charset=utf-8" on success.
//   - Returns 404 Not Found if the metric name is missing in the path.
//   - Returns 400 Bad Request if the service fails to process the update.
func (h *MetricHandler) UpdateMetricFromPath(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	serviceErr := h.metricService.UpdateMetric(r.Context(), metricType, metricName, metricValue)
	if serviceErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// UpdateMetricFromBody updates a single metric based on a JSON payload in the request body.
//
//   - Returns 200 OK with "text/plain; charset=utf-8" on success.
//   - Returns 415 Unsupported Media Type if Content-Type is not "application/json".
//   - Returns 400 Bad Request on invalid JSON, unknown fields, or service update failures.
//   - Returns 404 Not Found if the metric ID is missing in the payload.
func (h *MetricHandler) UpdateMetricFromBody(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var req models.Metrics
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		h.metricService.Logger.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	serviceErr := h.metricService.UpdateMetricFromStruct(r.Context(), req.MType, req.ID, req)
	if serviceErr != nil {
		h.metricService.Logger.Info(serviceErr.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if h.metricService.Publisher != nil {
		h.metricService.Publisher.Publish(models.AuditEvent{
			TS:        time.Now().Unix(),
			Metrics:   []string{req.ID},
			IPAddress: extractIP(r),
		})
	}
}

// BatchUpdateMetricsFromBody updates multiple metrics based on a JSON array in the request body.
//
//   - Returns 200 OK with "text/plain; charset=utf-8" on success.
//   - Returns 415 Unsupported Media Type if Content-Type is not "application/json".
//   - Returns 400 Bad Request on invalid JSON, unknown fields, or service update failures.
//   - Returns 404 Not Found if the payload contains no metrics with a valid ID.
func (h *MetricHandler) BatchUpdateMetricsFromBody(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var req []models.Metrics
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		h.metricService.Logger.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricIDs := make([]string, 0)
	validReq := make([]models.Metrics, 0)
	for _, m := range req {
		if m.ID != "" {
			validReq = append(validReq, m)
			metricIDs = append(metricIDs, m.ID)
		}
	}

	if len(validReq) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	serviceErr := h.metricService.UpdateMetricsFromStruct(r.Context(), validReq)
	if serviceErr != nil {
		h.metricService.Logger.Info(serviceErr.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if h.metricService.Publisher != nil {
		h.metricService.Publisher.Publish(models.AuditEvent{
			TS:        time.Now().Unix(),
			Metrics:   metricIDs,
			IPAddress: extractIP(r),
		})
	}
}

func extractIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}
