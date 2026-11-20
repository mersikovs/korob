package main

import (
	"fmt"
	"log"

	"github.com/mersikovs/korob.git/internal/crypt"
)

func main() {
	if err := crypt.GenerateAndSaveKeyPair("server_private.key", "server_public.key"); err != nil {
		log.Fatalf("generate server keys: %v", err)
	}

	fmt.Println("generated: server_private.key (for server), server_public.key (for client)")
}
