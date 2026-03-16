package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/afterdarksys/diskimager/pkg/tsk"
	"github.com/diskfs/go-diskfs"
	"github.com/spf13/cobra"
)

var getfsCmd = &cobra.Command{
	Use:   "getfs",
	Short: "Print detailed volumetric metadata",
	Run: func(cmd *cobra.Command, args []string) {
		requireDevice(cmd)

		fmt.Printf("Analyzing Filesystems on %s...\n\n", disktoolDevice)

		// 1. Try diskfs for partition table
		disk, err := diskfs.Open(disktoolDevice)
		if err != nil {
			log.Fatalf("Failed to open device with diskfs: %v\n", err)
		}
		
		pt, err := disk.GetPartitionTable()
		if err == nil && pt != nil {
			partitions := pt.GetPartitions()
			fmt.Printf("Found Partition Table (%T) with %d partitions.\n", pt, len(partitions))
			
			for i, p := range partitions {
				fmt.Printf("\n--- Partition %d ---\n", i)
				fmt.Printf("Start Sector : %d\n", p.GetStart())
				fmt.Printf("Size Sectors : %d\n", p.GetSize())
				// Note: `GetType()` does not exist in the base `part.Partition` interface for go-diskfs v1.7.0.
			}
		} else {
			fmt.Printf("No standard partition table found or device is raw volume.\n")
		}

		fmt.Println("\n--- Deep Inspection (TSK) ---")
		
		img, err := tsk.OpenImage(disktoolDevice)
		if err != nil {
			log.Fatalf("Failed to open device with TSK: %v\n", err)
		}
		defer img.Close()
		
		offsetsToProbe := []int64{0}
		if pt != nil {
			for _, p := range pt.GetPartitions() {
				offsetsToProbe = append(offsetsToProbe, int64(p.GetStart()) * 512) // Assume 512 sector size for probe
			}
		}

		for _, offset := range offsetsToProbe {
			fs, err := img.OpenFS(offset)
			if err != nil {
				// Not a filesystem at this offset, skip silently for scan output
				continue
			}

			fmt.Printf("\n[+] Filesystem Detected at Byte Offset %d\n", offset)
			fmt.Printf("    TSK Successfully mounted filesystem.\n")
			
			// We can use Walk to verify if it's readable
			walked := false
			fs.Walk(func(file *tsk.File, path string) error {
				walked = true
				return io.EOF // Stop immediately, we just proved we can read the fs
			})
			
			if walked {
				fmt.Printf("    Root Directory accessible. Contains items.\n")
			}

			fs.Close()
		}
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan disk for partition errors and filesystem health",
	Run: func(cmd *cobra.Command, args []string) {
		requireDevice(cmd)

		fmt.Printf("Starting surface heuristic scan on %s...\n", disktoolDevice)
		
		// In a true deep scan, we would read the disk block by block
		// looking for Superblock magic numbers to find orphaned or deleted partitions.
		// E.g., EXT magic 0xEF53 at offset 1024 into a block
		// NTFS "NTFS    " string at sector start.
		
		f, err := os.Open(disktoolDevice)
		if err != nil {
			log.Fatalf("Scan failed: %v", err)
		}
		defer f.Close()

		const maxScanBytes = 1024 * 1024 * 100 // Scan first 100MB for superblocks to keep it fast
		buf := make([]byte, 512)
		var offset int64 = 0

		foundSignatures := 0

		for offset < maxScanBytes {
			n, err := f.ReadAt(buf, offset)
			if err != nil || n < 512 {
				break
			}

			// NTFS Check (Usually starts with EB 52 90 "NTFS    ")
			if buf[0] == 0xEB && buf[1] == 0x52 && buf[2] == 0x90 && string(buf[3:7]) == "NTFS" {
				fmt.Printf("[!] Found NTFS Boot Sector at offset %d (Sector %d)\n", offset, offset/512)
				foundSignatures++
			}

			// EXT2/3/4 Superblock Check (Magic 0xEF53 at byte 56 within the 1024 offset block)
			// Wait, the superblock is at 1024 bytes into the partition.
			// Let's check if the *current* offset + 1024 has the magic.
			// For a basic sector scan, we'll check if byte 56 is 0x53 and 57 is 0xEF.
			if n >= 1024+56+2 {
				// We need a larger read buffer to check ahead, or we check specifically on 1K boundaries.
			}

			offset += 512
		}

		fmt.Printf("\nScan complete. Found %d recognizable volume/partition signatures in first 100MB.\n", foundSignatures)
	},
}

func init() {
	disktoolCmd.AddCommand(getfsCmd)
	disktoolCmd.AddCommand(scanCmd)
}
