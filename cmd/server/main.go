package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mersikovs/korob.git/internal/audit"
	"github.com/mersikovs/korob.git/internal/cache"
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
		fmt.Println("Ошибка парсинга параметров ", err)
		os.Exit(1)
	}

	logger, err := logger.NewZap("info")
	if err != nil {
		fmt.Println("Ошибка создания логгера")
		os.Exit(1)
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

	var dispatcher *audit.Dispatcher
	if cnf.AuditFile != "" || cnf.AuditUrl != "" {
		dispatcher = audit.NewDispatcher(500, 8, logger)

		if cnf.AuditFile != "" {
			fileObs, err := audit.NewFileObserver(cnf.AuditFile, cnf.AuditFile, logger)
			if err != nil {
				logger.Fatal("Ошибка файлового аудита :", err)
			}
			dispatcher.Register(fileObs)
		}

		if cnf.AuditUrl != "" {
			remoteObs := audit.NewRemoteObserver(
				cnf.AuditUrl,
				cnf.AuditUrl,
				logger,
			)
			dispatcher.Register(remoteObs)
		}
	}

	cacher := cache.NewNoopCache()
	if cnf.CacheTTLSec > 0 {
		cacher = cache.NewGoCache(time.Duration(cnf.CacheTTLSec*int(time.Second)), 2*time.Duration(cnf.CacheTTLSec*int(time.Second)))
	}

	service := service.NewMetricService(storage, dispatcher, cacher, logger)
	router := router.NewRouter(service, cnf.Key)

	err = http.ListenAndServe(cnf.Address, router)
	if err != nil {
		logger.Fatal("Ошибка запуска сервера:", err)
	}

	if closer, ok := storage.(io.Closer); ok {
		defer func() {
			if err := closer.Close(); err != nil {
				logger.Info("Ошибка закрытия хранилища %v", err)
			}
		}()
	}
	cancel()
}
