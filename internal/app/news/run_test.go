//nolint:testpackage // exercises unexported parseApp and News internals
package news

import (
	"context"
	"fmt"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/metrics"
	"mynews/internal/pkg/storage"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// feedServer serves an RSS feed whose item set can be swapped between requests,
// simulating a real feed where stories drop out of a fetch and reappear later.
type feedServer struct {
	mutex sync.Mutex
	body  string
}

func (f *feedServer) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	_, _ = w.Write([]byte(f.body))
}

func (f *feedServer) setBody(body string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.body = body
}

func rssFeedWith(items string) string {
	return `<?xml version="1.0"?><rss version="2.0"><channel><title>t</title>` + items + `</channel></rss>`
}

func rssItemXML(title, link string) string {
	return fmt.Sprintf(`<item><title>%s</title><link>%s</link><pubDate>%s</pubDate></item>`,
		title, link, time.Now().UTC().Format(time.RFC1123Z))
}

// TestStoryAbsentForOneCycleIsNotRebroadcast reproduces the "same stories over
// and over" bug: a story that is missing from a single fetch (feed jitter, a
// rotating front-page window, load-balanced caches) and then reappears must not
// be broadcast a second time.
func TestStoryAbsentForOneCycleIsNotRebroadcast(t *testing.T) {
	t.Parallel()

	storyItem := rssItemXML("Hello story", "https://ex.com/hello")

	server := &feedServer{mutex: sync.Mutex{}, body: rssFeedWith(storyItem)}
	testServer := httptest.NewServer(server)

	defer testServer.Close()

	news := News{ //nolint:exhaustruct // scoring and digests are not exercised here
		cfg:     &config.Config{Store: storage.New()}, //nolint:exhaustruct // only the store is needed
		metrics: metrics.New(time.Hour),
	}

	capture := &captureBroadcast{sent: nil}
	app := config.App{
		Sources: []*config.Source{
			{
				URL:                 testServer.URL,
				IgnoreStoriesBefore: time.Now().Add(-time.Hour),
			},
		},
		Broadcast: capture,
		Digest:    nil,
	}

	ctx := context.Background()
	log := logger.New(logger.Error)

	// Cycle 1: the story is in the feed and is broadcast.
	news.parseApp(ctx, app, nil, log)

	// Cycle 2: the story momentarily drops out of the feed.
	server.setBody(rssFeedWith(""))
	time.Sleep(2 * time.Millisecond) // ensure the cycle boundary has a later timestamp

	news.parseApp(ctx, app, nil, log)

	// Cycle 3: the story reappears.
	server.setBody(rssFeedWith(storyItem))
	time.Sleep(2 * time.Millisecond)

	news.parseApp(ctx, app, nil, log)

	if len(capture.sent) != 1 {
		t.Fatalf("story was broadcast %d times, want exactly 1", len(capture.sent))
	}
}

// TestDigestStoryAbsentForOneCycleIsNotRebroadcast pins the same guarantee for
// digest mode, where a story is marked sent only when the digest flushes rather
// than when it is first seen.
func TestDigestStoryAbsentForOneCycleIsNotRebroadcast(t *testing.T) {
	t.Parallel()

	storyItem := rssItemXML("Hello story", "https://ex.com/hello")

	server := &feedServer{mutex: sync.Mutex{}, body: rssFeedWith(storyItem)}
	testServer := httptest.NewServer(server)

	defer testServer.Close()

	capture := &captureBroadcast{sent: nil}

	// A nanosecond interval makes the digest due on every flushDigests call.
	dig := newDigest(&config.DigestConfig{Every: time.Nanosecond, TopN: 10}, capture, time.Now())

	news := News{ //nolint:exhaustruct // scoring is not exercised here
		cfg:     &config.Config{Store: storage.New()}, //nolint:exhaustruct // only the store is needed
		metrics: metrics.New(time.Hour),
		digests: []*digest{dig},
	}

	app := config.App{
		Sources: []*config.Source{
			{
				URL:                 testServer.URL,
				IgnoreStoriesBefore: time.Now().Add(-time.Hour),
			},
		},
		Broadcast: capture,
		Digest:    &config.DigestConfig{Every: time.Nanosecond, TopN: 10},
	}

	ctx := context.Background()
	log := logger.New(logger.Error)

	// Cycle 1: the story is buffered, then sent when the digest flushes.
	news.parseApp(ctx, app, dig, log)
	news.flushDigests(ctx, log)

	// Cycle 2: the story momentarily drops out of the feed.
	server.setBody(rssFeedWith(""))
	time.Sleep(2 * time.Millisecond) // ensure the cycle boundary has a later timestamp

	news.parseApp(ctx, app, dig, log)
	news.flushDigests(ctx, log)

	// Cycle 3: the story reappears.
	server.setBody(rssFeedWith(storyItem))
	time.Sleep(2 * time.Millisecond)

	news.parseApp(ctx, app, dig, log)
	news.flushDigests(ctx, log)

	if len(capture.sent) != 1 {
		t.Fatalf("story was broadcast %d times across digests, want exactly 1", len(capture.sent))
	}
}
