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
)

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

		// Read Audit Log if Resuming to find previous size and hash progress
		var existingBytesCopied int64 = 0
		if resume {
			stat, err := os.Stat(outputFile)
			if err == nil {
				existingBytesCopied = stat.Size()
				fmt.Printf("Resuming from %d bytes...\n", existingBytesCopied)
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

		if resume && existingBytesCopied > 0 {
			fmt.Printf("Hashing existing %d bytes from source for resume...\n", existingBytesCopied)
			if _, err := io.CopyN(hasher, in, existingBytesCopied); err != nil {
				log.Fatalf("Error hashing input file for resume: %v", err)
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
			Config      imager.Config
			Result      *imager.Result
			Error       string `json:",omitempty"`
		}{
			Source:      inputFile,
			Destination: outputFile,
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
		// Write Audit as truncation always to keep it clean (unless we merge bad sectors, simple rewrite is easier)
		if wErr := os.WriteFile(logFile, logBytes, 0644); wErr != nil {
			log.Printf("Error writing audit log: %v", wErr)
		} else {
			fmt.Printf("Audit log written to %s\n", logFile)
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
	lastPrint := atomic.LoadInt64(&pr.lastPrint)

	if newBytes-lastPrint >= 10*1024*1024 { // Print every 10MB
		fmt.Printf("\rCopied: %d bytes", newBytes)
		atomic.StoreInt64(&pr.lastPrint, newBytes)
	}
	return n, err
}

func (pr *ProgressReader) Seek(offset int64, whence int) (int64, error) {
	if s, ok := pr.Reader.(io.Seeker); ok {
		return s.Seek(offset, whence)
	}
	return 0, fmt.Errorf("source does not support seeking")
}
