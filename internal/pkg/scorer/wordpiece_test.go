package scorer //nolint:testpackage // tests exercise unexported tokenizer helpers.

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWordPieceTokenizeNormalizesAndSplits(t *testing.T) {
	t.Parallel()

	tokenizer, err := loadWordPieceTokenizer(writeTestTokenizer(t))
	if err != nil {
		t.Fatalf("load tokenizer: %v", err)
	}

	ids := tokenizer.tokenIDs("Hello, caf\u00e9s \u4e16\u754c")
	want := []int{2, 4, 5, 8, 6, 7}

	assertTokenIDs(t, ids, want)
}

func TestWordPieceTokenizeUsesUnknownForMissingPiece(t *testing.T) {
	t.Parallel()

	tokenizer, err := loadWordPieceTokenizer(writeTestTokenizer(t))
	if err != nil {
		t.Fatalf("load tokenizer: %v", err)
	}

	ids := tokenizer.tokenIDs("unmatched")
	want := []int{1}

	assertTokenIDs(t, ids, want)
}

func writeTestTokenizer(t *testing.T) string {
	t.Helper()

	tokenizerPath := filepath.Join(t.TempDir(), "tokenizer.json")
	//nolint:gosec // tokenizer fixture contains model token strings, not credentials.
	tokenizerJSON := `{
		"normalizer":{"type":"BertNormalizer","clean_text":true,"handle_chinese_chars":true,"lowercase":true},
		"model":{
			"type":"WordPiece",
			"unk_token":"[UNK]",
			"vocab":{
				"[PAD]":0,
				"[UNK]":1,
				"hello":2,
				"world":3,
				",":4,
				"cafe":5,
				"\u4e16":6,
				"\u754c":7,
				"##s":8
			}
		}
	}`

	err := os.WriteFile(tokenizerPath, []byte(tokenizerJSON), defaultFilePerm)
	if err != nil {
		t.Fatalf("write tokenizer: %v", err)
	}

	return tokenizerPath
}

func assertTokenIDs(t *testing.T, got, want []int) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
}
