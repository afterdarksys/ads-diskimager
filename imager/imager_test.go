package imager

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"testing"
)

func TestImage_Success(t *testing.T) {
	// Create random data
	data := make([]byte, 1024*1024) // 1MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Calculate expected hash
	hasher := sha256.New()
	hasher.Write(data)
	expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Source reader
	source := bytes.NewReader(data)
	// Destination buffer
	var dest bytes.Buffer

	cfg := Config{
		Source:      source,
		Destination: &dest,
		BlockSize:   1024,
		HashAlgo:    "sha256",
	}

	res, err := Image(cfg)
	if err != nil {
		t.Fatalf("Image() failed: %v", err)
	}

	if res.BytesCopied != int64(len(data)) {
		t.Errorf("Expected %d bytes copied, got %d", len(data), res.BytesCopied)
	}

	if res.Hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, res.Hash)
	}

	if !bytes.Equal(dest.Bytes(), data) {
		t.Errorf("Destination data does not match source data")
	}
}

// FaultyReader simulates a read error after some bytes
type FaultyReader struct {
	Data []byte
	Pos  int
	FailAt int
}

func (r *FaultyReader) Read(p []byte) (n int, err error) {
	if r.Pos >= len(r.Data) {
		return 0, io.EOF
	}
	if r.Pos >= r.FailAt {
		return 0, fmt.Errorf("simulated read error")
	}
	n = copy(p, r.Data[r.Pos:])
	r.Pos += n
	return n, nil
}

func TestImage_ReadError(t *testing.T) {
	data := []byte("hello world")
	source := &FaultyReader{Data: data, FailAt: 5}
	var dest bytes.Buffer

	cfg := Config{
		Source:      source,
		Destination: &dest,
		BlockSize:   1,
		HashAlgo:    "md5",
	}

	_, err := Image(cfg)
	if err == nil {
		t.Error("Expected error from faulty reader, got nil")
	}
}
