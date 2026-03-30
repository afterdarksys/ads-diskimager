package e01

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"io"
	"os"

	"github.com/afterdarksys/diskimager/imager"
)

const (
	chunkSize   = 32768 // Standard EWF chunk size (32KB)
	magicHeader = "EVF\x09\x0d\x0a\xff\x00" // Standard 8-byte magic
)

// Writer implements a minimal Expert Witness Format (E01) generator
// NOTE: A full E01 implementation requires handling Adler32 checksums per chunk,
// complex section headers (header, volume, table, data, desc), and table offsets.
// For the sake of this prototype, we will implement the scaffolding but
// acknowledge that an industry-compliant EWF generator typically uses libewf.
type Writer struct {
	out        io.WriteCloser
	metadata   imager.Metadata
	chunkCount int
	offsetList []int64 // Track chunk offsets

	currentOffset int64
}

// NewWriter initializes a new E01 writer
func NewWriter(out io.WriteCloser, appendMode bool, meta imager.Metadata) (*Writer, error) {
	w := &Writer{
		out:      out,
		metadata: meta,
	}

	if !appendMode {
		if err := w.writeHeader(); err != nil {
			out.Close()
			return nil, err
		}
	} else {
		// E01 Resume Support: Parse existing E01 file to recover chunk table
		// This is critical to avoid corrupting the E01 format structure
		if f, ok := out.(*os.File); ok {
			if err := w.parseExistingFile(f); err != nil {
				out.Close()
				return nil, fmt.Errorf("failed to parse existing E01 file for resume: %w", err)
			}
		} else {
			out.Close()
			return nil, fmt.Errorf("E01 resume only supported for local files")
		}
	}

	return w, nil
}

// parseExistingFile parses an existing E01 file to extract chunk table and verify integrity
func (w *Writer) parseExistingFile(f *os.File) error {
	// Seek to beginning to parse the file
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start: %w", err)
	}

	// Read and verify magic header
	magic := make([]byte, 8)
	if _, err := io.ReadFull(f, magic); err != nil {
		return fmt.Errorf("failed to read magic header: %w", err)
	}
	if string(magic) != magicHeader {
		return fmt.Errorf("invalid E01 magic header")
	}
	w.currentOffset = 8

	// Read header length
	var headerLen uint32
	if err := binary.Read(f, binary.LittleEndian, &headerLen); err != nil {
		return fmt.Errorf("failed to read header length: %w", err)
	}
	w.currentOffset += 4

	// Skip header content
	if _, err := f.Seek(int64(headerLen), io.SeekCurrent); err != nil {
		return fmt.Errorf("failed to skip header: %w", err)
	}
	w.currentOffset += int64(headerLen)

	// Parse chunks until we find the table section
	for {
		// Record chunk offset
		chunkOffset := w.currentOffset

		// Try to read chunk header (4-byte size)
		var flaggedSize uint32
		if err := binary.Read(f, binary.LittleEndian, &flaggedSize); err != nil {
			if err == io.EOF {
				// Reached end before table section - file might be incomplete
				break
			}
			return fmt.Errorf("failed to read chunk size at offset %d: %w", w.currentOffset, err)
		}

		// Check if this is the table section marker
		// Table section starts with "table2" magic
		currentPos := w.currentOffset
		if _, err := f.Seek(currentPos, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek to check table marker: %w", err)
		}

		tableMagic := make([]byte, 6)
		if n, _ := f.Read(tableMagic); n == 6 && string(tableMagic) == "table2" {
			// Found table section - we've parsed all data chunks
			// Seek back to before table to allow overwriting
			if _, err := f.Seek(currentPos, io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek back: %w", err)
			}
			w.currentOffset = currentPos

			// Truncate file at this point to remove old table
			if err := f.Truncate(w.currentOffset); err != nil {
				return fmt.Errorf("failed to truncate file: %w", err)
			}

			break
		}

		// This is a data chunk - record it and skip
		w.offsetList = append(w.offsetList, chunkOffset)
		w.chunkCount++

		// Extract actual compressed size (remove compression flag)
		compressedSize := flaggedSize & 0x7FFFFFFF

		// Skip compressed data + adler32 checksum (4 bytes)
		skipSize := int64(compressedSize) + 4 + 4 // 4 for size field we already read + 4 for checksum
		if _, err := f.Seek(w.currentOffset+skipSize, io.SeekStart); err != nil {
			return fmt.Errorf("failed to skip chunk data: %w", err)
		}
		w.currentOffset += skipSize
	}

	// Seek to end of file to continue writing
	if _, err := f.Seek(w.currentOffset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to resume position: %w", err)
	}

	return nil
}

// writeHeader writes the EWF magic signature and initial header section
func (w *Writer) writeHeader() error {
	// Write Magic
	if _, err := w.out.Write([]byte(magicHeader)); err != nil {
		return err
	}
	w.currentOffset += 8

	// To keep this MVP functional yet demonstrable, we write a simplified "custom" E01-like header.
	// True EWF1 format requires specific compression flags, volume sections, etc.
	// We will serialize the metadata here to satisfy the Chain of Custody requirement,
	// even if a standard tool (like FTK) might require the strict `libewf` ASCII format.

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

// Write compresses the incoming data into chunks and writes them
func (w *Writer) Write(p []byte) (n int, err error) {
	totalWritten := 0

	// We must chunk exactly `chunkSize` unless it's the final write
	for len(p) > 0 {
		writeLen := len(p)
		if writeLen > chunkSize {
			writeLen = chunkSize
		}

		chunk := p[:writeLen]
		p = p[writeLen:]

		// Calculate Adler32 checksum for uncompressed chunk (forensic integrity)
		checksum := adler32.Checksum(chunk)

		// Compress chunk
		var b bytes.Buffer
		zw := zlib.NewWriter(&b)
		_, err := zw.Write(chunk)
		if err != nil {
			return totalWritten, err
		}
		zw.Close()
		compressedData := b.Bytes()

		// EWF format: 4-byte flagged size + compressed data + 4-byte Adler32 checksum
		compressedSize := uint32(len(compressedData))
		// High bit set to indicate compression in EWF
		flaggedSize := compressedSize | 0x80000000

		w.offsetList = append(w.offsetList, w.currentOffset)

		// Write flagged size
		if err := binary.Write(w.out, binary.LittleEndian, flaggedSize); err != nil {
			return totalWritten, err
		}
		w.currentOffset += 4

		// Write compressed data
		if _, err := w.out.Write(compressedData); err != nil {
			return totalWritten, err
		}
		w.currentOffset += int64(compressedSize)

		// Write Adler32 checksum (improves EWF compliance)
		if err := binary.Write(w.out, binary.LittleEndian, checksum); err != nil {
			return totalWritten, err
		}
		w.currentOffset += 4
		
		w.chunkCount++
		totalWritten += writeLen
	}

	return totalWritten, nil
}

// Close finalizes the E01 file
func (w *Writer) Close() error {
	// A proper EWF file writes a "table" section at the end pointing to every chunk offset,
	// followed by a "done" section.
	
	// Write Table Magic/Section
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
		// EWF1 uses 32-bit offsets (limits to 4GB). 
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
