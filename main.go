package main

import (
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("missing verb")
	}

	verb := os.Args[1]

	if verb == "run" {
		command := strings.Join(os.Args[2:], " ")

		_ = command
	} else {
		log.Fatalf("unknown verb: %s", verb)
	}
}
