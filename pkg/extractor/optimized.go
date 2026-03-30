package extractor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// OptimizedExtractor provides high-performance file extraction
type OptimizedExtractor struct {
	*Extractor
	chunkSize int
}

// NewOptimizedExtractor creates an optimized extractor with larger read chunks
func NewOptimizedExtractor(imagePath, fsType string, chunkSize int) (*OptimizedExtractor, error) {
	base, err := NewExtractor(imagePath, fsType)
	if err != nil {
		return nil, err
	}

	if chunkSize <= 0 {
		chunkSize = 4 * 1024 * 1024 // Default 4MB chunks
	}

	return &OptimizedExtractor{
		Extractor: base,
		chunkSize: chunkSize,
	}, nil
}

// ExtractFileOptimized extracts a file using optimized chunk-based reading
func (e *OptimizedExtractor) ExtractFileOptimized(file FileInfo, output io.Writer) error {
	if e.useTSK && file.Inode > 0 {
		return e.extractFileWithTSKStreaming(file, output)
	}
	// extractFileWithMount returns ([]byte, error), we need to write to output
	data, err := e.extractFileWithMount(file)
	if err != nil {
		return err
	}
	_, err = output.Write(data)
	return err
}

// extractFileWithTSKStreaming uses icat with streaming and large buffers
func (e *OptimizedExtractor) extractFileWithTSKStreaming(file FileInfo, output io.Writer) error {
	cmd := exec.Command("icat", e.imagePath, fmt.Sprintf("%d", file.Inode))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("icat failed to start: %w", err)
	}

	// Use buffered reader with large buffer
	reader := bufio.NewReaderSize(stdout, e.chunkSize)

	// Stream data in large chunks
	_, copyErr := io.Copy(output, reader)

	// Capture any stderr output for errors
	var stderrBuf bytes.Buffer
	io.Copy(&stderrBuf, stderr)

	if err := cmd.Wait(); err != nil {
		if stderrBuf.Len() > 0 {
			return fmt.Errorf("icat process failed: %w, stderr: %s", err, stderrBuf.String())
		}
		return fmt.Errorf("icat process failed: %w", err)
	}

	if copyErr != nil {
		return fmt.Errorf("failed to copy data: %w", copyErr)
	}

	return nil
}

// FastScan performs optimized filesystem scanning with larger buffers
func (e *OptimizedExtractor) FastScan(patterns []string) ([]FileInfo, error) {
	if !e.useTSK {
		return e.FindFiles(patterns, true)
	}

	// Use optimized fls invocation with batching
	args := []string{"-r", "-p", "-m", "/", e.imagePath}

	cmd := exec.Command("fls", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("fls failed to start: %w", err)
	}

	var results []FileInfo
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, e.chunkSize), e.chunkSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse and filter in a single pass
		if fileInfo := e.parseFLSLine(line, patterns); fileInfo != nil {
			results = append(results, *fileInfo)
		}
	}

	if err := scanner.Err(); err != nil {
		cmd.Wait()
		return nil, fmt.Errorf("error reading fls output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("fls process failed: %w", err)
	}

	return results, nil
}

// parseFLSLine parses a single fls output line and checks patterns
func (e *OptimizedExtractor) parseFLSLine(line string, patterns []string) *FileInfo {
	// Quick rejection for directories
	if len(line) < 3 {
		return nil
	}

	// Parse basic structure (optimized for minimal allocations)
	// Format: "r/r 123: /path/to/file"
	colonIdx := -1
	for i := 0; i < len(line); i++ {
		if line[i] == ':' {
			colonIdx = i
			break
		}
	}

	if colonIdx == -1 || colonIdx >= len(line)-1 {
		return nil
	}

	filePath := line[colonIdx+2:] // Skip ": "
	if filePath == "" || filePath == "/" {
		return nil
	}

	// Quick pattern match without allocating
	if !matchesAnyPattern(filePath, patterns) {
		return nil
	}

	// Parse inode from first part
	var inode uint64
	fields := line[:colonIdx]
	fmt.Sscanf(fields, "%*s %d", &inode)

	return &FileInfo{
		Path:  filePath,
		Inode: inode,
	}
}

// BatchExtract extracts multiple files efficiently
func (e *OptimizedExtractor) BatchExtract(files []FileInfo, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Process files in parallel with worker pool
	numWorkers := 4 // Optimal for I/O bound operations
	jobs := make(chan FileInfo, len(files))
	errors := make(chan error, len(files))
	done := make(chan struct{})

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go func() {
			for file := range jobs {
				outPath := outputDir + file.Path
				if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
					errors <- fmt.Errorf("failed to create dir for %s: %w", file.Path, err)
					continue
				}

				outFile, err := os.Create(outPath)
				if err != nil {
					errors <- fmt.Errorf("failed to create %s: %w", file.Path, err)
					continue
				}

				err = e.ExtractFileOptimized(file, outFile)
				outFile.Close()

				if err != nil {
					errors <- fmt.Errorf("failed to extract %s: %w", file.Path, err)
				}
			}
		}()
	}

	// Submit jobs
	go func() {
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
		done <- struct{}{}
	}()

	// Wait for completion
	<-done

	close(errors)

	// Collect errors
	var firstError error
	for err := range errors {
		if firstError == nil {
			firstError = err
		}
	}

	return firstError
}
