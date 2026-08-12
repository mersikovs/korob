package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mersikovs/korob.git/internal/agent"
	"github.com/mersikovs/korob.git/internal/agent/transport"
	"github.com/mersikovs/korob.git/internal/config"
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
		fmt.Println("Ошибка парсинга параметров")
		return
	}

	retryClient := transport.New(delays)
	agent := agent.NewAgent(cnf, retryClient)
	agent.Run()
	fmt.Println("Работаю... Нажми Ctrl+C для завершения")

	// Ждём сигнал ОС (Ctrl+C или docker stop)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan // блокируем main до сигнала

	fmt.Println("Завершено")
}
