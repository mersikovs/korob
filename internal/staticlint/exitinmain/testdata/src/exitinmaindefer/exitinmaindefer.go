package main

import "os"

func main() {
	defer func() { /* cleanup */ }()

	os.Exit(1) // want "os\\.Exit in main!"
}
