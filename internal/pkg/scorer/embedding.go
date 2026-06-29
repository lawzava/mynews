package scorer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const (
	// DefaultModelName is the default model2vec potion model.
	DefaultModelName = "minishlab/potion-base-8M"

	defaultModelDirPerm = 0o755
	defaultScore        = 0.0
)

var errNoInterests = errors.New("at least one interest is required")

// EmbeddingScorer scores stories using static model2vec embeddings.
type EmbeddingScorer struct {
	model              *model2Vec
	interestTexts      []string
	interestEmbeddings [][]float32
}

// NewEmbeddingScorer creates a new embedding-based scorer.
func NewEmbeddingScorer(cfg Config) (*EmbeddingScorer, error) {
	if len(cfg.Interests) == 0 {
		return nil, errNoInterests
	}

	modelName := cfg.ModelName
	if modelName == "" {
		modelName = DefaultModelName
	}

	modelDir, err := embeddingModelDir(cfg.ModelDir)
	if err != nil {
		return nil, err
	}

	mkdirErr := os.MkdirAll(modelDir, defaultModelDirPerm)
	if mkdirErr != nil {
		return nil, fmt.Errorf("create model directory: %w", mkdirErr)
	}

	model, err := loadModel2Vec(context.Background(), model2VecLoadConfig{
		modelDir:  modelDir,
		modelName: modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("load model2vec model %q: %w", modelName, err)
	}

	embeddingScorer := &EmbeddingScorer{
		model:              model,
		interestTexts:      cfg.Interests,
		interestEmbeddings: make([][]float32, len(cfg.Interests)),
	}

	for interestIdx, interest := range cfg.Interests {
		embeddingScorer.interestEmbeddings[interestIdx] = model.encode(interest)
	}

	return embeddingScorer, nil
}

// Score computes semantic similarity between the story title and user interests.
func (scorer *EmbeddingScorer) Score(ctx context.Context, title string) (Score, error) {
	ctxErr := ctx.Err()
	if ctxErr != nil {
		return Score{}, fmt.Errorf("score embedding: %w", ctxErr)
	}

	titleEmbedding := scorer.model.encode(title)
	maxSimilarity := math.Inf(-1)
	bestMatch := ""

	for interestIdx, interestEmbedding := range scorer.interestEmbeddings {
		similarity := cosineSimilarity(titleEmbedding, interestEmbedding)
		if similarity > maxSimilarity {
			maxSimilarity = similarity
			bestMatch = scorer.interestTexts[interestIdx]
		}
	}

	return Score{
		Value:  min(1, max(defaultScore, maxSimilarity)),
		Reason: bestMatch,
	}, nil
}

// Name returns the scorer identifier.
func (scorer *EmbeddingScorer) Name() string {
	return ProviderEmbedding
}

// Close releases model resources.
func (scorer *EmbeddingScorer) Close() error {
	return nil
}

func embeddingModelDir(configuredDir string) (string, error) {
	if configuredDir != "" {
		return configuredDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "mynews", "models"), nil
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(vecA, vecB []float32) float64 {
	if len(vecA) != len(vecB) || len(vecA) == 0 {
		return 0
	}

	var dotProduct float64

	var normA float64

	var normB float64

	for valueIdx := range vecA {
		valueA := float64(vecA[valueIdx])
		valueB := float64(vecB[valueIdx])
		dotProduct += valueA * valueB
		normA += valueA * valueA
		normB += valueB * valueB
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
