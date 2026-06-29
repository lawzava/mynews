package broadcast

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mynews/internal/pkg/validate"
	"net/http"
	"time"
)

const (
	webhookTimeout          = 30 * time.Second
	maxWebhookResponseBytes = 1 << 20 // 1 MiB is ample for a webhook ack
	nameHashLen             = 12
)

var errWebhookBadStatus = errors.New("webhook returned a non-2xx status")

// webhookSender holds the shared state and transport for webhook-style targets.
type webhookSender struct {
	url  string
	name string
}

func newWebhookSender(kind, rawURL string) (webhookSender, error) {
	err := validate.RequiredString(rawURL, kind+" webhook URL")
	if err != nil {
		return webhookSender{}, fmt.Errorf("validating %s webhook URL: %w", kind, err)
	}

	return webhookSender{url: rawURL, name: targetName(kind, rawURL)}, nil
}

func (w webhookSender) Name() string { return w.name }

// Discord posts stories to a Discord channel via an incoming webhook URL.
type Discord struct{ webhookSender }

// NewDiscordClient validates the webhook URL and returns a Discord broadcaster.
//
//nolint:dupl // thin parallel constructors, one per webhook variant
func NewDiscordClient(webhookURL string) (*Discord, error) {
	sender, err := newWebhookSender("discord", webhookURL)
	if err != nil {
		return nil, err
	}

	return &Discord{sender}, nil
}

func (d *Discord) Send(message Story) error {
	return postJSON(d.url, map[string]string{"content": formatStoryText(message)})
}

// Slack posts stories to a Slack channel via an incoming webhook URL.
type Slack struct{ webhookSender }

// NewSlackClient validates the webhook URL and returns a Slack broadcaster.
//
//nolint:dupl // thin parallel constructors, one per webhook variant
func NewSlackClient(webhookURL string) (*Slack, error) {
	sender, err := newWebhookSender("slack", webhookURL)
	if err != nil {
		return nil, err
	}

	return &Slack{sender}, nil
}

func (s *Slack) Send(message Story) error {
	return postJSON(s.url, map[string]string{"text": formatStoryText(message)})
}

// Webhook posts the raw Story JSON to a user-supplied URL for custom automation.
type Webhook struct{ webhookSender }

// NewWebhookClient validates the URL and returns a generic webhook broadcaster.
//
//nolint:dupl // thin parallel constructors, one per webhook variant
func NewWebhookClient(webhookURL string) (*Webhook, error) {
	sender, err := newWebhookSender("webhook", webhookURL)
	if err != nil {
		return nil, err
	}

	return &Webhook{sender}, nil
}

func (w *Webhook) Send(message Story) error {
	return postJSON(w.url, message)
}

// formatStoryText renders a story as a plain-text message suitable for chat
// targets that auto-link bare URLs (Discord, Slack).
func formatStoryText(message Story) string {
	text := message.Title

	if message.Score > 0 {
		text += fmt.Sprintf(" (%.0f%%)", message.Score*scoreMultiplier)
	}

	if message.Summary != "" {
		text += "\n" + message.Summary
	}

	return text + "\n" + message.URL
}

// targetName derives a stable broadcast name from a secret URL without leaking
// the secret into the on-disk dedup store (names become storage keys).
func targetName(prefix, secret string) string {
	sum := sha256.Sum256([]byte(secret))

	return prefix + "-" + hex.EncodeToString(sum[:])[:nameHashLen]
}

// postJSON marshals payload as JSON and POSTs it to url, returning an error on a
// transport failure or any non-2xx response.
func postJSON(url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	//nolint:exhaustruct // only the timeout matters here
	client := &http.Client{Timeout: webhookTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("executing webhook request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	_, err = io.Copy(io.Discard, io.LimitReader(resp.Body, maxWebhookResponseBytes))
	if err != nil {
		return fmt.Errorf("reading webhook response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d", errWebhookBadStatus, resp.StatusCode)
	}

	return nil
}
