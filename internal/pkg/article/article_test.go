package article_test

import (
	"context"
	"mynews/internal/pkg/article"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchTextExtractsParagraphs(t *testing.T) {
	t.Parallel()

	page := `<html><body><nav>Menu</nav>` +
		`<p>First <b>bold</b> sentence.</p><script>noise()</script>` +
		`<p>Second &amp; clean.</p></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(page))
	}))
	defer server.Close()

	text, err := article.FetchText(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}

	want := "First bold sentence. Second & clean."
	if text != want {
		t.Errorf("got %q, want %q", text, want)
	}
}

func TestFetchTextNon2xxIsError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := article.FetchText(context.Background(), server.URL)
	if err == nil {
		t.Error("expected an error on a 404 response")
	}
}
