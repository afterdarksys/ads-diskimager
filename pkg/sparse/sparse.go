package sparse

import (
	"bytes"
	"io"
)

// Reader wraps an io.Reader and detects zero blocks
type Reader struct {
	r         io.Reader
	blockSize int
	stats     Stats
}

// Stats tracks sparse file statistics
type Stats struct {
	TotalBlocks  int64
	ZeroBlocks   int64
	DataBlocks   int64
	BytesSaved   int64
	SparseRatio  float64 // percentage of zero blocks
}

// NewReader creates a sparse-aware reader with the specified block size
func NewReader(r io.Reader, blockSize int) *Reader {
	if blockSize <= 0 {
		blockSize = 4096 // Default 4KB blocks
	}

	return &Reader{
		r:         r,
		blockSize: blockSize,
	}
}

// Read implements io.Reader
func (sr *Reader) Read(p []byte) (int, error) {
	return sr.r.Read(p)
}

// Stats returns sparse file statistics
func (sr *Reader) Stats() Stats {
	if sr.stats.TotalBlocks > 0 {
		sr.stats.SparseRatio = float64(sr.stats.ZeroBlocks) / float64(sr.stats.TotalBlocks) * 100
	}
	return sr.stats
}

// Writer wraps an io.Writer and skips zero blocks
type Writer struct {
	w           io.Writer
	seeker      io.Seeker
	blockSize   int
	buffer      []byte
	position    int64
	stats       Stats
	skipZeros   bool
}

// NewWriter creates a sparse-aware writer
func NewWriter(w io.Writer, blockSize int, skipZeros bool) *Writer {
	if blockSize <= 0 {
		blockSize = 4096
	}

	sw := &Writer{
		w:         w,
		blockSize: blockSize,
		buffer:    make([]byte, 0, blockSize),
		skipZeros: skipZeros,
	}

	// Check if writer supports seeking (needed for sparse writes)
	if seeker, ok := w.(io.Seeker); ok {
		sw.seeker = seeker
	}

	return sw
}

// Write implements io.Writer with sparse block detection
func (sw *Writer) Write(p []byte) (int, error) {
	if !sw.skipZeros {
		// No sparse optimization, write directly
		n, err := sw.w.Write(p)
		sw.position += int64(n)
		return n, err
	}

	totalWritten := 0

	for len(p) > 0 {
		// Add to buffer
		space := sw.blockSize - len(sw.buffer)
		if space > len(p) {
			space = len(p)
		}

		sw.buffer = append(sw.buffer, p[:space]...)
		p = p[space:]
		totalWritten += space

		// Process complete blocks
		if len(sw.buffer) >= sw.blockSize {
			if err := sw.writeBlock(sw.buffer); err != nil {
				return totalWritten, err
			}
			sw.buffer = sw.buffer[:0]
		}
	}

	return totalWritten, nil
}

// writeBlock writes a single block, skipping if all zeros
func (sw *Writer) writeBlock(block []byte) error {
	sw.stats.TotalBlocks++

	if isZeroBlock(block) {
		sw.stats.ZeroBlocks++
		sw.stats.BytesSaved += int64(len(block))

		// If we have a seeker, skip ahead (create hole)
		if sw.seeker != nil {
			_, err := sw.seeker.Seek(int64(len(block)), io.SeekCurrent)
			if err != nil {
				// If seek fails, write zeros anyway
				_, writeErr := sw.w.Write(block)
				sw.position += int64(len(block))
				return writeErr
			}
			sw.position += int64(len(block))
			return nil
		}

		// No seeker, must write zeros
		_, err := sw.w.Write(block)
		sw.position += int64(len(block))
		return err
	}

	// Write data block
	sw.stats.DataBlocks++
	_, err := sw.w.Write(block)
	sw.position += int64(len(block))
	return err
}

// Flush writes any remaining buffered data
func (sw *Writer) Flush() error {
	if len(sw.buffer) > 0 {
		if err := sw.writeBlock(sw.buffer); err != nil {
			return err
		}
		sw.buffer = sw.buffer[:0]
	}
	return nil
}

// Close flushes and closes the writer
func (sw *Writer) Close() error {
	if err := sw.Flush(); err != nil {
		return err
	}
	if closer, ok := sw.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Stats returns sparse write statistics
func (sw *Writer) Stats() Stats {
	if sw.stats.TotalBlocks > 0 {
		sw.stats.SparseRatio = float64(sw.stats.ZeroBlocks) / float64(sw.stats.TotalBlocks) * 100
	}
	return sw.stats
}

// isZeroBlock checks if a block contains all zeros
func isZeroBlock(block []byte) bool {
	for _, b := range block {
		if b != 0 {
			return false
		}
	}
	return true
}

// IsZeroBlock is exported for external use
func IsZeroBlock(block []byte) bool {
	return isZeroBlock(block)
}

// DetectSparseRatio samples data to estimate how sparse it is
func DetectSparseRatio(r io.Reader, sampleSize int64, blockSize int) (float64, error) {
	if sampleSize <= 0 {
		sampleSize = 1024 * 1024 // 1MB sample
	}

	if blockSize <= 0 {
		blockSize = 4096
	}

	buffer := make([]byte, blockSize)
	var totalBlocks, zeroBlocks int64

	for totalBytes := int64(0); totalBytes < sampleSize; {
		n, err := r.Read(buffer)
		if n > 0 {
			totalBlocks++
			if isZeroBlock(buffer[:n]) {
				zeroBlocks++
			}
			totalBytes += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	if totalBlocks == 0 {
		return 0, nil
	}

	return float64(zeroBlocks) / float64(totalBlocks) * 100, nil
}

// CopyWithSparseDetection copies from src to dst with sparse block detection
func CopyWithSparseDetection(dst io.Writer, src io.Reader, blockSize int) (written int64, stats Stats, err error) {
	sw := NewWriter(dst, blockSize, true)
	written, err = io.Copy(sw, src)

	if flushErr := sw.Flush(); flushErr != nil && err == nil {
		err = flushErr
	}

	stats = sw.Stats()
	return written, stats, err
}

// ZeroBlock returns a block of zeros of the specified size
func ZeroBlock(size int) []byte {
	return make([]byte, size)
}

// CompareBlocks compares two blocks and returns true if they are identical
func CompareBlocks(a, b []byte) bool {
	return bytes.Equal(a, b)
}

// SparseReader reads and reports on sparse characteristics
type SparseReader struct {
	r         io.Reader
	blockSize int
	analyzed  bool
	stats     Stats
}

// NewSparseReader creates a reader that analyzes sparseness
func NewSparseReader(r io.Reader, blockSize int) *SparseReader {
	if blockSize <= 0 {
		blockSize = 4096
	}

	return &SparseReader{
		r:         r,
		blockSize: blockSize,
	}
}

// Read implements io.Reader and tracks statistics
func (sr *SparseReader) Read(p []byte) (int, error) {
	n, err := sr.r.Read(p)

	// Analyze blocks
	if n > 0 {
		for offset := 0; offset < n; offset += sr.blockSize {
			end := offset + sr.blockSize
			if end > n {
				end = n
			}

			sr.stats.TotalBlocks++
			if isZeroBlock(p[offset:end]) {
				sr.stats.ZeroBlocks++
				sr.stats.BytesSaved += int64(end - offset)
			} else {
				sr.stats.DataBlocks++
			}
		}
	}

	return n, err
}

// Stats returns the sparse statistics
func (sr *SparseReader) Stats() Stats {
	if sr.stats.TotalBlocks > 0 {
		sr.stats.SparseRatio = float64(sr.stats.ZeroBlocks) / float64(sr.stats.TotalBlocks) * 100
	}
	return sr.stats
}
