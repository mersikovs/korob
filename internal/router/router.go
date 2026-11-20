// Package router configures the HTTP routing, middleware, and API endpoints
// for the metrics service.
package router

import (
	"crypto/hpke"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/mersikovs/korob.git/internal/audit"
	"github.com/mersikovs/korob.git/internal/crypt"
	"github.com/mersikovs/korob.git/internal/handler"
	"github.com/mersikovs/korob.git/internal/logger"
	"github.com/mersikovs/korob.git/internal/middleware"
	"github.com/mersikovs/korob.git/internal/service"
	"github.com/mersikovs/korob.git/internal/signing"
)

// NewRouter creates and configures a chi.Mux router for the metrics API.
func NewRouter(service *service.MetricService, dispatcher *audit.Dispatcher, logger logger.Logger, signKey *string, privateKey *hpke.PrivateKey) *chi.Mux {
	r := chi.NewRouter()

	metricHandler := handler.NewMetricHandler(service, dispatcher, logger)

	r.Use(middleware.Logger(logger))
	r.Use(chimiddleware.Compress(5, "application/json", "text/html"))

	if privateKey != nil {
		r.Use(middleware.DecryptMiddleware(*privateKey))
	} else {
		r.Use(middleware.RejectUnsupported(
			crypt.HeaderEncrypted,
			"encryption not supported: this server is not configured with a private key",
		))
	}

	r.Use(middleware.GzipRequestMiddleware)

	if signKey != nil {
		r.Use(middleware.VerifySha256Middleware([]byte(*signKey), logger))
		r.Use(middleware.SignResponseMiddleware([]byte(*signKey), logger))
	} else {
		r.Use(middleware.RejectUnsupported(
			signing.HeaderName,
			"signing not supported: this server is not configured with a signKey",
		))
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
