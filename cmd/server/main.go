package main

import (
	"flag"
	"log"
	"net/http"

	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/router"
	"github.com/mersikovs/korob.git/internal/service"
)

func main() {
	address := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	flag.Parse()

	modelStorage := models.NewStorage()
	service := service.NewMetricService(modelStorage)

	router := router.NewRouter(service)

	err := http.ListenAndServe(*address, router)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %s\n", err)
	}
}
