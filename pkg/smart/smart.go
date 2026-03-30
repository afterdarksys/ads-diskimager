package smart

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// DiskInfo holds SMART and disk identification data
type DiskInfo struct {
	DevicePath   string            `json:"device_path"`
	Model        string            `json:"model"`
	Serial       string            `json:"serial"`
	Firmware     string            `json:"firmware"`
	Capacity     string            `json:"capacity"`
	SMARTStatus  string            `json:"smart_status"`
	Temperature  string            `json:"temperature,omitempty"`
	PowerOnHours string            `json:"power_on_hours,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Available    bool              `json:"available"`
	Error        string            `json:"error,omitempty"`
}

// CollectDiskInfo gathers SMART and identification data from a device
func CollectDiskInfo(devicePath string) *DiskInfo {
	info := &DiskInfo{
		DevicePath: devicePath,
		Attributes: make(map[string]string),
	}

	// Try smartctl first (most reliable cross-platform)
	if err := collectSmartctl(devicePath, info); err == nil {
		info.Available = true
		return info
	}

	// Fallback to platform-specific tools
	switch runtime.GOOS {
	case "darwin":
		if err := collectDiskutilMacOS(devicePath, info); err == nil {
			info.Available = true
			return info
		}
	case "linux":
		if err := collectLinuxDiskInfo(devicePath, info); err == nil {
			info.Available = true
			return info
		}
	case "windows":
		if err := collectWindowsDiskInfo(devicePath, info); err == nil {
			info.Available = true
			return info
		}
	}

	info.Available = false
	info.Error = "SMART data not available (install smartmontools)"
	return info
}

// collectSmartctl uses smartctl to gather SMART data
func collectSmartctl(devicePath string, info *DiskInfo) error {
	// Check if smartctl is available
	if _, err := exec.LookPath("smartctl"); err != nil {
		return fmt.Errorf("smartctl not found")
	}

	// Get basic info
	cmd := exec.Command("smartctl", "-i", devicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("smartctl failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Device Model:") {
			info.Model = strings.TrimSpace(strings.TrimPrefix(line, "Device Model:"))
		} else if strings.HasPrefix(line, "Serial Number:") {
			info.Serial = strings.TrimSpace(strings.TrimPrefix(line, "Serial Number:"))
		} else if strings.HasPrefix(line, "Firmware Version:") {
			info.Firmware = strings.TrimSpace(strings.TrimPrefix(line, "Firmware Version:"))
		} else if strings.HasPrefix(line, "User Capacity:") {
			info.Capacity = strings.TrimSpace(strings.TrimPrefix(line, "User Capacity:"))
		}
	}

	// Get SMART health
	cmd = exec.Command("smartctl", "-H", devicePath)
	output, _ = cmd.CombinedOutput()
	if strings.Contains(string(output), "PASSED") {
		info.SMARTStatus = "PASSED"
	} else if strings.Contains(string(output), "FAILED") {
		info.SMARTStatus = "FAILED"
	} else {
		info.SMARTStatus = "UNKNOWN"
	}

	// Get detailed attributes
	cmd = exec.Command("smartctl", "-A", devicePath)
	output, _ = cmd.CombinedOutput()
	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Temperature") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				info.Temperature = fields[9] + "°C"
			}
		} else if strings.Contains(line, "Power_On_Hours") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				info.PowerOnHours = fields[9] + " hours"
			}
		} else if strings.Contains(line, "Reallocated_Sector") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				info.Attributes["Reallocated_Sectors"] = fields[9]
			}
		} else if strings.Contains(line, "Current_Pending_Sector") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				info.Attributes["Pending_Sectors"] = fields[9]
			}
		}
	}

	return nil
}

// collectDiskutilMacOS uses macOS diskutil for disk info
func collectDiskutilMacOS(devicePath string, info *DiskInfo) error {
	cmd := exec.Command("diskutil", "info", devicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("diskutil failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Device / Media Name:") {
			info.Model = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Disk Size:") {
			info.Capacity = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "SMART Status:") {
			info.SMARTStatus = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}

	info.Serial = "N/A (use smartctl for serial)"
	return nil
}

// collectLinuxDiskInfo uses Linux sysfs and hdparm
func collectLinuxDiskInfo(devicePath string, info *DiskInfo) error {
	// Try hdparm for disk info
	cmd := exec.Command("hdparm", "-I", devicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hdparm failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Model Number:") {
			info.Model = strings.TrimSpace(strings.TrimPrefix(line, "Model Number:"))
		} else if strings.HasPrefix(line, "Serial Number:") {
			info.Serial = strings.TrimSpace(strings.TrimPrefix(line, "Serial Number:"))
		} else if strings.HasPrefix(line, "Firmware Revision:") {
			info.Firmware = strings.TrimSpace(strings.TrimPrefix(line, "Firmware Revision:"))
		}
	}

	// Get size from blockdev
	cmd = exec.Command("blockdev", "--getsize64", devicePath)
	output, _ = cmd.CombinedOutput()
	info.Capacity = strings.TrimSpace(string(output)) + " bytes"

	info.SMARTStatus = "Unknown (install smartmontools)"
	return nil
}

// collectWindowsDiskInfo uses Windows WMIC
func collectWindowsDiskInfo(devicePath string, info *DiskInfo) error {
	// Extract physical drive number from path
	driveNum := strings.TrimPrefix(devicePath, `\\.\PhysicalDrive`)

	cmd := exec.Command("wmic", "diskdrive", "where", fmt.Sprintf("DeviceID='\\\\\\\\.\\\\PhysicalDrive%s'", driveNum), "get", "Model,SerialNumber,Size,Status", "/format:list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wmic failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Model=") {
			info.Model = strings.TrimPrefix(line, "Model=")
		} else if strings.HasPrefix(line, "SerialNumber=") {
			info.Serial = strings.TrimPrefix(line, "SerialNumber=")
		} else if strings.HasPrefix(line, "Size=") {
			info.Capacity = strings.TrimPrefix(line, "Size=") + " bytes"
		} else if strings.HasPrefix(line, "Status=") {
			info.SMARTStatus = strings.TrimPrefix(line, "Status=")
		}
	}

	return nil
}

// IsWriteProtected attempts to detect if a device has a hardware write-blocker
func IsWriteProtected(devicePath string) (bool, error) {
	switch runtime.GOOS {
	case "linux":
		return isWriteProtectedLinux(devicePath)
	case "darwin":
		return isWriteProtectedMacOS(devicePath)
	case "windows":
		return isWriteProtectedWindows(devicePath)
	}
	return false, fmt.Errorf("write-blocker detection not implemented for %s", runtime.GOOS)
}

// isWriteProtectedLinux checks if device is read-only on Linux
func isWriteProtectedLinux(devicePath string) (bool, error) {
	// Check /sys/block/*/ro
	deviceName := strings.TrimPrefix(devicePath, "/dev/")
	roPath := fmt.Sprintf("/sys/block/%s/ro", deviceName)

	cmd := exec.Command("cat", roPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("cannot read ro flag: %w", err)
	}

	return strings.TrimSpace(string(output)) == "1", nil
}

// isWriteProtectedMacOS checks if device is read-only on macOS
func isWriteProtectedMacOS(devicePath string) (bool, error) {
	cmd := exec.Command("diskutil", "info", devicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	// Look for "Read-Only Media: Yes"
	return strings.Contains(string(output), "Read-Only Media:     Yes"), nil
}

// isWriteProtectedWindows checks if device is read-only on Windows
func isWriteProtectedWindows(devicePath string) (bool, error) {
	// Windows write-blocker detection would require admin privileges
	// and checking disk properties via DeviceIoControl
	return false, fmt.Errorf("write-blocker detection requires admin privileges on Windows")
}
