package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileCarver(t *testing.T) {
	// Create a temporary file to act as our "disk image"
	tmpImage, err := os.CreateTemp("", "disktool_carver_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp image: %v", err)
	}
	defer os.Remove(tmpImage.Name())

	// Write some junk data
	junk := []byte("junk_data_preceding_file")
	tmpImage.Write(junk)

	// Write a fake JPEG file (starts with FFD8FFE0)
	fakeJPEG := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte("...jpeg_content...")...)
	tmpImage.Write(fakeJPEG)

	// Write more junk
	tmpImage.Write([]byte("more_junk"))

	tmpImage.Close()

	// Create a temp dir for recovered files
	outDir, err := os.MkdirTemp("", "recovered_files_*")
	if err != nil {
		t.Fatalf("Failed to create temp out dir: %v", err)
	}
	defer os.RemoveAll(outDir)

	// Run carver
	recoveredCount := 0
	err = FileCarver(tmpImage.Name(), outDir, func(offset int64, fileType string) {
		recoveredCount++
		t.Logf("Recovered %s at offset %d", fileType, offset)
		if offset != int64(len(junk)) {
			t.Errorf("Expected offset %d, got %d", len(junk), offset)
		}
		if fileType != ".jpg" {
			t.Errorf("Expected .jpg fileType, got %s", fileType)
		}
	})

	if err != nil {
		t.Fatalf("FileCarver failed: %v", err)
	}

	if recoveredCount != 1 {
		t.Fatalf("Expected to recover 1 file, got %d", recoveredCount)
	}

	// Verify the file was written
	files, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("Failed to read out dir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Expected 1 file in out dir, got %d", len(files))
	}

	recoveredPath := filepath.Join(outDir, files[0].Name())
	recoveredData, err := os.ReadFile(recoveredPath)
	if err != nil {
		t.Fatalf("Failed to read recovered file: %v", err)
	}

	// The carver extracts up to 5MB, so it will contain the jpeg + the trailing junk
	if !bytes.HasPrefix(recoveredData, fakeJPEG) {
		t.Errorf("Recovered data does not match the fake JPEG signature")
	}
}
