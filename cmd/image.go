package cmd

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/afterdarksys/diskimager/imager"
	"github.com/afterdarksys/diskimager/pkg/format/e01"
	"github.com/afterdarksys/diskimager/pkg/format/raw"
	"github.com/afterdarksys/diskimager/pkg/geometry"
	"github.com/afterdarksys/diskimager/pkg/smart"
	"github.com/afterdarksys/diskimager/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	inputFile  string
	outputFile string
	blockSize  int
	hashAlgo   string
	imgFormat  string
	resume     bool

	// Metadata flags
	caseNum  string
	evidence string
	examiner string
	desc     string
	notes    string

	// Safety flags
	collectSMART     bool
	verifyWriteBlock bool
	collectGeometry  bool
)

// ResumeMetadata stores state for resuming interrupted imaging sessions
type ResumeMetadata struct {
	BytesCopied    int64  `json:"bytes_copied"`
	HashAlgorithm  string `json:"hash_algorithm"`
	HashState      []byte `json:"hash_state,omitempty"` // Serialized hash state
	SourceChecksum string `json:"source_checksum"`       // Hash of source at time of interruption
	Timestamp      string `json:"timestamp"`
}

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Create a forensic image of a disk or file",
	Run: func(cmd *cobra.Command, args []string) {
		if inputFile == "" || outputFile == "" {
			cmd.Usage()
			os.Exit(1)
		}

		if blockSize <= 0 {
			log.Fatalf("Block size must be strictly positive")
		}

		switch hashAlgo {
		case "md5", "sha1", "sha256":
			// valid
		default:
			log.Fatalf("Unsupported hash algorithm: %s", hashAlgo)
		}

		// Collect disk geometry if requested
		var diskGeom *geometry.DiskGeometry
		if collectGeometry {
			fmt.Println("Collecting disk geometry...")
			var geoErr error
			diskGeom, geoErr = geometry.GetGeometry(inputFile)
			if geoErr != nil {
				fmt.Printf("⚠️  Geometry unavailable: %v\n", geoErr)
			} else {
				fmt.Printf("✓ Geometry: C=%d H=%d S=%d (Total: %d bytes)\n",
					diskGeom.Cylinders, diskGeom.Heads, diskGeom.Sectors, diskGeom.TotalSize)
			}
		}

		// Collect SMART data if requested
		var diskInfo *smart.DiskInfo
		if collectSMART {
			fmt.Println("Collecting SMART data...")
			diskInfo = smart.CollectDiskInfo(inputFile)
			if diskInfo.Available {
				fmt.Printf("✓ Device: %s %s (S/N: %s)\n", diskInfo.Model, diskInfo.Capacity, diskInfo.Serial)
				fmt.Printf("✓ SMART Status: %s\n", diskInfo.SMARTStatus)
				if diskInfo.Temperature != "" {
					fmt.Printf("✓ Temperature: %s\n", diskInfo.Temperature)
				}
				if diskInfo.PowerOnHours != "" {
					fmt.Printf("✓ Power-On Hours: %s\n", diskInfo.PowerOnHours)
				}
			} else {
				fmt.Printf("⚠️  SMART data unavailable: %s\n", diskInfo.Error)
			}
		}

		// Verify write-blocker if requested
		if verifyWriteBlock {
			fmt.Println("Checking write-blocker status...")
			isProtected, err := smart.IsWriteProtected(inputFile)
			if err != nil {
				fmt.Printf("⚠️  Cannot verify write-blocker: %v\n", err)
				fmt.Println("⚠️  Ensure hardware write-blocker is in use!")
			} else if isProtected {
				fmt.Println("✓ Device is write-protected")
			} else {
				fmt.Println("❌ WARNING: Device is NOT write-protected!")
				fmt.Println("❌ Use a hardware write-blocker for forensic acquisitions!")
				fmt.Print("Continue anyway? (type 'yes'): ")
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					log.Fatalf("Aborted: Device not write-protected")
				}
			}
		}

		// Read resume metadata if resuming
		var existingBytesCopied int64 = 0
		var resumeMeta *ResumeMetadata
		resumeMetaFile := outputFile + ".resume.json"

		if resume {
			// Try to load resume metadata
			if data, err := os.ReadFile(resumeMetaFile); err == nil {
				var meta ResumeMetadata
				if err := json.Unmarshal(data, &meta); err == nil {
					resumeMeta = &meta
					existingBytesCopied = meta.BytesCopied
					fmt.Printf("Resuming from %d bytes (saved state found)...\n", existingBytesCopied)

					// Verify hash algorithm matches
					if meta.HashAlgorithm != hashAlgo {
						log.Fatalf("Hash algorithm mismatch: resume file uses %s, but %s was specified", meta.HashAlgorithm, hashAlgo)
					}
				} else {
					fmt.Printf("Warning: Failed to parse resume metadata, will re-hash: %v\n", err)
				}
			} else {
				// Fallback to old behavior - use file size
				stat, err := os.Stat(outputFile)
				if err == nil {
					existingBytesCopied = stat.Size()
					fmt.Printf("Resuming from %d bytes (no saved state, will re-hash)...\n", existingBytesCopied)
				}
			}
		}

		// Open Input (Read-Only)
		in, err := os.Open(inputFile)
		if err != nil {
			log.Fatalf("Error opening input file: %v", err)
		}
		defer in.Close()

		var hasher hash.Hash
		switch hashAlgo {
		case "md5":
			hasher = md5.New()
		case "sha1":
			hasher = sha1.New()
		case "sha256":
			hasher = sha256.New()
		}

		// Restore hash state if available, otherwise re-hash existing data
		if resume && existingBytesCopied > 0 {
			if resumeMeta != nil && len(resumeMeta.HashState) > 0 {
				// TODO: Go's standard hash interfaces don't support state serialization
				// For now, we must re-hash. A future enhancement would use a custom
				// hash wrapper that supports marshaling/unmarshaling state.
				fmt.Printf("Note: Hash state restoration not yet implemented, re-hashing %d bytes...\n", existingBytesCopied)
				if _, err := io.CopyN(hasher, in, existingBytesCopied); err != nil {
					log.Fatalf("Error hashing input file for resume: %v", err)
				}
			} else {
				// Re-hash existing bytes from source
				fmt.Printf("Re-hashing existing %d bytes from source for resume...\n", existingBytesCopied)
				if _, err := io.CopyN(hasher, in, existingBytesCopied); err != nil {
					log.Fatalf("Error hashing input file for resume: %v", err)
				}
			}
		}

		stat, err := in.Stat()
		if err == nil {
			fmt.Printf("Source Total Size: %d bytes\n", stat.Size())
		}

		// Metadata mapping
		meta := imager.Metadata{
			CaseNumber:  caseNum,
			EvidenceNum: evidence,
			Examiner:    examiner,
			Description: desc,
			Notes:       notes,
		}

		// Format selection and Output Open
		outTarget, err := storage.OpenDestination(outputFile, resume)
		if err != nil {
			log.Fatalf("Error opening destination: %v", err)
		}

		var out io.WriteCloser
		fmtFormat := strings.ToLower(imgFormat)
		if fmtFormat == "e01" || fmtFormat == "ewf" {
			out, err = e01.NewWriter(outTarget, resume, meta)
		} else {
			out, err = raw.NewWriter(outTarget)
		}
		if err != nil {
			log.Fatalf("Error creating output format writer: %v", err)
		}
		defer out.Close()

		fmt.Printf("Starting imaging process...\nSource: %s\nDestination: %s\nFormat: %s\nHash: %s\n", 
			inputFile, outputFile, fmtFormat, hashAlgo)

		wrappedReader := &ProgressReader{
			Reader:    in,
			BytesRead: existingBytesCopied,
		}

		cfg := imager.Config{
			Source:      wrappedReader,
			Destination: out,
			BlockSize:   blockSize,
			HashAlgo:    hashAlgo,
			Hasher:      hasher,
			Metadata:    meta,
		}

		start := time.Now()
		res, err := imager.Image(cfg)
		if err != nil {
			log.Printf("\nError during imaging: %v", err)
		} else {
			fmt.Printf("\nImaging completed successfully in %v.\n", time.Since(start))
		}

		// Combine resume bytes with run bytes
		if res != nil {
			res.BytesCopied += existingBytesCopied
		}

		// Write Audit Log
		logFile := outputFile + ".log"
		logEntry := struct {
			Source      string
			Destination string
			DiskInfo    *smart.DiskInfo       `json:"disk_info,omitempty"`
			Geometry    *geometry.DiskGeometry `json:"geometry,omitempty"`
			Config      imager.Config
			Result      *imager.Result
			Error       string `json:",omitempty"`
		}{
			Source:      inputFile,
			Destination: outputFile,
			DiskInfo:    diskInfo,
			Geometry:    diskGeom,
			Config:      cfg,
			Result:      res,
		}
		// Zero out streams for serialization
		logEntry.Config.Source = nil
		logEntry.Config.Destination = nil

		if err != nil {
			logEntry.Error = err.Error()
		}

		logBytes, _ := json.MarshalIndent(logEntry, "", "  ")
		// Write Audit with secure permissions (0600 - owner read/write only)
		if wErr := os.WriteFile(logFile, logBytes, 0600); wErr != nil {
			log.Printf("Error writing audit log: %v", wErr)
		} else {
			fmt.Printf("Audit log written to %s (secure permissions)\n", logFile)
		}

		// Clean up resume metadata file on successful completion
		if err == nil && res != nil {
			os.Remove(resumeMetaFile)
		}

		if res != nil {
			fmt.Printf("Total Bytes Copied: %d\n", res.BytesCopied)
			fmt.Printf("Bad Sectors Encountered: %d\n", len(res.BadSectors))
			fmt.Printf("Hash (%s): %s\n", hashAlgo, res.Hash)
		}
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.Flags().StringVar(&inputFile, "in", "", "Input device or file path (required)")
	imageCmd.Flags().StringVar(&outputFile, "out", "", "Output image file path (required)")
	imageCmd.Flags().IntVar(&blockSize, "bs", 64*1024, "Block size in bytes")
	imageCmd.Flags().StringVar(&hashAlgo, "hash", "sha256", "Hash algorithm (md5, sha1, sha256)")
	
	// New Flags
	imageCmd.Flags().StringVar(&imgFormat, "format", "raw", "Output format (raw, e01)")
	imageCmd.Flags().BoolVar(&resume, "resume", false, "Resume from an interrupted imaging session")
	
	// Metadata Flags
	imageCmd.Flags().StringVar(&caseNum, "case", "", "Case Number")
	imageCmd.Flags().StringVar(&evidence, "evidence", "", "Evidence Number")
	imageCmd.Flags().StringVar(&examiner, "examiner", "", "Examiner Name")
	imageCmd.Flags().StringVar(&desc, "desc", "", "Description of evidence")
	imageCmd.Flags().StringVar(&notes, "notes", "", "Additional notes")

	// Safety and Forensic Flags
	imageCmd.Flags().BoolVar(&collectSMART, "smart", false, "Collect SMART data from source disk")
	imageCmd.Flags().BoolVar(&verifyWriteBlock, "verify-write-block", false, "Verify source is write-protected")
	imageCmd.Flags().BoolVar(&collectGeometry, "geometry", false, "Collect disk geometry (CHS)")

	imageCmd.MarkFlagRequired("in")
	imageCmd.MarkFlagRequired("out")
}

// ProgressReader wraps an io.Reader to print progress.
type ProgressReader struct {
	Reader    io.Reader
	Total     int64
	BytesRead int64
	lastPrint int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	newBytes := atomic.AddInt64(&pr.BytesRead, int64(n))

	// Use atomic compare-and-swap to prevent race conditions in progress reporting
	// Only print if we've read at least 10MB since last print
	for {
		lastPrint := atomic.LoadInt64(&pr.lastPrint)
		if newBytes-lastPrint >= 10*1024*1024 {
			// Try to update lastPrint atomically
			if atomic.CompareAndSwapInt64(&pr.lastPrint, lastPrint, newBytes) {
				// We won the race - print progress
				fmt.Printf("\rCopied: %d bytes", newBytes)
				break
			}
			// Someone else updated it, retry the loop
		} else {
			// Not enough progress yet, no need to print
			break
		}
	}

	return n, err
}

func (pr *ProgressReader) Seek(offset int64, whence int) (int64, error) {
	if s, ok := pr.Reader.(io.Seeker); ok {
		return s.Seek(offset, whence)
	}
	return 0, fmt.Errorf("source does not support seeking")
}
