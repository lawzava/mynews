//nolint:testpackage // uses the unexported baseURL seam to point at a test server
package broadcast

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramSendPayload(t *testing.T) {
	t.Parallel()

	var gotPath, gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		raw, _ := io.ReadAll(request.Body)
		gotBody = string(raw)

		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	const testChatID = "chat42"

	client := &Telegram{BotAPIToken: "secret-token", ChatID: testChatID, baseURL: server.URL}

	err := client.Send(Story{Title: "AI wins. Big!", URL: "https://ex.com/a", Summary: "recap", Score: 0.9, Reason: "ai"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if gotPath != "/botsecret-token/sendMessage" {
		t.Errorf("request path = %q, want /botsecret-token/sendMessage", gotPath)
	}

	var payload struct {
		ChatID    string `json:"chat_id"`    //nolint:tagliatelle // telegram API schema
		ParseMode string `json:"parse_mode"` //nolint:tagliatelle // telegram API schema
		Text      string `json:"text"`
	}

	err = json.Unmarshal([]byte(gotBody), &payload)
	if err != nil {
		t.Fatalf("body not JSON: %v (%q)", err, gotBody)
	}

	if payload.ChatID != testChatID {
		t.Errorf("chat_id = %q, want %q", payload.ChatID, testChatID)
	}

	if payload.ParseMode != telegramParseMode {
		t.Errorf("parse_mode = %q, want %q", payload.ParseMode, telegramParseMode)
	}

	// MarkdownV2 reserved characters in the title must be backslash-escaped.
	if !strings.Contains(payload.Text, `Big\!`) || !strings.Contains(payload.Text, `wins\.`) {
		t.Errorf("text not MarkdownV2-escaped: %q", payload.Text)
	}
}

func TestTelegramSendAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"ok":false,"description":"chat not found"}`))
	}))
	defer server.Close()

	client := &Telegram{BotAPIToken: "t", ChatID: "c", baseURL: server.URL}

	err := client.Send(Story{Title: "x", URL: "y", Summary: "", Score: 0, Reason: ""})
	if err == nil {
		t.Fatal("expected an error when Telegram responds ok:false")
	}
}
