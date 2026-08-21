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

	//Сервисный эндпоинт
	r.Get("/ping", metricHandler.PingDB)
	//Получить метрики
	r.Get("/", metricHandler.ListMetrics)
	r.Get("/value/{type}/{name}", metricHandler.GetMetricFromPath)
	r.Post("/value/", metricHandler.FetchMetricFromBody)
	//Обновить метрики
	r.Post("/update/{type}/{name}/{value}", metricHandler.UpdateMetricFromPath)
	r.Post("/update/", metricHandler.UpdateMetricFromBody)
	r.Post("/updates/", metricHandler.BatchUpdateMetricsFromBody)

	return r
}
