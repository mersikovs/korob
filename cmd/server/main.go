package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/mersikovs/korob.git/internal/config"
	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/router"
	"github.com/mersikovs/korob.git/internal/service"
)

func main() {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	cnf, err := config.ServerParseConfig(fs, os.Args[1:], config.ServerOSenv{})
	if err != nil {
		fmt.Println("Ошибка парсинга параметров")
		return
	}

	logger, err := logger.NewZap("info")
	if err != nil {
		logger.Fatal("Ошибка создания логгера:", err)
	}

	modelStorage := models.NewStorage(cnf.FileStoragePath)
	modelStorage.StartPeriodicSave(cnf.StoreInterval, logger)
	if cnf.Restore {
		modelStorage.Restore()
	}

	service := service.NewMetricService(modelStorage, logger)

	router := router.NewRouter(service)

	err = http.ListenAndServe(cnf.Address, router)
	if err != nil {
		logger.Fatal("Ошибка запуска сервера:", err)
	}
}
