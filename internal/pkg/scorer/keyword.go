package scorer

import (
	"context"
	"strings"
)

// KeywordScorer scores stories using simple keyword matching against interests.
// This is a minimal fallback when embedding models are not desired.
type KeywordScorer struct {
	interests []string
	keywords  map[string][]string // interest -> extracted keywords
}

// NewKeywordScorer creates a keyword-based scorer.
func NewKeywordScorer(cfg Config) (*KeywordScorer, error) {
	keywordScorer := &KeywordScorer{
		interests: cfg.Interests,
		keywords:  make(map[string][]string),
	}

	// Extract keywords from each interest phrase
	for _, interest := range cfg.Interests {
		keywordScorer.keywords[interest] = extractKeywords(interest)
	}

	return keywordScorer, nil
}

// Score computes a relevance score based on keyword matching.
func (k *KeywordScorer) Score(_ context.Context, title string) (Score, error) {
	titleLower := strings.ToLower(title)

	maxScore := 0.0
	bestMatch := ""

	for interest, keywords := range k.keywords {
		matchCount := 0

		for _, kw := range keywords {
			if strings.Contains(titleLower, kw) {
				matchCount++
			}
		}

		if len(keywords) > 0 {
			score := float64(matchCount) / float64(len(keywords))
			if score > maxScore {
				maxScore = score
				bestMatch = interest
			}
		}
	}

	return Score{
		Value:  maxScore,
		Reason: bestMatch,
	}, nil
}

// Name returns the scorer identifier.
func (k *KeywordScorer) Name() string {
	return ProviderKeyword
}

// Close is a no-op for keyword scorer.
func (k *KeywordScorer) Close() error {
	return nil
}

// extractKeywords extracts meaningful keywords from an interest phrase.
func extractKeywords(phrase string) []string {
	// Common stop words to filter out
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true,
		"of": true, "in": true, "to": true, "for": true, "with": true,
		"on": true, "at": true, "by": true, "from": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"being": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "shall": true,
		"about": true, "into": true, "through": true, "during": true,
		"before": true, "after": true, "above": true, "below": true,
		"between": true, "under": true, "again": true, "further": true,
		"then": true, "once": true, "here": true, "there": true, "when": true,
		"where": true, "why": true, "how": true, "all": true, "each": true,
		"few": true, "more": true, "most": true, "other": true, "some": true,
		"such": true, "no": true, "nor": true, "not": true, "only": true,
		"own": true, "same": true, "so": true, "than": true, "too": true,
		"very": true, "just": true, "also": true, "now": true, "new": true,
	}

	words := strings.Fields(strings.ToLower(phrase))
	keywords := make([]string, 0, len(words))

	for _, word := range words {
		// Clean punctuation
		word = strings.Trim(word, ".,!?;:'\"()-")

		// Skip short words and stop words
		if len(word) < 3 || stopWords[word] {
			continue
		}

		keywords = append(keywords, word)
	}

	return keywords
}
