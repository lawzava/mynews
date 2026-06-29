package main

import (
	"context"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	runErr := newsRunner.Run(ctx, log)

	stop() // restore default signal handling while we shut down

	shutdown(cfg, &newsRunner, log)

	if runErr != nil {
		log.Fatal("failed running feed", runErr)
	}
}

// shutdown releases runner resources and persists the dedup store. It runs after
// Run has returned, so no goroutine is mutating the store during the dump.
func shutdown(cfg *config.Config, newsRunner *news.News, log *logger.Log) {
	closeErr := newsRunner.Close()
	if closeErr != nil {
		log.WarnErr("failed to close news runner", closeErr)
	}

	dumpErr := cfg.Store.DumpToFile(cfg.StorageFilePath)
	if dumpErr != nil {
		log.Fatal("failed to dump storage file", dumpErr)
	}
}
