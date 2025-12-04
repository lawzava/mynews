package extractor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ContentSource indicates where the scoring text originated.
type ContentSource string

const (
	// SourceArticle indicates content was extracted from the article body.
	SourceArticle ContentSource = "article"
	// SourceOGDescription indicates content came from og:description meta tag.
	SourceOGDescription ContentSource = "og:description"
	// SourceTitle indicates fallback to title-only scoring.
	SourceTitle ContentSource = "title"
)

// ExtractedContent holds the result of content extraction.
type ExtractedContent struct {
	Content       string        // Main article text (best quality)
	OGDescription string        // OpenGraph description (fallback)
	Source        ContentSource // Which source the content came from
}

const (
	defaultTimeout        = 10 * time.Second
	defaultMaxContentSize = 1024 * 1024 // 1MB
	userAgent             = "Mynews/1.0"

	minArticleContentLength = 100 // Minimum viable content length for article extraction
	minBodyContentLength    = 200 // Minimum viable content length for body fallback
)

var errUnexpectedStatusCode = errors.New("unexpected status code")

// Extract fetches the URL and extracts content with fallback chain:
// Article content -> OG description -> empty (caller uses title).
func Extract(ctx context.Context, url string) (ExtractedContent, error) {
	result := ExtractedContent{
		Content:       "",
		OGDescription: "",
		Source:        SourceTitle, // default fallback
	}

	// Create HTTP request with timeout
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return result, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	//nolint:exhaustruct // no need to set all fields
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("fetching URL: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("%w: %d", errUnexpectedStatusCode, resp.StatusCode)
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, defaultMaxContentSize)

	doc, err := goquery.NewDocumentFromReader(limitedReader)
	if err != nil {
		return result, fmt.Errorf("parsing HTML: %w", err)
	}

	// Extract OG description as fallback
	result.OGDescription = extractOGDescription(doc)

	// Try to extract main article content
	result.Content = extractArticleContent(doc)

	// Determine source based on what we found
	if result.Content != "" {
		result.Source = SourceArticle
	} else if result.OGDescription != "" {
		result.Source = SourceOGDescription
	}

	return result, nil
}

// GetBestText returns the best available text for scoring.
// Falls back through: Content -> OGDescription -> title.
func (e ExtractedContent) GetBestText(title string) (string, ContentSource) {
	if e.Content != "" {
		return e.Content, SourceArticle
	}

	if e.OGDescription != "" {
		return e.OGDescription, SourceOGDescription
	}

	return title, SourceTitle
}

func extractOGDescription(doc *goquery.Document) string {
	// Try og:description first
	if content, exists := doc.Find(`meta[property="og:description"]`).Attr("content"); exists {
		return strings.TrimSpace(content)
	}

	// Fall back to regular description meta tag
	if content, exists := doc.Find(`meta[name="description"]`).Attr("content"); exists {
		return strings.TrimSpace(content)
	}

	return ""
}

func extractArticleContent(doc *goquery.Document) string {
	// Remove noise elements that don't contain article content
	doc.Find("script, style, nav, footer, header, aside, .sidebar, .comments, .advertisement, .ad, noscript").Remove()

	// Try semantic elements in order of preference
	selectors := []string{
		"article",
		"main",
		`[role="main"]`,
		".post-content",
		".article-content",
		".entry-content",
		".content",
	}

	for _, selector := range selectors {
		if selection := doc.Find(selector).First(); selection.Length() > 0 {
			text := cleanText(selection.Text())
			if len(text) > minArticleContentLength {
				return text
			}
		}
	}

	// Fall back to body with cleaned text
	bodyText := cleanText(doc.Find("body").Text())
	if len(bodyText) > minBodyContentLength {
		return bodyText
	}

	return ""
}

func cleanText(text string) string {
	// Normalize whitespace
	text = strings.Join(strings.Fields(text), " ")

	return strings.TrimSpace(text)
}
