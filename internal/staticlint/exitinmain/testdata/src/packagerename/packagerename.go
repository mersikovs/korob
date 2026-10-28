package main

import myos "os"

func main() {
	defer func() { /* cleanup */ }()
	myos.Exit(1) // want "os\\.Exit in main!"
}
