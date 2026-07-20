package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/router"
	"github.com/mersikovs/korob.git/internal/service"
)

func main() {
	address := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	flag.Parse()

	if addr, exists := os.LookupEnv("ADDRESS"); exists && addr != "" {
		*address = addr
	}

	logger, err := logger.NewZap("info")
	if err != nil {
		logger.Fatal("Ошибка создания логгера:", err)
	}

	modelStorage := models.NewStorage()
	service := service.NewMetricService(modelStorage, logger)

	router := router.NewRouter(service)

	err = http.ListenAndServe(*address, router)
	if err != nil {
		logger.Fatal("Ошибка запуска сервера:", err)
	}
}
