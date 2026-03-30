package extractor

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// FileInfo represents a file found in the filesystem
type FileInfo struct {
	Path        string
	Size        int64
	ModTime     time.Time
	Permissions string
	Inode       uint64
}

// Extractor handles file extraction from disk images
type Extractor struct {
	imagePath string
	fsType    string
	mountDir  string // For mount-based extraction
	useTSK    bool   // Use The Sleuth Kit if available
}

// NewExtractor creates a new file extractor
func NewExtractor(imagePath, fsType string) (*Extractor, error) {
	// Check if file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("image file not found: %s", imagePath)
	}

	ext := &Extractor{
		imagePath: imagePath,
		fsType:    fsType,
	}

	// Check if TSK is available
	if _, err := exec.LookPath("fls"); err == nil {
		ext.useTSK = true
	}

	return ext, nil
}

// DetectFilesystem attempts to detect the filesystem type
func DetectFilesystem(imagePath string) (string, error) {
	// Try using file command first
	cmd := exec.Command("file", "-s", imagePath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		result := strings.ToLower(string(output))

		if strings.Contains(result, "ext4") {
			return "ext4", nil
		} else if strings.Contains(result, "ext3") {
			return "ext3", nil
		} else if strings.Contains(result, "ext2") {
			return "ext2", nil
		} else if strings.Contains(result, "ntfs") {
			return "ntfs", nil
		} else if strings.Contains(result, "fat32") || strings.Contains(result, "vfat") {
			return "fat32", nil
		} else if strings.Contains(result, "fat16") {
			return "fat16", nil
		} else if strings.Contains(result, "apfs") {
			return "apfs", nil
		} else if strings.Contains(result, "hfs+") || strings.Contains(result, "hfsplus") {
			return "hfsplus", nil
		}
	}

	// Try using fsstat from TSK
	cmd = exec.Command("fsstat", imagePath)
	output, err = cmd.CombinedOutput()
	if err == nil {
		result := strings.ToLower(string(output))
		if strings.Contains(result, "ext4") {
			return "ext4", nil
		} else if strings.Contains(result, "ntfs") {
			return "ntfs", nil
		} else if strings.Contains(result, "fat") {
			return "fat32", nil
		}
	}

	return "", fmt.Errorf("unable to detect filesystem type")
}

// FindFiles searches for files matching patterns
func (e *Extractor) FindFiles(patterns []string, recursive bool) ([]FileInfo, error) {
	if e.useTSK {
		return e.findFilesWithTSK(patterns, recursive)
	}
	return e.findFilesWithMount(patterns, recursive)
}

// findFilesWithTSK uses The Sleuth Kit for file listing
func (e *Extractor) findFilesWithTSK(patterns []string, recursive bool) ([]FileInfo, error) {
	var results []FileInfo

	// Use fls to list files
	args := []string{"-r", "-p"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, e.imagePath)

	cmd := exec.Command("fls", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fls failed: %w\nOutput: %s", err, output)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse fls output: "r/r 123: /path/to/file"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		filePath := strings.TrimSpace(parts[1])
		if filePath == "" || filePath == "/" {
			continue
		}

		// Check if matches any pattern
		if matchesAnyPattern(filePath, patterns) {
			// Get file details
			fileInfo := FileInfo{
				Path: filePath,
			}

			// Try to get inode from first part
			inodePart := strings.TrimSpace(parts[0])
			fields := strings.Fields(inodePart)
			if len(fields) >= 2 {
				fmt.Sscanf(fields[1], "%d", &fileInfo.Inode)
			}

			results = append(results, fileInfo)
		}
	}

	// Get detailed info for each file
	for i := range results {
		if results[i].Inode > 0 {
			e.getFileDetailsWithTSK(&results[i])
		}
	}

	return results, nil
}

// getFileDetailsWithTSK gets detailed file information
func (e *Extractor) getFileDetailsWithTSK(file *FileInfo) error {
	// Use istat to get file details
	cmd := exec.Command("istat", e.imagePath, fmt.Sprintf("%d", file.Inode))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "size:") {
			fmt.Sscanf(line, "size: %d", &file.Size)
		} else if strings.HasPrefix(line, "Modified:") {
			// Parse timestamp
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				timeStr := strings.TrimSpace(parts[1])
				// TSK format: "2024-01-15 10:30:45 (EST)"
				t, err := time.Parse("2006-01-02 15:04:05", timeStr[:19])
				if err == nil {
					file.ModTime = t
				}
			}
		}
	}

	return nil
}

// findFilesWithMount uses temporary mount for file listing
func (e *Extractor) findFilesWithMount(patterns []string, recursive bool) ([]FileInfo, error) {
	// Create temporary mount point
	tmpDir, err := os.MkdirTemp("", "diskimager-mount-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Mount the image
	var mountCmd *exec.Cmd
	switch e.fsType {
	case "ext4", "ext3", "ext2":
		mountCmd = exec.Command("mount", "-o", "ro,loop", e.imagePath, tmpDir)
	case "ntfs":
		mountCmd = exec.Command("mount", "-t", "ntfs-3g", "-o", "ro", e.imagePath, tmpDir)
	case "fat32", "vfat":
		mountCmd = exec.Command("mount", "-t", "vfat", "-o", "ro", e.imagePath, tmpDir)
	default:
		return nil, fmt.Errorf("unsupported filesystem for mount: %s (try using TSK)", e.fsType)
	}

	if err := mountCmd.Run(); err != nil {
		return nil, fmt.Errorf("mount failed: %w (may need sudo)", err)
	}

	// Ensure umount happens even if panic occurs
	mounted := true
	defer func() {
		if mounted {
			if r := recover(); r != nil {
				// Unmount before re-panicking
				exec.Command("umount", tmpDir).Run()
				panic(r) // Re-throw panic after cleanup
			}
			exec.Command("umount", tmpDir).Run()
		}
	}()

	// Walk the filesystem
	var results []FileInfo
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, _ := filepath.Rel(tmpDir, path)
		relPath = "/" + relPath

		// Check patterns
		if matchesAnyPattern(relPath, patterns) {
			results = append(results, FileInfo{
				Path:    relPath,
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}

		return nil
	})

	// Mark as unmounted (will happen in defer)
	mounted = true

	if err != nil {
		return nil, err
	}

	return results, nil
}

// ExtractFile extracts a single file's content
func (e *Extractor) ExtractFile(file FileInfo) ([]byte, error) {
	if e.useTSK && file.Inode > 0 {
		return e.extractFileWithTSK(file)
	}
	// extractFileWithMount returns ([]byte, error)
	data, err := e.extractFileWithMount(file)
	return data, err
}

// extractFileWithTSK uses icat to extract file content
// Uses streaming to avoid loading entire file into memory
func (e *Extractor) extractFileWithTSK(file FileInfo) ([]byte, error) {
	cmd := exec.Command("icat", e.imagePath, fmt.Sprintf("%d", file.Inode))

	// Use StdoutPipe to stream data instead of loading all into memory
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("icat failed to start: %w", err)
	}

	// Read in chunks to limit memory usage
	const maxChunkSize = 4 * 1024 * 1024 // 4MB chunks
	var result []byte
	buf := make([]byte, maxChunkSize)

	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			cmd.Wait() // Clean up process
			return nil, fmt.Errorf("failed to read from icat: %w", err)
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("icat process failed: %w", err)
	}

	return result, nil
}

// extractFileWithMount extracts file from mounted filesystem
func (e *Extractor) extractFileWithMount(file FileInfo) ([]byte, error) {
	// Create temporary mount point
	tmpDir, err := os.MkdirTemp("", "diskimager-mount-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Mount the image
	var mountCmd *exec.Cmd
	switch e.fsType {
	case "ext4", "ext3", "ext2":
		mountCmd = exec.Command("mount", "-o", "ro,loop", e.imagePath, tmpDir)
	case "ntfs":
		mountCmd = exec.Command("mount", "-t", "ntfs-3g", "-o", "ro", e.imagePath, tmpDir)
	case "fat32", "vfat":
		mountCmd = exec.Command("mount", "-t", "vfat", "-o", "ro", e.imagePath, tmpDir)
	default:
		return nil, fmt.Errorf("unsupported filesystem: %s", e.fsType)
	}

	if err := mountCmd.Run(); err != nil {
		return nil, fmt.Errorf("mount failed: %w", err)
	}

	// Ensure umount happens even if panic occurs
	mounted := true
	defer func() {
		if mounted {
			if r := recover(); r != nil {
				// Unmount before re-panicking
				exec.Command("umount", tmpDir).Run()
				panic(r) // Re-throw panic after cleanup
			}
			exec.Command("umount", tmpDir).Run()
		}
	}()

	// Read file
	filePath := filepath.Join(tmpDir, strings.TrimPrefix(file.Path, "/"))
	data, err := os.ReadFile(filePath)

	// Mark as unmounted (will happen in defer)
	mounted = true

	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Close cleans up extractor resources
func (e *Extractor) Close() error {
	// Cleanup if needed
	return nil
}

// matchesAnyPattern checks if path matches any of the patterns
func matchesAnyPattern(path string, patterns []string) bool {
	filename := filepath.Base(path)

	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}

		// Also try as regex for more complex patterns
		pattern = strings.ReplaceAll(pattern, ".", "\\.")
		pattern = strings.ReplaceAll(pattern, "*", ".*")
		pattern = "^" + pattern + "$"

		matched, err = regexp.MatchString(pattern, filename)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// CalculateSHA256 calculates SHA256 hash of data
func CalculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}
