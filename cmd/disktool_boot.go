package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var getBootloaderCmd = &cobra.Command{
	Use:   "getbootloader",
	Short: "Extracts the MBR or boot sector binaries",
	Run: func(cmd *cobra.Command, args []string) {
		requireDevice(cmd)

		f, err := os.Open(disktoolDevice)
		if err != nil {
			log.Fatalf("Failed to open device: %v", err)
		}
		defer f.Close()

		buf := make([]byte, 512)
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatalf("Failed to read MBR: %v", err)
		}

		if n < 512 {
			log.Fatalf("Device too small for MBR (read %d bytes)", n)
		}

		outFile := "mbr_dump.bin"
		if err := os.WriteFile(outFile, buf, 0644); err != nil {
			log.Fatalf("Failed to write %s: %v", outFile, err)
		}
		
		fmt.Printf("Extracted 512-byte MBR/Boot block from %s to %s\n", disktoolDevice, outFile)

		// Basic Signature Check
		if buf[510] == 0x55 && buf[511] == 0xAA {
			fmt.Println("Valid Boot Signature 0x55AA detected at end of sector.")
		} else {
			fmt.Printf("Warning: Boot signature missing, found 0x%02X%02X\n", buf[510], buf[511])
		}
	},
}

var rewriteBootblockCmd = &cobra.Command{
	Use:   "rewrite-bootblock <backup_bin>",
	Short: "Restores a corrupted boot sector with a provided binary blob",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		requireDevice(cmd)
		backupFile := args[0]

		blob, err := os.ReadFile(backupFile)
		if err != nil {
			log.Fatalf("Failed to read backup binary: %v", err)
		}

		if len(blob) > 4096 {
			log.Fatalf("Protection: Input binary is too large (%d bytes) to be a boot block.", len(blob))
		}

		// Open disk for Writing
		f, err := os.OpenFile(disktoolDevice, os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open device for writing (are you root?): %v", err)
		}
		defer f.Close()

		n, err := f.Write(blob)
		if err != nil {
			log.Fatalf("Failed to write to device: %v", err)
		}

		fmt.Printf("Successfully wrote %d bytes from %s to the boot sector of %s.\n", n, backupFile, disktoolDevice)
	},
}

func init() {
	disktoolCmd.AddCommand(getBootloaderCmd)
	disktoolCmd.AddCommand(rewriteBootblockCmd)
}
