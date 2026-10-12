package router

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/mersikovs/korob.git/internal/handler"
	"github.com/mersikovs/korob.git/internal/middleware"
	"github.com/mersikovs/korob.git/internal/service"

	"net/http"
	_ "net/http/pprof"
)

func NewRouter(service *service.MetricService, signKey []byte) *chi.Mux {
	r := chi.NewRouter()

	metricHandler := handler.NewMetricHandler(service)

	r.Use(middleware.Logger(service.Logger))
	r.Use(chimiddleware.Compress(5, "application/json", "text/html"))

	if signKey != nil {
		r.Use(middleware.VerifySha256Middleware(signKey, service.Logger))
		r.Use(middleware.SignResponseMiddleware(signKey, service.Logger))
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

	r.Mount("/debug/pprof", http.HandlerFunc(http.DefaultServeMux.ServeHTTP))

	return r
}
