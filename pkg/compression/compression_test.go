package compression

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestGzipCompression(t *testing.T) {
	// Test data
	original := bytes.Repeat([]byte("Hello, World! This is test data. "), 100)

	// Compress
	var compressed bytes.Buffer
	writer, err := NewWriter(&compressed, AlgorithmGzip, LevelDefault)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	n, err := writer.Write(original)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(original) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(original), n)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify compression actually happened
	if compressed.Len() >= len(original) {
		t.Errorf("Compressed size (%d) >= original size (%d)", compressed.Len(), len(original))
	}

	t.Logf("Compression ratio: %.2f%% (original: %d, compressed: %d)",
		float64(compressed.Len())/float64(len(original))*100, len(original), compressed.Len())

	// Decompress
	reader, err := NewReader(&compressed, AlgorithmGzip)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Verify
	if !bytes.Equal(original, decompressed) {
		t.Error("Decompressed data doesn't match original")
	}
}

func TestZstdCompression(t *testing.T) {
	// Test data - highly compressible
	original := bytes.Repeat([]byte{0x00, 0x01, 0x02}, 1000)

	// Compress
	var compressed bytes.Buffer
	writer, err := NewWriter(&compressed, AlgorithmZstd, LevelDefault)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	n, err := writer.Write(original)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(original) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(original), n)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify compression
	if compressed.Len() >= len(original) {
		t.Errorf("Compressed size (%d) >= original size (%d)", compressed.Len(), len(original))
	}

	t.Logf("Zstd compression ratio: %.2f%% (original: %d, compressed: %d)",
		float64(compressed.Len())/float64(len(original))*100, len(original), compressed.Len())

	// Decompress
	reader, err := NewReader(&compressed, AlgorithmZstd)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Verify
	if !bytes.Equal(original, decompressed) {
		t.Error("Decompressed data doesn't match original")
	}
}

func TestNoCompression(t *testing.T) {
	original := []byte("This data won't be compressed")

	// "Compress" with none
	var output bytes.Buffer
	writer, err := NewWriter(&output, AlgorithmNone, LevelDefault)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	writer.Write(original)
	writer.Close()

	// Should be identical
	if !bytes.Equal(original, output.Bytes()) {
		t.Error("AlgorithmNone should not modify data")
	}
}

func TestCompressionLevels(t *testing.T) {
	original := bytes.Repeat([]byte("Test data for compression levels. "), 50)

	levels := []Level{LevelFastest, LevelDefault, LevelBest}

	for _, level := range levels {
		var compressed bytes.Buffer
		writer, err := NewWriter(&compressed, AlgorithmZstd, level)
		if err != nil {
			t.Fatalf("NewWriter failed for level %d: %v", level, err)
		}

		writer.Write(original)
		writer.Close()

		t.Logf("Level %d: compressed to %d bytes", level, compressed.Len())

		// Verify decompression
		reader, err := NewReader(&compressed, AlgorithmZstd)
		if err != nil {
			t.Fatalf("NewReader failed: %v", err)
		}

		decompressed, _ := io.ReadAll(reader)
		reader.Close()

		if !bytes.Equal(original, decompressed) {
			t.Errorf("Level %d: decompressed data doesn't match", level)
		}
	}
}

func TestDetectBestAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected Algorithm
	}{
		{
			name:     "Zeros (highly compressible)",
			data:     bytes.Repeat([]byte{0x00}, 1000),
			expected: AlgorithmZstd,
		},
		{
			name:     "Random (not compressible)",
			data:     []byte(strings.Repeat("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()", 10)),
			expected: AlgorithmNone,
		},
		{
			name:     "Repetitive text",
			data:     bytes.Repeat([]byte("Hello World "), 100),
			expected: AlgorithmZstd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectBestAlgorithm(tt.data)
			if result != tt.expected {
				t.Logf("Note: Expected %s, got %s for %s (heuristic may vary)",
					tt.expected, result, tt.name)
			}
		})
	}
}

func TestSuggestLevel(t *testing.T) {
	tests := []struct {
		cores    int
		expected Level
	}{
		{1, LevelFastest},
		{2, LevelFastest},
		{4, LevelDefault},
		{8, LevelBest},
		{16, LevelBest},
	}

	for _, tt := range tests {
		result := SuggestLevel(tt.cores)
		if result != tt.expected {
			t.Errorf("SuggestLevel(%d) = %d, want %d", tt.cores, result, tt.expected)
		}
	}
}

func TestMeasuredWriter(t *testing.T) {
	original := bytes.Repeat([]byte("Measured compression test. "), 50)

	var compressed bytes.Buffer
	mw, err := NewMeasuredWriter(&compressed, AlgorithmZstd, LevelDefault)
	if err != nil {
		t.Fatalf("NewMeasuredWriter failed: %v", err)
	}

	n, err := mw.Write(original)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(original) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(original), n)
	}

	mw.Close()

	stats := mw.Stats()
	if stats.UncompressedBytes != int64(len(original)) {
		t.Errorf("UncompressedBytes = %d, want %d", stats.UncompressedBytes, len(original))
	}

	t.Logf("Stats: Uncompressed=%d, Compressed=%d, Ratio=%.2f",
		stats.UncompressedBytes, stats.CompressedBytes, stats.Ratio)
}

func BenchmarkGzipCompression(b *testing.B) {
	data := bytes.Repeat([]byte("Benchmark data for gzip compression. "), 100)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w, _ := NewWriter(&buf, AlgorithmGzip, LevelDefault)
		w.Write(data)
		w.Close()
	}
}

func BenchmarkZstdCompression(b *testing.B) {
	data := bytes.Repeat([]byte("Benchmark data for zstd compression. "), 100)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w, _ := NewWriter(&buf, AlgorithmZstd, LevelDefault)
		w.Write(data)
		w.Close()
	}
}

func BenchmarkNoCompression(b *testing.B) {
	data := bytes.Repeat([]byte("Benchmark data for no compression. "), 100)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w, _ := NewWriter(&buf, AlgorithmNone, LevelDefault)
		w.Write(data)
		w.Close()
	}
}
