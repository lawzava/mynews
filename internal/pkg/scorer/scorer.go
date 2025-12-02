package scorer

import "context"

const (
	// ProviderEmbedding uses sentence embeddings for semantic similarity scoring.
	ProviderEmbedding = "embedding"
	// ProviderKeyword uses simple keyword matching for scoring.
	ProviderKeyword = "keyword"
)

// Score represents the AI scoring result for a story.
type Score struct {
	Value  float64 // 0.0 to 1.0 (normalized relevance score)
	Reason string  // Brief explanation of the score
}

// Scorer evaluates story relevance based on user interests.
type Scorer interface {
	// Score evaluates a story title against configured interests.
	// Returns a Score with Value between 0.0 and 1.0.
	Score(ctx context.Context, title string) (Score, error)

	// Name returns the scorer identifier (e.g., "embedding", "keyword").
	Name() string

	// Close releases any resources held by the scorer.
	Close() error
}

// Config holds scorer configuration.
type Config struct {
	// Provider specifies which scorer to use: "embedding" or "keyword"
	Provider string

	// Interests are the topics/themes to score stories against
	Interests []string

	// ModelDir is the directory to cache downloaded models
	ModelDir string

	// ModelName is the HuggingFace model name for embedding scorer
	// Defaults to "sentence-transformers/all-MiniLM-L6-v2"
	ModelName string
}

// NewScorer creates a scorer based on configuration.
func NewScorer(cfg Config) (*EmbeddingScorer, *KeywordScorer, error) {
	switch cfg.Provider {
	case ProviderKeyword:
		keywordScorer, err := NewKeywordScorer(cfg)

		return nil, keywordScorer, err
	default:
		embeddingScorer, err := NewEmbeddingScorer(cfg)

		return embeddingScorer, nil, err
	}
}
