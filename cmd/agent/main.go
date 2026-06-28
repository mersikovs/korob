package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mersikovs/korob.git/internal/agent"
)

func main() {

	agent := agent.NewAgent(2, 10)
	agent.Run()
	fmt.Println("Работаю... Нажми Ctrl+C для завершения")

	// Ждём сигнал ОС (Ctrl+C или docker stop)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan // блокируем main до сигнала

	fmt.Println("Завершено")
}
