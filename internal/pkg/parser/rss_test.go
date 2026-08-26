//nolint:testpackage // exercises unexported RSS decoding
package parser

import "testing"

func TestParseRSSPreservesCommentsURL(t *testing.T) {
	t.Parallel()

	items, err := parseRSS([]byte(`
		<rss version="2.0"><channel><item>
			<title>RAG Is Simpler Than You Think</title>
			<link>https://example.com/rag</link>
			<comments>https://news.ycombinator.com/item?id=49445727</comments>
			<pubDate>Wed, 26 Aug 2026 09:00:00 +0000</pubDate>
		</item></channel></rss>`))
	if err != nil {
		t.Fatalf("parseRSS: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("parsed %d items, want 1", len(items))
	}

	const want = "https://news.ycombinator.com/item?id=49445727"
	if items[0].CommentsURL != want {
		t.Fatalf("CommentsURL = %q, want %q", items[0].CommentsURL, want)
	}
}
