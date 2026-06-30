//nolint:testpackage // exercises unexported broadcastFeed/handleStory and News internals
package news

import (
	"context"
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

// fakeScorer scores a title by the interest token it contains: a title holding an
// interest string scores 0.9, otherwise 0.1.
type fakeScorer struct {
	interests []string
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
