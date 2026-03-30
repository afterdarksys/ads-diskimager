package e01

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"io"
	"sync"

	"github.com/afterdarksys/diskimager/imager"
)

// ParallelWriter implements parallel compression for E01 format
type ParallelWriter struct {
	out           io.WriteCloser
	metadata      imager.Metadata
	chunkCount    int
	offsetList    []int64
	currentOffset int64

	// Parallel compression
	numWorkers    int
	jobQueue      chan *compressionJob
	resultQueue   chan *compressionResult
	writerDone    chan struct{}
	compressorWG  sync.WaitGroup
	writerWG      sync.WaitGroup
	zlibPool      sync.Pool
	err           error
	errMu         sync.Mutex
}

type compressionJob struct {
	chunkID int
	data    []byte
}

type compressionResult struct {
	chunkID        int
	offset         int64
	compressedData []byte
	checksum       uint32
	err            error
}

// NewParallelWriter creates a new parallel E01 writer
func NewParallelWriter(out io.WriteCloser, numWorkers int, meta imager.Metadata) (*ParallelWriter, error) {
	if numWorkers <= 0 {
		numWorkers = 8 // Default to 8 workers
	}

	w := &ParallelWriter{
		out:         out,
		metadata:    meta,
		numWorkers:  numWorkers,
		jobQueue:    make(chan *compressionJob, numWorkers*2),
		resultQueue: make(chan *compressionResult, numWorkers*2),
		writerDone:  make(chan struct{}),
	}

	// Initialize zlib writer pool for reuse
	w.zlibPool = sync.Pool{
		New: func() interface{} {
			return zlib.NewWriter(nil)
		},
	}

	// Write header
	if err := w.writeHeader(); err != nil {
		out.Close()
		return nil, err
	}

	// Start compression workers
	w.compressorWG.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go w.compressionWorker()
	}

	// Start result writer
	w.writerWG.Add(1)
	go w.resultWriter()

	return w, nil
}

// writeHeader writes the EWF magic signature and initial header section
func (w *ParallelWriter) writeHeader() error {
	// Write Magic
	if _, err := w.out.Write([]byte(magicHeader)); err != nil {
		return err
	}
	w.currentOffset += 8

	// Write simplified header with metadata
	headerText := fmt.Sprintf("Case:%s\nEv:%s\nEx:%s\nDesc:%s\n",
		w.metadata.CaseNumber,
		w.metadata.EvidenceNum,
		w.metadata.Examiner,
		w.metadata.Description,
	)

	// Write length of header
	hLen := uint32(len(headerText))
	if err := binary.Write(w.out, binary.LittleEndian, hLen); err != nil {
		return err
	}
	w.currentOffset += 4

	// Write header
	if _, err := w.out.Write([]byte(headerText)); err != nil {
		return err
	}
	w.currentOffset += int64(len(headerText))

	return nil
}

// compressionWorker processes compression jobs
func (w *ParallelWriter) compressionWorker() {
	defer w.compressorWG.Done()

	for job := range w.jobQueue {
		result := &compressionResult{
			chunkID: job.chunkID,
		}

		// Calculate Adler32 checksum for uncompressed chunk
		result.checksum = adler32.Checksum(job.data)

		// Get zlib writer from pool
		var buf bytes.Buffer
		zw := w.zlibPool.Get().(*zlib.Writer)
		zw.Reset(&buf)

		// Compress chunk
		_, err := zw.Write(job.data)
		if err != nil {
			result.err = err
			w.resultQueue <- result
			continue
		}
		zw.Close()

		result.compressedData = buf.Bytes()

		// Return writer to pool
		w.zlibPool.Put(zw)

		w.resultQueue <- result
	}
}

// resultWriter writes compressed chunks in order
func (w *ParallelWriter) resultWriter() {
	defer w.writerWG.Done()
	defer close(w.writerDone)

	pendingResults := make(map[int]*compressionResult)
	nextChunkID := 0

	for result := range w.resultQueue {
		if result.err != nil {
			w.setError(result.err)
			return
		}

		// Store result
		pendingResults[result.chunkID] = result

		// Write all consecutive results
		for {
			res, exists := pendingResults[nextChunkID]
			if !exists {
				break
			}

			if err := w.writeChunk(res); err != nil {
				w.setError(err)
				return
			}

			delete(pendingResults, nextChunkID)
			nextChunkID++
		}
	}
}

// writeChunk writes a single compressed chunk
func (w *ParallelWriter) writeChunk(result *compressionResult) error {
	compressedSize := uint32(len(result.compressedData))
	flaggedSize := compressedSize | 0x80000000 // Set compression flag

	// Record offset
	result.offset = w.currentOffset
	w.offsetList = append(w.offsetList, result.offset)

	// Write flagged size
	if err := binary.Write(w.out, binary.LittleEndian, flaggedSize); err != nil {
		return err
	}
	w.currentOffset += 4

	// Write compressed data
	if _, err := w.out.Write(result.compressedData); err != nil {
		return err
	}
	w.currentOffset += int64(compressedSize)

	// Write Adler32 checksum
	if err := binary.Write(w.out, binary.LittleEndian, result.checksum); err != nil {
		return err
	}
	w.currentOffset += 4

	w.chunkCount++
	return nil
}

// Write compresses and writes data
func (w *ParallelWriter) Write(p []byte) (n int, err error) {
	if w.getError() != nil {
		return 0, w.getError()
	}

	totalWritten := 0

	// Chunk the data
	for len(p) > 0 {
		writeLen := len(p)
		if writeLen > chunkSize {
			writeLen = chunkSize
		}

		chunk := make([]byte, writeLen)
		copy(chunk, p[:writeLen])
		p = p[writeLen:]

		// Submit compression job
		job := &compressionJob{
			chunkID: w.chunkCount + len(w.jobQueue),
			data:    chunk,
		}

		select {
		case w.jobQueue <- job:
			totalWritten += writeLen
		case <-w.writerDone:
			return totalWritten, w.getError()
		}
	}

	return totalWritten, nil
}

// Close finalizes the E01 file
func (w *ParallelWriter) Close() error {
	// Close job queue to signal workers
	close(w.jobQueue)

	// Wait for all compression workers
	w.compressorWG.Wait()

	// Close result queue
	close(w.resultQueue)

	// Wait for writer
	w.writerWG.Wait()

	// Check for errors
	if err := w.getError(); err != nil {
		w.out.Close()
		return err
	}

	// Write table section
	tableMagic := "table2"
	if _, err := w.out.Write([]byte(tableMagic)); err != nil {
		return err
	}

	// Write number of chunks
	if err := binary.Write(w.out, binary.LittleEndian, uint32(w.chunkCount)); err != nil {
		return err
	}

	// Write offsets
	for _, offset := range w.offsetList {
		if offset > 0xFFFFFFFF {
			return fmt.Errorf("file size exceeds 4GB limit for EWF1 format")
		}
		if err := binary.Write(w.out, binary.LittleEndian, uint32(offset)); err != nil {
			return err
		}
	}

	// Write done section
	doneMagic := "done"
	if _, err := w.out.Write([]byte(doneMagic)); err != nil {
		return err
	}

	return w.out.Close()
}

// setError stores the first error encountered
func (w *ParallelWriter) setError(err error) {
	w.errMu.Lock()
	if w.err == nil {
		w.err = err
	}
	w.errMu.Unlock()
}

// getError retrieves the stored error
func (w *ParallelWriter) getError() error {
	w.errMu.Lock()
	defer w.errMu.Unlock()
	return w.err
}
