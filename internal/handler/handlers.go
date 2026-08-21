package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	var html string
	if err == nil {

		html = `<!DOCTYPE html>
<html>
<head><title>Список метрик</title></head>
<body>
    <h1>Список метрик пуст</h1>
</body>
</html>`

		if len(metricsNames) > 0 {
			html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Список метрик</title></head>
<body>
    <h1>Список метрик:</h1>
    <ul>
		<li>
			%s
		</li>
	</ul>
</body>
</html>`, strings.Join(metricsNames, "</li> <li>"))
		}

	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(html))
	if err != nil {
		h.metricService.Logger.Info("ошибка записи ответа: %v", err)
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

	for _, m := range req {
		if m.ID == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	serviceErr := h.metricService.UpdateMetricsFromStruct(r.Context(), req)
	if serviceErr != nil {
		h.metricService.Logger.Info(serviceErr.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}
