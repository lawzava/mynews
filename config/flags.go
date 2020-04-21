package config

import (
	"fmt"
	"strings"
	"time"
)

func fromFlags(f *flags) (*Config, error) {
	var (
		cfg Config
		err error

		ignoreBefore time.Time
	)

	if f.ignoreBefore != "" {
		ignoreBefore, err = time.Parse(time.RFC3339, f.ignoreBefore)
		if err != nil {
			return nil, fmt.Errorf("parsing ignore before: %w", err)
		}
	}

	for _, url := range strings.Split(f.sources, ",") {
		cfg.Sources = append(cfg.Sources, Source{URL: url, IgnoreStoriesBefore: ignoreBefore})
	}

	cfg.SleepDurationBetweenFeedParsing = f.intervalBetweenRuns
	cfg.SleepDurationBetweenBroadcasts = f.intervalBetweenBroadcast

	cfg.Store, err = parseStore(f.storeType, f.storeConfig)
	if err != nil {
		return nil, fmt.Errorf("parsing Store type: %w", err)
	}

	cfg.Broadcast, err = parseBroadcast(f.broadcastType, f.broadcastConfig)
	if err != nil {
		return nil, fmt.Errorf("parsing Broadcast type: %w", err)
	}

	return &cfg, nil
}
