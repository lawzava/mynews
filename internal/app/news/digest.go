package news

import (
	"context"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"sort"
	"sync"
	"time"
)

// digest batches the highest-scoring stories for an app and emits them on a fixed
// interval instead of broadcasting each story as it arrives.
type digest struct {
	every  time.Duration
	topN   int
	target broadcast.Broadcast

	mutex     sync.Mutex
	buffer    []broadcast.Story
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

// add buffers a story, keeping only the top-N highest-scoring entries so memory
// stays bounded regardless of feed volume or interval length.
func (d *digest) add(story broadcast.Story) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.buffer = append(d.buffer, story)
	if len(d.buffer) <= d.topN {
		return
	}

	lowestIdx := 0
	for idx := range d.buffer {
		if d.buffer[idx].Score < d.buffer[lowestIdx].Score {
			lowestIdx = idx
		}
	}

	d.buffer = append(d.buffer[:lowestIdx], d.buffer[lowestIdx+1:]...)
}

// flushDue emits the buffered stories (highest score first) and resets the timer
// once the interval has elapsed; otherwise it does nothing.
func (d *digest) flushDue(ctx context.Context, now time.Time, broadcastSleep time.Duration, log *logger.Log) {
	stories := d.drainIfDue(now)

	for idx := range stories {
		err := d.target.Send(stories[idx])
		if err != nil {
			log.WarnErr("broadcasting digest story", err)
		}

		if !sleep(ctx, broadcastSleep) {
			return
		}
	}
}

func (d *digest) drainIfDue(now time.Time) []broadcast.Story {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if now.Before(d.nextFlush) {
		return nil
	}

	stories := d.buffer
	d.buffer = nil
	d.nextFlush = now.Add(d.every)

	sort.Slice(stories, func(left, right int) bool {
		return stories[left].Score > stories[right].Score
	})

	return stories
}
