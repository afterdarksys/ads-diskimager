package geometry

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// DiskGeometry represents disk CHS geometry
type DiskGeometry struct {
	Cylinders   uint32 `json:"cylinders"`
	Heads       uint32 `json:"heads"`
	Sectors     uint32 `json:"sectors"`
	BytesPerSec uint32 `json:"bytes_per_sector"`
	TotalSize   uint64 `json:"total_size"`
}

// GetGeometry attempts to retrieve disk geometry from device
func GetGeometry(devicePath string) (*DiskGeometry, error) {
	switch runtime.GOOS {
	case "linux":
		return getGeometryLinux(devicePath)
	case "darwin":
		return getGeometryMacOS(devicePath)
	case "windows":
		return getGeometryWindows(devicePath)
	default:
		return nil, fmt.Errorf("geometry detection not supported on %s", runtime.GOOS)
	}
}

// getGeometryLinux retrieves geometry on Linux using fdisk
func getGeometryLinux(devicePath string) (*DiskGeometry, error) {
	cmd := exec.Command("fdisk", "-l", devicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fdisk failed: %w", err)
	}

	geom := &DiskGeometry{
		BytesPerSec: 512, // Default
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for "Disk /dev/sda: 465.76 GiB, 500107862016 bytes"
		if strings.HasPrefix(line, "Disk "+devicePath) {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				bytesStr := strings.TrimSpace(parts[1])
				bytesStr = strings.TrimSuffix(bytesStr, " bytes")
				if size, err := strconv.ParseUint(bytesStr, 10, 64); err == nil {
					geom.TotalSize = size
				}
			}
		}

		// Look for geometry line: "255 heads, 63 sectors/track, 60801 cylinders"
		if strings.Contains(line, "heads") && strings.Contains(line, "sectors") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "heads," && i > 0 {
					if heads, err := strconv.ParseUint(fields[i-1], 10, 32); err == nil {
						geom.Heads = uint32(heads)
					}
				}
				if field == "sectors/track," && i > 0 {
					if sectors, err := strconv.ParseUint(fields[i-1], 10, 32); err == nil {
						geom.Sectors = uint32(sectors)
					}
				}
				if field == "cylinders" && i > 0 {
					cylStr := strings.TrimSuffix(fields[i-1], ",")
					if cyls, err := strconv.ParseUint(cylStr, 10, 32); err == nil {
						geom.Cylinders = uint32(cyls)
					}
				}
			}
		}
	}

	// If we didn't get geometry, calculate it
	if geom.Heads == 0 || geom.Sectors == 0 {
		geom.Cylinders, geom.Heads, geom.Sectors = calculateCHS(geom.TotalSize)
	}

	return geom, nil
}

// getGeometryMacOS retrieves geometry on macOS using diskutil
func getGeometryMacOS(devicePath string) (*DiskGeometry, error) {
	cmd := exec.Command("diskutil", "info", devicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("diskutil failed: %w", err)
	}

	geom := &DiskGeometry{
		BytesPerSec: 512,
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Disk Size:") {
			// Parse "Disk Size: 500.1 GB (500107862016 Bytes)"
			if idx := strings.Index(line, "("); idx != -1 {
				bytesStr := line[idx+1:]
				bytesStr = strings.TrimSuffix(bytesStr, " Bytes)")
				bytesStr = strings.TrimSpace(bytesStr)
				if size, err := strconv.ParseUint(bytesStr, 10, 64); err == nil {
					geom.TotalSize = size
				}
			}
		}
	}

	// macOS doesn't typically expose CHS, calculate it
	geom.Cylinders, geom.Heads, geom.Sectors = calculateCHS(geom.TotalSize)

	return geom, nil
}

// getGeometryWindows retrieves geometry on Windows using fsutil
func getGeometryWindows(devicePath string) (*DiskGeometry, error) {
	// Extract physical drive number
	driveNum := strings.TrimPrefix(devicePath, `\\.\PhysicalDrive`)

	cmd := exec.Command("wmic", "diskdrive", "where", fmt.Sprintf("DeviceID='\\\\\\\\.\\\\PhysicalDrive%s'", driveNum), "get", "BytesPerSector,SectorsPerTrack,TotalCylinders,TotalHeads,TotalSectors,TotalTracks,Size", "/format:list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wmic failed: %w", err)
	}

	geom := &DiskGeometry{}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "BytesPerSector=") {
			if val, err := strconv.ParseUint(strings.TrimPrefix(line, "BytesPerSector="), 10, 32); err == nil {
				geom.BytesPerSec = uint32(val)
			}
		} else if strings.HasPrefix(line, "SectorsPerTrack=") {
			if val, err := strconv.ParseUint(strings.TrimPrefix(line, "SectorsPerTrack="), 10, 32); err == nil {
				geom.Sectors = uint32(val)
			}
		} else if strings.HasPrefix(line, "TotalCylinders=") {
			if val, err := strconv.ParseUint(strings.TrimPrefix(line, "TotalCylinders="), 10, 32); err == nil {
				geom.Cylinders = uint32(val)
			}
		} else if strings.HasPrefix(line, "TotalHeads=") {
			if val, err := strconv.ParseUint(strings.TrimPrefix(line, "TotalHeads="), 10, 32); err == nil {
				geom.Heads = uint32(val)
			}
		} else if strings.HasPrefix(line, "Size=") {
			if val, err := strconv.ParseUint(strings.TrimPrefix(line, "Size="), 10, 64); err == nil {
				geom.TotalSize = val
			}
		}
	}

	return geom, nil
}

// calculateCHS calculates CHS values from total disk size
func calculateCHS(totalBytes uint64) (cylinders, heads, sectors uint32) {
	// Standard defaults
	heads = 255
	sectors = 63

	// Calculate cylinders
	totalSectors := totalBytes / 512
	cylinders = uint32(totalSectors / uint64(heads*sectors))

	if cylinders == 0 {
		cylinders = 1
	}

	// Limit to max CHS values
	if cylinders > 1024 {
		// For large disks, use LBA mode (geometry is less relevant)
		cylinders = 1024
	}

	return cylinders, heads, sectors
}
