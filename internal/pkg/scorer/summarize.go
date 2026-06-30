package scorer

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxSummarySentences = 40  // cap encodings per article to bound CPU
	minSentenceLen      = 40  // skip fragments and boilerplate
	maxSentenceLen      = 320 // skip run-on / list blobs
	minProseWords       = 6   // shorter candidates are likely headings
	minLowercaseRatio   = 0.5 // prose has many lowercase function words; nav menus do not
)

var sentenceBoundaryRe = regexp.MustCompile(`[.!?]+\s+`)

// Summarize returns the article sentence most similar to the title, computed with
// the static embeddings already loaded for scoring. It is extractive: it selects
// an existing sentence rather than generating text.
func (scorer *EmbeddingScorer) Summarize(ctx context.Context, title, body string) (string, error) {
	err := ctx.Err()
	if err != nil {
		return "", fmt.Errorf("summarize: %w", err)
	}

	sentences := splitSentences(body)
	if len(sentences) == 0 {
		return "", nil
	}

	titleEmbedding := scorer.model.encode(title)
	best := ""
	bestScore := math.Inf(-1)

	for _, sentence := range sentences {
		similarity := cosineSimilarity(scorer.model.encode(sentence), titleEmbedding)
		if similarity > bestScore {
			bestScore = similarity
			best = sentence
		}
	}

	return best, nil
}

// splitSentences breaks body into candidate sentences, keeping only substantive
// ones and capping the count so summarizing stays cheap.
func splitSentences(body string) []string {
	parts := sentenceBoundaryRe.Split(body, -1)
	sentences := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < minSentenceLen || len(part) > maxSentenceLen {
			continue
		}

		if !looksLikeProse(part) {
			continue
		}

		sentences = append(sentences, part)
		if len(sentences) >= maxSummarySentences {
			break
		}
	}

	return sentences
}

// looksLikeProse rejects headings and navigation menus, which are dominated by
// capitalized words, by requiring a sentence to have several words of which a
// good fraction start lowercase (as English function words do).
func looksLikeProse(sentence string) bool {
	words := strings.Fields(sentence)
	if len(words) < minProseWords {
		return false
	}

	lowercase := 0

	for _, word := range words {
		first, _ := utf8.DecodeRuneInString(word)
		if unicode.IsLower(first) {
			lowercase++
		}
	}

	return float64(lowercase)/float64(len(words)) >= minLowercaseRatio
}
