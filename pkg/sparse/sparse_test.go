package sparse

import (
	"bytes"
	"io"
	"testing"
)

func TestIsZeroBlock(t *testing.T) {
	tests := []struct {
		name     string
		block    []byte
		expected bool
	}{
		{"All zeros", make([]byte, 4096), true},
		{"One non-zero", append(make([]byte, 4095), 1), false},
		{"All ones", bytes.Repeat([]byte{1}, 4096), false},
		{"Empty", []byte{}, true},
		{"Single zero", []byte{0}, true},
		{"Single non-zero", []byte{1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsZeroBlock(tt.block)
			if result != tt.expected {
				t.Errorf("IsZeroBlock() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSparseWriter(t *testing.T) {
	// Create test data: 3 blocks (zero, data, zero)
	blockSize := 4096
	testData := make([]byte, blockSize*3)

	// Block 0: all zeros
	// Block 1: data
	for i := 0; i < blockSize; i++ {
		testData[blockSize+i] = byte(i % 256)
	}
	// Block 2: all zeros

	var output bytes.Buffer
	sw := NewWriter(&output, blockSize, true)

	n, err := sw.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	if err := sw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	stats := sw.Stats()
	if stats.TotalBlocks != 3 {
		t.Errorf("TotalBlocks = %d, want 3", stats.TotalBlocks)
	}
	if stats.ZeroBlocks != 2 {
		t.Errorf("ZeroBlocks = %d, want 2", stats.ZeroBlocks)
	}
	if stats.DataBlocks != 1 {
		t.Errorf("DataBlocks = %d, want 1", stats.DataBlocks)
	}
	if stats.BytesSaved != int64(blockSize*2) {
		t.Errorf("BytesSaved = %d, want %d", stats.BytesSaved, blockSize*2)
	}

	expectedRatio := 2.0 / 3.0 * 100 // 66.67%
	if stats.SparseRatio < expectedRatio-1 || stats.SparseRatio > expectedRatio+1 {
		t.Errorf("SparseRatio = %.2f%%, want ~%.2f%%", stats.SparseRatio, expectedRatio)
	}

	t.Logf("Stats: Total=%d, Zero=%d, Data=%d, Saved=%d bytes, Ratio=%.2f%%",
		stats.TotalBlocks, stats.ZeroBlocks, stats.DataBlocks, stats.BytesSaved, stats.SparseRatio)
}

func TestSparseWriterNoSkip(t *testing.T) {
	blockSize := 4096
	testData := make([]byte, blockSize*2) // All zeros

	var output bytes.Buffer
	sw := NewWriter(&output, blockSize, false) // skipZeros = false

	n, err := sw.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Should write all data even though it's zeros
	if output.Len() != len(testData) {
		t.Errorf("Output size = %d, want %d (no skipping)", output.Len(), len(testData))
	}
}

func TestSparseReader(t *testing.T) {
	// Create sparse data
	blockSize := 4096
	data := make([]byte, blockSize*4)

	// Block 0: zeros
	// Block 1: data
	for i := 0; i < blockSize; i++ {
		data[blockSize+i] = byte(i % 256)
	}
	// Block 2: zeros
	// Block 3: data
	for i := 0; i < blockSize; i++ {
		data[blockSize*3+i] = byte(255 - (i % 256))
	}

	reader := bytes.NewReader(data)
	sr := NewSparseReader(reader, blockSize)

	// Read all data
	output := make([]byte, len(data))
	n, err := io.ReadFull(sr, output)
	if err != nil {
		t.Fatalf("ReadFull failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to read %d bytes, read %d", len(data), n)
	}

	stats := sr.Stats()
	if stats.TotalBlocks != 4 {
		t.Errorf("TotalBlocks = %d, want 4", stats.TotalBlocks)
	}
	if stats.ZeroBlocks != 2 {
		t.Errorf("ZeroBlocks = %d, want 2", stats.ZeroBlocks)
	}
	if stats.DataBlocks != 2 {
		t.Errorf("DataBlocks = %d, want 2", stats.DataBlocks)
	}

	t.Logf("SparseReader stats: Total=%d, Zero=%d, Data=%d, Ratio=%.2f%%",
		stats.TotalBlocks, stats.ZeroBlocks, stats.DataBlocks, stats.SparseRatio)

	// Verify data integrity
	if !bytes.Equal(data, output) {
		t.Error("Output data doesn't match input")
	}
}

func TestDetectSparseRatio(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		expectedRatio float64
		tolerance     float64
	}{
		{
			name:          "All zeros",
			data:          make([]byte, 10*4096),
			expectedRatio: 100.0,
			tolerance:     1.0,
		},
		{
			name:          "No zeros",
			data:          bytes.Repeat([]byte{1, 2, 3, 4}, 10*1024),
			expectedRatio: 0.0,
			tolerance:     1.0,
		},
		{
			name:          "Half zeros",
			data:          append(make([]byte, 5*4096), bytes.Repeat([]byte{1}, 5*4096)...),
			expectedRatio: 50.0,
			tolerance:     5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			ratio, err := DetectSparseRatio(reader, int64(len(tt.data)), 4096)
			if err != nil {
				t.Fatalf("DetectSparseRatio failed: %v", err)
			}

			if ratio < tt.expectedRatio-tt.tolerance || ratio > tt.expectedRatio+tt.tolerance {
				t.Errorf("DetectSparseRatio() = %.2f%%, want %.2f%% (±%.2f%%)",
					ratio, tt.expectedRatio, tt.tolerance)
			}

			t.Logf("Detected sparse ratio: %.2f%%", ratio)
		})
	}
}

func TestCopyWithSparseDetection(t *testing.T) {
	// Create sparse test data
	blockSize := 4096
	sourceData := make([]byte, blockSize*5)

	// Block 0: zeros
	// Block 1: data
	for i := 0; i < blockSize; i++ {
		sourceData[blockSize+i] = byte(i % 256)
	}
	// Blocks 2-4: zeros

	source := bytes.NewReader(sourceData)
	var dest bytes.Buffer

	written, stats, err := CopyWithSparseDetection(&dest, source, blockSize)
	if err != nil {
		t.Fatalf("CopyWithSparseDetection failed: %v", err)
	}

	if written != int64(len(sourceData)) {
		t.Errorf("Written = %d, want %d", written, len(sourceData))
	}

	expectedZeroBlocks := int64(4)
	if stats.ZeroBlocks != expectedZeroBlocks {
		t.Errorf("ZeroBlocks = %d, want %d", stats.ZeroBlocks, expectedZeroBlocks)
	}

	expectedRatio := 80.0 // 4 out of 5 blocks
	if stats.SparseRatio < expectedRatio-5 || stats.SparseRatio > expectedRatio+5 {
		t.Errorf("SparseRatio = %.2f%%, want ~%.2f%%", stats.SparseRatio, expectedRatio)
	}

	t.Logf("Copy stats: Written=%d, Zero blocks=%d, Data blocks=%d, Saved=%d bytes, Ratio=%.2f%%",
		written, stats.ZeroBlocks, stats.DataBlocks, stats.BytesSaved, stats.SparseRatio)
}

func TestCompareBlocks(t *testing.T) {
	block1 := make([]byte, 4096)
	block2 := make([]byte, 4096)
	block3 := bytes.Repeat([]byte{1}, 4096)

	if !CompareBlocks(block1, block2) {
		t.Error("Identical zero blocks should be equal")
	}

	if CompareBlocks(block1, block3) {
		t.Error("Different blocks should not be equal")
	}
}

func TestZeroBlock(t *testing.T) {
	sizes := []int{512, 4096, 65536}

	for _, size := range sizes {
		block := ZeroBlock(size)
		if len(block) != size {
			t.Errorf("ZeroBlock(%d) returned %d bytes", size, len(block))
		}
		if !IsZeroBlock(block) {
			t.Errorf("ZeroBlock(%d) didn't return all zeros", size)
		}
	}
}

func BenchmarkIsZeroBlock(b *testing.B) {
	block := make([]byte, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsZeroBlock(block)
	}
}

func BenchmarkIsZeroBlockWithData(b *testing.B) {
	block := bytes.Repeat([]byte{1, 2, 3, 4}, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsZeroBlock(block)
	}
}

func BenchmarkSparseWriter(b *testing.B) {
	data := make([]byte, 4096*10) // 50% sparse
	for i := 0; i < 4096*5; i++ {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		sw := NewWriter(&buf, 4096, true)
		sw.Write(data)
		sw.Flush()
	}
}
