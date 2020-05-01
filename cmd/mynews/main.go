package main

import (
	"mynews/internal/app/feed"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"os"
)

func main() {
	log := logger.New(logger.Info)

	cfg, err := config.New(log)
	if err != nil {
		log.Fatal("initiating config failed", err)
	}

	if cfg == nil {
		os.Exit(0)
	}

	if err = feed.New(cfg).Run(log); err != nil {
		log.Fatal("failed running feed", err)
	}
}
