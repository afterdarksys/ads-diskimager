package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	// DefaultPIDDir is the default directory for PID files
	DefaultPIDDir = "/var/run"
)

// PIDFile represents a process ID file
type PIDFile struct {
	Path string
	PID  int
}

// CreatePIDFile creates a PID file for the current process
func CreatePIDFile(path string) (*PIDFile, error) {
	// Get current process ID
	pid := os.Getpid()

	// Check if PID file already exists
	if err := checkStalePIDFile(path); err != nil {
		return nil, err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create PID directory: %w", err)
	}

	// Write PID to file
	pidStr := fmt.Sprintf("%d\n", pid)
	if err := os.WriteFile(path, []byte(pidStr), 0644); err != nil {
		return nil, fmt.Errorf("failed to write PID file: %w", err)
	}

	return &PIDFile{
		Path: path,
		PID:  pid,
	}, nil
}

// checkStalePIDFile checks if a PID file exists and if the process is still running
func checkStalePIDFile(path string) error {
	// Check if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, OK to proceed
		}
		return fmt.Errorf("failed to read existing PID file: %w", err)
	}

	// Parse PID
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// Invalid PID file, remove it
		os.Remove(path)
		return nil
	}

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist, remove stale PID file
		os.Remove(path)
		return nil
	}

	// Try to send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist, remove stale PID file
		os.Remove(path)
		return nil
	}

	// Process is running
	return fmt.Errorf("another instance is already running (PID: %d)", pid)
}

// RemovePIDFile removes a PID file
func RemovePIDFile(path string) error {
	if path == "" {
		return nil
	}

	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}

// ReadPIDFile reads a PID from a file
func ReadPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// IsProcessRunning checks if a process with the given PID is running
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GetDefaultPIDPath returns the default PID file path for a service
func GetDefaultPIDPath(serviceName string) string {
	// Try /var/run first (standard location)
	if isWritable(DefaultPIDDir) {
		return filepath.Join(DefaultPIDDir, fmt.Sprintf("%s.pid", serviceName))
	}

	// Fall back to /tmp if /var/run is not writable
	return filepath.Join("/tmp", fmt.Sprintf("%s.pid", serviceName))
}

// isWritable checks if a directory is writable
func isWritable(path string) bool {
	// Try to create a temporary file
	testFile := filepath.Join(path, ".write-test")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		return false
	}
	os.Remove(testFile)
	return true
}
