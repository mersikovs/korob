package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/mersikovs/korob.git/internal/handler"
	"github.com/mersikovs/korob.git/internal/middleware"
	"github.com/mersikovs/korob.git/internal/service"
)

func NewRouter(service *service.MetricService) *chi.Mux {
	r := chi.NewRouter()

	metricHandler := handler.NewMetricHandler(service)

	r.Use(middleware.Logger(service.Logger))
	r.Use(middleware.GzipResponseMiddleware)
	r.Use(middleware.GzipRequestMiddleware)

	r.Get("/", metricHandler.ListMetrics)
	r.Get("/value/{type}/{name}", metricHandler.GetMetricByUrlParam)
	r.Post("/value/", metricHandler.GetMetricByJSONParam)
	r.Post("/update/", metricHandler.UpdateMetricByJSONParam)
	r.Post("/update/{type}/{name}/{value}", metricHandler.UpdateMetricByParam)

	return r
}
