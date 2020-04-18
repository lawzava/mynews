package main

import (
	"fmt"
	"news/store"
	"time"

	"github.com/mmcdole/gofeed"
)

type parser struct {
	fp *gofeed.Parser

	sources []string
	store   store.Store

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

	return &parser{
		fp:      gofeed.NewParser(),
		sources: cfg.sources,
		config:  cfg,
		store:   stg,
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

				storyWasSent, err := p.store.KeyExists(storyID)
				if err != nil {
					return fmt.Errorf("checking if story was already sent: %w", err)
				}

				if storyWasSent {
					continue
				}

				if err := p.store.PutKey(storyID); err != nil {
					return fmt.Errorf("registering story as sent: %w", err)
				}

				fmt.Printf("%s | %s \n",
					story.Title, story.Link)
			}
		}

		time.Sleep(p.config.sleepDurationBetweenRuns)
	}
}
