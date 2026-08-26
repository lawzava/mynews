//nolint:testpackage // exercises the injectable unexported client constructor
package hackernews

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientStoryScore(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v0/item/49445727.json" {
			t.Errorf("request path = %q", request.URL.Path)
		}

		_, _ = fmt.Fprint(writer, `{"score":56}`)
	}))
	defer server.Close()

	client := newClient(server.URL, server.Client())

	score, err := client.StoryScore(context.Background(),
		"https://news.ycombinator.com/item?id=49445727")
	if err != nil {
		t.Fatalf("StoryScore: %v", err)
	}

	if score != 56 {
		t.Fatalf("score = %d, want 56", score)
	}
}

func TestClientRejectsNonHackerNewsCommentsURL(t *testing.T) {
	t.Parallel()

	client := newClient("https://hacker-news.firebaseio.com", http.DefaultClient)

	_, err := client.StoryScore(context.Background(), "https://example.com/item?id=49445727")
	if err == nil {
		t.Fatal("StoryScore error = nil, want invalid comments URL error")
	}
}
