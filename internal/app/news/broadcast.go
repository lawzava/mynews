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
	dig *digest,
	stories []parser.Item,
	source *config.Source,
	log *logger.Log,
) error {
	for storyIdx := range stories {
		stop, err := n.handleStory(ctx, broadcastClient, dig, &stories[storyIdx], source, log)
		if err != nil {
			return err
		}

		if stop {
			return nil
		}
	}

	return nil
}

// handleStory processes a single story: filter, dedup, score, then either buffer
// it for the digest or broadcast it immediately. It reports stop=true when the
// run context was canceled mid-broadcast (shutdown).
func (n News) handleStory(
	ctx context.Context,
	broadcastClient broadcast.Broadcast,
	dig *digest,
	story *parser.Item,
	source *config.Source,
	log *logger.Log,
) (bool, error) {
	if !storyMatchesConfig(story, source) {
		return false, nil
	}

	storyID := buildStoryID(story.PublishedAt, story.Link, source.StatusPage)

	storyWasAlreadySent, err := n.cfg.Store.KeyExists(broadcastClient.Name(), storyID)
	if err != nil {
		return false, fmt.Errorf("checking if story was already sent: %w", err)
	}

	if storyWasAlreadySent {
		return false, nil
	}

	scored := n.scoreStory(ctx, source, story.Title, log)
	if !scored.passes {
		// Below the threshold: record as seen so it is not re-scored each cycle.
		return false, n.markSent(broadcastClient, storyID)
	}

	message := broadcast.Story{
		Title:   story.Title,
		URL:     story.Link,
		Summary: cleanSummary(story.Description),
		Score:   scored.value,
		Reason:  scored.reason,
	}

	if dig != nil {
		// Buffer for the next digest and mark seen so it is collected only once.
		dig.add(message)

		return false, n.markSent(broadcastClient, storyID)
	}

	err = broadcastClient.Send(message)
	if err != nil {
		return false, fmt.Errorf("broadcasting story: %w", err)
	}

	// Record the story as sent only after a successful broadcast, so a transient
	// Send failure is retried on the next parse cycle.
	err = n.markSent(broadcastClient, storyID)
	if err != nil {
		return false, err
	}

	return !sleep(ctx, n.cfg.SleepDurationBetweenBroadcasts), nil
}

func (n News) markSent(broadcastClient broadcast.Broadcast, storyID string) error {
	err := n.cfg.Store.PutKey(broadcastClient.Name(), storyID)
	if err != nil {
		return fmt.Errorf("registering story as sent: %w", err)
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
func (n News) scoreStory(ctx context.Context, source *config.Source, title string, log *logger.Log) scoredStory {
	activeScorer := n.scorer
	if sourceScorer, ok := n.sourceScorers[source]; ok {
		activeScorer = sourceScorer
	}

	if activeScorer == nil {
		return scoredStory{value: 0, reason: "", passes: true}
	}

	ctx, cancel := context.WithTimeout(ctx, scoringTimeout)
	defer cancel()

	score, err := activeScorer.Score(ctx, title)
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
