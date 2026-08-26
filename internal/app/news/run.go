package news

import (
	"context"
	"fmt"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/parser"
	"sync"
	"time"
)

const (
	maxConcurrentFetches = 8

	// storageRetention bounds how long a dedup key is kept once it stops being
	// seen, so storage cannot grow without bound if a source stays unreachable.
	storageRetention = 30 * 24 * time.Hour
)

// Run polls every configured feed in a loop until ctx is canceled, returning
// nil on a clean shutdown.
func (n *News) Run(ctx context.Context, log *logger.Log) error {
	for {
		for idx := range n.cfg.Apps {
			n.parseApp(ctx, n.cfg.Apps[idx], n.digests[idx], log)
		}

		n.flushDigests(ctx, log)

		n.metrics.CycleCompleted()

		n.cfg.Store.CleanupAllBefore(time.Now().Add(-storageRetention))

		// Flush after each cycle so an unexpected exit loses at most one cycle of
		// dedup state rather than everything since startup.
		err := n.cfg.Store.DumpToFile(n.cfg.StorageFilePath)
		if err != nil {
			log.WarnErr("flushing storage to disk", err)
		}

		if !sleep(ctx, n.cfg.SleepDurationBetweenFeedParsing) {
			return nil
		}
	}
}

// flushDigests emits any due digests, recording each story as sent only after a
// successful broadcast so a failed or interrupted flush is retried next cycle.
func (n *News) flushDigests(ctx context.Context, log *logger.Log) {
	now := time.Now()

	for idx := range n.digests {
		dig := n.digests[idx]
		if dig == nil {
			continue
		}

		entries := dig.drainIfDue(now)
		for idx := range entries {
			err := dig.target.Send(entries[idx].story)
			if err != nil {
				log.WarnErr("broadcasting digest story", err)
				dig.add(&entries[idx]) // requeue for the next flush instead of dropping

				continue
			}

			err = n.markSent(dig.target, entries[idx].storyID)
			if err != nil {
				log.WarnErr("registering digest story as sent", err)
			}

			if !sleep(ctx, n.cfg.SleepDurationBetweenBroadcasts) {
				return
			}
		}
	}
}

// parseApp fetches every source concurrently, then broadcasts results in source
// order (preserving the global broadcast throttle). Dedup keys are retained for
// storageRetention after they were last seen (see Run), never pruned per cycle:
// a story that drops out of one fetch and reappears must not be re-broadcast.
func (n *News) parseApp(ctx context.Context, app config.App, dig *digest, log *logger.Log) {
	for _, result := range fetchSources(ctx, app.Sources) {
		if result.source == nil {
			continue // fetch skipped because shutdown began
		}

		if result.err != nil {
			n.metrics.ParseError()

			log.WarnErr(fmt.Sprintf("parsing feed of source '%s'", result.source.URL), result.err)

			continue
		}

		n.metrics.FeedParsed()

		err := n.broadcastFeed(ctx, app.Broadcast, dig, result.items, result.source, log)
		if err != nil {
			log.WarnErr(fmt.Sprintf("broadcasting items for source '%s'", result.source.URL), err)
		}
	}
}

type fetchResult struct {
	source *config.Source
	items  []parser.Item
	err    error
}

// fetchSources fetches every source concurrently with bounded parallelism so a
// single slow feed cannot stall the others, returning results in source order.
func fetchSources(ctx context.Context, sources []*config.Source) []fetchResult {
	results := make([]fetchResult, len(sources))

	var waitGroup sync.WaitGroup

	semaphore := make(chan struct{}, maxConcurrentFetches)

	for idx, source := range sources {
		if ctx.Err() != nil {
			break
		}

		waitGroup.Add(1)

		semaphore <- struct{}{}

		go func() {
			defer waitGroup.Done()
			defer func() { <-semaphore }()

			items, err := parser.ParseURL(ctx, source.URL)
			results[idx] = fetchResult{source: source, items: items, err: err}
		}()
	}

	waitGroup.Wait()

	return results
}

// sleep waits for d or until ctx is canceled, reporting whether d fully elapsed.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
