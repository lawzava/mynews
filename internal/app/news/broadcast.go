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
	"mynews/internal/pkg/article"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/parser"
	"mynews/internal/pkg/scorer"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
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
	select {
	case <-ctx.Done():
		return true, nil // shutting down: stop before scoring/sending
	default:
	}

	filterResult := n.storyPassesSourceFilters(story, source)
	if !filterResult.matchesSource {
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
	if !passesRelevanceFilter(scored, filterResult.keywordPasses) {
		// Below the threshold: record as seen so it is not re-scored each cycle.
		return false, n.markSent(broadcastClient, storyID)
	}

	message := broadcast.Story{
		Title:   story.Title,
		URL:     story.Link,
		Summary: n.storySummary(ctx, story, log),
		Score:   scored.value,
		Reason:  scored.reason,
	}

	if dig != nil {
		// Buffer for the next digest; it is marked sent only after a successful
		// flush, so an unsent story is re-collected (and re-ranked) next cycle.
		dig.add(&digestEntry{story: message, storyID: storyID})

		return false, nil
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

type sourceFilterResult struct {
	matchesSource bool
	keywordPasses bool
}

func (n News) storyPassesSourceFilters(story *parser.Item, source *config.Source) sourceFilterResult {
	if source.MatchKeywordsOrScore && n.scorer != nil {
		keywordPasses := sourceIncludesKeywords(
			story.Title, source.MustIncludeKeywords, source.MatchKeywordsAsWords)

		return sourceFilterResult{
			matchesSource: storyPassesRequiredFilters(story, source),
			keywordPasses: keywordPasses,
		}
	}

	return sourceFilterResult{
		matchesSource: storyMatchesConfig(story, source),
		keywordPasses: false,
	}
}

func passesRelevanceFilter(scored scoredStory, keywordPasses bool) bool {
	return scored.passes || keywordPasses
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

	minScore := n.cfg.Scoring.MinScore
	if source.MinScore != nil {
		minScore = *source.MinScore
	}

	return scoredStory{
		value:  score.Value,
		reason: score.Reason,
		passes: score.Value >= minScore,
	}
}

// storySummary returns the feed entry's own description when present, otherwise
// (if article summarization is enabled and an embedding scorer is available) it
// fetches the article and extracts its title-most-relevant sentence. Best-effort:
// any failure falls back to an empty summary.
func (n News) storySummary(ctx context.Context, story *parser.Item, log *logger.Log) string {
	summary := cleanSummary(story.Description)
	if summary != "" {
		return summary
	}

	if n.cfg.Scoring == nil || !n.cfg.Scoring.SummarizeArticles {
		return ""
	}

	summarizer, ok := n.scorer.(scorer.Summarizer)
	if !ok {
		return "" // summarization needs the embedding provider
	}

	text, err := article.FetchText(ctx, story.Link)
	if err != nil {
		log.WarnErr("fetching article for summary", err)

		return ""
	}

	sentence, err := summarizer.Summarize(ctx, story.Title, text)
	if err != nil {
		log.WarnErr("summarizing article", err)

		return ""
	}

	return cleanSummary(sentence)
}

const maxSummaryLen = 280

var (
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)

	// feedBoilerplateRe matches aggregator metadata (notably hnrss) so we don't
	// leak the originating site, its comment links, points, or comment counts.
	feedBoilerplateRe = regexp.MustCompile(
		`(?i)\s*(?:article url|comments url)\s*:\s*\S+|\s*points\s*:\s*\d+|\s*#\s*comments\s*:\s*\d+`)
)

// cleanSummary strips HTML, aggregator boilerplate, and whitespace from a feed
// description, then truncates to a short single-line snippet for broadcasting.
func cleanSummary(raw string) string {
	if raw == "" {
		return ""
	}

	text := htmlTagRe.ReplaceAllString(raw, " ")
	text = html.UnescapeString(text)
	text = feedBoilerplateRe.ReplaceAllString(text, " ")
	text = strings.Join(strings.Fields(text), " ")

	runes := []rune(text)
	if len(runes) > maxSummaryLen {
		text = strings.TrimSpace(string(runes[:maxSummaryLen])) + "…"
	}

	return text
}

func storyMatchesConfig(story *parser.Item, source *config.Source) bool {
	if !storyPassesRequiredFilters(story, source) {
		return false
	}

	if len(source.MustIncludeKeywords) != 0 {
		if !sourceIncludesKeywords(story.Title, source.MustIncludeKeywords, source.MatchKeywordsAsWords) {
			return false
		}
	}

	return true
}

func storyPassesRequiredFilters(story *parser.Item, source *config.Source) bool {
	if story.PublishedAtParsed.IsZero() {
		return false
	}

	if story.PublishedAtParsed.Before(source.IgnoreStoriesBefore) {
		return false
	}

	if len(source.MustExcludeKeywords) != 0 {
		if sourceIncludesKeywords(story.Title, source.MustExcludeKeywords, source.MatchKeywordsAsWords) {
			return false
		}
	}

	return true
}

func sourceIncludesKeywords(target string, keywords []string, matchAsWords bool) bool {
	if matchAsWords {
		return includesWholeWordKeywords(target, keywords)
	}

	return includesKeywords(target, keywords)
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

func includesWholeWordKeywords(target string, keywords []string) bool {
	target = strings.ToLower(target)

	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && containsWholeKeyword(target, keyword) {
			return true
		}
	}

	return false
}

func containsWholeKeyword(target, keyword string) bool {
	for searchFrom := 0; searchFrom <= len(target); {
		relativeStart := strings.Index(target[searchFrom:], keyword)
		if relativeStart < 0 {
			return false
		}

		start := searchFrom + relativeStart
		end := start + len(keyword)
		beforeWord := false

		if start > 0 {
			before, _ := utf8.DecodeLastRuneInString(target[:start])
			beforeWord = isKeywordWordRune(before)
		}

		afterWord := false

		if end < len(target) {
			after, _ := utf8.DecodeRuneInString(target[end:])
			afterWord = isKeywordWordRune(after)
		}

		if !beforeWord && !afterWord {
			return true
		}

		searchFrom = end
	}

	return false
}

func isKeywordWordRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsNumber(value) || value == '_'
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
