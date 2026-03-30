package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"gocloud.dev/blob"
)

// BufferedCloudWriter provides write-behind caching for cloud storage
type BufferedCloudWriter struct {
	bucket       *blob.Bucket
	objectKey    string
	bufferSize   int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	partNumber   int64
	partSize     int64
	buffer       *bytes.Buffer
	mu           sync.Mutex
	uploadQueue  chan *uploadPart
	err          atomic.Value
	closed       bool
	multipartID  string
	completedParts []completedPart
}

type uploadPart struct {
	partNumber int64
	data       []byte
}

type completedPart struct {
	PartNumber int64
	ETag       string
}

// NewBufferedCloudWriter creates a buffered writer for cloud storage
// bufferSize should be at least 5MB for multipart uploads (AWS requirement)
func NewBufferedCloudWriter(bucket *blob.Bucket, objectKey string, bufferSize int) (*BufferedCloudWriter, error) {
	if bufferSize < 5*1024*1024 {
		bufferSize = 10 * 1024 * 1024 // Default 10MB
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &BufferedCloudWriter{
		bucket:      bucket,
		objectKey:   objectKey,
		bufferSize:  bufferSize,
		ctx:         ctx,
		cancel:      cancel,
		buffer:      bytes.NewBuffer(make([]byte, 0, bufferSize)),
		uploadQueue: make(chan *uploadPart, 4),
	}

	// Start upload workers
	numWorkers := 4
	w.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go w.uploadWorker()
	}

	return w, nil
}

// uploadWorker processes upload jobs
func (w *BufferedCloudWriter) uploadWorker() {
	defer w.wg.Done()

	for part := range w.uploadQueue {
		if err := w.uploadPart(part); err != nil {
			w.setError(err)
			return
		}
	}
}

// uploadPart uploads a single part
func (w *BufferedCloudWriter) uploadPart(part *uploadPart) error {
	// For simplicity, we'll use single-object writes with part number in key
	// In production, you'd want to use proper multipart upload APIs
	partKey := fmt.Sprintf("%s.part.%d", w.objectKey, part.partNumber)

	writer, err := w.bucket.NewWriter(w.ctx, partKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create part writer: %w", err)
	}

	_, err = writer.Write(part.data)
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to write part: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close part writer: %w", err)
	}

	atomic.AddInt64(&w.partSize, int64(len(part.data)))

	return nil
}

// Write implements io.Writer with buffering
func (w *BufferedCloudWriter) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, fmt.Errorf("writer is closed")
	}

	if storedErr := w.getError(); storedErr != nil {
		return 0, storedErr
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	totalWritten := 0
	for len(p) > 0 {
		// Calculate space available in buffer
		available := w.bufferSize - w.buffer.Len()

		if available == 0 {
			// Buffer is full, flush it
			if err := w.flushLocked(); err != nil {
				return totalWritten, err
			}
			available = w.bufferSize
		}

		// Write what we can to the buffer
		writeSize := len(p)
		if writeSize > available {
			writeSize = available
		}

		n, err := w.buffer.Write(p[:writeSize])
		totalWritten += n
		p = p[writeSize:]

		if err != nil {
			return totalWritten, err
		}
	}

	return totalWritten, nil
}

// flushLocked flushes the buffer (must be called with lock held)
func (w *BufferedCloudWriter) flushLocked() error {
	if w.buffer.Len() == 0 {
		return nil
	}

	// Create a copy of the buffer data
	data := make([]byte, w.buffer.Len())
	copy(data, w.buffer.Bytes())
	w.buffer.Reset()

	partNum := atomic.AddInt64(&w.partNumber, 1)

	part := &uploadPart{
		partNumber: partNum,
		data:       data,
	}

	select {
	case w.uploadQueue <- part:
		return nil
	case <-w.ctx.Done():
		return w.ctx.Err()
	}
}

// Flush flushes any buffered data
func (w *BufferedCloudWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flushLocked()
}

// Close flushes remaining data and finalizes the upload
func (w *BufferedCloudWriter) Close() error {
	if w.closed {
		return nil
	}

	// Flush any remaining data
	if err := w.Flush(); err != nil {
		w.cancel()
		return err
	}

	// Close upload queue
	close(w.uploadQueue)

	// Wait for all uploads to complete
	w.wg.Wait()

	w.closed = true
	w.cancel()

	// Check for errors
	if err := w.getError(); err != nil {
		return err
	}

	// Combine parts into final object
	return w.combineParts()
}

// combineParts combines all uploaded parts into the final object
func (w *BufferedCloudWriter) combineParts() error {
	// If only one part, just rename it
	numParts := atomic.LoadInt64(&w.partNumber)
	if numParts == 1 {
		return w.renamePart(1)
	}

	// For multiple parts, we need to concatenate them
	writer, err := w.bucket.NewWriter(w.ctx, w.objectKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create final writer: %w", err)
	}

	// Read and write each part
	for i := int64(1); i <= numParts; i++ {
		partKey := fmt.Sprintf("%s.part.%d", w.objectKey, i)

		reader, err := w.bucket.NewReader(w.ctx, partKey, nil)
		if err != nil {
			writer.Close()
			return fmt.Errorf("failed to read part %d: %w", i, err)
		}

		if _, err := io.Copy(writer, reader); err != nil {
			reader.Close()
			writer.Close()
			return fmt.Errorf("failed to copy part %d: %w", i, err)
		}
		reader.Close()

		// Delete the part after copying
		w.bucket.Delete(w.ctx, partKey)
	}

	return writer.Close()
}

// renamePart renames a single part to the final object
func (w *BufferedCloudWriter) renamePart(partNumber int64) error {
	partKey := fmt.Sprintf("%s.part.%d", w.objectKey, partNumber)

	// Read the part
	reader, err := w.bucket.NewReader(w.ctx, partKey, nil)
	if err != nil {
		return fmt.Errorf("failed to read part: %w", err)
	}

	// Write to final location
	writer, err := w.bucket.NewWriter(w.ctx, w.objectKey, nil)
	if err != nil {
		reader.Close()
		return fmt.Errorf("failed to create final writer: %w", err)
	}

	if _, err := io.Copy(writer, reader); err != nil {
		reader.Close()
		writer.Close()
		return fmt.Errorf("failed to copy data: %w", err)
	}
	reader.Close()

	if err := writer.Close(); err != nil {
		return err
	}

	// Delete the temporary part
	return w.bucket.Delete(w.ctx, partKey)
}

// setError stores the first error
func (w *BufferedCloudWriter) setError(err error) {
	w.err.CompareAndSwap(nil, err)
	w.cancel()
}

// getError retrieves the stored error
func (w *BufferedCloudWriter) getError() error {
	if v := w.err.Load(); v != nil {
		return v.(error)
	}
	return nil
}

// GetBytesWritten returns total bytes written
func (w *BufferedCloudWriter) GetBytesWritten() int64 {
	return atomic.LoadInt64(&w.partSize)
}
