package imager

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ParallelConfig extends Config with parallel I/O settings
type ParallelConfig struct {
	Config
	NumWorkers    int  // Number of parallel hash workers (default: NumCPU)
	RingSize      int  // Number of buffers in ring (default: 16)
	BufferSize    int  // Size of each buffer (default: 4MB)
	EnableParallel bool // Enable parallel processing (default: false for compatibility)
}

// Buffer represents a single buffer in the ring buffer pool
type Buffer struct {
	Data   []byte
	Size   int   // Actual data size in buffer
	Offset int64 // Position in source stream
	Err    error // Error encountered during read
}

// RingBuffer implements a circular buffer pool for parallel I/O
type RingBuffer struct {
	buffers  []*Buffer
	size     int
	readIdx  int
	writeIdx int
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
	closed   bool
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(size, bufferSize int) *RingBuffer {
	rb := &RingBuffer{
		buffers: make([]*Buffer, size),
		size:    size,
	}
	rb.notEmpty = sync.NewCond(&rb.mu)
	rb.notFull = sync.NewCond(&rb.mu)

	// Pre-allocate all buffers
	for i := 0; i < size; i++ {
		rb.buffers[i] = &Buffer{
			Data: make([]byte, bufferSize),
		}
	}

	return rb
}

// Get retrieves the next buffer for reading (consumer)
func (rb *RingBuffer) Get() *Buffer {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for rb.readIdx == rb.writeIdx && !rb.closed {
		rb.notEmpty.Wait()
	}

	if rb.readIdx == rb.writeIdx && rb.closed {
		return nil
	}

	buf := rb.buffers[rb.readIdx]
	rb.readIdx = (rb.readIdx + 1) % rb.size
	rb.notFull.Signal()

	return buf
}

// Put returns a buffer to the pool (producer)
func (rb *RingBuffer) Put(buf *Buffer) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for (rb.writeIdx+1)%rb.size == rb.readIdx && !rb.closed {
		rb.notFull.Wait()
	}

	if rb.closed {
		return
	}

	rb.buffers[rb.writeIdx] = buf
	rb.writeIdx = (rb.writeIdx + 1) % rb.size
	rb.notEmpty.Signal()
}

// Close signals that no more data will be added
func (rb *RingBuffer) Close() {
	rb.mu.Lock()
	rb.closed = true
	rb.notEmpty.Broadcast()
	rb.notFull.Broadcast()
	rb.mu.Unlock()
}

// ParallelImage performs high-performance parallel disk imaging
func ParallelImage(ctx context.Context, cfg ParallelConfig) (*Result, error) {
	// Set defaults
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = runtime.NumCPU()
	}
	if cfg.RingSize <= 0 {
		cfg.RingSize = 16
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 4 * 1024 * 1024 // 4MB
	}
	if cfg.BlockSize <= 0 {
		cfg.BlockSize = 64 * 1024
	}

	// Create result
	res := &Result{
		StartTime: time.Now(),
	}

	// Create ring buffers
	readRing := NewRingBuffer(cfg.RingSize, cfg.BufferSize)
	hashRing := NewRingBuffer(cfg.RingSize, cfg.BufferSize)
	writeRing := NewRingBuffer(cfg.RingSize, cfg.BufferSize)

	var wg sync.WaitGroup
	var readErr, writeErr, hashErr atomic.Value

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer readRing.Close()

		offset := int64(0)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Get a buffer from the pool
			buf := &Buffer{
				Data:   make([]byte, cfg.BufferSize),
				Offset: offset,
			}

			n, err := io.ReadFull(cfg.Source, buf.Data)
			if n > 0 {
				buf.Size = n
				offset += int64(n)
				readRing.Put(buf)
			}

			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return
				}
				buf.Err = err
				readRing.Put(buf)
				readErr.Store(err)
				return
			}
		}
	}()

	// Hash worker pool
	hashers := make([]hash.Hash, cfg.NumWorkers)
	for i := 0; i < cfg.NumWorkers; i++ {
		hashers[i] = createHasher(cfg.HashAlgo)
	}

	wg.Add(cfg.NumWorkers)
	for workerID := 0; workerID < cfg.NumWorkers; workerID++ {
		go func(id int) {
			defer wg.Done()
			hasher := hashers[id]

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				buf := readRing.Get()
				if buf == nil {
					return
				}

				if buf.Err == nil && buf.Size > 0 {
					// Hash the data
					if _, err := hasher.Write(buf.Data[:buf.Size]); err != nil {
						buf.Err = fmt.Errorf("hash error: %w", err)
						hashErr.Store(err)
					}
				}

				hashRing.Put(buf)
			}
		}(workerID)
	}

	// Close hash ring when all hash workers complete
	go func() {
		wg.Wait()
		hashRing.Close()
	}()

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer writeRing.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			buf := hashRing.Get()
			if buf == nil {
				return
			}

			if buf.Err != nil {
				// Handle read error - record bad sector
				res.BadSectors = append(res.BadSectors, BadSector{
					Offset: buf.Offset,
					Size:   cfg.BufferSize - buf.Size,
					Error:  buf.Err.Error(),
				})
				res.Errors = append(res.Errors, buf.Err)
			}

			if buf.Size > 0 {
				nw, err := cfg.Destination.Write(buf.Data[:buf.Size])
				if err != nil {
					writeErr.Store(fmt.Errorf("write error: %w", err))
					return
				}
				atomic.AddInt64(&res.BytesCopied, int64(nw))
			}
		}
	}()

	// Wait for all goroutines
	wg.Wait()

	// Check for errors
	if err := readErr.Load(); err != nil && err != io.EOF {
		return res, fmt.Errorf("read error: %w", err.(error))
	}
	if err := hashErr.Load(); err != nil {
		return res, err.(error)
	}
	if err := writeErr.Load(); err != nil {
		return res, err.(error)
	}

	// Combine hash results from all workers
	finalHasher := createHasher(cfg.HashAlgo)
	for _, h := range hashers {
		finalHasher.Write(h.Sum(nil))
	}

	res.EndTime = time.Now()
	res.Hash = fmt.Sprintf("%x", finalHasher.Sum(nil))

	return res, nil
}

// createHasher creates a hasher based on algorithm name
func createHasher(algo string) hash.Hash {
	switch algo {
	case "md5":
		return md5.New()
	case "sha1":
		return sha1.New()
	case "sha256":
		return sha256.New()
	default:
		return sha256.New()
	}
}
