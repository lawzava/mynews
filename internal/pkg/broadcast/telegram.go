package broadcast

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mynews/internal/pkg/validate"
	"net/http"
	"strings"
	"time"
)

const (
	telegramTimeout          = 30 * time.Second
	maxTelegramResponseBytes = 1 << 20 // 1 MiB is ample for the Telegram API JSON response
)

type Telegram struct {
	BotAPIToken string
	ChatID      string
}

func NewTelegramClient(botAPIToken, chatID string) (*Telegram, error) {
	client := Telegram{
		BotAPIToken: botAPIToken,
		ChatID:      chatID,
	}

	err := validate.RequiredString(client.BotAPIToken, "Telegram API Token")
	if err != nil {
		return nil, fmt.Errorf("validating Telegram API Token: %w", err)
	}

	err = validate.RequiredString(client.ChatID, "Telegram Chat ID")
	if err != nil {
		return nil, fmt.Errorf("validating Telegram Chat ID: %w", err)
	}

	return &client, nil
}

func (t Telegram) Name() string {
	return "telegram-" + t.ChatID
}

var (
	errUnacceptableResponseFromTelegram = errors.New("unacceptable response from Telegram bot API")
	errTelegramResponseTooLarge         = errors.New("Telegram API response exceeds size limit")
)

func (t Telegram) Send(message Story) error {
	//nolint:tagliatelle // required structure for telegram requests
	type inlineKeyboard struct {
		Text              string `json:"text"`
		URL               string `json:"url"`
		SwitchInlineQuery string `json:"switch_inline_query"`
	}

	//nolint:tagliatelle // required structure for telegram requests
	type replyMarkup struct {
		InlineKeyboard [][]inlineKeyboard `json:"inline_keyboard"`
	}

	// Build message text with optional score
	text := buildTelegramText(message)

	//nolint:tagliatelle // required structure for telegram requests
	telegramMessage := struct {
		ChatID      string      `json:"chat_id"`
		ParseMode   string      `json:"parse_mode"`
		Text        string      `json:"text"`
		ReplyMarkup replyMarkup `json:"reply_markup"`
	}{
		ChatID:    t.ChatID,
		ParseMode: "MarkdownV2",
		Text:      text,
		ReplyMarkup: replyMarkup{
			InlineKeyboard: [][]inlineKeyboard{
				{{Text: "Read", URL: message.URL, SwitchInlineQuery: ""}},
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

	//nolint:exhaustruct // no need to set any other fields
	client := &http.Client{Timeout: telegramTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request to Telegram API: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes+1))
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if int64(len(body)) > maxTelegramResponseBytes {
		return errTelegramResponseTooLarge
	}

	var telegramResponse struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}

	err = json.Unmarshal(body, &telegramResponse)
	if err != nil {
		return fmt.Errorf("unmarshaling response body: %w", err)
	}

	if !telegramResponse.OK {
		return fmt.Errorf("%w: %s", errUnacceptableResponseFromTelegram, telegramResponse.Description)
	}

	return nil
}

const scoreMultiplier = 100

func buildTelegramText(message Story) string {
	text := "*" + escapeTelegramText(message.Title) + "*\n"

	if message.Score > 0 {
		text += fmt.Sprintf("📊 Score: %.0f%%\n", message.Score*scoreMultiplier)
	}

	if message.Summary != "" {
		text += "\n" + escapeTelegramText(message.Summary) + "\n"
	}

	text += "\n" + escapeTelegramText(message.URL)

	return text
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
