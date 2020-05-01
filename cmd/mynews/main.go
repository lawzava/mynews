package main

import (
	"log"
	"mynews/internal/app/feed"
	"mynews/internal/pkg/config"
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

	if err = feed.New(cfg).Run(); err != nil {
		log.Fatal(err)
	}
}
