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

	newsRunner, err := news.New(cfg, log)
	if err != nil {
		log.Fatal("initializing news runner failed", err)
	}

	handleInterrupt(cfg, &newsRunner, log)

	err = newsRunner.Run(log)
	if err != nil {
		log.Fatal("failed running feed", err)
	}
}

func handleInterrupt(cfg *config.Config, newsRunner *news.News, log *logger.Log) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		closeErr := newsRunner.Close()
		if closeErr != nil {
			log.WarnErr("failed to close news runner", closeErr)
		}

		dumpErr := cfg.Store.DumpToFile(cfg.StorageFilePath)
		if dumpErr != nil {
			log.Fatal("failed to dump storage file", dumpErr)
		}

		os.Exit(0)
	}()
}
