package cmd

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	verifyImageFile string
	verifyExpected  string
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify the integrity of a disk image",
	Run: func(cmd *cobra.Command, args []string) {
		if verifyImageFile == "" || verifyExpected == "" {
			cmd.Usage()
			os.Exit(1)
		}

		f, err := os.Open(verifyImageFile)
		if err != nil {
			log.Fatalf("Failed to open image file: %v", err)
		}
		defer f.Close()

		fmt.Printf("Verifying %s...\n", verifyImageFile)
		fmt.Printf("Expected SHA256 : %s\n", verifyExpected)

		start := time.Now()
		
		h := sha256.New()
		var totalRead int64 = 0

		// Check if it's an EVF file
		headerMagic := make([]byte, 8)
		f.Read(headerMagic)
		isEWF := string(headerMagic) == "EVF\x09\x0d\x0a\xff\x00"

		if isEWF {
			fmt.Println("Detected E01 format. Hashing raw uncompressed data...")
			// Read header length
			var headerLen uint32
			binary.Read(f, binary.LittleEndian, &headerLen)
			f.Seek(int64(headerLen), io.SeekCurrent)

			for {
				var flaggedSize uint32
				err := binary.Read(f, binary.LittleEndian, &flaggedSize)
				if err == io.EOF {
					break // Reached EOF without error, but usually we'd hit the table section
				}
				if err != nil {
					log.Fatalf("\nError reading chunk size: %v", err)
				}

				// Check if we hit the 'table' magic string by interpreting the 4 bytes read as 'tabl'
				// The string "tabl" in little-endian uint32 is 0x6c626174
				if flaggedSize == 0x6c626174 {
					// We hit "table2", we are done parsing chunks
					break
				}

				compressedSize := flaggedSize &^ 0x80000000
				isCompressed := (flaggedSize & 0x80000000) != 0

				chunkData := make([]byte, compressedSize)
				if _, err := io.ReadFull(f, chunkData); err != nil {
					log.Fatalf("\nError reading chunk data: %v", err)
				}

				if isCompressed {
					zr, err := zlib.NewReader(bytes.NewReader(chunkData))
					if err != nil {
						log.Fatalf("\nError decompressing chunk: %v", err)
					}
					uncompressedData, _ := io.ReadAll(zr)
					zr.Close()
					h.Write(uncompressedData)
					totalRead += int64(len(uncompressedData))
				} else {
					h.Write(chunkData)
					totalRead += int64(compressedSize)
				}

				if totalRead%(1024*1024*50) == 0 {
					fmt.Printf("\rHashed: %d MB", totalRead/(1024*1024))
				}
			}
		} else {
			f.Seek(0, io.SeekStart)
			buf := make([]byte, 256*1024) // 256KB buffer for speed
			for {
				n, err := f.Read(buf)
				if n > 0 {
					h.Write(buf[:n])
					totalRead += int64(n)
					if totalRead%(1024*1024*50) == 0 {
						fmt.Printf("\rHashed: %d MB", totalRead/(1024*1024))
					}
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Fatalf("\nError reading file: %v", err)
				}
			}
		}

		actualHash := fmt.Sprintf("%x", h.Sum(nil))
		fmt.Printf("\nActual SHA256   : %s\n", actualHash)
		
		if actualHash == verifyExpected {
			fmt.Printf("VERIFICATION SUCCESS in %v\n", time.Since(start))
		} else {
			fmt.Printf("VERIFICATION FAILED in %v\n", time.Since(start))
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().StringVar(&verifyImageFile, "image", "", "Image file to verify")
	verifyCmd.Flags().StringVar(&verifyExpected, "expected-hash", "", "Expected SHA256 hash")
}
