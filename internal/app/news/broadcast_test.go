//nolint:testpackage // exercises unexported helpers (normalizeURL, cleanSummary)
package news

import "testing"

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
