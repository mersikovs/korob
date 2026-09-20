package main

import (
	"context"
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
	"github.com/mersikovs/korob.git/internal/logger"
)

func main() {
	delays := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	cnf, err := config.Parse(fs, os.Args[1:], config.OSenv{})
	if err != nil {
		fmt.Println("Ошибка парсинга параметров ", err)
		return
	}

	logger, err := logger.NewZap("info")
	if err != nil {
		fmt.Println("Ошибка создания логгера")
		os.Exit(1)
	}

	compression := transport.NewCompressTransport(
		http.DefaultTransport,
		logger,
	)

	t := &transport.SigningTransport{
		Base: compression,
		Key:  cnf.Key,
	}

	client := &http.Client{
		Transport: t,
	}

	retryClient := transport.New(client, delays)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agent := agent.NewAgent(cnf, retryClient, logger)
	done := make(chan struct{})
	go func() {
		agent.Run(ctx)
		close(done)
	}()

	logger.Info("Работаю... Нажми Ctrl+C для завершения")

	// Ждём сигнал ОС (Ctrl+C или docker stop)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan // блокируем main до сигнала
	cancel()
	<-done
	logger.Info("Завершено")
}
