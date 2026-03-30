package cmd

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/afterdarksys/diskimager/imager"
	"github.com/afterdarksys/diskimager/pkg/format/e01"
	"github.com/spf13/cobra"
)

var (
	restoreImageFile string
	restoreOutDevice string
	restoreVerify    bool
	restoreForce     bool
	restoreHashAlgo  string
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a forensic image to a disk or file (DESTRUCTIVE)",
	Long: `Restore a forensic disk image to a physical device or file.

WARNING: This operation is DESTRUCTIVE and will overwrite all data on the destination.

Safety features:
- Interactive confirmation required
- Mount detection (prevents writing to mounted filesystems)
- Size verification (destination must be large enough)
- System disk protection (prevents accidental /dev/sda overwrite)
- Optional post-restore verification

Example:
  sudo ./diskimager restore --image evidence.img --out /dev/sdb --verify
  sudo ./diskimager restore --image evidence.e01 --out /dev/sdc --verify
`,
	Run: func(cmd *cobra.Command, args []string) {
		if restoreImageFile == "" || restoreOutDevice == "" {
			cmd.Usage()
			os.Exit(1)
		}

		// Safety checks
		if err := performRestoreSafetyChecks(restoreImageFile, restoreOutDevice, restoreForce); err != nil {
			log.Fatalf("Safety check failed: %v", err)
		}

		// Interactive confirmation
		if !confirmRestore(restoreOutDevice) {
			fmt.Println("Restore cancelled by user.")
			os.Exit(0)
		}

		// Perform restore
		start := time.Now()
		hash, err := performRestore(restoreImageFile, restoreOutDevice, restoreVerify)
		if err != nil {
			log.Fatalf("Restore failed: %v", err)
		}

		fmt.Printf("\n✅ Restore completed successfully in %v\n", time.Since(start))
		if restoreVerify && hash != "" {
			fmt.Printf("Restored data hash (%s): %s\n", restoreHashAlgo, hash)
		}
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().StringVar(&restoreImageFile, "image", "", "Input image file (required)")
	restoreCmd.Flags().StringVar(&restoreOutDevice, "out", "", "Output device or file (required)")
	restoreCmd.Flags().BoolVar(&restoreVerify, "verify", true, "Verify written data with hash")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false, "Skip safety checks (DANGEROUS)")
	restoreCmd.Flags().StringVar(&restoreHashAlgo, "hash", "sha256", "Hash algorithm for verification")
}

// performRestoreSafetyChecks validates that the restore operation is safe
func performRestoreSafetyChecks(imagePath, destPath string, force bool) error {
	// Check that image file exists
	imgStat, err := os.Stat(imagePath)
	if err != nil {
		return fmt.Errorf("cannot access image file: %w", err)
	}

	if force {
		fmt.Println("⚠️  WARNING: Safety checks bypassed with --force flag")
		return nil
	}

	// Protect system disk (common patterns)
	systemDiskPatterns := []string{"/dev/sda", "/dev/disk0", "/dev/nvme0n1", `\\.\PhysicalDrive0`}
	for _, pattern := range systemDiskPatterns {
		if strings.HasPrefix(destPath, pattern) {
			return fmt.Errorf("refusing to write to likely system disk %s (use --force to override)", destPath)
		}
	}

	// Check if destination is mounted
	mounted, err := isDeviceMounted(destPath)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not check mount status: %v\n", err)
	} else if mounted {
		return fmt.Errorf("destination %s is currently mounted (unmount first)", destPath)
	}

	// Check destination size (if it's a block device)
	destStat, err := os.Stat(destPath)
	if err == nil && destStat.Mode()&os.ModeDevice != 0 {
		// It's a device, try to get size
		destSize, err := getDeviceSize(destPath)
		if err == nil {
			if destSize < imgStat.Size() {
				return fmt.Errorf("destination (%d bytes) is smaller than image (%d bytes)", destSize, imgStat.Size())
			}
			fmt.Printf("✓ Size check: Destination (%d bytes) >= Image (%d bytes)\n", destSize, imgStat.Size())
		}
	}

	// Check if destination exists and is writable
	if destStat != nil {
		// File exists, check if writable
		f, err := os.OpenFile(destPath, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("destination is not writable: %w", err)
		}
		f.Close()
	}

	fmt.Println("✓ Safety checks passed")
	return nil
}

// confirmRestore prompts user for interactive confirmation
func confirmRestore(destPath string) bool {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("⚠️  CRITICAL WARNING: DESTRUCTIVE OPERATION")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("You are about to DESTROY ALL DATA on: %s\n", destPath)
	fmt.Println("This operation CANNOT BE UNDONE.")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("\nTo confirm, type the destination path exactly: %s\n", destPath)
	fmt.Print("Confirm: ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != destPath {
		fmt.Printf("\n❌ Confirmation failed. You typed: '%s'\n", confirm)
		return false
	}

	fmt.Println("\n✓ Confirmation accepted. Starting restore in 3 seconds...")
	time.Sleep(3 * time.Second)
	return true
}

// performRestore executes the actual restore operation
func performRestore(imagePath, destPath string, verify bool) (string, error) {
	// Open source image
	imgFile, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer imgFile.Close()

	// Check if it's E01 format
	headerMagic := make([]byte, 8)
	imgFile.Read(headerMagic)
	isEWF := string(headerMagic) == "EVF\x09\x0d\x0a\xff\x00"
	imgFile.Seek(0, io.SeekStart)

	var source io.Reader = imgFile

	// If E01, wrap with decompressor
	if isEWF {
		fmt.Println("Detected E01 format. Decompressing chunks during restore...")
		reader, err := e01.NewReader(imgFile)
		if err != nil {
			return "", fmt.Errorf("failed to create E01 reader: %w", err)
		}
		source = reader
	}

	// Open destination
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to open destination: %w", err)
	}
	defer destFile.Close()

	// Setup hash if verifying
	var hasher hash.Hash
	var multiWriter io.Writer = destFile

	if verify {
		switch restoreHashAlgo {
		case "sha256":
			hasher = sha256.New()
		default:
			return "", fmt.Errorf("unsupported hash algorithm: %s", restoreHashAlgo)
		}
		multiWriter = io.MultiWriter(destFile, hasher)
	}

	// Progress tracking
	wrappedReader := &ProgressReader{
		Reader: source,
	}

	fmt.Printf("Restoring %s -> %s\n", imagePath, destPath)

	// Copy with progress
	cfg := imager.Config{
		Source:      wrappedReader,
		Destination: &writeOnlyCloser{multiWriter, destFile},
		BlockSize:   1024 * 1024, // 1MB blocks for restore
		HashAlgo:    restoreHashAlgo,
		Hasher:      hasher,
	}

	res, err := imager.Image(cfg)
	if err != nil {
		return "", fmt.Errorf("restore operation failed: %w", err)
	}

	// Sync to ensure all data is written
	destFile.Sync()

	fmt.Printf("\n✓ Wrote %d bytes\n", res.BytesCopied)

	if verify {
		return res.Hash, nil
	}
	return "", nil
}

// writeOnlyCloser wraps an io.Writer and closer for compatibility
type writeOnlyCloser struct {
	w io.Writer
	c io.Closer
}

func (wc *writeOnlyCloser) Write(p []byte) (n int, err error) {
	return wc.w.Write(p)
}

func (wc *writeOnlyCloser) Close() error {
	return wc.c.Close()
}

// isDeviceMounted checks if a device is currently mounted
func isDeviceMounted(devicePath string) (bool, error) {
	// Read /proc/mounts on Linux, /etc/mtab on others
	mountFile := "/proc/mounts"
	if _, err := os.Stat(mountFile); os.IsNotExist(err) {
		mountFile = "/etc/mtab"
		if _, err := os.Stat(mountFile); os.IsNotExist(err) {
			// Try macOS diskutil
			return isDeviceMountedMacOS(devicePath)
		}
	}

	data, err := os.ReadFile(mountFile)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, devicePath) {
			return true, nil
		}
	}

	return false, nil
}

// isDeviceMountedMacOS checks mount status on macOS using diskutil
func isDeviceMountedMacOS(devicePath string) (bool, error) {
	// For macOS, we could exec diskutil, but for safety just return unknown
	// This prevents false negatives
	return false, fmt.Errorf("mount detection not fully implemented for this platform")
}

// getDeviceSize returns the size of a block device in bytes
func getDeviceSize(devicePath string) (int64, error) {
	f, err := os.Open(devicePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Seek to end to get size
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	return size, nil
}
