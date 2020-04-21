package config

import (
	"flag"
	"mynews/broadcast"
	"mynews/store"
	"time"
)

type Source struct {
	URL                 string    `json:"url"`
	IgnoreStoriesBefore time.Time `json:"ignoreStoriesBefore"`
	MustIncludeKeywords []string  `json:"mustIncludeKeywords"`
	MustExcludeKeywords []string  `json:"mustExcludeKeywords"`
}

type Config struct {
	SleepDurationBetweenFeedParsing time.Duration
	SleepDurationBetweenBroadcasts  time.Duration

	Sources []*Source

	Store     store.Store
	Broadcast broadcast.Broadcast
}

func New() (*Config, error) {
	f := parseFlags()

	if f.configFilePath != "" {
		if f.createSampleConfigFile {
			return nil, createSampleFile(f.configFilePath)
		}

		return fromFile(f.configFilePath)
	}

	return fromFlags(&f)
}

type flags struct {
	configFilePath         string
	createSampleConfigFile bool

	sources string

	intervalBetweenRuns      time.Duration
	intervalBetweenBroadcast time.Duration

	ignoreBefore string

	storeType     string
	broadcastType string

	storeConfig     store.Config
	broadcastConfig broadcast.Config
}

func parseFlags() flags {
	const (
		nameConfigFilePath         = "configFile"
		nameCreateSampleConfigFile = "createSampleConfigFile"

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

	var f flags

	// Config file configuration
	flag.StringVar(&f.configFilePath, nameConfigFilePath, "",
		`path to config file. If specified, the remaining flags will be ignored. 
				If file does not exist, it will be created with sample configuration`)
	flag.BoolVar(&f.createSampleConfigFile, nameCreateSampleConfigFile, false,
		"if true, will create a sample config file and exit")

	// Feed configuration
	flag.StringVar(&f.sources, nameSources, "https://hnrss.org/newest.atom",
		"rss/atom source URLs separated by a comma")
	flag.DurationVar(&f.intervalBetweenRuns, nameFeedParseInterval, 5*time.Minute, "duration between each feed parse")      // nolint:gomnd ignore magic number in example
	flag.DurationVar(&f.intervalBetweenBroadcast, nameBroadcastInterval, 10*time.Second, "duration between each Broadcast") // nolint:gomnd ignore magic number in example

	// Stories filtering
	flag.StringVar(&f.ignoreBefore, nameIgnoreBefore, "2020-04-20T00:00:00Z",
		"if specified, stories published before this date will be ignored, must be in RFC3339 format")

	// Store configuration
	flag.StringVar(&f.storeType, nameStore, "memory",
		"Store type to use. Valid values are: 'memory', 'postgres', 'redis'")
	flag.StringVar(&f.storeConfig.RedisDB.RedisURI, nameStoreRedisURI,
		"redis://localhost:6379",
		"redis access URI")
	flag.StringVar(&f.storeConfig.PostgresDB.DatabaseURI, nameStorePostgresURI,
		"postgres://user:password@localhost:6379/db",
		"postgres access URI")

	flag.StringVar(&f.broadcastType, nameBroadcastType, "stdout",
		"Broadcast type to use. Valid values are: 'telegram', 'stdout'")

	// Broadcast configuration
	flag.StringVar(&f.broadcastConfig.Telegram.BotAPIToken, nameTelegramBotToken, "",
		"telegram bot token to use with 'telegram' Broadcast type")
	flag.StringVar(&f.broadcastConfig.Telegram.ChatID, nameTelegramChatID, "",
		"telegram chatID to use with 'telegram' Broadcast type")

	flag.Parse()

	return f
}
