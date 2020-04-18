package broadcast

import (
	"fmt"
)

type Message struct {
	Title string
	Link  string
}

type Broadcast interface {
	Send(message Message) error
}

type Config struct {
	Type

	Telegram telegram
}

func (c Config) Validate() error {
	if c.Type == TypeTelegram {
		if err := c.Telegram.validate(); err != nil {
			return fmt.Errorf("validating telegram config: %w", err)
		}
	}

	return nil
}

func New(cfg Config) (Broadcast, error) {
	if cfg.Type == TypeTelegram {
		return &cfg.Telegram, nil
	}

	return nil, nil
}
