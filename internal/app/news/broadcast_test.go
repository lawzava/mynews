//nolint:testpackage // exercises unexported helpers (normalizeURL, cleanSummary)
package news

import (
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/parser"
	"strings"
	"testing"
	"time"
)

func TestStoryMatchesConfigWholeWordKeywords(t *testing.T) {
	t.Parallel()

	source := &config.Source{ //nolint:exhaustruct // only matching fields matter
		IgnoreStoriesBefore:  time.Now().Add(-time.Hour),
		MustIncludeKeywords:  []string{"ai"},
		MatchKeywordsAsWords: true,
	}

	tests := []struct {
		title string
		want  bool
	}{
		{title: "AI Chip Architectures", want: true},
		{title: "Building an AI-powered assistant", want: true},
		{title: "FDA clears blood test to aid evaluation", want: false},
		{title: "Repair rules take effect", want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.title, func(t *testing.T) {
			t.Parallel()

			story := parser.Item{ //nolint:exhaustruct // matching uses title and parsed time
				Title:             testCase.title,
				PublishedAtParsed: time.Now(),
			}

			if got := storyMatchesConfig(&story, source); got != testCase.want {
				t.Errorf("storyMatchesConfig(%q) = %t, want %t", testCase.title, got, testCase.want)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	const baseURL = "https://example.com/a"

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"strips utm params", baseURL + "?utm_source=x&utm_medium=y", baseURL},
		{"strips fbclid", baseURL + "?fbclid=abc", baseURL},
		{"keeps content params", baseURL + "?id=42", baseURL + "?id=42"},
		{"drops fragment", baseURL + "#section", baseURL},
		{"lowercases host", "https://Example.COM/a", baseURL},
		{"mixed", "https://EX.com/p?id=1&utm_campaign=z#f", "https://ex.com/p?id=1"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeURL(testCase.input)
			if got != testCase.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestCleanSummary(t *testing.T) {
	t.Parallel()

	got := cleanSummary("<p>Hello &amp; <b>world</b></p>\n  of  news")

	want := "Hello & world of news"
	if got != want {
		t.Errorf("cleanSummary() = %q, want %q", got, want)
	}
}

func TestCleanSummaryStripsAggregatorBoilerplate(t *testing.T) {
	t.Parallel()

	// A typical hnrss description: it must not leak the comments link, points,
	// comment count, or that it came from HackerNews.
	raw := "A real summary sentence. " +
		"Article URL: https://example.com/a " +
		"Comments URL: https://news.ycombinator.com/item?id=123 " +
		"Points: 42 # Comments: 7"

	got := cleanSummary(raw)

	if got != "A real summary sentence." {
		t.Errorf("cleanSummary() = %q, want only the real sentence", got)
	}

	for _, leak := range []string{"news.ycombinator.com", "Points", "Comments URL", "# Comments"} {
		if strings.Contains(got, leak) {
			t.Errorf("summary leaks %q: %q", leak, got)
		}
	}
}
