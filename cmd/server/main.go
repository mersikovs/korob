// Package main is the entry point for the metrics server application.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
		fmt.Fprintln(os.Stderr, "Failed to parse command-line and environment parameters:", err)
		os.Exit(2)
	}

	logger, err := logger.NewZap("info")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create logger:", err)
		os.Exit(1)
	}

	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cnf.DatabaseDSN != "" {
		if err := database.MigrateUp(cnf.DatabaseDSN, "file://migrations"); err != nil {
			logger.Fatal("Database migration failed", "error", err)
		}
	}

	storage, err := repository.NewStorage(appCtx, cnf, logger)
	if err != nil {
		logger.Fatal("Failed to initialize storage", "error", err)
	}

	var dispatcher *audit.Dispatcher
	if cnf.AuditFile != "" || cnf.AuditURL != "" {
		dispatcher = audit.NewDispatcher(500, 8, logger)

		if cnf.AuditFile != "" {
			fileObs, err := audit.NewFileObserver(cnf.AuditFile, cnf.AuditFile, logger)
			if err != nil {
				logger.Fatal("File audit failed", "error", err)
			}
			dispatcher.Register(fileObs)
		}

		if cnf.AuditURL != "" {
			remoteObs := audit.NewRemoteObserver(
				cnf.AuditURL,
				cnf.AuditURL,
				logger,
			)
			dispatcher.Register(remoteObs)
		}

		dispatcher.Start()
	}

	cacher := cache.NewNoopCache()
	if cnf.CacheTTLSec > 0 {
		cacher = cache.NewGoCache(time.Duration(cnf.CacheTTLSec*int(time.Second)), 2*time.Duration(cnf.CacheTTLSec*int(time.Second)))
	}

	service := service.NewMetricService(storage, cacher, logger)
	router := router.NewRouter(service, dispatcher, logger, cnf.Key)

	srv := &http.Server{
		Addr:    cnf.Address,
		Handler: router,
	}

	go func() {
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("The web server failed to start", "error", err)
		}
	}()

	<-appCtx.Done()
	logger.Info("Graceful shutdown...")

	httpShutdownCtx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := srv.Shutdown(httpShutdownCtx); err != nil {
		logger.Info("Graceful shutdown of HTTP server failed", "error", err)
	}
	httpCancel()

	if dispatcher != nil {
		dispatcherCtx, dispatcherCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := dispatcher.Shutdown(dispatcherCtx); err != nil {
			logger.Info("dispatcher shutdown completed with errors", "error", err)
		}
		logger.Info("dispatcher shut down successfully")
		dispatcherCancel()
	}

	if closer, ok := storage.(io.Closer); ok {
		defer func() {
			if err := closer.Close(); err != nil {
				logger.Info("Failed to close storage", "error", err)
			}
		}()
	}

}
