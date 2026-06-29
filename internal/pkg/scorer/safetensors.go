package scorer

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
)

const (
	safetensorsHeaderLengthBytes = 8
	bytesPerFloat32              = 4
	defaultFilePerm              = 0o644
)

var (
	errShortSafetensorsFile       = errors.New("safetensors file is shorter than header length")
	errSafetensorsHeaderTooLarge  = errors.New("safetensors header length exceeds file size")
	errMultipleEmbeddingTensors   = errors.New("multiple 2-D F32 tensors found")
	errEmbeddingTensorMissing     = errors.New("no 2-D F32 tensor found")
	errTensorOffsetsOutOfRange    = errors.New("tensor data offsets are out of range")
	errTensorByteLengthMismatched = errors.New("tensor byte length does not match shape byte length")
)

type embeddingMatrix struct {
	values    []float32
	vocabSize int
	dim       int
}

type safetensorsTensor struct {
	DType string `json:"dtype"`
	Shape []int  `json:"shape"`

	DataOffsets [2]int64 `json:"data_offsets"` //nolint:tagliatelle // safetensors schema uses snake_case.
}

func loadSafetensorsEmbeddings(modelPath string) (*embeddingMatrix, string, error) {
	fileBytes, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, "", fmt.Errorf("read safetensors file: %w", err)
	}

	header, tensorData, err := splitSafetensors(fileBytes)
	if err != nil {
		return nil, "", err
	}

	tensors, err := parseSafetensorsHeader(header)
	if err != nil {
		return nil, "", err
	}

	tensorName, tensor, err := selectEmbeddingTensor(tensors)
	if err != nil {
		return nil, "", err
	}

	matrix, err := decodeEmbeddingTensor(tensorData, tensor)
	if err != nil {
		return nil, "", fmt.Errorf("decode tensor %q: %w", tensorName, err)
	}

	return matrix, tensorName, nil
}

//nolint:gocritic // named returns conflict with nonamedreturns.
func splitSafetensors(fileBytes []byte) ([]byte, []byte, error) {
	if len(fileBytes) < safetensorsHeaderLengthBytes {
		return nil, nil, errShortSafetensorsFile
	}

	headerLength := binary.LittleEndian.Uint64(fileBytes[:safetensorsHeaderLengthBytes])
	//nolint:gosec // len is non-negative and this is only a bounds comparison.
	if headerLength > uint64(len(fileBytes)-safetensorsHeaderLengthBytes) {
		return nil, nil, errSafetensorsHeaderTooLarge
	}

	//nolint:gosec // guarded against overflow and file bounds above.
	headerEnd := safetensorsHeaderLengthBytes + int(headerLength)

	return fileBytes[safetensorsHeaderLengthBytes:headerEnd], fileBytes[headerEnd:], nil
}

func parseSafetensorsHeader(header []byte) (map[string]safetensorsTensor, error) {
	rawHeader := make(map[string]json.RawMessage)

	err := json.Unmarshal(header, &rawHeader)
	if err != nil {
		return nil, fmt.Errorf("parse safetensors header: %w", err)
	}

	tensors := make(map[string]safetensorsTensor, len(rawHeader))
	for tensorName, rawTensor := range rawHeader {
		if tensorName == "__metadata__" {
			continue
		}

		var tensor safetensorsTensor

		err = json.Unmarshal(rawTensor, &tensor)
		if err != nil {
			return nil, fmt.Errorf("parse tensor %q metadata: %w", tensorName, err)
		}

		tensors[tensorName] = tensor
	}

	return tensors, nil
}

func selectEmbeddingTensor(tensors map[string]safetensorsTensor) (string, safetensorsTensor, error) {
	selectedName := ""

	var selectedTensor safetensorsTensor

	for tensorName, tensor := range tensors {
		if tensor.DType != "F32" || len(tensor.Shape) != 2 {
			continue
		}

		if selectedName != "" {
			return "", safetensorsTensor{}, errMultipleEmbeddingTensors
		}

		selectedName = tensorName
		selectedTensor = tensor
	}

	if selectedName == "" {
		return "", safetensorsTensor{}, errEmbeddingTensorMissing
	}

	return selectedName, selectedTensor, nil
}

func decodeEmbeddingTensor(tensorData []byte, tensor safetensorsTensor) (*embeddingMatrix, error) {
	startOffset := tensor.DataOffsets[0]
	endOffset := tensor.DataOffsets[1]

	if startOffset < 0 || endOffset < startOffset || endOffset > int64(len(tensorData)) {
		return nil, errTensorOffsetsOutOfRange
	}

	rawValues := tensorData[startOffset:endOffset]
	valueCount := tensor.Shape[0] * tensor.Shape[1]
	expectedBytes := valueCount * bytesPerFloat32

	if len(rawValues) != expectedBytes {
		return nil, fmt.Errorf("%w: %d != %d", errTensorByteLengthMismatched, len(rawValues), expectedBytes)
	}

	values := make([]float32, valueCount)
	for valueIdx := range values {
		startByte := valueIdx * bytesPerFloat32
		bits := binary.LittleEndian.Uint32(rawValues[startByte : startByte+bytesPerFloat32])
		values[valueIdx] = math.Float32frombits(bits)
	}

	return &embeddingMatrix{
		values:    values,
		vocabSize: tensor.Shape[0],
		dim:       tensor.Shape[1],
	}, nil
}

func (matrix *embeddingMatrix) row(tokenID int) []float32 {
	if tokenID < 0 || tokenID >= matrix.vocabSize {
		return nil
	}

	startOffset := tokenID * matrix.dim
	endOffset := startOffset + matrix.dim

	return matrix.values[startOffset:endOffset]
}
