package raw

import (
	"io"
)

// Writer is a simple wrapper for raw file output
type Writer struct {
	out io.WriteCloser
}

// NewWriter creates a new raw output writer
func NewWriter(out io.WriteCloser) (*Writer, error) {
	return &Writer{out: out}, nil
}

// Write implements io.Writer
func (w *Writer) Write(p []byte) (n int, err error) {
	return w.out.Write(p)
}

// Close implements io.Closer
func (w *Writer) Close() error {
	return w.out.Close()
}
