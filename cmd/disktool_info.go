package cmd

import (
	"fmt"
	"strings"

	"github.com/diskfs/go-diskfs"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get Volume/Partition and FS Info (blkid/diskutil equivalent)",
	Run: func(cmd *cobra.Command, args []string) {
		requireDevice(cmd)
		targetPath := disktoolDevice // comes from root disktool scope

		fmt.Printf("Analyzing %s...\n\n", targetPath)
		report, err := GetDiskInfo(targetPath)
		if err != nil {
			fmt.Printf("Error analyzing disk: %v\n", err)
			return
		}
		fmt.Println(report)
	},
}

func init() {
	disktoolCmd.AddCommand(infoCmd)
}

// GetDiskInfo opens the raw block device or image file, parses the
// partition table (MBR/GPT) natively in Go, and attempts to identify filesystems.
func GetDiskInfo(path string) (string, error) {
	// Open the disk read-only
	disk, err := diskfs.Open(path, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return "", fmt.Errorf("failed to open device/image: %w", err)
	}
	defer disk.Close()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Disk: %s\n", path))
	sb.WriteString(fmt.Sprintf("Logical Sector Size: %d bytes\n", disk.LogicalBlocksize))
	sb.WriteString(fmt.Sprintf("Total Size: %d bytes (%.2f GB)\n\n", disk.Size, float64(disk.Size)/(1024*1024*1024)))

	// Attempt to get the partition table
	pt, err := disk.GetPartitionTable()
	if err != nil {
		// If no partition table, try to see if it's a raw filesystem
		sb.WriteString("Partition Table: None (or unrecognizable)\n")
		fs, fsErr := disk.GetFilesystem(0)
		if fsErr == nil {
			sb.WriteString(fmt.Sprintf("Raw Filesystem Detected: %v\n", fs.Type()))
		} else {
			sb.WriteString("No raw filesystem detected.\n")
		}
		return sb.String(), nil
	}

	sb.WriteString(fmt.Sprintf("Partition Table Type: %s\n\n", pt.Type()))
	partitions := pt.GetPartitions()
	
	if len(partitions) == 0 {
		sb.WriteString("No partitions found.\n")
		return sb.String(), nil
	}

	for i, part := range partitions {
		sb.WriteString(fmt.Sprintf("Partition %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  Start: %d\n", part.GetStart()))
		sb.WriteString(fmt.Sprintf("  Size:  %d bytes (%.2f GB)\n", part.GetSize(), float64(part.GetSize())/(1024*1024*1024)))
		sb.WriteString(fmt.Sprintf("  UUID:  %s\n", part.UUID()))

		// Try to identify the filesystem in this partition
		fs, fsErr := disk.GetFilesystem(i + 1) // 1-indexed for diskfs
		if fsErr == nil {
			sb.WriteString(fmt.Sprintf("  Filesystem: %v\n", fs.Type()))
		} else {
			sb.WriteString(fmt.Sprintf("  Filesystem: Unknown FS or could not parse (%v)\n", fsErr))
		}
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String()), nil
}
