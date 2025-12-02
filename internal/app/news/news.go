package news

import (
	"fmt"
	"mynews/internal/pkg/config"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/scorer"
	"path/filepath"
)

// News handles RSS feed parsing and broadcasting.
type News struct {
	cfg    *config.Config
	scorer scorer.Scorer
}

// New creates a new News instance with optional scoring.
func New(cfg *config.Config, log *logger.Log) (News, error) {
	newsInstance := News{
		cfg:    cfg,
		scorer: nil,
	}

	if cfg.Scoring != nil && cfg.Scoring.Enabled {
		modelDir := cfg.Scoring.ModelDir
		if modelDir == "" {
			modelDir = filepath.Join(filepath.Dir(cfg.StorageFilePath), "models")
		}

		log.Info(fmt.Sprintf("Initializing %s scorer with %d interests...", cfg.Scoring.Provider, len(cfg.Scoring.Interests)))

		embeddingScorer, keywordScorer, err := scorer.NewScorer(scorer.Config{
			Provider:  cfg.Scoring.Provider,
			Interests: cfg.Scoring.Interests,
			ModelDir:  modelDir,
			ModelName: cfg.Scoring.ModelName,
		})
		if err != nil {
			return News{}, fmt.Errorf("failed to initialize scorer: %w", err)
		}

		if embeddingScorer != nil {
			newsInstance.scorer = embeddingScorer
			log.Info(fmt.Sprintf("Scorer initialized successfully (provider: %s)", embeddingScorer.Name()))
		} else if keywordScorer != nil {
			newsInstance.scorer = keywordScorer
			log.Info(fmt.Sprintf("Scorer initialized successfully (provider: %s)", keywordScorer.Name()))
		}
	}

	return newsInstance, nil
}

// Close releases resources held by News.
func (n News) Close() error {
	if n.scorer != nil {
		closeErr := n.scorer.Close()
		if closeErr != nil {
			return fmt.Errorf("failed to close scorer: %w", closeErr)
		}
	}

	return nil
}
