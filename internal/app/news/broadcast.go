package news

import (
	"context"
	//nolint:gosec // md5 used for key generation, nothing sensitive
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/parser"
	"strings"
	"time"
)

const scoringTimeout = 30 * time.Second

func (n News) broadcastFeed(
	briadcastClient broadcast.Broadcast,
	stories []parser.Item,
	source *config.Source,
	log *logger.Log,
) error {
	for _, story := range stories {
		if !storyMatchesConfig(story, source) {
			continue
		}

		storyID := buildStoryID(story.PublishedAt, story.Link, source.StatusPage)

		storyWasAlreadySent, err := n.cfg.Store.KeyExists(briadcastClient.Name(), storyID)
		if err != nil {
			return fmt.Errorf("checking if story was already sent: %w", err)
		}

		if storyWasAlreadySent {
			continue
		}

		err = n.cfg.Store.PutKey(briadcastClient.Name(), storyID)
		if err != nil {
			return fmt.Errorf("registering story as sent: %w", err)
		}

		newBroadcastMessage := broadcast.Story{
			Title:  story.Title,
			URL:    story.Link,
			Score:  0,
			Reason: "",
		}

		// Score the story if scoring is enabled
		if n.scorer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), scoringTimeout)
			score, scoreErr := n.scorer.Score(ctx, story.Title)

			cancel()

			if scoreErr != nil {
				log.WarnErr("scoring story", scoreErr)
			} else {
				newBroadcastMessage.Score = score.Value
				newBroadcastMessage.Reason = score.Reason
			}
		}

		err = briadcastClient.Send(newBroadcastMessage)
		if err != nil {
			return fmt.Errorf("broadcasting story: %w", err)
		}

		time.Sleep(n.cfg.SleepDurationBetweenBroadcasts)
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

func buildStoryID(published, link string, statusPage bool) string {
	hash := md5.New() //nolint:gosec // speed is higher concern than security in this use case

	if statusPage {
		_, _ = hash.Write([]byte(published + link))
	} else {
		_, _ = hash.Write([]byte(link))
	}

	return hex.EncodeToString(hash.Sum(nil))
}
