package scorer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	modelFileName     = "model.safetensors"
	tokenizerFileName = "tokenizer.json" //nolint:gosec // model filename, not an auth token.
	configFileName    = "config.json"

	modelDownloadTimeout = 5 * time.Minute
	modelUserAgent       = "mynews-model2vec/1.0"
)

var (
	errHiddenDimMismatch       = errors.New("config hidden_dim does not match embedding dim")
	errUnexpectedDownloadState = errors.New("unexpected model download status")
)

type model2Vec struct {
	tokenizer *wordPieceTokenizer
	matrix    *embeddingMatrix
	normalize bool
	dim       int
}

type model2VecLoadConfig struct {
	modelDir  string
	modelName string
}

type model2VecConfigFile struct {
	HiddenDim int `json:"hidden_dim"` //nolint:tagliatelle // external model config uses snake_case.

	Normalize bool `json:"normalize"`
}

func loadModel2Vec(ctx context.Context, cfg model2VecLoadConfig) (*model2Vec, error) {
	err := ensureModelFiles(ctx, cfg)
	if err != nil {
		return nil, err
	}

	modelConfig, err := loadModel2VecConfig(filepath.Join(cfg.modelDir, configFileName))
	if err != nil {
		return nil, err
	}

	matrix, _, err := loadSafetensorsEmbeddings(filepath.Join(cfg.modelDir, modelFileName))
	if err != nil {
		return nil, err
	}

	if modelConfig.HiddenDim != 0 && modelConfig.HiddenDim != matrix.dim {
		return nil, fmt.Errorf("%w: %d != %d", errHiddenDimMismatch, modelConfig.HiddenDim, matrix.dim)
	}

	tokenizer, err := loadWordPieceTokenizer(filepath.Join(cfg.modelDir, tokenizerFileName))
	if err != nil {
		return nil, err
	}

	return &model2Vec{
		tokenizer: tokenizer,
		matrix:    matrix,
		normalize: modelConfig.Normalize,
		dim:       matrix.dim,
	}, nil
}

func ensureModelFiles(ctx context.Context, cfg model2VecLoadConfig) error {
	client := &http.Client{ //nolint:exhaustruct // timeout is the only non-default behavior required.
		Timeout: modelDownloadTimeout,
	}
	fileNames := []string{modelFileName, tokenizerFileName, configFileName}

	for _, fileName := range fileNames {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			return fmt.Errorf("ensure model files: %w", ctxErr)
		}

		filePath := filepath.Join(cfg.modelDir, fileName)

		_, statErr := os.Stat(filePath)
		if statErr == nil {
			continue
		}

		if !os.IsNotExist(statErr) {
			return fmt.Errorf("stat model file %q: %w", filePath, statErr)
		}

		err := downloadModelFile(ctx, client, cfg.modelName, fileName, filePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadModelFile(
	ctx context.Context,
	client *http.Client,
	modelName string,
	fileName string,
	filePath string,
) error {
	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", modelName, fileName)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("create model download request: %w", err)
	}

	request.Header.Set("User-Agent", modelUserAgent)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download model file %q: %w", fileName, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w for %q: %s", errUnexpectedDownloadState, fileName, response.Status)
	}

	return writeResponseAtomically(response.Body, filePath)
}

func writeResponseAtomically(body io.Reader, filePath string) error {
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp model file: %w", err)
	}

	tempPath := tempFile.Name()

	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	_, err = io.Copy(tempFile, body)
	if err != nil {
		closeErr := tempFile.Close()
		if closeErr != nil {
			return fmt.Errorf("copy model file and close temp file: %w", errors.Join(err, closeErr))
		}

		return fmt.Errorf("copy model file: %w", err)
	}

	err = tempFile.Close()
	if err != nil {
		return fmt.Errorf("close temp model file: %w", err)
	}

	err = os.Rename(tempPath, filePath)
	if err != nil {
		return fmt.Errorf("rename temp model file: %w", err)
	}

	removeTemp = false

	return nil
}

func loadModel2VecConfig(configPath string) (model2VecConfigFile, error) {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return model2VecConfigFile{}, fmt.Errorf("read model config: %w", err)
	}

	var cfg model2VecConfigFile

	err = json.Unmarshal(configBytes, &cfg)
	if err != nil {
		return model2VecConfigFile{}, fmt.Errorf("parse model config: %w", err)
	}

	return cfg, nil
}

func (model *model2Vec) encode(text string) []float32 {
	tokenIDs := model.tokenizer.tokenIDs(text)
	vector := make([]float32, model.dim)
	rowCount := 0

	for _, tokenID := range tokenIDs {
		// model2vec drops unknown tokens before pooling so the [UNK] vector does
		// not dilute the mean for titles with out-of-vocabulary words.
		if tokenID == model.tokenizer.unkTokenID {
			continue
		}

		row := model.matrix.row(tokenID)
		if row == nil {
			continue
		}

		for valueIdx, value := range row {
			vector[valueIdx] += value
		}

		rowCount++
	}

	if rowCount == 0 {
		return vector
	}

	scale := float32(1) / float32(rowCount)
	for valueIdx := range vector {
		vector[valueIdx] *= scale
	}

	if model.normalize {
		l2Normalize(vector)
	}

	return vector
}

func l2Normalize(vector []float32) {
	var normSquared float64

	for _, value := range vector {
		floatValue := float64(value)
		normSquared += floatValue * floatValue
	}

	if normSquared == 0 {
		return
	}

	scale := float32(1 / math.Sqrt(normSquared))
	for valueIdx := range vector {
		vector[valueIdx] *= scale
	}
}
