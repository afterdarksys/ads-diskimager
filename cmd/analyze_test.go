package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

func TestAnalyzeFAT32(t *testing.T) {
	// 1. Create a dummy disk image
	imgFile := "test_fat32.img"
	diskSize := int64(10 * 1024 * 1024) // 10MB
	defer os.Remove(imgFile)

	myDisk, err := diskfs.Create(imgFile, diskSize, diskfs.SectorSizeDefault)
	if err != nil {
		t.Fatalf("Failed to create disk: %v", err)
	}

	// 2. Partition it (GPT)
	table := &gpt.Table{
		Partitions: []*gpt.Partition{
			{
				Start: 2048,
				End:   uint64(diskSize/512) - 1,
				Type:  gpt.Type("ebd0a0a2-b9e5-4433-87c0-68b6b72699c7"), // Basic Data GUID
				Name:  "Test",
			},
		},
	}
	if err := myDisk.Partition(table); err != nil {
		t.Fatalf("Failed to partition: %v", err)
	}

	// 3. Format as FAT32
	// Re-open to pick up partition
	myDisk.Close()
	myDisk, err = diskfs.Open(imgFile)
	if err != nil {
		t.Fatalf("Failed to re-open disk: %v", err)
	}
	
	fs, err := myDisk.CreateFilesystem(disk.FilesystemSpec{Partition: 1, FSType: filesystem.TypeFat32, VolumeLabel: "TEST"})
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	// 4. Write a file
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	content := []byte("Hello World")
	f.Write(content)
	f.Close()

	// 5. Run Analyze (by calling the logic or CLI)
	// We'll invoke the global logic via the variable for simplicity in this test
	analyzeInput = imgFile
	// Redirect stdout/stderr to avoid clutter
	// In a real test we'd extract the run function to be more testable without globals
	
	// We just check if it crashes for now, and check if json exists
	defer os.Remove("system_hashes.json")
	
	// Manually run the body of what analyzeCmd.Run would do, or just Run it
	analyzeCmd.Run(analyzeCmd, []string{})

	// 6. Verify JSON
	jsonContent, err := os.ReadFile("system_hashes.json")
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}
	fmt.Printf("Report:\n%s\n", string(jsonContent))
	
	// TODO: Assert hash matches "Hello World" sha256
}
