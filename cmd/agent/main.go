package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mersikovs/korob.git/internal/agent"
	"github.com/mersikovs/korob.git/internal/config"
)

func main() {
	cnf, err := config.Parse()
	if err != nil {
		fmt.Println("Ошибка парсинга параметров")
		return
	}

	agent := agent.NewAgent(cnf)
	agent.Run()
	fmt.Println("Работаю... Нажми Ctrl+C для завершения")

	// Ждём сигнал ОС (Ctrl+C или docker stop)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan // блокируем main до сигнала

	fmt.Println("Завершено")
}
