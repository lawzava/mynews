package broadcast

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"mynews/internal/pkg/validate"
)

type Telegram struct {
	BotAPIToken string
	ChatID      string
}

func (t Telegram) New() (Broadcast, error) {
	if err := validate.RequiredString(t.BotAPIToken, "Telegram API Token"); err != nil {
		return nil, err
	}

	if err := validate.RequiredString(t.ChatID, "Telegram Chat ID"); err != nil {
		return nil, err
	}

	return t, nil
}

var errUnacceptableResponseFromTelegram = errors.New("unacceptable response from Telegram bot API")

func (t Telegram) Send(message Story) error {
	type inlineKeyboard struct {
		Text              string `json:"text"`
		URL               string `json:"url"`
		SwitchInlineQuery string `json:"switch_inline_query"`
	}

	type replyMarkup struct {
		InlineKeyboard [][]inlineKeyboard `json:"inline_keyboard"`
	}

	telegramMessage := struct {
		ChatID      string      `json:"chat_id"`
		ParseMode   string      `json:"parse_mode"`
		Text        string      `json:"text"`
		ReplyMarkup replyMarkup `json:"reply_markup"`
	}{
		ChatID:    t.ChatID,
		ParseMode: "MarkdownV2",
		Text: fmt.Sprintf(`*%s* 

%s`, // empty line is intended
			escapeTelegramText(message.Title),
			escapeTelegramText(message.URL),
		),
		ReplyMarkup: replyMarkup{
			InlineKeyboard: [][]inlineKeyboard{
				{{Text: "Read", URL: message.URL}},
			},
		},
	}

	requestBody, err := json.Marshal(telegramMessage)
	if err != nil {
		return fmt.Errorf("preparing request body: %w", err)
	}

	requestURL := fmt.Sprintf("https://api.Telegram.org/bot%s/sendMessage", t.BotAPIToken)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, requestURL, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request to Telegram API: %w", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	var telegramResponse struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}

	if err = json.Unmarshal(body, &telegramResponse); err != nil {
		return fmt.Errorf("unmarshaling response body: %w", err)
	}

	if !telegramResponse.OK {
		return fmt.Errorf("%w: %s", errUnacceptableResponseFromTelegram, telegramResponse.Description)
	}

	return nil
}

func escapeTelegramText(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		",", "\\,",
		"!", "\\!",
	)

	return replacer.Replace(text)
}
