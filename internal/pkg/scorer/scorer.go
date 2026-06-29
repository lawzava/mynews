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

	// Derive returns a scorer that scores against the given interests, reusing any
	// already-loaded model. It returns the receiver unchanged when interests is empty.
	Derive(interests []string) (Scorer, error)

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

	// ModelName is the HuggingFace model2vec model for the embedding scorer.
	// Defaults to "minishlab/potion-base-8M".
	ModelName string
}

// NewScorer creates a scorer based on configuration.
//
//nolint:ireturn // factory deliberately returns the Scorer impl selected by provider
func NewScorer(cfg Config) (Scorer, error) {
	switch cfg.Provider {
	case ProviderKeyword:
		keywordScorer, err := NewKeywordScorer(cfg)
		if err != nil {
			return nil, err
		}

		return keywordScorer, nil
	default:
		embeddingScorer, err := NewEmbeddingScorer(cfg)
		if err != nil {
			return nil, err
		}

		return embeddingScorer, nil
	}
}
