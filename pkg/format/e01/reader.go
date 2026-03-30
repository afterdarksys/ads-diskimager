package e01

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"io"
)

// Reader decompresses and reconstructs E01 format images
type Reader struct {
	source       io.ReadSeeker
	currentChunk int
	chunkOffsets []int64
	eof          bool
	buffer       *bytes.Buffer
}

// NewReader creates a new E01 reader
func NewReader(source io.ReadSeeker) (*Reader, error) {
	r := &Reader{
		source: source,
		buffer: &bytes.Buffer{},
	}

	// Verify magic header
	magic := make([]byte, 8)
	if _, err := io.ReadFull(source, magic); err != nil {
		return nil, fmt.Errorf("failed to read magic: %w", err)
	}

	if string(magic) != magicHeader {
		return nil, fmt.Errorf("invalid E01 magic header")
	}

	// Read header length
	var headerLen uint32
	if err := binary.Read(source, binary.LittleEndian, &headerLen); err != nil {
		return nil, fmt.Errorf("failed to read header length: %w", err)
	}

	// Skip header text
	if _, err := source.Seek(int64(headerLen), io.SeekCurrent); err != nil {
		return nil, fmt.Errorf("failed to skip header: %w", err)
	}

	return r, nil
}

// Read implements io.Reader by decompressing chunks on-demand
func (r *Reader) Read(p []byte) (n int, err error) {
	if r.eof {
		return 0, io.EOF
	}

	// If buffer has data, read from it first
	if r.buffer.Len() > 0 {
		return r.buffer.Read(p)
	}

	// Read next chunk
	var flaggedSize uint32
	err = binary.Read(r.source, binary.LittleEndian, &flaggedSize)
	if err == io.EOF {
		r.eof = true
		return 0, io.EOF
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read chunk size: %w", err)
	}

	// Check for table section (magic "tabl" = 0x6c626174)
	if flaggedSize == 0x6c626174 || flaggedSize == 0x656c6261 { // "tabl" or "able"
		r.eof = true
		return 0, io.EOF
	}

	// Check for done section
	if flaggedSize == 0x656e6f64 { // "done"
		r.eof = true
		return 0, io.EOF
	}

	compressedSize := flaggedSize &^ 0x80000000
	isCompressed := (flaggedSize & 0x80000000) != 0

	chunkData := make([]byte, compressedSize)
	if _, err := io.ReadFull(r.source, chunkData); err != nil {
		return 0, fmt.Errorf("failed to read chunk data: %w", err)
	}

	// Try to read Adler32 checksum (newer E01 format)
	var expectedChecksum uint32
	hasChecksum := false
	if err := binary.Read(r.source, binary.LittleEndian, &expectedChecksum); err == nil {
		hasChecksum = true
	} else {
		// No checksum present, seek back (older format or different implementation)
		r.source.Seek(-4, io.SeekCurrent)
	}

	var uncompressedData []byte

	if isCompressed {
		// Decompress using zlib
		zr, err := zlib.NewReader(bytes.NewReader(chunkData))
		if err != nil {
			return 0, fmt.Errorf("failed to create zlib reader: %w", err)
		}
		uncompressedData, err = io.ReadAll(zr)
		zr.Close()
		if err != nil {
			return 0, fmt.Errorf("failed to decompress chunk: %w", err)
		}
	} else {
		uncompressedData = chunkData
	}

	// Verify checksum if present
	if hasChecksum {
		actualChecksum := adler32.Checksum(uncompressedData)
		if actualChecksum != expectedChecksum {
			return 0, fmt.Errorf("chunk checksum mismatch: expected 0x%x, got 0x%x", expectedChecksum, actualChecksum)
		}
	}

	// Write to buffer
	r.buffer.Write(uncompressedData)

	// Read from buffer into output
	return r.buffer.Read(p)
}
