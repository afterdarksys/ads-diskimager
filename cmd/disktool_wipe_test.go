package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestWipeDrive(t *testing.T) {
	// Create a temporary file with known data
	tmpFile, err := os.CreateTemp("", "disktool_wipe_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	data := []byte("secret_data_to_be_wiped")
	if _, err := tmpFile.Write(data); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Wipe the file (1 pass for speed in test)
	err = WipeDrive(tmpFile.Name(), 1, nil)
	if err != nil {
		t.Fatalf("WipeDrive failed: %v", err)
	}

	// Verify the file was overwritten (assuming pass 1 is zeros)
	wipedData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read wiped file: %v", err)
	}

	if bytes.Equal(data, wipedData) {
		t.Errorf("File data was not wiped")
	}

	for _, b := range wipedData {
		if b != 0x00 { // We know pass 1 writes 0x00
			t.Errorf("Expected all zeros after 1 wipe pass, got byte: 0x%02x", b)
			break
		}
	}
}
