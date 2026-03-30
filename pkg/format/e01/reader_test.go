package e01

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"testing"
)

func TestE01Reader(t *testing.T) {
	// Create a simple E01-like structure for testing
	var buf bytes.Buffer

	// Write magic header
	buf.Write([]byte(magicHeader))

	// Write header length (0 for simplicity)
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	// Write a compressed chunk
	testData := []byte("Hello, World! This is test data for E01 format.")

	// Compress the data
	var compressedBuf bytes.Buffer
	zw := zlib.NewWriter(&compressedBuf)
	zw.Write(testData)
	zw.Close()
	compressed := compressedBuf.Bytes()

	// Write flagged size (compressed with high bit set)
	flaggedSize := uint32(len(compressed)) | 0x80000000
	binary.Write(&buf, binary.LittleEndian, flaggedSize)

	// Write compressed data
	buf.Write(compressed)

	// Write Adler32 checksum (if present in newer format)
	// For this test, we'll skip it to test backward compatibility

	// Write table section marker to signal end
	binary.Write(&buf, binary.LittleEndian, uint32(0x6c626174)) // "tabl"

	// Create reader
	reader, err := NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to create E01 reader: %v", err)
	}

	// Read and verify data
	output := make([]byte, len(testData)+10) // Extra space
	n, err := reader.Read(output)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Expected to read %d bytes, read %d", len(testData), n)
	}

	if !bytes.Equal(output[:n], testData) {
		t.Errorf("Data mismatch.\nExpected: %s\nGot: %s", testData, output[:n])
	}
}
