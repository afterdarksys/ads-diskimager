package imager

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"time"
)

// Metadata holds Chain of Custody information
type Metadata struct {
	CaseNumber  string `json:"case_number,omitempty"`
	EvidenceNum string `json:"evidence_number,omitempty"`
	Examiner    string `json:"examiner,omitempty"`
	Description string `json:"description,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type Config struct {
	Source         io.Reader
	Destination    io.Writer
	BlockSize      int
	HashAlgo       string   // "md5", "sha1", "sha256" (primary algorithm for backward compatibility)
	HashAlgorithms []string // Multiple algorithms: ["md5", "sha1", "sha256"]
	Hasher         hash.Hash
	MultiHasher    io.Writer // Optional multi-hash writer
	Metadata       Metadata
}

// BadSector records a read error at a specific offset.
type BadSector struct {
	Offset int64  `json:"offset"`
	Size   int    `json:"size"`
	Error  string `json:"error"`
}

// Result holds the results of the imaging process.
type Result struct {
	BytesCopied int64              `json:"bytes_copied"`
	Hash        string             `json:"hash"` // Primary hash for backward compatibility
	Hashes      map[string]string  `json:"hashes,omitempty"` // All computed hashes (md5, sha1, sha256)
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	BadSectors  []BadSector        `json:"bad_sectors,omitempty"`
	Errors      []error            `json:"-"` // Internal errors unmarshaled safely
}

// Image performs the disk imaging process.
func Image(cfg Config) (*Result, error) {
	if cfg.BlockSize <= 0 {
		cfg.BlockSize = 64 * 1024 // Default 64KB
	}

	// Determine hash writer (single or multi)
	var hashWriter io.Writer
	var hasher hash.Hash
	usingMultiHash := cfg.MultiHasher != nil

	if usingMultiHash {
		// Use provided MultiHasher
		hashWriter = cfg.MultiHasher
	} else if cfg.Hasher != nil {
		// Use provided single hasher
		hasher = cfg.Hasher
		hashWriter = hasher
	} else {
		// Create single hasher from HashAlgo
		switch cfg.HashAlgo {
		case "md5":
			hasher = md5.New()
		case "sha1":
			hasher = sha1.New()
		case "sha256":
			hasher = sha256.New()
		default:
			return nil, fmt.Errorf("unsupported hash algorithm: %s", cfg.HashAlgo)
		}
		hashWriter = hasher
	}

	multiWriter := io.MultiWriter(cfg.Destination, hashWriter)

	res := &Result{
		StartTime: time.Now(),
		Hashes:    make(map[string]string),
	}

	buf := make([]byte, cfg.BlockSize)
	for {
		nr, err := cfg.Source.Read(buf)
		if nr > 0 {
			// Write the successfully read bytes
			nw, wErr := multiWriter.Write(buf[0:nr])
			if wErr != nil {
				return nil, fmt.Errorf("write error: %w", wErr)
			}
			if nr != nw {
				return nil, io.ErrShortWrite
			}
			res.BytesCopied += int64(nw)
		}

		if err != nil {
			if err == io.EOF {
				break
			}

			// We encountered a read error (Bad Sector).
			// Implement exponential backoff retry with smaller read sizes
			remaining := cfg.BlockSize - nr
			if remaining > 0 {
				recovered := tryRecoverBadSector(cfg.Source, multiWriter, res, remaining, err)
				if !recovered {
					// Recovery failed - zero fill the bad region
					res.BadSectors = append(res.BadSectors, BadSector{
						Offset: res.BytesCopied,
						Size:   remaining,
						Error:  err.Error(),
					})

					zeroBuf := make([]byte, remaining)
					nw, wErr := multiWriter.Write(zeroBuf)
					if wErr != nil {
						return nil, fmt.Errorf("write error during zero-padding: %w", wErr)
					}
					res.BytesCopied += int64(nw)

					res.Errors = append(res.Errors, err)
				}
				continue // Try reading the next block
			}
		}
	}

	res.EndTime = time.Now()

	// Populate hash results
	if usingMultiHash {
		// MultiHasher results will be extracted in calling code
		// Mark that multi-hash was used
		res.Hash = "multi-hash-enabled"
	} else if hasher != nil {
		// Single hash
		res.Hash = fmt.Sprintf("%x", hasher.Sum(nil))
		res.Hashes[cfg.HashAlgo] = res.Hash
	}

	return res, nil
}

// tryRecoverBadSector attempts to read around bad sectors with exponential backoff
// Returns true if recovery succeeded, false if the entire region is bad
func tryRecoverBadSector(source io.Reader, dest io.Writer, res *Result, size int, originalErr error) bool {
	// Only attempt recovery if source is seekable
	seeker, ok := source.(io.Seeker)
	if !ok {
		return false
	}

	// Get current position
	currentPos, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return false
	}

	// Try reading with exponentially smaller chunks: 4KB, 2KB, 1KB, 512B
	chunkSizes := []int{4096, 2048, 1024, 512}
	recovered := 0
	badOffset := res.BytesCopied

	for recovered < size {
		readSize := size - recovered
		success := false

		// Try each chunk size with exponential backoff
		for _, chunkSize := range chunkSizes {
			if readSize > chunkSize {
				readSize = chunkSize
			}

			// Seek to the position we want to read
			if _, err := seeker.Seek(currentPos+int64(recovered), io.SeekStart); err != nil {
				continue
			}

			// Retry read with smaller chunk
			buf := make([]byte, readSize)
			nr, err := source.Read(buf)
			if nr > 0 {
				// Successfully read some data - write it
				if nw, wErr := dest.Write(buf[0:nr]); wErr == nil {
					recovered += nw
					res.BytesCopied += int64(nw)
					success = true
					break
				}
			}

			if err == io.EOF {
				return recovered > 0
			}

			// If this chunk size failed, try smaller (loop continues)
		}

		// If all chunk sizes failed for this position, mark it as bad
		if !success {
			// Mark this sector as bad
			res.BadSectors = append(res.BadSectors, BadSector{
				Offset: badOffset + int64(recovered),
				Size:   512, // Assume sector size
				Error:  originalErr.Error(),
			})

			// Zero fill this sector
			zeroBuf := make([]byte, 512)
			if nw, wErr := dest.Write(zeroBuf); wErr == nil {
				recovered += nw
				res.BytesCopied += int64(nw)
			} else {
				return false // Write error is fatal
			}
		}
	}

	// Seek to end of recovered region
	seeker.Seek(currentPos+int64(recovered), io.SeekStart)

	return true
}
