package main

import (
	"crypto/md5" // nolint:gosec speed is higher concern than security in this use case
	"encoding/hex"
	"fmt"
	"mynews/broadcast"
	"mynews/config"
	"time"

	"github.com/mmcdole/gofeed"
)

type feed struct {
	fp     *gofeed.Parser
	config *config.Config
}

func newFeed(cfg *config.Config) *feed {
	const defaultSleepDuration = 10 * time.Second

	if cfg.SleepDurationBetweenBroadcasts == 0 {
		cfg.SleepDurationBetweenBroadcasts = defaultSleepDuration
	}

	return &feed{
		fp:     gofeed.NewParser(),
		config: cfg,
	}
}

func (f *feed) run() error {
	for {
		for _, source := range f.config.Sources {
			sourceFeed, err := f.fp.ParseURL(source.URL)
			if err != nil {
				return fmt.Errorf("parsing sourceFeed of source '%s': %w", source, err)
			}

			if err = f.broadcastFeed(sourceFeed.Items, source); err != nil {
				return fmt.Errorf("processing feed of source '%s': %w", source, err)
			}
		}

		time.Sleep(f.config.SleepDurationBetweenFeedParsing)
	}
}

func (f *feed) broadcastFeed(stories []*gofeed.Item, source config.Source) (err error) {
	for _, story := range stories {
		if story.PublishedParsed.Before(source.IgnoreStoriesBefore) {
			return nil
		}

		storyID := buildStoryIDFromURL(story.Link)

		var storyWasSent bool

		storyWasSent, err = f.config.Store.KeyExists(storyID)
		if err != nil {
			return fmt.Errorf("checking if story was already sent: %w", err)
		}

		if storyWasSent {
			return nil
		}

		if err = f.config.Store.PutKey(storyID); err != nil {
			return fmt.Errorf("registering story as sent: %w", err)
		}

		newBroadcastMessage := broadcast.Story{
			Title: story.Title,
			URL:   story.Link,
		}

		if err = f.config.Broadcast.Send(newBroadcastMessage); err != nil {
			return fmt.Errorf("broadcasting story: %w", err)
		}

		time.Sleep(f.config.SleepDurationBetweenBroadcasts)
	}

	return nil
}

func buildStoryIDFromURL(link string) string {
	h := md5.New() // nolint:gosec speed is higher concern than security in this use case

	_, _ = h.Write([]byte(link))

	return hex.EncodeToString(h.Sum(nil))
}
