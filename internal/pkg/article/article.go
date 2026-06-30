// Package article fetches a web page and extracts its readable paragraph text,
// used as a best-effort source for summarizing stories whose feed entry has no
// description of its own.
package article

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	fetchTimeout    = 20 * time.Second
	maxArticleBytes = 4 << 20 // 4 MiB is plenty for an article's HTML
)

var errBadResponseCode = errors.New("bad response code")

var (
	paragraphRe = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	tagRe       = regexp.MustCompile(`<[^>]+>`)
)

// FetchText downloads url and returns the concatenated text of its <p> elements.
func FetchText(ctx context.Context, url string) (string, error) {
	//nolint:exhaustruct // only the timeout matters here
	client := http.Client{Timeout: fetchTimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("building article request: %w", err)
	}

	req.Header.Set("User-Agent", "Mynews/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching article: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errBadResponseCode
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxArticleBytes))
	if err != nil {
		return "", fmt.Errorf("reading article body: %w", err)
	}

	return extractText(body), nil
}

// extractText pulls the text out of every <p> element, dropping inner tags and
// HTML entities and collapsing whitespace.
func extractText(htmlBytes []byte) string {
	var builder strings.Builder

	for _, match := range paragraphRe.FindAllSubmatch(htmlBytes, -1) {
		stripped := tagRe.ReplaceAll(match[1], []byte(" "))
		text := strings.Join(strings.Fields(html.UnescapeString(string(stripped))), " ")

		if text != "" {
			builder.WriteString(text)
			builder.WriteString(" ")
		}
	}

	return strings.TrimSpace(builder.String())
}
