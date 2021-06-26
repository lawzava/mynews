package main

import (
	"mynews/internal/app/news"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log := logger.New(logger.Info)

	cfg, err := config.New(log)
	if err != nil {
		log.Fatal("initiating config failed", err)
	}

	if cfg == nil {
		log.Warn("config is empty, exiting (if you have just created config, start the app again without create action)")
		os.Exit(0)
	}

	handleInterrupt(cfg, log)

	if err = news.New(cfg).Run(log); err != nil {
		log.Fatal("failed running feed", err)
	}
}

func handleInterrupt(cfg *config.Config, log *logger.Log) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		if err := cfg.Store.DumpToFile(cfg.StorageFilePath); err != nil {
			log.Fatal("failed to dump storage file", err)
		}

		os.Exit(0)
	}()
}
