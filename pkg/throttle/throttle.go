package throttle

import (
	"context"
	"io"
	"time"

	"golang.org/x/time/rate"
)

// Reader wraps an io.Reader with bandwidth throttling
type Reader struct {
	r       io.Reader
	limiter *rate.Limiter
	ctx     context.Context
}

// NewReader creates a throttled reader with the specified bytes per second limit
// Set bytesPerSecond to 0 for unlimited bandwidth
func NewReader(r io.Reader, bytesPerSecond int64) *Reader {
	if bytesPerSecond <= 0 {
		// No throttling
		return &Reader{
			r:       r,
			limiter: nil,
			ctx:     context.Background(),
		}
	}

	// Set burst equal to the bandwidth limit (1 second worth of data)
	// This ensures proper throttling
	burst := int(bytesPerSecond)

	return &Reader{
		r:       r,
		limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), burst),
		ctx:     context.Background(),
	}
}

// NewReaderWithContext creates a throttled reader with context support
func NewReaderWithContext(ctx context.Context, r io.Reader, bytesPerSecond int64) *Reader {
	if bytesPerSecond <= 0 {
		return &Reader{
			r:       r,
			limiter: nil,
			ctx:     ctx,
		}
	}

	burst := int(bytesPerSecond)

	return &Reader{
		r:       r,
		limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), burst),
		ctx:     ctx,
	}
}

// Read implements io.Reader with bandwidth throttling
func (tr *Reader) Read(p []byte) (int, error) {
	if tr.limiter == nil {
		// No throttling
		return tr.r.Read(p)
	}

	// Read in chunks respecting the burst size
	burst := tr.limiter.Burst()
	readSize := len(p)
	if readSize > burst {
		readSize = burst
	}

	// Wait for bandwidth allowance before reading
	if err := tr.limiter.WaitN(tr.ctx, readSize); err != nil {
		return 0, err
	}

	// Read from underlying reader
	n, err := tr.r.Read(p[:readSize])

	return n, err
}

// Close closes the reader if it implements io.Closer
func (tr *Reader) Close() error {
	if closer, ok := tr.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Writer wraps an io.Writer with bandwidth throttling
type Writer struct {
	w       io.Writer
	limiter *rate.Limiter
	ctx     context.Context
}

// NewWriter creates a throttled writer with the specified bytes per second limit
func NewWriter(w io.Writer, bytesPerSecond int64) *Writer {
	if bytesPerSecond <= 0 {
		return &Writer{
			w:       w,
			limiter: nil,
			ctx:     context.Background(),
		}
	}

	burst := int(bytesPerSecond)

	return &Writer{
		w:       w,
		limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), burst),
		ctx:     context.Background(),
	}
}

// NewWriterWithContext creates a throttled writer with context support
func NewWriterWithContext(ctx context.Context, w io.Writer, bytesPerSecond int64) *Writer {
	if bytesPerSecond <= 0 {
		return &Writer{
			w:       w,
			limiter: nil,
			ctx:     ctx,
		}
	}

	burst := int(bytesPerSecond)

	return &Writer{
		w:       w,
		limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), burst),
		ctx:     ctx,
	}
}

// Write implements io.Writer with bandwidth throttling
func (tw *Writer) Write(p []byte) (int, error) {
	if tw.limiter == nil {
		// No throttling
		return tw.w.Write(p)
	}

	// Write in chunks respecting the burst size
	totalWritten := 0
	burst := tw.limiter.Burst()

	for totalWritten < len(p) {
		writeSize := len(p) - totalWritten
		if writeSize > burst {
			writeSize = burst
		}

		// Wait for bandwidth allowance before writing
		if err := tw.limiter.WaitN(tw.ctx, writeSize); err != nil {
			if totalWritten > 0 {
				return totalWritten, err
			}
			return 0, err
		}

		// Write chunk
		n, err := tw.w.Write(p[totalWritten : totalWritten+writeSize])
		totalWritten += n

		if err != nil {
			return totalWritten, err
		}
	}

	return totalWritten, nil
}

// Close closes the writer if it implements io.Closer
func (tw *Writer) Close() error {
	if closer, ok := tw.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// ReadWriter wraps an io.ReadWriter with bandwidth throttling
type ReadWriter struct {
	*Reader
	*Writer
}

// NewReadWriter creates a throttled ReadWriter
func NewReadWriter(rw io.ReadWriter, bytesPerSecond int64) *ReadWriter {
	return &ReadWriter{
		Reader: NewReader(rw, bytesPerSecond),
		Writer: NewWriter(rw, bytesPerSecond),
	}
}

// SetLimit updates the bandwidth limit (bytes per second)
// Set to 0 for unlimited bandwidth
func (tr *Reader) SetLimit(bytesPerSecond int64) {
	if bytesPerSecond <= 0 {
		tr.limiter = nil
		return
	}

	burst := int(bytesPerSecond)

	if tr.limiter == nil {
		tr.limiter = rate.NewLimiter(rate.Limit(bytesPerSecond), burst)
	} else {
		tr.limiter.SetLimit(rate.Limit(bytesPerSecond))
		tr.limiter.SetBurst(burst)
	}
}

// SetLimit updates the bandwidth limit for writer
func (tw *Writer) SetLimit(bytesPerSecond int64) {
	if bytesPerSecond <= 0 {
		tw.limiter = nil
		return
	}

	burst := int(bytesPerSecond)

	if tw.limiter == nil {
		tw.limiter = rate.NewLimiter(rate.Limit(bytesPerSecond), burst)
	} else {
		tw.limiter.SetLimit(rate.Limit(bytesPerSecond))
		tw.limiter.SetBurst(burst)
	}
}

// MeasuredReader tracks bandwidth usage
type MeasuredReader struct {
	r         io.Reader
	throttle  *Reader
	bytesRead int64
	startTime time.Time
}

// NewMeasuredReader creates a reader that tracks bandwidth usage
func NewMeasuredReader(r io.Reader, bytesPerSecond int64) *MeasuredReader {
	return &MeasuredReader{
		r:         r,
		throttle:  NewReader(r, bytesPerSecond),
		startTime: time.Now(),
	}
}

// Read implements io.Reader with measurement
func (mr *MeasuredReader) Read(p []byte) (int, error) {
	n, err := mr.throttle.Read(p)
	mr.bytesRead += int64(n)
	return n, err
}

// BytesRead returns total bytes read
func (mr *MeasuredReader) BytesRead() int64 {
	return mr.bytesRead
}

// AverageSpeed returns average speed in bytes per second
func (mr *MeasuredReader) AverageSpeed() float64 {
	elapsed := time.Since(mr.startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(mr.bytesRead) / elapsed
}

// CurrentLimit returns the current bandwidth limit
func (mr *MeasuredReader) CurrentLimit() int64 {
	if mr.throttle.limiter == nil {
		return 0 // Unlimited
	}
	return int64(mr.throttle.limiter.Limit())
}

// SetLimit updates the bandwidth limit
func (mr *MeasuredReader) SetLimit(bytesPerSecond int64) {
	mr.throttle.SetLimit(bytesPerSecond)
}
