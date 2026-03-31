package compression

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

// Algorithm represents a compression algorithm
type Algorithm string

const (
	AlgorithmNone Algorithm = "none"
	AlgorithmGzip Algorithm = "gzip"
	AlgorithmZstd Algorithm = "zstd"
)

// Level represents compression level
type Level int

const (
	LevelFastest Level = 1
	LevelDefault Level = 5
	LevelBest    Level = 9
)

// Reader wraps an io.Reader with decompression
type Reader struct {
	underlying io.Reader
	reader     io.ReadCloser
	algorithm  Algorithm
}

// NewReader creates a decompression reader
func NewReader(r io.Reader, algo Algorithm) (*Reader, error) {
	cr := &Reader{
		underlying: r,
		algorithm:  algo,
	}

	var err error
	switch algo {
	case AlgorithmNone:
		cr.reader = io.NopCloser(r)
	case AlgorithmGzip:
		cr.reader, err = gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
	case AlgorithmZstd:
		decoder, err := zstd.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("zstd reader: %w", err)
		}
		cr.reader = decoder.IOReadCloser()
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}

	return cr, nil
}

// Read implements io.Reader
func (cr *Reader) Read(p []byte) (int, error) {
	return cr.reader.Read(p)
}

// Close closes the reader
func (cr *Reader) Close() error {
	return cr.reader.Close()
}

// Writer wraps an io.Writer with compression
type Writer struct {
	underlying io.Writer
	writer     io.WriteCloser
	algorithm  Algorithm
	level      Level
}

// NewWriter creates a compression writer
func NewWriter(w io.Writer, algo Algorithm, level Level) (*Writer, error) {
	cw := &Writer{
		underlying: w,
		algorithm:  algo,
		level:      level,
	}

	var err error
	switch algo {
	case AlgorithmNone:
		cw.writer = nopWriteCloser{w}
	case AlgorithmGzip:
		gzipLevel := gzip.DefaultCompression
		switch level {
		case LevelFastest:
			gzipLevel = gzip.BestSpeed
		case LevelBest:
			gzipLevel = gzip.BestCompression
		}
		cw.writer, err = gzip.NewWriterLevel(w, gzipLevel)
		if err != nil {
			return nil, fmt.Errorf("gzip writer: %w", err)
		}
	case AlgorithmZstd:
		zstdLevel := zstd.SpeedDefault
		switch level {
		case LevelFastest:
			zstdLevel = zstd.SpeedFastest
		case LevelBest:
			zstdLevel = zstd.SpeedBestCompression
		}
		encoder, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstdLevel))
		if err != nil {
			return nil, fmt.Errorf("zstd writer: %w", err)
		}
		cw.writer = encoder
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}

	return cw, nil
}

// Write implements io.Writer
func (cw *Writer) Write(p []byte) (int, error) {
	return cw.writer.Write(p)
}

// Close closes the writer and flushes any buffered data
func (cw *Writer) Close() error {
	return cw.writer.Close()
}

// nopWriteCloser wraps an io.Writer with a no-op Close method
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// Stats tracks compression statistics
type Stats struct {
	UncompressedBytes int64
	CompressedBytes   int64
	Ratio             float64 // compression ratio (compressed/uncompressed)
}

// MeasuredWriter wraps a compression writer with statistics tracking
type MeasuredWriter struct {
	*Writer
	uncompressedBytes int64
	compressedBytes   int64
}

// NewMeasuredWriter creates a writer that tracks compression statistics
func NewMeasuredWriter(w io.Writer, algo Algorithm, level Level) (*MeasuredWriter, error) {
	writer, err := NewWriter(w, algo, level)
	if err != nil {
		return nil, err
	}

	return &MeasuredWriter{
		Writer: writer,
	}, nil
}

// Write implements io.Writer and tracks bytes
func (mw *MeasuredWriter) Write(p []byte) (int, error) {
	n, err := mw.Writer.Write(p)
	mw.uncompressedBytes += int64(n)
	// Note: We can't easily track compressed bytes without wrapping the underlying writer
	// This would require a more complex implementation
	return n, err
}

// Stats returns compression statistics
func (mw *MeasuredWriter) Stats() Stats {
	ratio := 0.0
	if mw.uncompressedBytes > 0 {
		ratio = float64(mw.compressedBytes) / float64(mw.uncompressedBytes)
	}

	return Stats{
		UncompressedBytes: mw.uncompressedBytes,
		CompressedBytes:   mw.compressedBytes,
		Ratio:             ratio,
	}
}

// DetectBestAlgorithm samples data to determine the best compression algorithm
func DetectBestAlgorithm(sample []byte) Algorithm {
	// Check if data is highly compressible
	// Simple heuristic: count zero bytes
	zeros := 0
	for _, b := range sample {
		if b == 0 {
			zeros++
		}
	}

	// If >30% zeros, data is likely compressible
	if float64(zeros)/float64(len(sample)) > 0.3 {
		return AlgorithmZstd // Zstd is faster than gzip
	}

	// Check for randomness (entropy)
	// High entropy data (encrypted, compressed) won't benefit from compression
	unique := make(map[byte]bool)
	for _, b := range sample {
		unique[b] = true
	}

	// If >250 unique bytes in sample, data is likely random/compressed
	if len(unique) > 250 {
		return AlgorithmNone
	}

	// Default to zstd for moderate compression benefit
	return AlgorithmZstd
}

// SuggestLevel suggests a compression level based on CPU availability
func SuggestLevel(cpuCores int) Level {
	if cpuCores <= 2 {
		return LevelFastest
	} else if cpuCores >= 8 {
		return LevelBest
	}
	return LevelDefault
}
