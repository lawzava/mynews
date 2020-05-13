package news

import (
	"fmt"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/parser"
	"time"
)

func (n News) Run(log *logger.Log) error {
	for {
		parsingStarted := time.Now()

		for _, source := range n.config.Sources {
			items, err := parser.ParseURL(source.URL)
			if err != nil {
				log.WarnErr(fmt.Sprintf("parsing feed of source '%s'", source), err)
				continue
			}

			if err = n.broadcastFeed(items, source); err != nil {
				log.WarnErr(fmt.Sprintf("broadcasting items for source '%s'", source), err)
			}
		}

		n.config.Store.CleanupBefore(parsingStarted)
		time.Sleep(n.config.SleepDurationBetweenFeedParsing)
	}
}
