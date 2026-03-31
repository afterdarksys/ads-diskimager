package recovery

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

// errorReader simulates a reader with bad sectors
type errorReader struct {
	data        []byte
	position    int64
	errorRanges []errorRange
}

type errorRange struct {
	start int64
	end   int64
}

func newErrorReader(data []byte, errors []errorRange) *errorReader {
	return &errorReader{
		data:        data,
		errorRanges: errors,
	}
}

func (er *errorReader) Read(p []byte) (n int, err error) {
	// Check if current position is in error range
	for _, errRange := range er.errorRanges {
		if er.position >= errRange.start && er.position < errRange.end {
			return 0, errors.New("simulated read error")
		}
	}

	// Normal read
	if er.position >= int64(len(er.data)) {
		return 0, io.EOF
	}

	n = copy(p, er.data[er.position:])
	er.position += int64(n)
	return n, nil
}

func (er *errorReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		er.position = offset
	case io.SeekCurrent:
		er.position += offset
	case io.SeekEnd:
		er.position = int64(len(er.data)) + offset
	}

	if er.position < 0 {
		er.position = 0
	}
	if er.position > int64(len(er.data)) {
		er.position = int64(len(er.data))
	}

	return er.position, nil
}

func TestAdaptiveRecovery(t *testing.T) {
	// Create test data: 1MB of data with a bad sector at offset 512KB
	dataSize := 1024 * 1024
	testData := make([]byte, dataSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Simulate bad sector from 512KB to 512KB+512B
	badStart := int64(512 * 1024)
	badEnd := badStart + 512

	reader := newErrorReader(testData, []errorRange{{start: badStart, end: badEnd}})
	output := &bytes.Buffer{}

	ar := NewAdaptiveRecovery(StrategyAdaptive, 3)

	// Try to recover the bad block
	recovered, err := ar.RecoverBlock(reader, output, badStart, int(badEnd-badStart), errors.New("test error"))

	if err != nil && err != io.EOF {
		t.Errorf("RecoverBlock failed: %v", err)
	}

	if recovered != int(badEnd-badStart) {
		t.Errorf("Expected to recover %d bytes, got %d", badEnd-badStart, recovered)
	}

	if ar.GetErrorCount() == 0 {
		t.Error("Expected error to be logged in error map")
	}
}

func TestGetBlockSizes(t *testing.T) {
	ar := NewAdaptiveRecovery(StrategyForward, 3)

	sizes := ar.getBlockSizes(128 * 1024)

	if len(sizes) == 0 {
		t.Error("Expected non-empty block sizes")
	}

	// Verify decreasing order
	for i := 1; i < len(sizes); i++ {
		if sizes[i] > sizes[i-1] {
			t.Error("Block sizes should be in decreasing order")
		}
	}

	// Verify minimum size is included
	if sizes[len(sizes)-1] != ar.MinimumBlock {
		t.Errorf("Expected last size to be %d, got %d", ar.MinimumBlock, sizes[len(sizes)-1])
	}
}

func TestErrorMap(t *testing.T) {
	em := &ErrorMap{}

	// Add some errors
	em.Add(1024, 512, errors.New("error 1"))
	em.Add(2048, 512, errors.New("error 2"))
	em.Add(4096, 512, errors.New("error 3"))

	if em.Count() != 3 {
		t.Errorf("Expected 3 errors, got %d", em.Count())
	}

	// Verify error details
	for i, err := range em.Errors {
		if err.Offset == 0 {
			t.Errorf("Error %d has zero offset", i)
		}
		if err.Size != 512 {
			t.Errorf("Error %d has wrong size: %d", i, err.Size)
		}
	}
}

func TestCalculateStats(t *testing.T) {
	ar := NewAdaptiveRecovery(StrategyAdaptive, 3)

	// Add some errors
	ar.ErrorMap.Add(1024, 512, errors.New("error 1"))
	ar.ErrorMap.Add(2048, 512, errors.New("error 2"))

	totalBytes := int64(1024 * 1024)
	recoveredBytes := int64(1023 * 1024) // 1MB - 1KB lost
	elapsed := 10 * time.Second

	stats := ar.CalculateStats(totalBytes, recoveredBytes, elapsed)

	if stats.TotalBytes != totalBytes {
		t.Errorf("Expected total bytes %d, got %d", totalBytes, stats.TotalBytes)
	}

	if stats.RecoveredBytes != recoveredBytes {
		t.Errorf("Expected recovered bytes %d, got %d", recoveredBytes, stats.RecoveredBytes)
	}

	if stats.BadSectors != 2 {
		t.Errorf("Expected 2 bad sectors, got %d", stats.BadSectors)
	}

	expectedSpeed := recoveredBytes / 10
	if stats.AverageSpeed != expectedSpeed {
		t.Errorf("Expected speed %d bytes/sec, got %d", expectedSpeed, stats.AverageSpeed)
	}
}

func TestShouldSkipArea(t *testing.T) {
	ar := NewAdaptiveRecovery(StrategyAdaptive, 3)
	ar.SkipBadAreas = true

	// Add an error at offset 1024
	ar.ErrorMap.Add(1024, 512, errors.New("test error"))

	// Areas near the error should be skipped
	if !ar.ShouldSkipArea(2048) {
		t.Error("Expected area near error to be skipped")
	}

	// Areas far from errors should not be skipped
	if ar.ShouldSkipArea(1024 * 1024) {
		t.Error("Expected distant area to not be skipped")
	}
}

func BenchmarkRecoverBlock(b *testing.B) {
	dataSize := 64 * 1024
	testData := make([]byte, dataSize)

	// Bad sector in the middle
	badStart := int64(32 * 1024)
	badEnd := badStart + 512

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := newErrorReader(testData, []errorRange{{start: badStart, end: badEnd}})
		output := &bytes.Buffer{}
		ar := NewAdaptiveRecovery(StrategyAdaptive, 3)

		ar.RecoverBlock(reader, output, badStart, int(badEnd-badStart), errors.New("test"))
	}
}
