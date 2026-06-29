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

const maxConcurrentFetches = 8

// Run polls every configured feed in a loop until ctx is canceled, returning
// nil on a clean shutdown.
func (n News) Run(ctx context.Context, log *logger.Log) error {
	for {
		for _, app := range n.cfg.Apps {
			n.parseApp(ctx, app, log)
		}

		if !sleep(ctx, n.cfg.SleepDurationBetweenFeedParsing) {
			return nil
		}
	}
}

// parseApp fetches every source concurrently, then broadcasts results in source
// order (preserving the global broadcast throttle). Stale dedup keys are pruned
// only when the whole app parsed cleanly and we are not shutting down.
func (n News) parseApp(ctx context.Context, app config.App, log *logger.Log) {
	parsingStartedAt := time.Now()
	sourceHadIssues := false

	for _, result := range fetchSources(ctx, app.Sources) {
		if result.source == nil {
			continue // fetch skipped because shutdown began
		}

		if result.err != nil {
			log.WarnErr(fmt.Sprintf("parsing feed of source '%s'", result.source.URL), result.err)

			sourceHadIssues = true

			continue
		}

		err := n.broadcastFeed(ctx, app.Broadcast, result.items, result.source, log)
		if err != nil {
			log.WarnErr(fmt.Sprintf("broadcasting items for source '%s'", result.source.URL), err)

			sourceHadIssues = true
		}
	}

	if !sourceHadIssues && ctx.Err() == nil {
		n.cfg.Store.CleanupBefore(app.Broadcast.Name(), parsingStartedAt)
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
