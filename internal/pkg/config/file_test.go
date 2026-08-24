//nolint:testpackage // this test exercises unexported file decoding
package config

import (
	"mynews/internal/pkg/logger"
	"path/filepath"
	"testing"
)

func TestSourceScoringOptionsLoadFromFileStructure(t *testing.T) {
	t.Parallel()

	minScore := 0.12
	file := fileStructure{ //nolint:exhaustruct // the test exercises only source options
		Elements: []fileStructureElement{
			{ //nolint:exhaustruct // unrelated app fields use defaults
				BroadcastType: stdoutBroadcastType,
				Sources: []fileStructureSource{
					{ //nolint:exhaustruct // unrelated source fields use defaults
						URL:                  "https://news.ycombinator.com/rss",
						MinScore:             &minScore,
						MatchKeywordsAsWords: true,
					},
				},
			},
		},
	}

	cfg, err := file.toConfig(filepath.Join(t.TempDir(), "data.json"), logger.New(logger.Error))
	if err != nil {
		t.Fatalf("toConfig: %v", err)
	}

	source := cfg.Apps[0].Sources[0]
	if source.MinScore == nil || *source.MinScore != minScore {
		t.Fatalf("source MinScore = %v, want %v", source.MinScore, minScore)
	}

	if !source.MatchKeywordsAsWords {
		t.Fatal("source MatchKeywordsAsWords = false, want true")
	}
}
