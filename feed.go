package main

import (
	"crypto/md5" // nolint:gosec speed is higher concern than security in this use case
	"encoding/hex"
	"fmt"
	"mynews/broadcast"
	"time"

	"github.com/mmcdole/gofeed"
)

type feed struct {
	fp *gofeed.Parser

	sources []string

	config *config
}

func newFeed(cfg *config) *feed {
	const defaultSleepDuration = 10 * time.Second

	if cfg.sleepDurationBetweenBroadcasts == 0 {
		cfg.sleepDurationBetweenBroadcasts = defaultSleepDuration
	}

	return &feed{
		fp:      gofeed.NewParser(),
		sources: cfg.sources,
		config:  cfg,
	}
}

func (f *feed) run() error {
	for {
		for _, source := range f.sources {
			sourceFeed, err := f.fp.ParseURL(source)
			if err != nil {
				return fmt.Errorf("parsing sourceFeed of source '%s': %w", source, err)
			}

			if err = f.broadcastFeed(sourceFeed.Items); err != nil {
				return fmt.Errorf("processing feed of source '%s': %w", source, err)
			}
		}

		time.Sleep(f.config.sleepDurationBetweenFeedParsing)
	}
}

func (f *feed) broadcastFeed(stories []*gofeed.Item) (err error) {
	for _, story := range stories {
		if story.PublishedParsed.Before(f.config.ignoreStoriesBefore) {
			return nil
		}

		storyID := buildStoryIDFromURL(story.Link)

		var storyWasSent bool

		storyWasSent, err = f.config.store.KeyExists(storyID)
		if err != nil {
			return fmt.Errorf("checking if story was already sent: %w", err)
		}

		if storyWasSent {
			return nil
		}

		if err = f.config.store.PutKey(storyID); err != nil {
			return fmt.Errorf("registering story as sent: %w", err)
		}

		newBroadcastMessage := broadcast.Story{
			Title: story.Title,
			URL:   story.Link,
		}

		if err = f.config.broadcast.Send(newBroadcastMessage); err != nil {
			return fmt.Errorf("broadcasting story: %w", err)
		}

		time.Sleep(f.config.sleepDurationBetweenBroadcasts)
	}

	return nil
}

func buildStoryIDFromURL(link string) string {
	h := md5.New() // nolint:gosec speed is higher concern than security in this use case

	_, _ = h.Write([]byte(link))

	return hex.EncodeToString(h.Sum(nil))
}
