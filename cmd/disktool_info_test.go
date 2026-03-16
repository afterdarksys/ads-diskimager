package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/stretchr/testify/assert"
)

func TestGetDiskInfo(t *testing.T) {
	tmpDir := t.TempDir()
	imgFile := filepath.Join(tmpDir, "test.img")

	// Create a mock disk image (10MB)
	var size int64 = 10 * 1024 * 1024
	disk, err := diskfs.Create(imgFile, size, diskfs.SectorSizeDefault)
	assert.NoError(t, err)

	// Since it has no partitions or filesystem, it should say "No raw filesystem detected"
	disk.Close()

	report, err := GetDiskInfo(imgFile)
	assert.NoError(t, err)

	assert.Contains(t, report, "Logical Sector Size")
	assert.Contains(t, report, "Total Size")
	assert.Contains(t, report, "Partition Table: None")
	assert.Contains(t, report, "No raw filesystem detected")
}

func TestGetDiskInfo_InvalidFile(t *testing.T) {
	_, err := GetDiskInfo("/path/to/nonexistent/file.img")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to open device/image") || strings.Contains(err.Error(), "no such file"))
}
