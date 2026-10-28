package main

import "os"

func main() {
	defer func() { /* cleanup */ }()
	f := func() { os.Exit(1) } // want `os.Exit in main!`
	_ = f
}

var exit = func() { os.Exit(1) } // no error

func other() { os.Exit(1) }
