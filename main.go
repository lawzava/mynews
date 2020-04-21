package main

import (
	"log"
	"mynews/config"
	"os"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	if cfg == nil {
		os.Exit(0)
	}

	if err = newFeed(cfg).run(); err != nil {
		log.Fatal(err)
	}
}
