package config

import (
	"errors"
	"flag"
	"fmt"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/storage"
	"os"
	"time"
)

var ErrCreatedNewFile = errors.New("created new file")

type Source struct {
	URL                  string
	IgnoreStoriesBefore  time.Time
	MustIncludeKeywords  []string
	MustExcludeKeywords  []string
	MatchKeywordsAsWords bool
	MatchKeywordsOrScore bool
	StatusPage           bool     // used when links in feed does not change but timestamp changes
	Interests            []string // per-source scoring interests; falls back to the global list
	MinScore             *float64 // optional per-source threshold; nil falls back to the global threshold
	MinHackerNewsScore   *int     // optional native HN points threshold
}

type Config struct {
	SleepDurationBetweenFeedParsing time.Duration
	SleepDurationBetweenBroadcasts  time.Duration

	Store           storage.Storage
	StorageFilePath string

	Apps []App

	Scoring *ScoringConfig

	// MetricsAddr, when set (e.g. ":8080"), serves /healthz and /metrics.
	MetricsAddr string
}

type ScoringConfig struct {
	Enabled   bool
	Provider  string   // "embedding" or "keyword"
	Interests []string // Topics to score stories against
	ModelName string   // HuggingFace model name (for embedding provider)
	ModelDir  string   // Directory to cache models
	MinScore  float64  // Stories scoring below this (0-1) are not broadcast

	// SummarizeArticles, with the embedding provider, fetches the article and
	// attaches its title-most-relevant sentence when a feed entry has no summary.
	SummarizeArticles bool
}

type App struct {
	Sources   []*Source
	Broadcast broadcast.Broadcast
	Digest    *DigestConfig // when set, batch the top-N stories per interval
}

type DigestConfig struct {
	Every time.Duration // how often to emit a digest
	TopN  int           // how many highest-scoring stories to include
}

const (
	configFilePathEnvironmentVariable = "MYNEWS_CONFIG_FILE"
	configFileDefaultLocation         = "$HOME/.config/mynews/config.json"

	storageFilePathEnvironmentVariable = "MYNEWS_STORAGE_FILE"
	storageFileDefaultLocation         = "$HOME/.config/mynews/data.json"

	// Defaults applied when the corresponding config field is omitted, so a
	// minimal config works well out of the box.
	defaultFeedParsingInterval = 5 * time.Minute
	defaultBroadcastInterval   = 10 * time.Second
	defaultIgnoreStoriesBefore = 24 * time.Hour
)

func New(log *logger.Log) (*Config, error) {
	var (
		configFileLocation, storageFileLocation, importOPML string
		createSample                                        bool
	)

	flag.StringVar(&configFileLocation, "config", "",
		fmt.Sprintf("Path to config file. Defaults to '%s'.", configFileDefaultLocation))

	flag.StringVar(&storageFileLocation, "storage", "",
		fmt.Sprintf("Path to storage file. Defaults to '%s'.", storageFileDefaultLocation))

	flag.BoolVar(&createSample, "create", false, `Creates a sample config file.`)
	flag.StringVar(&importOPML, "import-opml", "", "Path to an OPML file to import feeds from into a new config.")
	flag.Parse()

	if configFileLocation == "" {
		configFileLocation = os.ExpandEnv(configFileDefaultLocation)

		if e := os.Getenv(configFilePathEnvironmentVariable); e != "" {
			configFileLocation = e
		}
	}

	if createSample {
		err := createSampleFile(configFileLocation)
		if err != nil {
			return nil, fmt.Errorf("creating new sample config: %w", err)
		}

		log.Info(fmt.Sprintf(`Created a sample config file at '%s'`, configFileLocation))

		return nil, fmt.Errorf("created sample config file: %w", ErrCreatedNewFile)
	}

	if importOPML != "" {
		err := importOPMLFile(importOPML, configFileLocation, log)
		if err != nil {
			return nil, fmt.Errorf("importing OPML: %w", err)
		}

		log.Info(fmt.Sprintf(`Created config from OPML at '%s'`, configFileLocation))

		return nil, fmt.Errorf("created config from OPML: %w", ErrCreatedNewFile)
	}

	config, err := fromFile(configFileLocation, storageFileLocation, log)
	if err != nil {
		return nil, fmt.Errorf("parsing config from file: %w", err)
	}

	return config, nil
}
