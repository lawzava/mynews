//nolint:testpackage // exercises the unexported extractText helper
package article

import (
	"context"
	"errors"
	"mynews/internal/pkg/safehttp"
	"testing"
)

func TestExtractText(t *testing.T) {
	t.Parallel()

	page := []byte(`<html><body><nav>Menu</nav>` +
		`<p>First <b>bold</b> sentence.</p><script>noise()</script>` +
		`<p>Second &amp; clean.</p></body></html>`)

	got := extractText(page)

	want := "First bold sentence. Second & clean."
	if got != want {
		t.Errorf("extractText() = %q, want %q", got, want)
	}
}

// TestFetchTextBlocksPrivate confirms FetchText uses the SSRF-safe client, so a
// feed link pointing at a private/loopback host is refused.
func TestFetchTextBlocksPrivate(t *testing.T) {
	t.Parallel()

	_, err := FetchText(context.Background(), "http://127.0.0.1:80/article")
	if !errors.Is(err, safehttp.ErrBlockedAddress) {
		t.Errorf("expected a blocked-address error, got %v", err)
	}
}
