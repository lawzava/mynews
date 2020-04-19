package main

import (
	"crypto/md5" // nolint:gosec speed is higher concern than security in this use case
	"encoding/hex"
	"fmt"
	"mynews/broadcast"
	"mynews/store"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type feed struct {
	fp *gofeed.Parser

	sources   []string
	store     store.Store
	broadcast broadcast.Broadcast

	config
}

func newFeed(cfg config) (*feed, error) {
	const (
		defaultSleepDuration = 10 * time.Second
	)

	if cfg.sleepDurationBetweenBroadcasts == 0 {
		cfg.sleepDurationBetweenBroadcasts = defaultSleepDuration
	}

	stg, err := store.New(cfg.store)
	if err != nil {
		return nil, fmt.Errorf("creating store: %w", err)
	}

	brc, err := broadcast.New(cfg.broadcast)
	if err != nil {
		return nil, fmt.Errorf("creating broadcast: %w", err)
	}

	return &feed{
		fp:        gofeed.NewParser(),
		sources:   cfg.sources,
		config:    cfg,
		store:     stg,
		broadcast: brc,
	}, nil
}

func (p feed) run() error {
	startedAt := time.Now() // used to prevent broadcasting historical stories

	for {
		for _, source := range p.sources {
			sourceFeed, err := p.fp.ParseURL(source)
			if err != nil {
				return fmt.Errorf("parsing sourceFeed of source '%s': %w", source, err)
			}

			for _, story := range sourceFeed.Items {
				if story.PublishedParsed.Before(startedAt) {
					continue
				}

				storyID := buildStoryID(story.Title, story.Link)

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

				time.Sleep(p.config.sleepDurationBetweenBroadcasts)
			}
		}

		time.Sleep(p.config.sleepDurationBetweenFeedParsing)
	}
}

func buildStoryID(in ...string) string {
	h := md5.New() // nolint:gosec speed is higher concern than security in this use case

	_, _ = h.Write([]byte(strings.Join(in, "-")))

	return hex.EncodeToString(h.Sum(nil))
}
