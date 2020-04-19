package broadcast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mynews/validate"
	"net/http"
	"strings"
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

func (t Telegram) Send(message Message) error {
	telegramMessage := struct {
		ChatID    string `json:"chat_id"`
		ParseMode string `json:"parse_mode"`
		Text      string `json:"text"`
	}{
		ChatID:    t.ChatID,
		ParseMode: "MarkdownV2",
		Text: fmt.Sprintf(`*%s* 

			*Link:* [%s](%s)`,
			escapeTelegramText(message.Title),
			escapeTelegramText(message.Link),
			escapeTelegramLink(message.Link),
		),
	}

	requestBody, err := json.Marshal(telegramMessage)
	if err != nil {
		return fmt.Errorf("preparing request body: %w", err)
	}

	requestURL := fmt.Sprintf("https://api.Telegram.org/bot%s/sendMessage", t.BotAPIToken)

	req, _ := http.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(requestBody))
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
		return fmt.Errorf("unacceptable response from Telegram bot API: %s", telegramResponse.Description)
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
	)

	return replacer.Replace(text)
}

func escapeTelegramLink(link string) string {
	replacer := strings.NewReplacer(
		")", "\\)",
		"\\", "\\\\",
	)

	return replacer.Replace(link)
}
