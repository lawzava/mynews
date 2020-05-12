package feed

import (
	"crypto/md5" // nolint:gosec speed is higher concern than security in this use case
	"encoding/hex"
	"fmt"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/parser"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type Feed struct {
	fp     *gofeed.Parser
	config *config.Config
}

func New(cfg *config.Config) *Feed {
	const defaultSleepDuration = 10 * time.Second

	if cfg.SleepDurationBetweenBroadcasts == 0 {
		cfg.SleepDurationBetweenBroadcasts = defaultSleepDuration
	}

	return &Feed{
		fp:     gofeed.NewParser(),
		config: cfg,
	}
}

func (f *Feed) Run(log *logger.Log) error {
	for {
		for _, source := range f.config.Sources {
			items, err := parser.ParseURL(source.URL)
			if err != nil {
				log.WarnErr(fmt.Sprintf("parsing feed of source '%s'", source), err)
				continue
			}

			if err = f.broadcastFeed(items, source); err != nil {
				log.WarnErr(fmt.Sprintf("broadcasting items for source '%s'", source), err)
			}
		}

		time.Sleep(f.config.SleepDurationBetweenFeedParsing)
	}
}

func (f *Feed) broadcastFeed(stories []parser.Item, source *config.Source) error {
	for _, story := range stories {
		if !storyMatchesConfig(story, source) {
			continue
		}

		storyID := buildStoryID(story.PublishedAt, story.Link)

		storyWasAlreadySent, err := f.config.Store.KeyExists(storyID)
		if err != nil {
			return fmt.Errorf("checking if story was already sent: %w", err)
		}

		if storyWasAlreadySent {
			continue
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

func storyMatchesConfig(story parser.Item, source *config.Source) bool {
	if story.PublishedAtParsed.IsZero() {
		return false
	}

	if story.PublishedAtParsed.Before(source.IgnoreStoriesBefore) {
		return false
	}

	if len(source.MustExcludeKeywords) != 0 {
		if includesKeywords(story.Title, source.MustExcludeKeywords) {
			return false
		}
	}

	if len(source.MustIncludeKeywords) != 0 {
		if !includesKeywords(story.Title, source.MustIncludeKeywords) {
			return false
		}
	}

	return true
}

func includesKeywords(target string, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}

	target = strings.ToLower(target)

	for _, keyword := range keywords {
		if strings.Contains(target, keyword) {
			return true
		}
	}

	return false
}

func buildStoryID(published, link string) string {
	h := md5.New() // nolint:gosec speed is higher concern than security in this use case

	_, _ = h.Write([]byte(published + link))

	return hex.EncodeToString(h.Sum(nil))
}
