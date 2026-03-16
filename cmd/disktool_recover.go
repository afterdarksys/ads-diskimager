package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var recoverOutDir string

var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "File carving and unallocated space recovery",
	Run: func(cmd *cobra.Command, args []string) {
		requireDevice(cmd)
		if recoverOutDir == "" {
			fmt.Println("Error: --out-dir required")
			return
		}

		if err := os.MkdirAll(recoverOutDir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}

		fmt.Printf("Starting recovery/carving on %s -> %s\n", disktoolDevice, recoverOutDir)
		
		err := FileCarver(disktoolDevice, recoverOutDir, func(offset int64, fileType string) {
			fmt.Printf("\rRecovered %s at offset 0x%x        ", fileType, offset)
		})
		
		if err != nil {
			log.Fatalf("\nError during recovery: %v", err)
		}
		fmt.Println("\nFile carving complete.")
	},
}

var (
	magicJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0}
	magicPDF  = []byte{0x25, 0x50, 0x44, 0x46, 0x2D} // "%PDF-"
	magicPNG  = []byte{0x89, 0x50, 0x4E, 0x47} // "\x89PNG"
)

// FileCarver scans a raw image or device for known file signatures and extracts them.
func FileCarver(devicePath string, outDir string, onRecover func(offset int64, fileType string)) error {
	f, err := os.Open(devicePath)
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer f.Close()

	// 1 MB buffer for reading chunks, using 512 byte overlap to catch signatures on boundaries
	bufSize := 1024 * 1024 
	overlap := 512
	buf := make([]byte, bufSize+overlap)
	
	var globalOffset int64 = 0
	
	for {
		// We read bufSize bytes, leaving the overlap region at the beginning filled with the end of the last read
		readStart := 0
		if globalOffset > 0 {
			readStart = overlap
		}
		
		n, err := io.ReadFull(f, buf[readStart:])
		actualRead := n + readStart
		
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("read error at offset %d: %w", globalOffset, err)
		}
		
		if actualRead == 0 {
			break
		}

		// Search for signatures in the current window
		searchData := buf[:actualRead]
		
		carveFile := func(magic []byte, ext string, maxSize int) {
			idx := bytes.Index(searchData, magic)
			if idx != -1 {
				absOffset := globalOffset - int64(overlap) + int64(idx)
				if globalOffset == 0 {
					absOffset = int64(idx)
				}
				
				if onRecover != nil {
					onRecover(absOffset, ext)
				}
				
				// Attempt to extract the file up to maxSize or EOF
				// For a production carver, we would look for trailers (e.g. FFD9 for JPEG)
				// For this prototype, we carve a fixed chunk of bytes to demonstrate extraction.
				outPath := filepath.Join(outDir, fmt.Sprintf("recovered_0x%x%s", absOffset, ext))
				outFile, createErr := os.Create(outPath)
				if createErr != nil {
					log.Printf("\nFailed to create carved file: %v", createErr)
					return
				}
				
				// Re-seek and read from the absolute offset
				currentPos, _ := f.Seek(0, io.SeekCurrent) // Save position
				f.Seek(absOffset, io.SeekStart)
				
				carveBuf := make([]byte, maxSize)
				cn, _ := io.ReadFull(f, carveBuf)
				outFile.Write(carveBuf[:cn])
				outFile.Close()
				
				f.Seek(currentPos, io.SeekStart) // Restore position
				
				// Zero out the found magic in our buffer to prevent infinite loops if we rescan
				copy(searchData[idx:idx+len(magic)], make([]byte, len(magic)))
			}
		}

		// Keep carving out multiple files in the same block if they exist
		for bytes.Contains(searchData, magicJPEG) { carveFile(magicJPEG, ".jpg", 5*1024*1024) } // 5MB limit
		for bytes.Contains(searchData, magicPDF) { carveFile(magicPDF, ".pdf", 10*1024*1024) } // 10MB limit
		for bytes.Contains(searchData, magicPNG) { carveFile(magicPNG, ".png", 5*1024*1024) } // 5MB limit

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		
		// Copy the last `overlap` bytes to the front of the buffer for the next iteration
		copy(buf[:overlap], buf[actualRead-overlap:actualRead])
		globalOffset += int64(n)
	}

	return nil
}

func init() {
	disktoolCmd.AddCommand(recoverCmd)
	recoverCmd.Flags().StringVarP(&recoverOutDir, "out-dir", "o", "", "Output directory for carved files")
}
