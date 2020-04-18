package parser

import (
	"fmt"
	"news/internal/pkg/identity"
	"news/internal/pkg/storage"
	"time"

	"github.com/mmcdole/gofeed"
)

type Parser struct {
	fp *gofeed.Parser

	sources []string
	storage storage.Storage

	Config
}

func New(cfg Config) (*Parser, error) {
	const (
		defaultSleepDuration = 10 * time.Second
	)

	if cfg.SleepDurationBetweenRuns == 0 {
		cfg.SleepDurationBetweenRuns = defaultSleepDuration
	}

	stg, err := storage.New(cfg.StorageType, cfg.StorageAccessDetails)
	if err != nil {
		return nil, fmt.Errorf("creating storage: %w", err)
	}

	return &Parser{
		fp:      gofeed.NewParser(),
		sources: cfg.Sources,
		Config:  cfg,
		storage: stg,
	}, nil
}

func (p Parser) Run() error {
	for {
		for _, source := range p.sources {
			feed, err := p.fp.ParseURL(source)
			if err != nil {
				return fmt.Errorf("parsing feed of source '%s': %w", source, err)
			}

			for _, story := range feed.Items {
				storyID := identity.New(source, story.Title, story.Link, story.Published)

				storyWasSent, err := p.storage.KeyExists(storyID)
				if err != nil {
					return fmt.Errorf("checking if story was already sent: %w", err)
				}

				if storyWasSent {
					continue
				}

				if err := p.storage.PutKey(storyID); err != nil {
					return fmt.Errorf("registering story as sent: %w", err)
				}

				fmt.Printf("%s | %s \n",
					story.Title, story.Link)
			}
		}

		time.Sleep(p.Config.SleepDurationBetweenRuns)
	}
}
