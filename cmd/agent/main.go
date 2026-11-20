// Package main is the entry point for the metrics agent application.
package main

import (
	"cmp"
	"context"
	"crypto/hpke"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mersikovs/korob.git/internal/agent"
	"github.com/mersikovs/korob.git/internal/agent/transport"
	"github.com/mersikovs/korob.git/internal/config"
	"github.com/mersikovs/korob.git/internal/crypt"
	"github.com/mersikovs/korob.git/internal/logger"
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
		fmt.Fprintln(os.Stderr, "Agent failed:", err)
		//nolint:exitinmain // все defer выполнились в run, игнорируем
		os.Exit(1)
	}
}

func run(ctx context.Context) error {

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	delays := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	cnf, err := config.AgentParseConfig(fs, os.Args[1:], config.OSenv{})
	if err != nil {
		return fmt.Errorf("failed to parse command-line and environment parameters: %w", err)
	}

	logger, err := logger.NewZap("info")
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	var publicKey *hpke.PublicKey
	if cnf.CryptoKey != nil {
		pk, loadKeyErr := crypt.LoadPublicKey(*cnf.CryptoKey)
		if loadKeyErr != nil {
			return fmt.Errorf("failed to load public key: %w", loadKeyErr)
		}
		publicKey = &pk
	}

	var baseRT http.RoundTripper = http.DefaultTransport

	if publicKey != nil {
		baseRT = transport.NewEncryptingTransport(baseRT, *publicKey)
	}

	compression := transport.NewCompressTransport(
		baseRT,
		logger,
	)

	t := &transport.SigningTransport{
		Base: compression,
	}

	if cnf.Key != nil {
		t.Key = []byte(*cnf.Key)
	}

	client := &http.Client{
		Transport: t,
	}

	retryClient := transport.New(client, delays)
	agent := agent.NewAgent(cnf, retryClient, logger)
	done := make(chan struct{})
	go func() {
		agent.Run(runCtx)
		close(done)
	}()

	logger.Info("Работаю... Нажми Ctrl+C для завершения")

	<-runCtx.Done()
	logger.Info("A termination signal has been received.")

	<-done
	logger.Info("Done")

	return nil
}
