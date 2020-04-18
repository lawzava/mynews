package main

import (
	"flag"
	"fmt"
	"mynews/broadcast"
	"mynews/store"
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
	const (
		nameSources            = "sources"
		nameInterval           = "interval"
		nameStore              = "store"
		nameStoreAccessDetails = "storeAccessDetails"
		nameBroadcastType      = "broadcastType"
		nameTelegramBotToken   = "telegramBotToken"
		nameTelegramChatID     = "telegramChatID"
	)

	var (
		sources             string
		storeType           string
		storeAccessDetails  string
		intervalBetweenRuns uint64

		broadcastType    string
		telegramBotToken string
		telegramChatID   string
	)

	flag.StringVar(&sources, nameSources, "https://hnrss.org/newest.atom", "rss/atom source URLs separated by a comma")
	flag.Uint64Var(&intervalBetweenRuns, nameInterval, 60, "interval in seconds between each feed parsing run")
	flag.StringVar(&storeType, nameStore, "memory",
		"store type to use. Valid values are: 'memory' (persistent hash map), 'postgres', 'redis'")
	flag.StringVar(&storeAccessDetails, nameStoreAccessDetails, "redis://localhost:6379",
		"store access URI if the type is not 'memory'")

	flag.StringVar(&broadcastType, nameBroadcastType, "telegram", "broadcast type to use. Valid values are: 'telegram'")
	flag.StringVar(&telegramBotToken, nameTelegramBotToken, "", "telegram bot token to use with 'telegram' broadcast type")
	flag.StringVar(&telegramChatID, nameTelegramChatID, "", "telegram chatID to use with 'telegram' broadcast type")

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
