package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mersikovs/korob.git/internal/reset"
)

func main() {
	root := flag.String("root", "", "корневая директория (обязательное)")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "Ошибка: флаг -root является обязательным.")
		flag.Usage()
		os.Exit(2)
	}

	if err := reset.Run(*root); err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("Generation OK!")
}
