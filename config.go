package main

import (
	"flag"
	"fmt"
	"news/broadcast"
	"news/store"
	"strings"
	"time"
)

type config struct {
	sleepDurationBetweenRuns time.Duration
	sources                  []string

	store     store.Config
	broadcast broadcast.Config
}

func (cfg config) validate() error {
	if err := cfg.store.Validate(); err != nil {
		return fmt.Errorf("validating store config: %w", err)
	}

	if err := cfg.broadcast.Validate(); err != nil {
		return fmt.Errorf("validating broadcast config: %w", err)
	}

	return nil
}

func parseConfig() (config, error) {
	var (
		sources             string
		storeType           string
		storeAccessDetails  string
		intervalBetweenRuns uint64

		broadcastType    string
		telegramBotToken string
		telegramChatID   int64
	)

	flag.StringVar(&sources, "sources", "https://hnrss.org/newest.atom", "rss/atom source URLs separated by a comma")
	flag.Uint64Var(&intervalBetweenRuns, "interval", 60, "interval in seconds between each feed parsing run")
	flag.StringVar(&storeType, "store", "memory",
		"store type to use. Valid values are: 'memory' (persistent hash map), 'postgres', 'redis'")
	flag.StringVar(&storeAccessDetails, "storeAccessDetails", "redis://localhost:6379",
		"store access URI if the type is not 'memory'")

	flag.StringVar(&broadcastType, "broadcastType", "telegram", "broadcast type to use. Valid values are: 'telegram'")
	flag.StringVar(&telegramBotToken, "telegramBottoken", "", "telegram bot token to use with 'telegram' broadcast type")
	flag.Int64Var(&telegramChatID, "telegramChatID", 0, "telegram chatID to use with 'telegram' broadcast type")

	flag.Parse()

	var cfg config

	cfg.sources = strings.Split(sources, ",")
	cfg.sleepDurationBetweenRuns = time.Second * time.Duration(intervalBetweenRuns)

	// Store config
	st, err := store.ParseType(storeType)
	if err != nil {
		return cfg, fmt.Errorf("parsing store type: %w", err)
	}

	cfg.store.Type = st
	cfg.store.AccessDetails = storeAccessDetails

	// Broadcast config
	bt, err := broadcast.ParseType(broadcastType)
	if err != nil {
		return cfg, fmt.Errorf("parsing broadcast type: %w", err)
	}

	cfg.broadcast.Type = bt
	cfg.broadcast.Telegram.BotAPIToken = telegramBotToken
	cfg.broadcast.Telegram.ChatID = telegramChatID

	return cfg, cfg.validate()
}
