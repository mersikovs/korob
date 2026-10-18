// Package router configures the HTTP routing, middleware, and API endpoints
// for the metrics service.
package router

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/mersikovs/korob.git/internal/audit"
	"github.com/mersikovs/korob.git/internal/handler"
	"github.com/mersikovs/korob.git/internal/logger"
	"github.com/mersikovs/korob.git/internal/middleware"
	"github.com/mersikovs/korob.git/internal/service"
)

// NewRouter creates and configures a chi.Mux router for the metrics API.
func NewRouter(service *service.MetricService, dispatcher *audit.Dispatcher, logger logger.Logger, signKey []byte) *chi.Mux {
	r := chi.NewRouter()

	metricHandler := handler.NewMetricHandler(service, dispatcher, logger)

	r.Use(middleware.Logger(logger))
	r.Use(chimiddleware.Compress(5, "application/json", "text/html"))
	r.Use(middleware.GzipRequestMiddleware)

	if signKey != nil {
		r.Use(middleware.VerifySha256Middleware(signKey, logger))
		r.Use(middleware.SignResponseMiddleware(signKey, logger))
	}

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
