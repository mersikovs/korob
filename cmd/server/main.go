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
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/repository"
	"github.com/mersikovs/korob.git/internal/router"
	"github.com/mersikovs/korob.git/internal/service"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "Application failed:", err)
		//nolint:exitinmain // все defer выполнились в run, игнорируем
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	var err error
	var cnf *models.ServerConfig
	if cnf, err = config.ServerParseConfig(fs, os.Args[1:], config.OSenv{}); err != nil {
		return fmt.Errorf("failed to parse command-line and environment parameters: %w", err)
	}

	var appLog logger.Logger
	if appLog, err = logger.NewZap("info"); err != nil {
		return fmt.Errorf("Failed to create logger: %w", err)
	}

	appCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cnf.DatabaseDSN != "" {
		if err = database.MigrateUp(cnf.DatabaseDSN, "file://migrations"); err != nil {
			return fmt.Errorf("database migration failed: %w", err)
		}
	}

	var storage repository.Storage
	if storage, err = repository.NewStorage(appCtx, cnf, appLog); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	cleanupStorage := true
	defer func() {
		if cleanupStorage {
			if closer, ok := storage.(io.Closer); ok {
				appLog.Info("Cleaning up storage due to initialization error", "error", closer.Close())
			}
		}
	}()

	var dispatcher *audit.Dispatcher
	if cnf.AuditFile != "" || cnf.AuditURL != "" {
		dispatcher = audit.NewDispatcher(500, 8, appLog)

		if cnf.AuditFile != "" {
			var fileObs *audit.FileObserver
			if fileObs, err = audit.NewFileObserver(cnf.AuditFile, cnf.AuditFile, appLog); err != nil {
				return fmt.Errorf("file audit failed: %w", err)
			}
			dispatcher.Register(fileObs)
		}

		if cnf.AuditURL != "" {
			remoteObs := audit.NewRemoteObserver(
				cnf.AuditURL,
				cnf.AuditURL,
				appLog,
			)
			dispatcher.Register(remoteObs)
		}

		dispatcher.Start()
	}

	cacher := cache.NewNoopCache()
	if cnf.CacheTTLSec > 0 {
		cacher = cache.NewGoCache(time.Duration(cnf.CacheTTLSec*int(time.Second)), 2*time.Duration(cnf.CacheTTLSec*int(time.Second)))
	}

	service := service.NewMetricService(storage, cacher, appLog)
	router := router.NewRouter(service, dispatcher, appLog, cnf.Key)

	srv := &http.Server{
		Addr:    cnf.Address,
		Handler: router,
	}

	cleanupStorage = false

	errCh := make(chan error, 1)
	go func() {
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-appCtx.Done():
		appLog.Info("Graceful shutdown...")
	case err = <-errCh:
		appLog.Info("Server error detected, initiating shutdown", "error", err)
	}

	httpShutdownCtx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = srv.Shutdown(httpShutdownCtx)
	httpCancel()
	if err != nil {
		appLog.Info("Graceful shutdown of HTTP server failed", "error", err)
	} else {
		appLog.Info("HTTP server shut down successfully")
	}

	if dispatcher != nil {
		dispatcherCtx, dispatcherCancel := context.WithTimeout(context.Background(), 30*time.Second)
		err = dispatcher.Shutdown(dispatcherCtx)
		dispatcherCancel()
		if err != nil {
			appLog.Info("dispatcher shutdown completed with errors", "error", err)
		} else {
			appLog.Info("dispatcher shut down successfully")
		}
	}

	if closer, ok := storage.(io.Closer); ok {
		if err = closer.Close(); err != nil {
			appLog.Info("Failed to close storage", "error", err)
		} else {
			appLog.Info("Storage closed successfully")
		}
	}

	return err
}
