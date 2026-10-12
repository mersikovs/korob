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

type MetricHandler struct {
	metricService *service.MetricService
}

func NewMetricHandler(metricService *service.MetricService) *MetricHandler {
	return &MetricHandler{metricService: metricService}
}

func (h *MetricHandler) PingDB(w http.ResponseWriter, r *http.Request) {
	err := h.metricService.Ping(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

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
			Ts:        time.Now().Unix(),
			Metrics:   []string{req.ID},
			IPAddress: extractIP(r),
		})
	}
}

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
			Ts:        time.Now().Unix(),
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
