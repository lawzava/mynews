package main

import (
	"os"
	"os/signal"
	"syscall"

	"mynews/internal/app/news"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
)

func main() {
	log := logger.New(logger.Info)

	cfg, err := config.New(log)
	if err != nil {
		log.Fatal("initiating config failed", err)
	}

	if cfg == nil {
		log.Warn("cfg is nil")
		os.Exit(0)
	}

	handleInterrupt(cfg, log)

	if err = cfg.Store.RecoverFromFile(cfg.StorageFilePath, log); err != nil {
		log.Fatal("recovering data from file", err)
	}

	if err = news.New(cfg).Run(log); err != nil {
		log.Fatal("failed running feed", err)
	}
}

func handleInterrupt(cfg *config.Config, log *logger.Log) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		if err := cfg.Store.DumpToFile(cfg.StorageFilePath); err != nil {
			log.Fatal("failed to dump storage file", err)
		}

		os.Exit(0)
	}()
}
