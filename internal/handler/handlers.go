package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mersikovs/korob.git/internal/service"
)

type MetricHandler struct {
	metricService *service.MetricService
}

func NewMetricHandler(metricService *service.MetricService) *MetricHandler {
	return &MetricHandler{metricService: metricService}
}

func (h *MetricHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	serviceErr := h.metricService.UpdateMetric(metricType, metricName, metricValue)
	if serviceErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (h *MetricHandler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	metricsNames := h.metricService.ListMetrics()

	html := `<!DOCTYPE html>
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(html))
	if err != nil {
		fmt.Printf("ошибка записи ответа: %v", err)
	}
}

func (h *MetricHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	value, serviceErr := h.metricService.GetMetric(metricType, metricName)
	if serviceErr != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(value))
	if err != nil {
		fmt.Printf("ошибка записи ответа: %v", err)
	}
}
