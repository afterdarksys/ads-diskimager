package recovery

import (
	"fmt"
	"io"
	"time"
)

// RecoveryStrategy defines how to approach bad sector recovery
type RecoveryStrategy string

const (
	StrategyForward  RecoveryStrategy = "forward"  // Read sequentially, skip bad sectors
	StrategyBackward RecoveryStrategy = "backward" // Read backwards from end
	StrategyRandom   RecoveryStrategy = "random"   // Random access pattern
	StrategyAdaptive RecoveryStrategy = "adaptive" // Dynamic strategy based on errors
)

// ErrorInfo tracks information about read errors
type ErrorInfo struct {
	Offset    int64
	Size      int
	Timestamp time.Time
	Retries   int
	Error     error
}

// ErrorMap tracks all encountered errors
type ErrorMap struct {
	Errors []ErrorInfo
}

// Add adds an error to the map
func (em *ErrorMap) Add(offset int64, size int, err error) {
	em.Errors = append(em.Errors, ErrorInfo{
		Offset:    offset,
		Size:      size,
		Timestamp: time.Now(),
		Retries:   0,
		Error:     err,
	})
}

// Count returns the number of errors
func (em *ErrorMap) Count() int {
	return len(em.Errors)
}

// AdaptiveRecovery implements intelligent bad sector recovery
type AdaptiveRecovery struct {
	Strategy       RecoveryStrategy
	MaxRetries     int
	InitialBlock   int
	MinimumBlock   int // Minimum block size to try (typically 512 bytes)
	ErrorMap       *ErrorMap
	SkipBadAreas   bool // Skip bad areas initially, return later
	BadAreasBypass int  // How many blocks to skip around bad areas
}

// NewAdaptiveRecovery creates a new adaptive recovery handler
func NewAdaptiveRecovery(strategy RecoveryStrategy, maxRetries int) *AdaptiveRecovery {
	return &AdaptiveRecovery{
		Strategy:       strategy,
		MaxRetries:     maxRetries,
		InitialBlock:   64 * 1024,     // 64KB
		MinimumBlock:   512,            // 512 bytes (sector size)
		ErrorMap:       &ErrorMap{},
		SkipBadAreas:   true,
		BadAreasBypass: 8, // Skip 8 blocks around errors
	}
}

// RecoverBlock attempts to recover data from a problematic block
func (ar *AdaptiveRecovery) RecoverBlock(
	source io.ReadSeeker,
	dest io.Writer,
	offset int64,
	size int,
	originalErr error,
) (recovered int, err error) {
	// Seek to the problem offset
	if _, err := source.Seek(offset, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek error: %w", err)
	}

	// Try progressively smaller block sizes
	blockSizes := ar.getBlockSizes(size)
	recovered = 0

	for recovered < size {
		readSize := size - recovered
		success := false

		for _, blockSize := range blockSizes {
			if readSize > blockSize {
				readSize = blockSize
			}

			// Seek to current position
			currentOffset := offset + int64(recovered)
			if _, err := source.Seek(currentOffset, io.SeekStart); err != nil {
				continue
			}

			// Attempt read with current block size
			buf := make([]byte, readSize)
			nr, readErr := source.Read(buf)

			if nr > 0 {
				// Write successfully read data
				if nw, wErr := dest.Write(buf[0:nr]); wErr == nil {
					recovered += nw
					success = true
					break
				}
			}

			if readErr == io.EOF {
				return recovered, io.EOF
			}

			// Try smaller block size
		}

		// If all block sizes failed
		if !success {
			// Record error
			ar.ErrorMap.Add(offset+int64(recovered), ar.MinimumBlock, originalErr)

			// Zero-fill the bad sector
			zeroBuf := make([]byte, ar.MinimumBlock)
			if nw, wErr := dest.Write(zeroBuf); wErr != nil {
				return recovered, wErr
			} else {
				recovered += nw
			}
		}
	}

	return recovered, nil
}

// getBlockSizes returns a list of block sizes to try for recovery
func (ar *AdaptiveRecovery) getBlockSizes(maxSize int) []int {
	sizes := []int{}

	// Start from large blocks and go smaller
	for size := ar.InitialBlock; size >= ar.MinimumBlock; size /= 2 {
		if size <= maxSize {
			sizes = append(sizes, size)
		}
	}

	// Always include the minimum block size
	if len(sizes) == 0 || sizes[len(sizes)-1] != ar.MinimumBlock {
		sizes = append(sizes, ar.MinimumBlock)
	}

	return sizes
}

// ShouldSkipArea determines if an area should be skipped on first pass
func (ar *AdaptiveRecovery) ShouldSkipArea(offset int64) bool {
	if !ar.SkipBadAreas {
		return false
	}

	// Check if offset is near a known error
	for _, errInfo := range ar.ErrorMap.Errors {
		distance := offset - errInfo.Offset
		if distance >= 0 && distance < int64(ar.BadAreasBypass*ar.InitialBlock) {
			return true
		}
	}

	return false
}

// GetErrorCount returns the total error count
func (ar *AdaptiveRecovery) GetErrorCount() int {
	return ar.ErrorMap.Count()
}

// GetErrorMap returns the error map for visualization
func (ar *AdaptiveRecovery) GetErrorMap() *ErrorMap {
	return ar.ErrorMap
}

// RecoveryStats provides statistics about the recovery process
type RecoveryStats struct {
	TotalBytes      int64
	RecoveredBytes  int64
	BadSectors      int
	RetryAttempts   int
	Strategy        RecoveryStrategy
	ElapsedTime     time.Duration
	AverageSpeed    int64 // bytes per second
	WorstErrorZone  int64 // offset with most errors
}

// CalculateStats calculates recovery statistics
func (ar *AdaptiveRecovery) CalculateStats(totalBytes, recoveredBytes int64, elapsed time.Duration) RecoveryStats {
	stats := RecoveryStats{
		TotalBytes:     totalBytes,
		RecoveredBytes: recoveredBytes,
		BadSectors:     ar.ErrorMap.Count(),
		Strategy:       ar.Strategy,
		ElapsedTime:    elapsed,
	}

	if elapsed.Seconds() > 0 {
		stats.AverageSpeed = int64(float64(recoveredBytes) / elapsed.Seconds())
	}

	return stats
}
