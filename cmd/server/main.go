// Package main is the entry point for the metrics server application.
package main

import (
	"cmp"
	"context"
	"errors"
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

var (
	BuildVersion, BuildDate, BuildCommit string
)

func main() {
	fmt.Printf(
		"Build version: %s\nBuild date: %s\nBuild commit: %s\n",
		cmp.Or(BuildVersion, "N/A"),
		cmp.Or(BuildDate, "N/A"),
		cmp.Or(BuildCommit, "N/A"),
	)
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "Application failed:", err)
		//nolint:exitinmain // все defer выполнились в run, игнорируем
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	cnf, err := config.ServerParseConfig(fs, os.Args[1:], config.OSenv{})
	if err != nil {
		return fmt.Errorf("failed to parse command-line and environment parameters: %w", err)
	}

	appLog, err := logger.NewZap("info")
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	appCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cnf.DatabaseDSN != "" {
		if migrationErr := database.MigrateUp(cnf.DatabaseDSN, "file://migrations"); migrationErr != nil {
			return fmt.Errorf("database migration failed: %w", migrationErr)
		}
	}

	storage, err := repository.NewStorage(appCtx, cnf, appLog)
	if err != nil {
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
			fileObs, err := audit.NewFileObserver(cnf.AuditFile, cnf.AuditFile, appLog)
			if err != nil {
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
		ttl := time.Duration(cnf.CacheTTLSec) * time.Second
		cacher = cache.NewGoCache(ttl, 2*ttl)
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
		if serverErr := srv.ListenAndServe(); serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			errCh <- serverErr
		}
	}()

	var serverErr error
	select {
	case <-appCtx.Done():
		appLog.Info("Graceful shutdown...")
	case serverErr = <-errCh:
		appLog.Info("Server error detected, initiating shutdown", "error", serverErr)
	}

	httpShutdownCtx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	shutdownErr := srv.Shutdown(httpShutdownCtx)
	httpCancel()
	if shutdownErr != nil {
		appLog.Info("Graceful shutdown of HTTP server failed", "error", shutdownErr)
	} else {
		appLog.Info("HTTP server shut down successfully")
	}

	var errs []error
	if serverErr != nil {
		errs = append(errs, serverErr)
	}
	if shutdownErr != nil {
		errs = append(errs, shutdownErr)
	}

	if dispatcher != nil {
		dispatcherCtx, dispatcherCancel := context.WithTimeout(context.Background(), 30*time.Second)
		dispatcherErr := dispatcher.Shutdown(dispatcherCtx)
		dispatcherCancel()
		if dispatcherErr != nil {
			appLog.Info("dispatcher shutdown completed with errors", "error", dispatcherErr)
			errs = append(errs, dispatcherErr)
		} else {
			appLog.Info("dispatcher shut down successfully")
		}
	}

	if closer, ok := storage.(io.Closer); ok {
		if closerErr := closer.Close(); closerErr != nil {
			appLog.Info("Failed to close storage", "error", closerErr)
			errs = append(errs, closerErr)
		} else {
			appLog.Info("Storage closed successfully")
		}
	}

	return errors.Join(errs...)
}
