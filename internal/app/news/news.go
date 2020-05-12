package news

import (
	"mynews/internal/pkg/config"
	"time"
)

type News struct {
	config *config.Config
}

func New(cfg *config.Config) News {
	const defaultSleepDuration = 10 * time.Second

	if cfg.SleepDurationBetweenBroadcasts == 0 {
		cfg.SleepDurationBetweenBroadcasts = defaultSleepDuration
	}

	return News{cfg}
}
