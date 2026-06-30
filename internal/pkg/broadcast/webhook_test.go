package broadcast_test

import (
	"encoding/json"
	"io"
	"mynews/internal/pkg/broadcast"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testTitle = "AI model breaks records"
	testURL   = "https://example.com/ai"
)

type capture struct {
	server *httptest.Server
	body   *string
}

// newCapture returns a test server that records the body of the last request.
func newCapture(t *testing.T, status int) capture {
	t.Helper()

	var body string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		body = string(raw)

		writer.WriteHeader(status)
	}))

	t.Cleanup(server.Close)

	return capture{server: server, body: &body}
}

func sampleStory() broadcast.Story {
	return broadcast.Story{
		Title:   testTitle,
		URL:     testURL,
		Summary: "A new model tops the charts.",
		Score:   0.9,
		Reason:  "artificial intelligence",
	}
}

func TestDiscordSendPayload(t *testing.T) {
	t.Parallel()

	mock := newCapture(t, http.StatusOK)

	client, err := broadcast.NewDiscordClient(mock.server.URL)
	if err != nil {
		t.Fatalf("new discord client: %v", err)
	}

	err = client.Send(sampleStory())
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	var payload struct {
		Content string `json:"content"`
	}

	err = json.Unmarshal([]byte(*mock.body), &payload)
	if err != nil {
		t.Fatalf("payload not valid JSON: %v (%q)", err, *mock.body)
	}

	for _, want := range []string{testTitle, "90%", testURL} {
		if !strings.Contains(payload.Content, want) {
			t.Errorf("discord content %q missing %q", payload.Content, want)
		}
	}
}

func TestSlackSendPayload(t *testing.T) {
	t.Parallel()

	mock := newCapture(t, http.StatusOK)

	client, err := broadcast.NewSlackClient(mock.server.URL)
	if err != nil {
		t.Fatalf("new slack client: %v", err)
	}

	err = client.Send(sampleStory())
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	var payload struct {
		Text string `json:"text"`
	}

	err = json.Unmarshal([]byte(*mock.body), &payload)
	if err != nil {
		t.Fatalf("payload not valid JSON: %v (%q)", err, *mock.body)
	}

	if !strings.Contains(payload.Text, testTitle) {
		t.Errorf("slack text %q missing title", payload.Text)
	}
}

func TestWebhookSendPayloadIsRawStory(t *testing.T) {
	t.Parallel()

	mock := newCapture(t, http.StatusOK)

	client, err := broadcast.NewWebhookClient(mock.server.URL)
	if err != nil {
		t.Fatalf("new webhook client: %v", err)
	}

	err = client.Send(sampleStory())
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	var got broadcast.Story

	err = json.Unmarshal([]byte(*mock.body), &got)
	if err != nil {
		t.Fatalf("payload not valid Story JSON: %v (%q)", err, *mock.body)
	}

	if got != sampleStory() {
		t.Errorf("webhook payload = %+v, want %+v", got, sampleStory())
	}
}

func TestWebhookNon2xxIsError(t *testing.T) {
	t.Parallel()

	mock := newCapture(t, http.StatusInternalServerError)

	client, err := broadcast.NewWebhookClient(mock.server.URL)
	if err != nil {
		t.Fatalf("new webhook client: %v", err)
	}

	err = client.Send(sampleStory())
	if err == nil {
		t.Error("expected an error on a 500 response, got nil")
	}
}

func TestTargetNameHidesURLSecret(t *testing.T) {
	t.Parallel()

	client, err := broadcast.NewWebhookClient("https://hooks.example.com/secret-token-abc123")
	if err != nil {
		t.Fatalf("new webhook client: %v", err)
	}

	if strings.Contains(client.Name(), "secret-token-abc123") {
		t.Errorf("Name() %q leaks the URL secret", client.Name())
	}

	if !strings.HasPrefix(client.Name(), "webhook-") {
		t.Errorf("Name() %q missing expected prefix", client.Name())
	}
}
