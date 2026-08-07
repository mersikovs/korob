package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mersikovs/korob.git/internal/config"
	"github.com/mersikovs/korob.git/internal/database"
	"github.com/mersikovs/korob.git/internal/logger"
	"github.com/mersikovs/korob.git/internal/repository"
	"github.com/mersikovs/korob.git/internal/router"
	"github.com/mersikovs/korob.git/internal/service"
)

func main() {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	cnf, err := config.ServerParseConfig(fs, os.Args[1:], config.OSenv{})
	if err != nil {
		fmt.Println("Ошибка парсинга параметров")
		return
	}

	logger, err := logger.NewZap("info")
	if err != nil {
		logger.Fatal("Ошибка создания логгера:", err)
	}

	if cnf.DatabaseDSN != "" {
		if err := database.MigrateUp(cnf.DatabaseDSN, "file://migrations"); err != nil {
			logger.Fatal("Ошибка миграции базы данных:", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	storage, err := repository.NewStorage(ctx, cnf, logger)
	if err != nil {
		logger.Fatal("Ошибка создания объекта хранилища:", err)
	}
	service := service.NewMetricService(storage, logger)

	router := router.NewRouter(service)

	err = http.ListenAndServe(cnf.Address, router)
	if err != nil {
		logger.Fatal("Ошибка запуска сервера:", err)
	}

	if closer, ok := storage.(io.Closer); ok {
		defer closer.Close()
	}
	cancel()
}
