package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/mersikovs/korob.git/internal/reset"
)

func main() {
	root := flag.String("root", "", "корневая директория (обязательное)")
	clean := flag.String("clean", "", "что делать с устаревшими сгенерированными файлами: delete|warn (обязательное)")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "Ошибка: флаг -root является обязательным.")
		flag.Usage()
		os.Exit(2)
	}

	mode, err := reset.ParseCleanMode(*clean)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		flag.Usage()
		os.Exit(2)
	}

	if err := reset.Run(*root, mode); err != nil {
		printNonStale(err)
		os.Exit(1)
	}

	fmt.Println("Generation OK!")
}

func printNonStale(err error) {
	if err == nil {
		return
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range joined.Unwrap() {
			printNonStale(e)
		}
		return
	}
	if errors.Is(err, reset.ErrStaleFound) {
		return
	}
	fmt.Fprintln(os.Stderr, err)
}
