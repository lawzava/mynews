package main

import (
	"log"
)

func main() {
	cfg, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err = newFeed(cfg).run(); err != nil {
		log.Fatal(err)
	}
}
