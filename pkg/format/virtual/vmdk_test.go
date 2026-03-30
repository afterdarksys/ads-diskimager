package virtual

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVMDKWriter(t *testing.T) {
	tmpDir := t.TempDir()
	vmdkPath := filepath.Join(tmpDir, "test.vmdk")
	diskSize := int64(10 * 1024 * 1024) // 10MB

	// Create VMDK writer
	writer, err := NewVMDKWriter(vmdkPath, diskSize)
	if err != nil {
		t.Fatalf("Failed to create VMDK writer: %v", err)
	}

	// Write some data
	testData := make([]byte, 1024*1024) // 1MB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Close writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify descriptor file exists
	if _, err := os.Stat(vmdkPath); os.IsNotExist(err) {
		t.Error("Descriptor file was not created")
	}

	// Verify flat file exists
	flatPath := strings.TrimSuffix(vmdkPath, ".vmdk") + "-flat.vmdk"
	if _, err := os.Stat(flatPath); os.IsNotExist(err) {
		t.Error("Flat file was not created")
	}

	// Read and verify descriptor
	descriptorContent, err := os.ReadFile(vmdkPath)
	if err != nil {
		t.Fatalf("Failed to read descriptor: %v", err)
	}

	descriptor := string(descriptorContent)
	if !strings.Contains(descriptor, "# Disk DescriptorFile") {
		t.Error("Descriptor missing header")
	}
	if !strings.Contains(descriptor, "createType=\"monolithicFlat\"") {
		t.Error("Descriptor missing createType")
	}
	if !strings.Contains(descriptor, "ddb.geometry.cylinders") {
		t.Error("Descriptor missing geometry")
	}
}

func TestVHDWriter(t *testing.T) {
	tmpDir := t.TempDir()
	vhdPath := filepath.Join(tmpDir, "test.vhd")
	diskSize := int64(10 * 1024 * 1024) // 10MB

	// Create VHD writer
	writer, err := NewVHDWriter(vhdPath, diskSize)
	if err != nil {
		t.Fatalf("Failed to create VHD writer: %v", err)
	}

	// Write some data
	testData := make([]byte, 1024*1024) // 1MB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Close writer (writes footer)
	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify VHD file exists
	stat, err := os.Stat(vhdPath)
	if os.IsNotExist(err) {
		t.Error("VHD file was not created")
	}

	// Verify footer was written (file should be data + 512 byte footer)
	expectedSize := int64(len(testData) + 512)
	if stat.Size() != expectedSize {
		t.Errorf("Expected VHD size %d, got %d", expectedSize, stat.Size())
	}

	// Read footer and verify magic
	vhdData, err := os.ReadFile(vhdPath)
	if err != nil {
		t.Fatalf("Failed to read VHD: %v", err)
	}

	// Check for "conectix" magic in footer
	footerStart := len(vhdData) - 512
	if footerStart < 0 {
		t.Fatal("VHD file too small for footer")
	}

	footer := vhdData[footerStart:]
	magic := string(footer[0:8])
	if magic != "conectix" {
		t.Errorf("Expected 'conectix' magic, got '%s'", magic)
	}
}
