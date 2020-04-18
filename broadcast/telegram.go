package broadcast

import (
	"news/validate"
)

type telegramConfig struct {
	BotAPIToken string
	ChatID      int64
}

func (c telegramConfig) validate() error {
	if err := validate.RequiredString(c.BotAPIToken, "BotAPIToken"); err != nil {
		return err
	}

	if err := validate.RequiredInt64(c.ChatID, "ChatID"); err != nil {
		return err
	}

	return nil
}

type telegram struct {
}

func newTelegram(cfg telegramConfig) (*telegram, error) {
	return nil, nil
}

func (t *telegram) Send(message Message) error {
	return nil
}
