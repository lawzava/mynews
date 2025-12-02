package scorer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/nlpodyssey/cybertron/pkg/models/bert"
	"github.com/nlpodyssey/cybertron/pkg/tasks"
	"github.com/nlpodyssey/cybertron/pkg/tasks/textencoding"
)

const (
	// DefaultModelName is the default sentence-transformers model.
	DefaultModelName = "sentence-transformers/all-MiniLM-L6-v2"

	defaultModelDirPerm = 0o755
	scoreNormalizer     = 2
)

var errNoInterests = errors.New("at least one interest is required")

// EmbeddingScorer scores stories using semantic similarity with sentence embeddings.
type EmbeddingScorer struct {
	model              textencoding.Interface
	interestTexts      []string
	interestEmbeddings [][]float64
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

	modelDir := cfg.ModelDir

	if modelDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		modelDir = filepath.Join(homeDir, ".config", "mynews", "models")
	}

	mkdirErr := os.MkdirAll(modelDir, defaultModelDirPerm)
	if mkdirErr != nil {
		return nil, fmt.Errorf("failed to create model directory: %w", mkdirErr)
	}

	// Load or download the model
	model, err := tasks.Load[textencoding.Interface](&tasks.Config{
		ModelsDir:           modelDir,
		ModelName:           modelName,
		HubAccessToken:      "",
		DownloadPolicy:      tasks.DownloadMissing,
		ConversionPolicy:    tasks.ConvertMissing,
		ConversionPrecision: tasks.F32,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load model %s: %w", modelName, err)
	}

	embeddingScorer := &EmbeddingScorer{
		model:              model,
		interestTexts:      cfg.Interests,
		interestEmbeddings: make([][]float64, len(cfg.Interests)),
	}

	// Pre-compute embeddings for all interests
	ctx := context.Background()

	for interestIdx, interest := range cfg.Interests {
		result, encodeErr := model.Encode(ctx, interest, int(bert.MeanPooling))
		if encodeErr != nil {
			return nil, fmt.Errorf("failed to encode interest %q: %w", interest, encodeErr)
		}

		embeddingScorer.interestEmbeddings[interestIdx] = result.Vector.Data().F64()
	}

	return embeddingScorer, nil
}

// Score computes semantic similarity between the story title and user interests.
func (e *EmbeddingScorer) Score(ctx context.Context, title string) (Score, error) {
	// Encode the story title
	result, err := e.model.Encode(ctx, title, int(bert.MeanPooling))
	if err != nil {
		return Score{}, fmt.Errorf("failed to encode title: %w", err)
	}

	titleEmbedding := result.Vector.Data().F64()

	// Find the highest similarity to any interest
	maxSim := 0.0
	bestMatch := ""

	for i, interestEmb := range e.interestEmbeddings {
		sim := cosineSimilarity(titleEmbedding, interestEmb)
		if sim > maxSim {
			maxSim = sim
			bestMatch = e.interestTexts[i]
		}
	}

	// Normalize similarity to 0-1 range (cosine similarity can be negative)
	// For sentence-transformers, values typically range from -1 to 1
	normalizedScore := (maxSim + 1) / scoreNormalizer

	if normalizedScore < 0 {
		normalizedScore = 0
	}

	if normalizedScore > 1 {
		normalizedScore = 1
	}

	return Score{
		Value:  normalizedScore,
		Reason: bestMatch,
	}, nil
}

// Name returns the scorer identifier.
func (e *EmbeddingScorer) Name() string {
	return ProviderEmbedding
}

// Close releases model resources.
func (e *EmbeddingScorer) Close() error {
	return nil
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(vecA, vecB []float64) float64 {
	if len(vecA) != len(vecB) || len(vecA) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64

	for i := range vecA {
		dotProduct += vecA[i] * vecB[i]
		normA += vecA[i] * vecA[i]
		normB += vecB[i] * vecB[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
