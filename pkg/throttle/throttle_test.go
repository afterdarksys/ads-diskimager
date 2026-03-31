package throttle

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestThrottledReader(t *testing.T) {
	// Create 10KB test data
	data := make([]byte, 10*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Throttle to 5KB/sec
	limit := int64(5 * 1024)
	reader := NewReader(bytes.NewReader(data), limit)

	start := time.Now()
	buf := make([]byte, len(data))
	n, err := io.ReadFull(reader, buf)

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to read %d bytes, got %d", len(data), n)
	}

	// With burst = limit, first `limit` bytes read immediately, rest throttled
	// Expected: (totalBytes - burst) / rate = (10KB - 5KB) / 5KB/sec = 1 second
	expectedDuration := time.Duration(float64(len(data)-int(limit))/float64(limit)) * time.Second
	tolerance := 500 * time.Millisecond

	if elapsed < expectedDuration-tolerance {
		t.Errorf("Read too fast: expected ~%v, got %v", expectedDuration, elapsed)
	}

	t.Logf("Read %d bytes in %v (expected ~%v)", len(data), elapsed, expectedDuration)

	// Verify data integrity
	if !bytes.Equal(data, buf) {
		t.Error("Data corruption detected")
	}
}

func TestThrottledWriter(t *testing.T) {
	data := make([]byte, 5*1024) // 5KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	output := &bytes.Buffer{}
	limit := int64(2 * 1024) // 2KB/sec
	writer := NewWriter(output, limit)

	start := time.Now()
	n, err := writer.Write(data)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Write error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, got %d", len(data), n)
	}

	// Writer waits before writing (not after), so full throttling applies
	// Expected: totalBytes / rate = 5KB / 2KB/sec = 2.5 seconds
	expectedDuration := time.Duration(float64(len(data))/float64(limit)) * time.Second
	tolerance := 500 * time.Millisecond

	if elapsed < expectedDuration-tolerance {
		t.Errorf("Write too fast: expected ~%v, got %v", expectedDuration, elapsed)
	}

	t.Logf("Wrote %d bytes in %v (expected ~%v)", len(data), elapsed, expectedDuration)
}

func TestUnlimitedBandwidth(t *testing.T) {
	data := make([]byte, 10*1024)
	reader := NewReader(bytes.NewReader(data), 0) // 0 = unlimited

	start := time.Now()
	buf := make([]byte, len(data))
	_, err := io.ReadFull(reader, buf)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	// Should be very fast (< 100ms for 10KB in memory)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Unlimited read too slow: %v", elapsed)
	}
}

func TestSetLimit(t *testing.T) {
	data := make([]byte, 10*1024)
	reader := NewReader(bytes.NewReader(data), 1024) // 1KB/sec initially

	// Change to unlimited
	reader.SetLimit(0)

	start := time.Now()
	buf := make([]byte, len(data))
	_, err := io.ReadFull(reader, buf)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	// Should be fast now
	if elapsed > 100*time.Millisecond {
		t.Errorf("Read after SetLimit(0) too slow: %v", elapsed)
	}
}

func TestMeasuredReader(t *testing.T) {
	data := make([]byte, 5*1024) // 5KB
	mr := NewMeasuredReader(bytes.NewReader(data), 0) // Unlimited for quick test

	buf := make([]byte, len(data))
	n, err := io.ReadFull(mr, buf)

	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected %d bytes, got %d", len(data), n)
	}

	if mr.BytesRead() != int64(len(data)) {
		t.Errorf("BytesRead() = %d, want %d", mr.BytesRead(), len(data))
	}

	avgSpeed := mr.AverageSpeed()
	if avgSpeed <= 0 {
		t.Error("Average speed should be > 0")
	}

	t.Logf("Average speed: %.2f bytes/sec", avgSpeed)
}

func TestMeasuredReaderWithThrottle(t *testing.T) {
	data := make([]byte, 8*1024) // 8KB (more data for accurate measurement)
	limit := int64(2 * 1024)     // 2KB/sec
	mr := NewMeasuredReader(bytes.NewReader(data), limit)

	buf := make([]byte, len(data))
	start := time.Now()
	_, err := io.ReadFull(mr, buf)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	avgSpeed := mr.AverageSpeed()

	// Expected time: (totalBytes - burst) / rate = (8KB - 2KB) / 2KB/sec = 3 seconds
	// Average speed over full duration: 8KB / 3s = 2.67 KB/sec
	// This is higher than the limit due to initial burst

	// Just verify the read was throttled (took at least 2 seconds)
	minExpectedDuration := 2 * time.Second
	if elapsed < minExpectedDuration {
		t.Errorf("Read too fast: expected at least %v, got %v", minExpectedDuration, elapsed)
	}

	t.Logf("Elapsed: %v, Avg Speed: %.2f bytes/sec, Limit: %d bytes/sec",
		elapsed, avgSpeed, limit)
}

func TestCurrentLimit(t *testing.T) {
	data := make([]byte, 1024)
	limit := int64(5 * 1024)
	mr := NewMeasuredReader(bytes.NewReader(data), limit)

	currentLimit := mr.CurrentLimit()
	if currentLimit != limit {
		t.Errorf("CurrentLimit() = %d, want %d", currentLimit, limit)
	}

	// Test unlimited
	mr.SetLimit(0)
	currentLimit = mr.CurrentLimit()
	if currentLimit != 0 {
		t.Errorf("CurrentLimit() after SetLimit(0) = %d, want 0", currentLimit)
	}
}

func BenchmarkThrottledRead(b *testing.B) {
	data := make([]byte, 64*1024) // 64KB
	limit := int64(10 * 1024 * 1024) // 10MB/sec

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		reader := NewReader(bytes.NewReader(data), limit)
		io.Copy(io.Discard, reader)
	}
}

func BenchmarkUnthrottledRead(b *testing.B) {
	data := make([]byte, 64*1024)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		reader := NewReader(bytes.NewReader(data), 0) // Unlimited
		io.Copy(io.Discard, reader)
	}
}
