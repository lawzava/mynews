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
	sleepDurationBetweenFeedParsing time.Duration
	sleepDurationBetweenBroadcasts  time.Duration
	sources                         []string

	ignoreStoriesBefore time.Time

	store     store.Store
	broadcast broadcast.Broadcast
}

func parseConfig() (*config, error) {
	const (
		nameSources           = "sources"
		nameFeedParseInterval = "feedParseInterval"
		nameBroadcastInterval = "broadcastInterval"
		nameIgnoreBefore      = "ignoreBefore"

		nameStore            = "store"
		nameStoreRedisURI    = "storeRedisURI"
		nameStorePostgresURI = "storePostgresURI"

		nameBroadcastType    = "broadcastType"
		nameTelegramBotToken = "telegramBotToken"
		nameTelegramChatID   = "telegramChatID"
	)

	var (
		sources string

		intervalBetweenRuns      time.Duration
		intervalBetweenBroadcast time.Duration

		ignoreBefore string

		storeType     string
		broadcastType string

		storeConfig     store.Config
		broadcastConfig broadcast.Config

		cfg config
		err error
	)

	// Feed configuration
	flag.StringVar(&sources, nameSources, "https://hnrss.org/newest.atom",
		"rss/atom source URLs separated by a comma")
	flag.DurationVar(&intervalBetweenRuns, nameFeedParseInterval, 5*time.Minute, "duration between each feed parse")      // nolint:gomnd ignore magic number in example
	flag.DurationVar(&intervalBetweenBroadcast, nameBroadcastInterval, 10*time.Second, "duration between each broadcast") // nolint:gomnd ignore magic number in example

	// Stories filtering
	flag.StringVar(&ignoreBefore, nameIgnoreBefore, "2020-04-20T00:00:00Z",
		"if specified, stories published before this date will be ignored, must be in RFC3339 format")

	// Store configuration
	flag.StringVar(&storeType, nameStore, "memory",
		"store type to use. Valid values are: 'memory', 'postgres', 'redis'")
	flag.StringVar(&storeConfig.RedisDB.RedisURI, nameStoreRedisURI,
		"redis://localhost:6379",
		"redis access URI")
	flag.StringVar(&storeConfig.PostgresDB.DatabaseURI, nameStorePostgresURI,
		"postgres://user:password@localhost:6379/db",
		"postgres access URI")

	flag.StringVar(&broadcastType, nameBroadcastType, "stdout",
		"broadcast type to use. Valid values are: 'telegram', 'stdout'")

	// Broadcast configuration
	flag.StringVar(&broadcastConfig.Telegram.BotAPIToken, nameTelegramBotToken, "",
		"telegram bot token to use with 'telegram' broadcast type")
	flag.StringVar(&broadcastConfig.Telegram.ChatID, nameTelegramChatID, "",
		"telegram chatID to use with 'telegram' broadcast type")

	flag.Parse()

	cfg.sources = strings.Split(sources, ",")
	cfg.sleepDurationBetweenFeedParsing = intervalBetweenRuns
	cfg.sleepDurationBetweenBroadcasts = intervalBetweenBroadcast

	if ignoreBefore != "" {
		cfg.ignoreStoriesBefore, err = time.Parse(time.RFC3339, ignoreBefore)
		if err != nil {
			return nil, fmt.Errorf("parsing ignore before: %w", err)
		}
	}

	cfg.store, err = parseStore(storeType, storeConfig)
	if err != nil {
		return nil, fmt.Errorf("parsing store type: %w", err)
	}

	cfg.broadcast, err = parseBroadcast(broadcastType, broadcastConfig)
	if err != nil {
		return nil, fmt.Errorf("parsing broadcast type: %w", err)
	}

	return &cfg, nil
}

// Defaults to "STDOUT"
func parseBroadcast(name string, cfg broadcast.Config) (broadcast.Broadcast, error) {
	switch strings.ToUpper(name) {
	case "TELEGRAM":
		return cfg.Telegram.New()
	default:
		return cfg.StdOut.New()
	}
}

// Defaults to "MEMORY"
func parseStore(name string, cfg store.Config) (store.Store, error) {
	switch strings.ToUpper(name) {
	case "REDIS":
		return cfg.RedisDB.New()
	case "POSTGRES":
		return cfg.PostgresDB.New()
	default:
		return cfg.MemoryDB.New()
	}
}
