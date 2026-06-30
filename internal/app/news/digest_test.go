//nolint:testpackage // exercises the unexported digest internals
package news

import (
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/config"
	"testing"
	"time"
)

func scoredEntry(score float64, id string) *digestEntry {
	return &digestEntry{
		story:   broadcast.Story{Title: id, URL: "", Summary: "", Score: score, Reason: ""},
		storyID: id,
	}
}

func TestDigestKeepsTopNDedupedAndOrdered(t *testing.T) {
	t.Parallel()

	start := time.Now()
	dig := newDigest(&config.DigestConfig{Every: time.Hour, TopN: 2}, &captureBroadcast{sent: nil}, start)

	dig.add(scoredEntry(0.1, "a"))
	dig.add(scoredEntry(0.5, "b"))
	dig.add(scoredEntry(0.9, "c"))
	dig.add(scoredEntry(0.3, "d"))
	dig.add(scoredEntry(0.9, "c")) // duplicate id is ignored

	// Not due yet.
	if got := dig.drainIfDue(start); got != nil {
		t.Fatalf("drainIfDue before interval returned %d entries, want nil", len(got))
	}

	entries := dig.drainIfDue(start.Add(2 * time.Hour))
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want top-2", len(entries))
	}

	if entries[0].storyID != "c" || entries[1].storyID != "b" {
		t.Errorf("expected top-2 [c, b] by score, got [%s, %s]", entries[0].storyID, entries[1].storyID)
	}

	// Draining clears the buffer; a second due drain yields nothing.
	if got := dig.drainIfDue(start.Add(4 * time.Hour)); got != nil {
		t.Errorf("second drain returned %d entries, want nil", len(got))
	}
}
