package scorer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	wordPiecePrefix        = "##"
	defaultMaxCharsPerWord = 100   // HuggingFace WordPiece default
	maxTokenizerInputRunes = 20000 // hard cap so a giant title cannot stall scoring
)

var errUnkTokenMissing = errors.New("unk token missing from vocab")

type wordPieceTokenizer struct {
	vocab              map[string]int
	unkTokenID         int
	maxCharsPerWord    int
	lowercase          bool
	stripAccents       bool
	cleanText          bool
	handleChineseChars bool
}

type tokenizerJSON struct {
	Normalizer tokenizerNormalizerJSON `json:"normalizer"`
	Model      tokenizerModelJSON      `json:"model"`
}

type tokenizerNormalizerJSON struct {
	Lowercase *bool `json:"lowercase"`

	StripAccents *bool `json:"strip_accents"` //nolint:tagliatelle // HuggingFace tokenizer JSON uses snake_case.

	CleanText *bool `json:"clean_text"` //nolint:tagliatelle // HuggingFace tokenizer JSON uses snake_case.

	//nolint:tagliatelle // HuggingFace tokenizer JSON uses snake_case.
	HandleChineseChars *bool `json:"handle_chinese_chars"`
}

type tokenizerModelJSON struct {
	Vocab map[string]int `json:"vocab"`

	UnkToken string `json:"unk_token"` //nolint:tagliatelle // HuggingFace tokenizer JSON uses snake_case.

	MaxInputCharsPerWord int `json:"max_input_chars_per_word"` //nolint:tagliatelle // HF snake_case
}

func loadWordPieceTokenizer(tokenizerPath string) (*wordPieceTokenizer, error) {
	tokenizerBytes, err := os.ReadFile(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("read tokenizer: %w", err)
	}

	var parsed tokenizerJSON

	err = json.Unmarshal(tokenizerBytes, &parsed)
	if err != nil {
		return nil, fmt.Errorf("parse tokenizer: %w", err)
	}

	unkTokenID, ok := parsed.Model.Vocab[parsed.Model.UnkToken]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errUnkTokenMissing, parsed.Model.UnkToken)
	}

	lowercase := boolValue(parsed.Normalizer.Lowercase, true)
	stripAccents := boolValue(parsed.Normalizer.StripAccents, lowercase)

	maxCharsPerWord := parsed.Model.MaxInputCharsPerWord
	if maxCharsPerWord <= 0 {
		maxCharsPerWord = defaultMaxCharsPerWord
	}

	return &wordPieceTokenizer{
		vocab:              parsed.Model.Vocab,
		unkTokenID:         unkTokenID,
		maxCharsPerWord:    maxCharsPerWord,
		lowercase:          lowercase,
		stripAccents:       stripAccents,
		cleanText:          boolValue(parsed.Normalizer.CleanText, true),
		handleChineseChars: boolValue(parsed.Normalizer.HandleChineseChars, true),
	}, nil
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}

	return *value
}

func (tokenizer *wordPieceTokenizer) tokenIDs(text string) []int {
	if len([]rune(text)) > maxTokenizerInputRunes {
		text = string([]rune(text)[:maxTokenizerInputRunes])
	}

	normalized := tokenizer.normalize(text)
	basicTokens := basicTokenize(normalized)
	tokenIDs := make([]int, 0, len(basicTokens))

	for _, token := range basicTokens {
		tokenIDs = append(tokenIDs, tokenizer.wordPieceIDs(token)...)
	}

	return tokenIDs
}

func (tokenizer *wordPieceTokenizer) normalize(text string) string {
	if tokenizer.cleanText {
		text = cleanTokenText(text)
	}

	if tokenizer.handleChineseChars {
		text = spaceCJKChars(text)
	}

	if tokenizer.lowercase {
		text = strings.ToLower(text)
	}

	if tokenizer.stripAccents {
		text = stripAccentMarks(text)
	}

	return text
}

func cleanTokenText(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))

	for _, tokenRune := range text {
		if tokenRune == 0 || tokenRune == utf8.RuneError || isControl(tokenRune) {
			continue
		}

		if unicode.IsSpace(tokenRune) {
			builder.WriteRune(' ')

			continue
		}

		builder.WriteRune(tokenRune)
	}

	return builder.String()
}

func spaceCJKChars(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))

	for _, tokenRune := range text {
		if isCJKChar(tokenRune) {
			builder.WriteRune(' ')
			builder.WriteRune(tokenRune)
			builder.WriteRune(' ')

			continue
		}

		builder.WriteRune(tokenRune)
	}

	return builder.String()
}

func stripAccentMarks(text string) string {
	var builder strings.Builder

	for _, tokenRune := range norm.NFD.String(text) {
		if unicode.Is(unicode.Mn, tokenRune) {
			continue
		}

		builder.WriteRune(tokenRune)
	}

	return builder.String()
}

func basicTokenize(text string) []string {
	tokens := make([]string, 0)

	var builder strings.Builder

	for _, tokenRune := range text {
		if unicode.IsSpace(tokenRune) {
			flushToken(&tokens, &builder)

			continue
		}

		if isPunctuation(tokenRune) {
			flushToken(&tokens, &builder)
			tokens = append(tokens, string(tokenRune))

			continue
		}

		builder.WriteRune(tokenRune)
	}

	flushToken(&tokens, &builder)

	return tokens
}

func flushToken(tokens *[]string, builder *strings.Builder) {
	if builder.Len() == 0 {
		return
	}

	*tokens = append(*tokens, builder.String())
	builder.Reset()
}

func (tokenizer *wordPieceTokenizer) wordPieceIDs(token string) []int {
	if token == "" {
		return nil
	}

	tokenRunes := []rune(token)
	if len(tokenRunes) > tokenizer.maxCharsPerWord {
		return []int{tokenizer.unkTokenID} // matches HF; also bounds the O(n^2) match
	}

	startRune := 0
	tokenIDs := make([]int, 0)

	for startRune < len(tokenRunes) {
		endRune := len(tokenRunes)
		currentID := -1

		for startRune < endRune {
			piece := string(tokenRunes[startRune:endRune])
			if startRune > 0 {
				piece = wordPiecePrefix + piece
			}

			tokenID, ok := tokenizer.vocab[piece]
			if ok {
				currentID = tokenID

				break
			}

			endRune--
		}

		if currentID < 0 {
			return []int{tokenizer.unkTokenID}
		}

		tokenIDs = append(tokenIDs, currentID)
		startRune = endRune
	}

	return tokenIDs
}

func isControl(tokenRune rune) bool {
	if tokenRune == '\t' || tokenRune == '\n' || tokenRune == '\r' {
		return false
	}

	return unicode.IsControl(tokenRune)
}

func isPunctuation(tokenRune rune) bool {
	if tokenRune >= '!' && tokenRune <= '/' {
		return true
	}

	if tokenRune >= ':' && tokenRune <= '@' {
		return true
	}

	if tokenRune >= '[' && tokenRune <= '`' {
		return true
	}

	if tokenRune >= '{' && tokenRune <= '~' {
		return true
	}

	return unicode.IsPunct(tokenRune)
}

func isCJKChar(tokenRune rune) bool {
	type runeRange struct {
		start rune
		end   rune
	}

	cjkRanges := [...]runeRange{
		{start: '\u4E00', end: '\u9FFF'},
		{start: '\u3400', end: '\u4DBF'},
		{start: '\U00020000', end: '\U0002A6DF'},
		{start: '\U0002A700', end: '\U0002B73F'},
		{start: '\U0002B740', end: '\U0002B81F'},
		{start: '\U0002B820', end: '\U0002CEAF'},
		{start: '\uF900', end: '\uFAFF'},
		{start: '\U0002F800', end: '\U0002FA1F'},
	}

	for _, cjkRange := range cjkRanges {
		if tokenRune >= cjkRange.start && tokenRune <= cjkRange.end {
			return true
		}
	}

	return false
}
