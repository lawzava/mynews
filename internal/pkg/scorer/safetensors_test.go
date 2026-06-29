package scorer //nolint:testpackage // tests exercise unexported safetensors helpers.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSafetensorsEmbeddingsSelectsSingleF32Matrix(t *testing.T) {
	t.Parallel()

	modelPath := filepath.Join(t.TempDir(), "model.safetensors")
	values := []float32{1, 2, 3, 4, 5, 6}

	writeTestSafetensors(t, modelPath, "embeddings", []int{3, 2}, values)

	matrix, tensorName, err := loadSafetensorsEmbeddings(modelPath)
	if err != nil {
		t.Fatalf("load safetensors embeddings: %v", err)
	}

	if tensorName != "embeddings" {
		t.Fatalf("tensor name = %q, want embeddings", tensorName)
	}

	if matrix.vocabSize != 3 || matrix.dim != 2 {
		t.Fatalf("shape = [%d,%d], want [3,2]", matrix.vocabSize, matrix.dim)
	}

	for valueIdx, wantValue := range values {
		if matrix.values[valueIdx] != wantValue {
			t.Fatalf("value %d = %v, want %v", valueIdx, matrix.values[valueIdx], wantValue)
		}
	}
}

func TestCosineSimilarityHandlesFloat32Vectors(t *testing.T) {
	t.Parallel()

	got := cosineSimilarity([]float32{1, 1}, []float32{1, 0})
	want := 1 / math.Sqrt2

	if math.Abs(got-want) > floatTolerance {
		t.Fatalf("cosine = %v, want %v", got, want)
	}
}

func writeTestSafetensors(
	t *testing.T,
	modelPath string,
	tensorName string,
	shape []int,
	values []float32,
) {
	t.Helper()

	dataSize := len(values) * bytesPerFloat32
	header := fmt.Appendf(nil,
		`{%q:{"dtype":"F32","shape":[%d,%d],"data_offsets":[0,%d]}}`,
		tensorName,
		shape[0],
		shape[1],
		dataSize,
	)
	buffer := bytes.NewBuffer(nil)

	err := binary.Write(buffer, binary.LittleEndian, uint64(len(header)))
	if err != nil {
		t.Fatalf("write header length: %v", err)
	}

	buffer.Write(header)

	for _, value := range values {
		writeErr := binary.Write(buffer, binary.LittleEndian, value)
		if writeErr != nil {
			t.Fatalf("write value: %v", writeErr)
		}
	}

	if len(shape) != 2 {
		t.Fatalf("test fixture shape = %v, want two dimensions", shape)
	}

	writeErr := os.WriteFile(modelPath, buffer.Bytes(), defaultFilePerm)
	if writeErr != nil {
		t.Fatalf("write model file: %v", writeErr)
	}
}
