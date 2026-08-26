//nolint:testpackage // exercises unexported broadcastFeed/handleStory and News internals
package news

import (
	"context"
	"errors"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/metrics"
	"mynews/internal/pkg/parser"
	"mynews/internal/pkg/scorer"
	"mynews/internal/pkg/storage"
	"strings"
	"testing"
	"time"
)

const (
	fakeProvider = "fake"
	linuxKeyword = "linux"
)

var errHackerNewsUnavailable = errors.New("hacker news unavailable")

// fakeScorer scores a title by the interest token it contains: a title holding an
// interest string scores 0.9, otherwise 0.1.
type fakeScorer struct {
	interests []string
}

type fakeHackerNewsScoreClient struct {
	scores []int
	err    error
	calls  int
}

func (f *fakeHackerNewsScoreClient) StoryScore(_ context.Context, _ string) (int, error) {
	if f.err != nil {
		return 0, f.err
	}

	score := f.scores[min(f.calls, len(f.scores)-1)]
	f.calls++

	return score, nil
}

func (f fakeScorer) Score(_ context.Context, title string) (scorer.Score, error) {
	for _, interest := range f.interests {
		if strings.Contains(strings.ToLower(title), interest) {
			return scorer.Score{Value: 0.9, Reason: interest}, nil
		}
	}

	return scorer.Score{Value: 0.1, Reason: ""}, nil
}

func (f fakeScorer) Name() string { return fakeProvider }

//nolint:ireturn // implements the Scorer interface method
func (f fakeScorer) Derive(interests []string) (scorer.Scorer, error) {
	return fakeScorer{interests: interests}, nil
}

func (f fakeScorer) Close() error { return nil }

// captureBroadcast records every story it is asked to send.
type captureBroadcast struct {
	sent []broadcast.Story
}

func (c *captureBroadcast) Send(message broadcast.Story) error {
	c.sent = append(c.sent, message)

	return nil
}

func (c *captureBroadcast) Name() string { return "capture" }

func recentItem(title, link string) parser.Item {
	return parser.Item{
		Title:             title,
		Link:              link,
		CommentsURL:       "https://news.ycombinator.com/item?id=123",
		Description:       "",
		PublishedAt:       "",
		PublishedAtParsed: time.Now(),
	}
}

func newTestNews(storyScorer scorer.Scorer, minScore float64) News {
	cfg := &config.Config{ //nolint:exhaustruct // only the fields used by broadcastFeed are set
		Store: storage.New(),
		//nolint:exhaustruct // only the threshold matters for these tests
		Scoring: &config.ScoringConfig{Enabled: true, Provider: fakeProvider, MinScore: minScore},
	}

	return News{ //nolint:exhaustruct // metrics/digests not exercised here
		cfg:     cfg,
		scorer:  storyScorer,
		metrics: metrics.New(time.Hour),
	}
}

func openSource() *config.Source {
	return &config.Source{ //nolint:exhaustruct // only the matching-relevant field is needed
		IgnoreStoriesBefore: time.Now().Add(-time.Hour),
	}
}

func TestMinScoreFiltersBelowThreshold(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	capture := &captureBroadcast{sent: nil}

	items := []parser.Item{
		recentItem("Big Linux kernel release", "https://ex.com/1"),
		recentItem("Celebrity gossip roundup", "https://ex.com/2"),
		recentItem("Another Linux distro launches", "https://ex.com/3"),
	}

	err := news.broadcastFeed(context.Background(), capture, nil, items, openSource(), logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 2 {
		t.Fatalf("sent %d stories, want 2 (the two Linux ones)", len(capture.sent))
	}

	for _, story := range capture.sent {
		if !strings.Contains(strings.ToLower(story.Title), linuxKeyword) {
			t.Errorf("a below-threshold story was sent: %q", story.Title)
		}
	}
}

func TestPerSourceMinScoreOverridesGlobalThreshold(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.95)
	capture := &captureBroadcast{sent: nil}
	source := openSource()
	minScore := 0.5
	source.MinScore = &minScore

	items := []parser.Item{
		recentItem("Big Linux kernel release", "https://ex.com/1"),
	}

	err := news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 1 {
		t.Fatalf("sent %d stories, want 1 using the per-source threshold", len(capture.sent))
	}
}

func TestKeywordOrScoreAllowsEitherSignal(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	capture := &captureBroadcast{sent: nil}
	source := openSource()
	source.MustIncludeKeywords = []string{aiKeyword}
	source.MustExcludeKeywords = []string{"spam"}
	source.MatchKeywordsAsWords = true
	source.MatchKeywordsOrScore = true

	items := []parser.Item{
		recentItem("AI product announcement", "https://ex.com/keyword"),
		recentItem("Linux kernel release", "https://ex.com/score"),
		recentItem("Celebrity roundup", "https://ex.com/neither"),
		recentItem("Linux spam bulletin", "https://ex.com/excluded"),
	}

	err := news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 2 {
		t.Fatalf("sent %d stories, want keyword match and score match", len(capture.sent))
	}

	if capture.sent[0].Title != items[0].Title || capture.sent[1].Title != items[1].Title {
		t.Fatalf("sent titles = %q, %q; want %q, %q", capture.sent[0].Title, capture.sent[1].Title,
			items[0].Title, items[1].Title)
	}
}

func TestKeywordOrScoreRequiresScoring(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	news.scorer = nil
	capture := &captureBroadcast{sent: nil}
	source := openSource()
	source.MustIncludeKeywords = []string{aiKeyword}
	source.MatchKeywordsOrScore = true

	items := []parser.Item{
		recentItem("Celebrity roundup", "https://ex.com/no-score"),
	}

	err := news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 0 {
		t.Fatalf("sent %d stories without scoring, want 0", len(capture.sent))
	}
}

func TestHackerNewsScoreAllowsRelevantStory(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	hackerNewsClient := &fakeHackerNewsScoreClient{scores: []int{56}, err: nil, calls: 0}
	news.hackerNewsClient = hackerNewsClient
	capture := &captureBroadcast{sent: nil}
	source := openSource()
	minHackerNewsScore := 10
	source.MinHackerNewsScore = &minHackerNewsScore

	items := []parser.Item{recentItem("RAG Is Simpler Than You Think", "https://ex.com/rag")}

	err := news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 1 {
		t.Fatalf("sent %d stories, want 1 passing HN score", len(capture.sent))
	}
}

func TestKeywordOrHackerNewsScoreWorksWithoutRelevanceScorer(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	news.scorer = nil
	hackerNewsClient := &fakeHackerNewsScoreClient{scores: []int{56}, err: nil, calls: 0}
	news.hackerNewsClient = hackerNewsClient
	capture := &captureBroadcast{sent: nil}
	source := openSource()
	source.MustIncludeKeywords = []string{aiKeyword}
	source.MatchKeywordsOrScore = true
	minHackerNewsScore := 10
	source.MinHackerNewsScore = &minHackerNewsScore

	items := []parser.Item{recentItem("Compiler implementation notes", "https://ex.com/hn-only")}

	err := news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 1 {
		t.Fatalf("sent %d stories, want HN score match without relevance scorer", len(capture.sent))
	}

	if hackerNewsClient.calls != 1 {
		t.Fatalf("HN score calls = %d, want 1", hackerNewsClient.calls)
	}
}

func TestHackerNewsScoreRejectsBelowThresholdWithoutRelevanceScorer(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	news.scorer = nil
	hackerNewsClient := &fakeHackerNewsScoreClient{scores: []int{9}, err: nil, calls: 0}
	news.hackerNewsClient = hackerNewsClient
	capture := &captureBroadcast{sent: nil}
	source := openSource()
	minHackerNewsScore := 10
	source.MinHackerNewsScore = &minHackerNewsScore

	items := []parser.Item{recentItem("Compiler implementation notes", "https://ex.com/hn-below-threshold")}

	err := news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 0 {
		t.Fatalf("sent %d stories, want 0 below HN threshold", len(capture.sent))
	}

	if hackerNewsClient.calls != 1 {
		t.Fatalf("HN score calls = %d, want 1", hackerNewsClient.calls)
	}
}

func TestHackerNewsScoreRechecksRejectedStory(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	news.hackerNewsClient = &fakeHackerNewsScoreClient{scores: []int{9, 10}, err: nil, calls: 0}
	capture := &captureBroadcast{sent: nil}
	source := openSource()
	minHackerNewsScore := 10
	source.MinHackerNewsScore = &minHackerNewsScore
	items := []parser.Item{recentItem("Compiler implementation notes", "https://ex.com/compiler")}

	err := news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("first broadcastFeed: %v", err)
	}

	err = news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("second broadcastFeed: %v", err)
	}

	if len(capture.sent) != 1 {
		t.Fatalf("sent %d stories, want 1 after HN score reached threshold", len(capture.sent))
	}
}

func TestHackerNewsScoreFailureFailsOpen(t *testing.T) {
	t.Parallel()

	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	news.hackerNewsClient = &fakeHackerNewsScoreClient{
		scores: nil,
		err:    errHackerNewsUnavailable,
		calls:  0,
	}
	capture := &captureBroadcast{sent: nil}
	source := openSource()
	minHackerNewsScore := 10
	source.MinHackerNewsScore = &minHackerNewsScore

	items := []parser.Item{recentItem("Unknown but ranked story", "https://ex.com/fail-open")}

	err := news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 1 {
		t.Fatalf("sent %d stories, want fail-open delivery", len(capture.sent))
	}
}

func TestPerSourceInterestsRouteToDerivedScorer(t *testing.T) {
	t.Parallel()

	// Global scorer matches "linux"; the source overrides interests to "gossip".
	news := newTestNews(fakeScorer{interests: []string{linuxKeyword}}, 0.5)
	source := openSource()
	source.Interests = []string{"gossip"}

	derived, err := news.scorer.Derive(source.Interests)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}

	news.sourceScorers = map[*config.Source]scorer.Scorer{source: derived}

	capture := &captureBroadcast{sent: nil}

	items := []parser.Item{
		recentItem("Big Linux kernel release", "https://ex.com/1"),
		recentItem("Celebrity gossip roundup", "https://ex.com/2"),
	}

	err = news.broadcastFeed(context.Background(), capture, nil, items, source, logger.New(logger.Error))
	if err != nil {
		t.Fatalf("broadcastFeed: %v", err)
	}

	if len(capture.sent) != 1 || !strings.Contains(strings.ToLower(capture.sent[0].Title), "gossip") {
		t.Fatalf("per-source interests not applied: sent=%v", capture.sent)
	}
}
