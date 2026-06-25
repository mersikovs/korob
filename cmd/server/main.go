package main

import (
	"log"
	"net/http"

	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/router"
	"github.com/mersikovs/korob.git/internal/service"
)

func main() {
	modelStorage := models.NewStorage()
	service := service.NewMetricService(modelStorage)

	router := router.NewRouter(service)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %s\n", err)
	}
}
