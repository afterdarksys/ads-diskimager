package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/afterdarksys/diskimager/pkg/extractor"
	"github.com/dsnet/compress/bzip2"
	"github.com/spf13/cobra"
)

var (
	findDisk      string
	findFS        string
	findPatterns  []string
	findOutput    string
	findFormat    string
	findRecursive bool
	findVerbose   bool
	findHashFiles bool
)

var findCmd = &cobra.Command{
	Use:   "find",
	Short: "Search and extract files from disk images",
	Long: `Search for files within disk images using pattern matching and extract them.

Supports multiple filesystems (auto-detected or specified) and can create
archives for easy evidence collection and analysis.

Examples:
  # Find all Office documents
  ./diskimager find \
    --disk evidence.img \
    --fs auto \
    --patterns "*.doc,*.docx,*.xls,*.xlsx,*.ppt,*.pptx"

  # Extract to tar.bz2 archive
  ./diskimager find \
    --disk evidence.img \
    --fs ext4 \
    --patterns "*.pdf,*.jpg" \
    --output evidence_files.tar.bz2 \
    --format tar.bz2

  # Extract to cpio.bz2 archive
  ./diskimager find \
    --disk evidence.img \
    --patterns "*.exe,*.dll" \
    --output executables.cpio.bz2 \
    --format cpio.bz2

  # Print results only (no extraction)
  ./diskimager find \
    --disk evidence.img \
    --patterns "*.doc" \
    --output print \
    --verbose

  # With hash calculation
  ./diskimager find \
    --disk evidence.img \
    --patterns "*.exe" \
    --output malware.tar.bz2 \
    --hash
`,
	Run: func(cmd *cobra.Command, args []string) {
		if findDisk == "" {
			cmd.Usage()
			os.Exit(1)
		}

		if len(findPatterns) == 0 {
			log.Fatalf("Error: --patterns required (e.g., *.doc,*.pdf)")
		}

		// Detect filesystem if auto
		fsType := findFS
		if fsType == "auto" {
			fmt.Println("Auto-detecting filesystem...")
			detected, err := extractor.DetectFilesystem(findDisk)
			if err != nil {
				log.Fatalf("Failed to detect filesystem: %v\nTry specifying with --fs", err)
			}
			fsType = detected
			fmt.Printf("✓ Detected filesystem: %s\n", fsType)
		}

		// Create extractor
		ext, err := extractor.NewExtractor(findDisk, fsType)
		if err != nil {
			log.Fatalf("Failed to open disk image: %v", err)
		}
		defer ext.Close()

		fmt.Printf("Searching for patterns: %s\n", strings.Join(findPatterns, ", "))

		// Search for files
		start := time.Now()
		results, err := ext.FindFiles(findPatterns, findRecursive)
		if err != nil {
			log.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			fmt.Println("No files found matching patterns")
			return
		}

		fmt.Printf("\n✓ Found %d file(s) in %v\n\n", len(results), time.Since(start))

		// Handle output
		switch findOutput {
		case "print", "":
			// Print mode
			printResults(results, findVerbose, findHashFiles, ext)

		default:
			// Archive mode
			if findFormat == "" {
				// Auto-detect format from extension
				if strings.HasSuffix(findOutput, ".tar.bz2") {
					findFormat = "tar.bz2"
				} else if strings.HasSuffix(findOutput, ".cpio.bz2") {
					findFormat = "cpio.bz2"
				} else if strings.HasSuffix(findOutput, ".tar.gz") {
					findFormat = "tar.gz"
				} else if strings.HasSuffix(findOutput, ".tar") {
					findFormat = "tar"
				} else {
					findFormat = "tar.bz2" // Default
					findOutput = findOutput + ".tar.bz2"
				}
			}

			fmt.Printf("Creating archive: %s (format: %s)\n", findOutput, findFormat)
			err = createArchive(findOutput, findFormat, results, ext, findHashFiles)
			if err != nil {
				log.Fatalf("Failed to create archive: %v", err)
			}

			// Print summary
			stat, _ := os.Stat(findOutput)
			if stat != nil {
				fmt.Printf("\n✓ Archive created: %s (%d bytes)\n", findOutput, stat.Size())
				fmt.Printf("✓ Files archived: %d\n", len(results))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(findCmd)
	findCmd.Flags().StringVar(&findDisk, "disk", "", "Disk image to search (required)")
	findCmd.Flags().StringVar(&findFS, "fs", "auto", "Filesystem type (auto, ext4, ntfs, fat32, apfs)")
	findCmd.Flags().StringSliceVar(&findPatterns, "patterns", []string{}, "File patterns to search (e.g., *.doc,*.pdf)")
	findCmd.Flags().StringVar(&findOutput, "output", "print", "Output destination (print, or file path)")
	findCmd.Flags().StringVar(&findFormat, "format", "", "Archive format (tar, tar.bz2, tar.gz, cpio.bz2)")
	findCmd.Flags().BoolVarP(&findRecursive, "recursive", "r", true, "Recursive search")
	findCmd.Flags().BoolVarP(&findVerbose, "verbose", "v", false, "Verbose output")
	findCmd.Flags().BoolVar(&findHashFiles, "hash", false, "Calculate SHA256 hash for each file")
}

// printResults displays search results
func printResults(results []extractor.FileInfo, verbose, calcHash bool, ext *extractor.Extractor) {
	fmt.Println("Files found:")
	fmt.Println(strings.Repeat("-", 80))

	for i, file := range results {
		fmt.Printf("%4d. %s\n", i+1, file.Path)

		if verbose {
			fmt.Printf("      Size:     %d bytes\n", file.Size)
			fmt.Printf("      Modified: %s\n", file.ModTime.Format(time.RFC3339))
			if file.Permissions != "" {
				fmt.Printf("      Perms:    %s\n", file.Permissions)
			}
		}

		if calcHash {
			// Extract and hash the file
			data, err := ext.ExtractFile(file)
			if err != nil {
				fmt.Printf("      Hash:     Error: %v\n", err)
			} else {
				hash := extractor.CalculateSHA256(data)
				fmt.Printf("      SHA256:   %s\n", hash)
			}
		}

		if verbose || calcHash {
			fmt.Println()
		}
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Total: %d file(s)\n", len(results))
}

// createArchive creates tar or cpio archive
func createArchive(outputPath, format string, files []extractor.FileInfo, ext *extractor.Extractor, calcHash bool) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer outFile.Close()

	var archiver archiveWriter
	switch format {
	case "tar", "tar.bz2", "tar.gz":
		archiver, err = newTarArchiver(outFile, format)
	case "cpio", "cpio.bz2":
		archiver, err = newCpioArchiver(outFile, format == "cpio.bz2")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return err
	}
	defer archiver.Close()

	// Add files to archive
	for i, file := range files {
		fmt.Printf("\r[%d/%d] Adding: %s", i+1, len(files), file.Path)

		// Extract file data
		data, err := ext.ExtractFile(file)
		if err != nil {
			fmt.Printf("\n⚠️  Warning: Failed to extract %s: %v\n", file.Path, err)
			continue
		}

		// Add to archive
		if err := archiver.AddFile(file, data); err != nil {
			return fmt.Errorf("failed to add %s: %w", file.Path, err)
		}

		// Calculate hash if requested
		if calcHash {
			hash := extractor.CalculateSHA256(data)
			// Write hash file alongside
			hashEntry := extractor.FileInfo{
				Path:    file.Path + ".sha256",
				Size:    int64(len(hash)),
				ModTime: file.ModTime,
			}
			if err := archiver.AddFile(hashEntry, []byte(hash+"\n")); err != nil {
				fmt.Printf("\n⚠️  Warning: Failed to add hash for %s\n", file.Path)
			}
		}
	}
	fmt.Println()

	return nil
}

// archiveWriter interface for different archive formats
type archiveWriter interface {
	AddFile(file extractor.FileInfo, data []byte) error
	Close() error
}

// tarArchiver implements tar archives
type tarArchiver struct {
	writer     *tar.Writer
	compressor io.WriteCloser
}

func newTarArchiver(w io.Writer, format string) (*tarArchiver, error) {
	var writer io.Writer = w
	var compressor io.WriteCloser

	switch format {
	case "tar.bz2":
		// bzip2 compression
		bz2, err := bzip2.NewWriter(w, &bzip2.WriterConfig{Level: 9})
		if err != nil {
			return nil, fmt.Errorf("failed to create bzip2 writer: %w", err)
		}
		compressor = bz2
		writer = compressor
	case "tar.gz":
		// gzip compression
		compressor = gzip.NewWriter(w)
		writer = compressor
	}

	return &tarArchiver{
		writer:     tar.NewWriter(writer),
		compressor: compressor,
	}, nil
}

func (t *tarArchiver) AddFile(file extractor.FileInfo, data []byte) error {
	header := &tar.Header{
		Name:    file.Path,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: file.ModTime,
	}

	if err := t.writer.WriteHeader(header); err != nil {
		return err
	}

	if _, err := t.writer.Write(data); err != nil {
		return err
	}

	return nil
}

func (t *tarArchiver) Close() error {
	if err := t.writer.Close(); err != nil {
		return err
	}
	if t.compressor != nil {
		return t.compressor.Close()
	}
	return nil
}

// cpioArchiver implements cpio archives
type cpioArchiver struct {
	writer     io.Writer
	compressor io.WriteCloser
}

func newCpioArchiver(w io.Writer, compressed bool) (*cpioArchiver, error) {
	var writer io.Writer = w
	var compressor io.WriteCloser

	if compressed {
		bz2, err := bzip2.NewWriter(w, &bzip2.WriterConfig{Level: 9})
		if err != nil {
			return nil, fmt.Errorf("failed to create bzip2 writer: %w", err)
		}
		compressor = bz2
		writer = compressor
	}

	return &cpioArchiver{
		writer:     writer,
		compressor: compressor,
	}, nil
}

func (c *cpioArchiver) AddFile(file extractor.FileInfo, data []byte) error {
	// Write cpio header (newc format)
	header := fmt.Sprintf("070701%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x",
		0,                      // inode
		0100644,                // mode (regular file)
		0, 0,                   // uid, gid
		1,                      // nlink
		file.ModTime.Unix(),    // mtime
		len(data),              // filesize
		0, 0, 0, 0,             // devmajor, devminor, rdevmajor, rdevminor
		len(file.Path)+1,       // namesize (including null)
		0,                      // check
	)

	if _, err := c.writer.Write([]byte(header)); err != nil {
		return err
	}

	if _, err := c.writer.Write([]byte(file.Path + "\x00")); err != nil {
		return err
	}

	// Alignment (4-byte boundary)
	nameLen := len(file.Path) + 1
	padding := (4 - (len(header)+nameLen)%4) % 4
	if padding > 0 {
		if _, err := c.writer.Write(make([]byte, padding)); err != nil {
			return err
		}
	}

	// Write file data
	if _, err := c.writer.Write(data); err != nil {
		return err
	}

	// Alignment after data
	dataPadding := (4 - len(data)%4) % 4
	if dataPadding > 0 {
		if _, err := c.writer.Write(make([]byte, dataPadding)); err != nil {
			return err
		}
	}

	return nil
}

func (c *cpioArchiver) Close() error {
	// Write TRAILER!!! marker using proper cpio newc format
	// Format: 070701 + 13 hex fields (8 chars each) + name (TRAILER!!!) + null + padding
	trailerName := "TRAILER!!!"
	header := fmt.Sprintf("070701%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x",
		0,                  // inode
		0,                  // mode
		0, 0,               // uid, gid
		1,                  // nlink
		0,                  // mtime
		0,                  // filesize
		0, 0, 0, 0,         // devmajor, devminor, rdevmajor, rdevminor
		len(trailerName)+1, // namesize (including null terminator)
		0,                  // check
	)

	if _, err := c.writer.Write([]byte(header)); err != nil {
		return err
	}

	// Write trailer name with null terminator
	if _, err := c.writer.Write([]byte(trailerName + "\x00")); err != nil {
		return err
	}

	// Calculate padding to 4-byte boundary
	// header is 110 bytes, name is 11 bytes (including null) = 121 bytes
	// Need padding to make it multiple of 4
	nameLen := len(trailerName) + 1
	totalLen := 110 + nameLen
	padding := (4 - (totalLen % 4)) % 4
	if padding > 0 {
		if _, err := c.writer.Write(make([]byte, padding)); err != nil {
			return err
		}
	}

	if c.compressor != nil {
		return c.compressor.Close()
	}
	return nil
}

