package config

import (
	"flag"
	"fmt"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/storage"
	"os"
	"time"
)

type Source struct {
	URL                 string
	IgnoreStoriesBefore time.Time
	MustIncludeKeywords []string
	MustExcludeKeywords []string
	StatusPage          bool // used when links in feed does not change but timestamp changes
}

type Config struct {
	SleepDurationBetweenFeedParsing time.Duration
	SleepDurationBetweenBroadcasts  time.Duration

	Store           storage.Storage
	StorageFilePath string

	Apps []App
}

type App struct {
	Sources   []*Source
	Broadcast broadcast.Broadcast
}

const (
	configFilePathEnvironmentVariable = "MYNEWS_CONFIG_FILE"
	configFileDefaultLocation         = "$HOME/.config/mynews/config.json"

	storageFilePathEnvironmentVariable = "MYNEWS_STORAGE_FILE"
	storageFileDefaultLocation         = "$HOME/.config/mynews/data.json"

	defaultSleepDuration = 10 * time.Second
)

func New(log *logger.Log) (*Config, error) {
	var (
		configFileLocation, storageFileLocation string
		createSample                            bool
	)

	flag.StringVar(&configFileLocation, "config", "",
		fmt.Sprintf("Path to config file. Defaults to '%s'.", configFileDefaultLocation))

	flag.StringVar(&storageFileLocation, "storage", "",
		fmt.Sprintf("Path to storage file. Defaults to '%s'.", storageFileDefaultLocation))

	flag.BoolVar(&createSample, "create", false, `Creates a sample config file.`)
	flag.Parse()

	if configFileLocation == "" {
		configFileLocation = configFileDefaultLocation

		if e := os.Getenv(configFilePathEnvironmentVariable); e != "" {
			configFileLocation = e
		}
	}

	if createSample {
		if err := createSampleFile(configFileLocation); err != nil {
			return nil, fmt.Errorf("creating new sample config: %w", err)
		}

		log.Info(fmt.Sprintf(`Created a sample config file at '%s'`, configFileLocation))

		return nil, nil
	}

	config, err := fromFile(configFileLocation, storageFileLocation, log)
	if err != nil {
		return nil, fmt.Errorf("parsing config from file: %w", err)
	}

	return config, nil
}
