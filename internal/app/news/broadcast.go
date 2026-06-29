package news

import (
	"context"
	"html"
	"net/url"
	"regexp"

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
	ctx context.Context,
	broadcastClient broadcast.Broadcast,
	stories []parser.Item,
	source *config.Source,
	log *logger.Log,
) error {
	for _, story := range stories {
		if !storyMatchesConfig(&story, source) {
			continue
		}

		storyID := buildStoryID(story.PublishedAt, story.Link, source.StatusPage)

		storyWasAlreadySent, err := n.cfg.Store.KeyExists(broadcastClient.Name(), storyID)
		if err != nil {
			return fmt.Errorf("checking if story was already sent: %w", err)
		}

		if storyWasAlreadySent {
			continue
		}

		scored := n.scoreStory(ctx, story.Title, log)

		if !scored.passes {
			// Below the relevance threshold: record as seen so it is not
			// re-scored every cycle, but do not broadcast it.
			err = n.cfg.Store.PutKey(broadcastClient.Name(), storyID)
			if err != nil {
				return fmt.Errorf("registering story as sent: %w", err)
			}

			continue
		}

		newBroadcastMessage := broadcast.Story{
			Title:   story.Title,
			URL:     story.Link,
			Summary: cleanSummary(story.Description),
			Score:   scored.value,
			Reason:  scored.reason,
		}

		err = broadcastClient.Send(newBroadcastMessage)
		if err != nil {
			return fmt.Errorf("broadcasting story: %w", err)
		}

		// Record the story as sent only after a successful broadcast, so a
		// transient Send failure is retried on the next parse cycle.
		err = n.cfg.Store.PutKey(broadcastClient.Name(), storyID)
		if err != nil {
			return fmt.Errorf("registering story as sent: %w", err)
		}

		if !sleep(ctx, n.cfg.SleepDurationBetweenBroadcasts) {
			return nil
		}
	}

	return nil
}

type scoredStory struct {
	value  float64
	reason string
	passes bool // false when below the configured MinScore threshold
}

// scoreStory scores a title against the configured interests and reports whether
// it clears the relevance threshold. When scoring is disabled or errors, the story
// passes (value 0) so it is never silently dropped on a scorer failure.
func (n News) scoreStory(ctx context.Context, title string, log *logger.Log) scoredStory {
	if n.scorer == nil {
		return scoredStory{value: 0, reason: "", passes: true}
	}

	ctx, cancel := context.WithTimeout(ctx, scoringTimeout)
	defer cancel()

	score, err := n.scorer.Score(ctx, title)
	if err != nil {
		log.WarnErr("scoring story", err)

		return scoredStory{value: 0, reason: "", passes: true}
	}

	return scoredStory{
		value:  score.Value,
		reason: score.Reason,
		passes: score.Value >= n.cfg.Scoring.MinScore,
	}
}

const maxSummaryLen = 280

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// cleanSummary strips HTML tags and entities from a feed description, collapses
// whitespace, and truncates to a short single-line snippet for broadcasting.
func cleanSummary(raw string) string {
	if raw == "" {
		return ""
	}

	text := htmlTagRe.ReplaceAllString(raw, " ")
	text = html.UnescapeString(text)
	text = strings.Join(strings.Fields(text), " ")

	runes := []rune(text)
	if len(runes) > maxSummaryLen {
		text = strings.TrimSpace(string(runes[:maxSummaryLen])) + "…"
	}

	return text
}

func storyMatchesConfig(story *parser.Item, source *config.Source) bool {
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
	link = normalizeURL(link)

	hash := md5.New() //nolint:gosec // speed is higher concern than security in this use case

	if statusPage {
		_, _ = hash.Write([]byte(published + link))
	} else {
		_, _ = hash.Write([]byte(link))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// trackingParams are query parameters that identify a referral, not the content,
// so the same article shared via different sources dedups to one story.
//
//nolint:gochecknoglobals // read-only lookup table
var trackingParams = map[string]bool{
	"fbclid": true, "gclid": true, "msclkid": true, "igshid": true,
	"mc_cid": true, "mc_eid": true, "ref": true, "ref_src": true,
	"source": true, "cmpid": true, "spm": true,
}

// normalizeURL lower-cases the host and strips the fragment and tracking query
// parameters so cosmetically different links to the same article collide.
func normalizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""

	if parsed.RawQuery != "" {
		query := parsed.Query()
		for key := range query {
			lowered := strings.ToLower(key)
			if strings.HasPrefix(lowered, "utm_") || trackingParams[lowered] {
				query.Del(key)
			}
		}

		parsed.RawQuery = query.Encode()
	}

	return parsed.String()
}
