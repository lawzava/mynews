package news

import (
	"fmt"
	"time"

	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/parser"
)

func (n News) Run(log *logger.Log) error {
	var sourceHadIssues bool

	for {
		for _, app := range n.cfg.Apps {
			parsingStartedAt := time.Now()

			for _, source := range app.Sources {
				items, err := parser.ParseURL(source.URL)
				if err != nil {
					log.WarnErr(fmt.Sprintf("parsing feed of source '%s'", source.URL), err)

					sourceHadIssues = true

					continue
				}

				if err = n.broadcastFeed(app.Broadcast, items, source); err != nil {
					log.WarnErr(fmt.Sprintf("broadcasting items for source '%s'", source.URL), err)

					sourceHadIssues = true
				}
			}

			if !sourceHadIssues {
				n.cfg.Store.CleanupBefore(app.Broadcast.Name(), parsingStartedAt)
			}

			sourceHadIssues = false
		}

		time.Sleep(n.cfg.SleepDurationBetweenFeedParsing)
	}
}
