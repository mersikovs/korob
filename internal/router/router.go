package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/mersikovs/korob.git/internal/handler"
	"github.com/mersikovs/korob.git/internal/service"
)

func NewRouter(service *service.MetricService) *chi.Mux {
	r := chi.NewRouter()

	metricHandler := handler.NewMetricHandler(service)

	r.Get("/", metricHandler.ListMetrics)
	r.Get("/value/{type}/{name}", metricHandler.GetMetric)
	r.Post("/update/{type}/{name}/{value}", metricHandler.UpdateMetric)

	return r
}
