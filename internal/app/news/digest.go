package news

import (
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/config"
	"sort"
	"sync"
	"time"
)

// digestEntry pairs a buffered story with its dedup key so the digest can mark
// it sent only after a successful flush.
type digestEntry struct {
	story   broadcast.Story
	storyID string
}

// digest batches the highest-scoring stories for an app and emits them on a fixed
// interval instead of broadcasting each story as it arrives.
type digest struct {
	every  time.Duration
	topN   int
	target broadcast.Broadcast

	mutex     sync.Mutex
	buffer    []digestEntry
	nextFlush time.Time
}

// buildDigests returns a digest per app (nil for apps without a digest config),
// aligned by index with cfg.Apps.
func buildDigests(cfg *config.Config) []*digest {
	digests := make([]*digest, len(cfg.Apps))
	now := time.Now()

	for idx := range cfg.Apps {
		if cfg.Apps[idx].Digest != nil {
			digests[idx] = newDigest(cfg.Apps[idx].Digest, cfg.Apps[idx].Broadcast, now)
		}
	}

	return digests
}

func newDigest(cfg *config.DigestConfig, target broadcast.Broadcast, now time.Time) *digest {
	return &digest{
		every:     cfg.Every,
		topN:      cfg.TopN,
		target:    target,
		mutex:     sync.Mutex{},
		buffer:    nil,
		nextFlush: now.Add(cfg.Every),
	}
}

// add buffers a story, deduping by storyID and keeping only the top-N highest
// scoring entries so memory stays bounded regardless of feed volume or interval.
func (d *digest) add(entry *digestEntry) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for idx := range d.buffer {
		if d.buffer[idx].storyID == entry.storyID {
			if entry.story.Score > d.buffer[idx].story.Score {
				d.buffer[idx] = *entry // keep the higher-scoring view of the story
			}

			return
		}
	}

	d.buffer = append(d.buffer, *entry)
	if len(d.buffer) <= d.topN {
		return
	}

	lowestIdx := 0
	for idx := range d.buffer {
		if d.buffer[idx].story.Score < d.buffer[lowestIdx].story.Score {
			lowestIdx = idx
		}
	}

	d.buffer = append(d.buffer[:lowestIdx], d.buffer[lowestIdx+1:]...)
}

// drainIfDue returns the buffered entries (highest score first) and resets the
// timer once the interval has elapsed; otherwise it returns nil. Entries not
// successfully sent are left unmarked and re-collected on the next cycle.
func (d *digest) drainIfDue(now time.Time) []digestEntry {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if now.Before(d.nextFlush) {
		return nil
	}

	entries := d.buffer
	d.buffer = nil
	d.nextFlush = now.Add(d.every)

	sort.Slice(entries, func(left, right int) bool {
		return entries[left].story.Score > entries[right].story.Score
	})

	return entries
}
