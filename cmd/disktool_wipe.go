package cmd

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var wipePasses int

var wipeCmd = &cobra.Command{
	Use:   "wipe <device/file>",
	Short: "Securely overwrite a drive or file (DoD 5220.22-M standard)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		requireDevice(cmd)
		targetPath := disktoolDevice // comes from root disktool scope

		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Println("⚠️  CRITICAL WARNING: DESTRUCTIVE WIPE OPERATION")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("You are about to SECURELY WIPE: %s\n", targetPath)
		fmt.Println("This operation will DESTROY ALL DATA and is IRREVERSIBLE.")
		fmt.Printf("Wipe pattern: %d-pass DoD 5220.22-M\n", wipePasses)
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("\nTo confirm, type 'WIPE' in capital letters: ")

		var confirm string
		fmt.Scanln(&confirm)

		if confirm != "WIPE" {
			fmt.Printf("\n❌ Confirmation failed. You typed: '%s'\n", confirm)
			fmt.Println("Aborting wipe operation.")
			os.Exit(1)
		}

		fmt.Println("\n✓ Confirmation accepted. Starting wipe in 3 seconds...")
		time.Sleep(3 * time.Second)
		fmt.Printf("Executing %d-pass wipe...\n", wipePasses)

		err := WipeDrive(targetPath, wipePasses, func(pass int, copied, total int64) {
			if copied%(1024*1024*10) == 0 || copied == total {
				fmt.Printf("\rPass %d/%d - Overwritten: %d / %d bytes", pass, wipePasses, copied, total)
			}
		})
		if err != nil {
			log.Fatalf("\nError during wipe: %v", err)
		}
		fmt.Println("\nSecure Wipe Complete.")
	},
}

func init() {
	disktoolCmd.AddCommand(wipeCmd)
	wipeCmd.Flags().IntVar(&wipePasses, "passes", 3, "Number of overwrite passes (1-7)")
}

// WipeDrive performs a secure multi-pass overwrite on a file or block device.
func WipeDrive(path string, passes int, progressCallback func(pass int, copied, total int64)) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open device for writing: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat device: %w", err)
	}
	
	totalSize := stat.Size()
	if totalSize == 0 {
		return fmt.Errorf("device size is 0 or cannot be determined")
	}

	bufSize := 1024 * 1024 // 1MB blocks
	buf := make([]byte, bufSize)

	start := time.Now()

	for pass := 1; pass <= passes; pass++ {
		// Prepare buffer for this pass
		switch pass % 3 {
		case 1:
			// Pass 1: Zeros
			for i := range buf {
				buf[i] = 0x00
			}
		case 2:
			// Pass 2: Ones
			for i := range buf {
				buf[i] = 0xFF
			}
		case 0:
			// Pass 3: Random
			// We will re-generate random data inside the write loop
		}

		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek to start: %w", err)
		}

		var written int64 = 0
		for written < totalSize {
			writeLen := int64(bufSize)
			if totalSize-written < writeLen {
				writeLen = totalSize - written
			}

			if pass%3 == 0 {
				io.ReadFull(rand.Reader, buf[:writeLen])
			}

			n, err := f.Write(buf[:writeLen])
			if err != nil {
				return fmt.Errorf("write failed at offset %d: %w", written, err)
			}
			
			written += int64(n)
			if progressCallback != nil {
				progressCallback(pass, written, totalSize)
			}
		}
		
		// Ensure writes hit the disk
		f.Sync()
	}

	fmt.Printf("\nTotal Wipe Time: %v\n", time.Since(start))
	return nil
}
