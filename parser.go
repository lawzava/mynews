package main

import (
	"fmt"
	"news/broadcast"
	"news/store"
	"time"

	"github.com/mmcdole/gofeed"
)

type parser struct {
	fp *gofeed.Parser

	sources   []string
	store     store.Store
	broadcast broadcast.Broadcast

	config
}

func newParser(cfg config) (*parser, error) {
	const (
		defaultSleepDuration = 10 * time.Second
	)

	if cfg.sleepDurationBetweenRuns == 0 {
		cfg.sleepDurationBetweenRuns = defaultSleepDuration
	}

	stg, err := store.New(cfg.store)
	if err != nil {
		return nil, fmt.Errorf("creating store: %w", err)
	}

	brc, err := broadcast.New(cfg.broadcast)
	if err != nil {
		return nil, fmt.Errorf("creating broadcast: %w", err)
	}

	return &parser{
		fp:        gofeed.NewParser(),
		sources:   cfg.sources,
		config:    cfg,
		store:     stg,
		broadcast: brc,
	}, nil
}

func (p parser) run() error {
	for {
		for _, source := range p.sources {
			feed, err := p.fp.ParseURL(source)
			if err != nil {
				return fmt.Errorf("parsing feed of source '%s': %w", source, err)
			}

			for _, story := range feed.Items {
				storyID := buildStoryID(source, story.Title, story.Link, story.Published)

				var storyWasSent bool

				storyWasSent, err = p.store.KeyExists(storyID)
				if err != nil {
					return fmt.Errorf("checking if story was already sent: %w", err)
				}

				if storyWasSent {
					continue
				}

				if err = p.store.PutKey(storyID); err != nil {
					return fmt.Errorf("registering story as sent: %w", err)
				}

				newBroadcastMessage := broadcast.Message{
					Title: story.Title,
					Link:  story.Link,
				}

				if err = p.broadcast.Send(newBroadcastMessage); err != nil {
					return fmt.Errorf("broadcasting story: %w", err)
				}
			}
		}

		time.Sleep(p.config.sleepDurationBetweenRuns)
	}
}
