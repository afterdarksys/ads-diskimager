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
	Source      io.Reader
	Destination io.Writer
	BlockSize   int
	HashAlgo    string // "md5", "sha1", "sha256"
	Hasher      hash.Hash
	Metadata    Metadata
}

// BadSector records a read error at a specific offset.
type BadSector struct {
	Offset int64  `json:"offset"`
	Size   int    `json:"size"`
	Error  string `json:"error"`
}

// Result holds the results of the imaging process.
type Result struct {
	BytesCopied int64       `json:"bytes_copied"`
	Hash        string      `json:"hash"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     time.Time   `json:"end_time"`
	BadSectors  []BadSector `json:"bad_sectors,omitempty"`
	Errors      []error     `json:"-"` // Internal errors unmarshaled safely
}

// Image performs the disk imaging process.
func Image(cfg Config) (*Result, error) {
	if cfg.BlockSize <= 0 {
		cfg.BlockSize = 64 * 1024 // Default 64KB
	}

	var hasher hash.Hash
	if cfg.Hasher != nil {
		hasher = cfg.Hasher
	} else {
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
	}

	multiWriter := io.MultiWriter(cfg.Destination, hasher)

	res := &Result{
		StartTime: time.Now(),
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
			// If we didn't read the full block, we need to pad the rest of the expected block size with zeros.
			remaining := cfg.BlockSize - nr
			if remaining > 0 {
				res.BadSectors = append(res.BadSectors, BadSector{
					Offset: res.BytesCopied,
					Size:   remaining,
					Error:  err.Error(),
				})
				
				// Zero fill
				zeroBuf := make([]byte, remaining)
				nw, wErr := multiWriter.Write(zeroBuf)
				if wErr != nil {
					return nil, fmt.Errorf("write error during zero-padding: %w", wErr)
				}
				res.BytesCopied += int64(nw)
				
				// If the source is a Seeker, try to seek past the bad block.
				if seeker, ok := cfg.Source.(io.Seeker); ok {
					if _, seekErr := seeker.Seek(int64(remaining), io.SeekCurrent); seekErr != nil {
						return res, fmt.Errorf("failed to seek past bad sector: %w", seekErr)
					}
				}
				
				res.Errors = append(res.Errors, err)
				continue // Try reading the next block
			}
		}
	}

	res.EndTime = time.Now()
	res.Hash = fmt.Sprintf("%x", hasher.Sum(nil))

	return res, nil
}
