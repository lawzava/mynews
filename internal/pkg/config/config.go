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

	Sources []*Source

	Store     storage.Storage
	Broadcast broadcast.Broadcast
}

func New(log *logger.Log) (*Config, error) {
	const (
		configFilePathEnvironmentVariable = "MYNEWS_CONFIG_FILE"
		configFileDefaultLocation         = "$HOME/.config/mynews/config.json"
	)

	var (
		fileLocation string
		createSample bool
	)

	flag.StringVar(&fileLocation, "config", "",
		fmt.Sprintf("Path to config file. Defaults to '%s'.", configFileDefaultLocation))
	flag.BoolVar(&createSample, "create", false, `Creates a sample config file.`)
	flag.Parse()

	if fileLocation == "" {
		fileLocation = configFileDefaultLocation

		if e := os.Getenv(configFilePathEnvironmentVariable); e != "" {
			fileLocation = e
		}
	}

	if createSample {
		if err := createSampleFile(fileLocation); err != nil {
			return nil, fmt.Errorf("creating new sample config: %w", err)
		}

		log.Info(fmt.Sprintf(`Created a sample config file at '%s'`, fileLocation))

		return nil, nil
	}

	return fromFile(fileLocation, log)
}
