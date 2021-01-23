package news

import (
	"fmt"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/parser"
	"time"
)

func (n News) Run(log *logger.Log) error {
	var sourceHadIssues bool

	for {
		parsingStartedAt := time.Now()

		for _, source := range n.config.Sources {
			items, err := parser.ParseURL(source.URL)
			if err != nil {
				log.WarnErr(fmt.Sprintf("parsing feed of source '%s'", source.URL), err)

				sourceHadIssues = true

				continue
			}

			if err = n.broadcastFeed(items, source); err != nil {
				log.WarnErr(fmt.Sprintf("broadcasting items for source '%s'", source.URL), err)

				sourceHadIssues = true
			}
		}

		if !sourceHadIssues {
			n.config.Store.CleanupBefore(parsingStartedAt)
		}

		sourceHadIssues = false

		time.Sleep(n.config.SleepDurationBetweenFeedParsing)
	}
}
