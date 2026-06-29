package news

import (
	"context"
	"fmt"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/parser"
	"time"
)

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

// parseApp fetches and broadcasts every source of a single app. Stale dedup keys
// are pruned only when the whole app parsed cleanly and we are not shutting down.
func (n News) parseApp(ctx context.Context, app config.App, log *logger.Log) {
	parsingStartedAt := time.Now()
	sourceHadIssues := false

	for _, source := range app.Sources {
		if ctx.Err() != nil {
			return
		}

		items, err := parser.ParseURL(ctx, source.URL)
		if err != nil {
			log.WarnErr(fmt.Sprintf("parsing feed of source '%s'", source.URL), err)

			sourceHadIssues = true

			continue
		}

		err = n.broadcastFeed(ctx, app.Broadcast, items, source, log)
		if err != nil {
			log.WarnErr(fmt.Sprintf("broadcasting items for source '%s'", source.URL), err)

			sourceHadIssues = true
		}
	}

	if !sourceHadIssues && ctx.Err() == nil {
		n.cfg.Store.CleanupBefore(app.Broadcast.Name(), parsingStartedAt)
	}
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
