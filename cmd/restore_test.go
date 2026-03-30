package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreSafetyChecks(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create test image
	testImage := filepath.Join(tmpDir, "test.img")
	testData := make([]byte, 1024*1024) // 1MB
	if err := os.WriteFile(testImage, testData, 0600); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Create destination file (smaller than source)
	testDest := filepath.Join(tmpDir, "dest.img")
	smallData := make([]byte, 512*1024) // 512KB
	if err := os.WriteFile(testDest, smallData, 0600); err != nil {
		t.Fatalf("Failed to create destination: %v", err)
	}

	t.Run("RejectsNonExistentImage", func(t *testing.T) {
		err := performRestoreSafetyChecks("/nonexistent.img", testDest, false)
		if err == nil {
			t.Error("Expected error for non-existent image")
		}
	})

	t.Run("RejectsSystemDisk", func(t *testing.T) {
		systemDisks := []string{"/dev/sda", "/dev/disk0", "/dev/nvme0n1"}
		for _, disk := range systemDisks {
			err := performRestoreSafetyChecks(testImage, disk, false)
			if err == nil {
				t.Errorf("Expected error for system disk %s", disk)
			}
		}
	})

	t.Run("AllowsForceOverride", func(t *testing.T) {
		err := performRestoreSafetyChecks(testImage, "/dev/sda", true)
		if err != nil {
			t.Errorf("Force flag should bypass safety checks: %v", err)
		}
	})
}

func TestGetDeviceSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")

	// Create 1MB test file
	testData := make([]byte, 1024*1024)
	if err := os.WriteFile(testFile, testData, 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size, err := getDeviceSize(testFile)
	if err != nil {
		t.Errorf("Failed to get device size: %v", err)
	}

	expectedSize := int64(1024 * 1024)
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestIsDeviceMounted(t *testing.T) {
	// Test with non-existent device (should return false, no error)
	mounted, err := isDeviceMounted("/dev/nonexistent999")
	if err != nil {
		// Platform-specific, error is acceptable
		t.Logf("Mount detection returned error (acceptable): %v", err)
	}
	if mounted {
		t.Error("Non-existent device should not be mounted")
	}
}
