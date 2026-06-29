package scorer //nolint:testpackage // tests exercise unexported parser/model helpers.

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
)

const (
	floatTolerance = 1e-6
	testModelName  = "unit/model"
	testWorldToken = "world"
)

func TestModel2VecEncodeMeansRowsAndNormalizes(t *testing.T) {
	t.Parallel()

	modelDir := t.TempDir()
	writeTestModel(t, modelDir)

	model, err := loadModel2Vec(context.Background(), model2VecLoadConfig{
		modelDir:  modelDir,
		modelName: testModelName,
	})
	if err != nil {
		t.Fatalf("load model2vec: %v", err)
	}

	vector := model.encode("hello world")
	wantScale := 1 / math.Sqrt(2)
	want := []float32{float32(wantScale), float32(wantScale)}

	assertFloat32Slice(t, vector, want)
}

func TestEmbeddingScorerUsesBestInterest(t *testing.T) {
	t.Parallel()

	modelDir := t.TempDir()
	writeTestModel(t, modelDir)

	scorer, err := NewEmbeddingScorer(Config{
		Provider:  ProviderEmbedding,
		Interests: []string{"hello", testWorldToken},
		ModelDir:  modelDir,
		ModelName: testModelName,
	})
	if err != nil {
		t.Fatalf("new embedding scorer: %v", err)
	}

	score, err := scorer.Score(context.Background(), testWorldToken)
	if err != nil {
		t.Fatalf("score title: %v", err)
	}

	if score.Reason != testWorldToken {
		t.Fatalf("reason = %q, want world", score.Reason)
	}

	if score.Value != 1 {
		t.Fatalf("score = %v, want 1", score.Value)
	}
}

func writeTestModel(t *testing.T, modelDir string) {
	t.Helper()

	writeTestJSON(t, filepath.Join(modelDir, "config.json"), `{"hidden_dim":2,"normalize":true}`)
	writeTestTokenizerFile(t, filepath.Join(modelDir, "tokenizer.json"))
	writeTestSafetensors(t, filepath.Join(modelDir, "model.safetensors"), "embeddings", []int{4, 2}, []float32{
		0, 0,
		0, 0,
		1, 0,
		0, 1,
	})
}

func writeTestTokenizerFile(t *testing.T, tokenizerPath string) {
	t.Helper()

	//nolint:gosec // tokenizer fixture contains model token strings, not credentials.
	tokenizerJSON := `{
		"normalizer":{"type":"BertNormalizer","clean_text":true,"handle_chinese_chars":true,"lowercase":true},
		"model":{
			"type":"WordPiece",
			"unk_token":"[UNK]",
			"vocab":{"[PAD]":0,"[UNK]":1,"hello":2,"world":3}
		}
	}`

	writeTestJSON(t, tokenizerPath, tokenizerJSON)
}

func writeTestJSON(t *testing.T, filePath, data string) {
	t.Helper()

	err := os.WriteFile(filePath, []byte(data), defaultFilePerm)
	if err != nil {
		t.Fatalf("write json: %v", err)
	}
}

func assertFloat32Slice(t *testing.T, got, want []float32) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("vector = %v, want %v", got, want)
	}

	for valueIdx := range got {
		if math.Abs(float64(got[valueIdx]-want[valueIdx])) > floatTolerance {
			t.Fatalf("vector = %v, want %v", got, want)
		}
	}
}
