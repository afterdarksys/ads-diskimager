package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/afterdarksys/diskimager/pkg/tsk"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/spf13/cobra"
)

var (
	analyzeInput string
	analyzeDir   string
	useTSK       bool
)

type FileHash struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type PartitionReport struct {
	Index int        `json:"index"`
	Type  string     `json:"type"`
	Files []FileHash `json:"files"`
}

type SystemHashesReport struct {
	ImagePath   string            `json:"image_path"`
	GeneratedAt time.Time         `json:"generated_at"`
	Partitions  []PartitionReport `json:"partitions"`
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze disk image or directory and generate system hashes",
	Run: func(cmd *cobra.Command, args []string) {
		if analyzeInput == "" && analyzeDir == "" {
			cmd.Usage()
			os.Exit(1)
		}
		if analyzeInput != "" && analyzeDir != "" {
			log.Fatal("Please specify either -in or -dir, not both.")
		}
		
		target := analyzeInput
		if analyzeDir != "" {
			target = analyzeDir
		}

		report := SystemHashesReport{
			ImagePath:   target,
			GeneratedAt: time.Now(),
			Partitions:  []PartitionReport{},
		}

		if analyzeDir != "" {
			fmt.Printf("Analyzing directory: %s\n", analyzeDir)
			part, err := analyzeDirectory(analyzeDir)
			if err != nil {
				log.Fatalf("Analysis failed: %v", err)
			}
			report.Partitions = append(report.Partitions, part)
		} else {
			// Image analysis logic
			if useTSK {
				if err := analyzeImageTSK(analyzeInput, &report); err != nil {
					log.Fatalf("TSK analysis failed: %v", err)
				}
			} else {
				if err := analyzeImage(analyzeInput, &report); err != nil {
					log.Fatalf("Image analysis failed: %v", err)
				}
			}
		}

		// Write Report
		reportJSON, _ := json.MarshalIndent(report, "", "  ")
		outName := "system_hashes.json"
		if err := os.WriteFile(outName, reportJSON, 0644); err != nil {
			log.Fatalf("Failed to write report: %v", err)
		}
		fmt.Printf("Analysis complete. Report written to %s\n", outName)
	},
}

func analyzeDirectory(root string) (PartitionReport, error) {
	report := PartitionReport{
		Index: 0,
		Type:  "Directory/Mount",
		Files: []FileHash{},
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Optional: Skip the report itself if writing to same dir
		if filepath.Base(path) == "system_hashes.json" {
			return nil
		}

		fmt.Printf("\r  Hashing: %s", path)

		// Open file
		f, err := os.Open(path)
		if err != nil {
			log.Printf("    Error opening %s: %v", path, err)
			return nil
		}
		defer f.Close()

		// Hash
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			log.Printf("    Error hashing %s: %v", path, err)
			return nil
		}

		report.Files = append(report.Files, FileHash{
			Path:   path,
			Size:   info.Size(),
			SHA256: fmt.Sprintf("%x", h.Sum(nil)),
		})
		return nil
	})

	fmt.Println() // Newline
	return report, err
}

func analyzeImage(path string, report *SystemHashesReport) error {
	// Open disk image
	disk, err := diskfs.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open disk image: %v", err)
	}

	// Get partition table
	pt, err := disk.GetPartitionTable()
	if err != nil {
		log.Printf("Warning: could not read partition table (might be raw fs): %v", err)
	}

	if pt != nil {
		partitions := pt.GetPartitions()
		fmt.Printf("Found %d partitions.\n", len(partitions))

		for i, p := range partitions {
			fmt.Printf("Analyzing Partition %d (Start: %d, Size: %d)...\n", i, p.GetStart(), p.GetSize())
			partReport := PartitionReport{
				Index: i,
				Type:  "Unknown",
				Files: []FileHash{},
			}

			fs, err := disk.GetFilesystem(i + 1) // 1-based index usually
			if err != nil {
				log.Printf("  Skipping partition %d: %v", i, err)
				report.Partitions = append(report.Partitions, partReport)
				continue
			}

			walker := func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}

				fmt.Printf("\r  Hashing: %s", path)

				// Open file
				f, err := fs.OpenFile(path, os.O_RDONLY)
				if err != nil {
					log.Printf("    Error opening %s: %v", path, err)
					return nil
				}
				defer f.Close()

				// Hash
				h := sha256.New()
				if _, err := io.Copy(h, f); err != nil {
					log.Printf("    Error hashing %s: %v", path, err)
					return nil
				}

				partReport.Files = append(partReport.Files, FileHash{
					Path:   path,
					Size:   info.Size(),
					SHA256: fmt.Sprintf("%x", h.Sum(nil)),
				})
				return nil
			}

			// DiskFS doesn't have a standardized Walk. We might need to implement it recursively using ReadDir.
			// Let's implement a quick recursive walker.
			err = walkFS(fs, "/", walker)
			if err != nil {
				log.Printf("Error walking partition %d: %v", i, err)
			}
			fmt.Println() // Newline after progress

			report.Partitions = append(report.Partitions, partReport)
		}
	} else {
		log.Println("No partition table found.")
	}
	return nil
}

func analyzeImageTSK(path string, report *SystemHashesReport) error {
	// 1. Get Partitions using go-diskfs (it's fast and easy for MBR/GPT)
	disk, err := diskfs.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open disk image: %v", err)
	}
	
	pt, err := disk.GetPartitionTable()
	if err != nil {
		log.Printf("Warning: could not read partition table: %v", err)
	}

	// 2. Open Image with TSK
	tskImg, err := tsk.OpenImage(path)
	if err != nil {
		return fmt.Errorf("tsk open image failed: %v", err)
	}
	defer tskImg.Close()

	if pt != nil {
		partitions := pt.GetPartitions()
		fmt.Printf("Found %d partitions (using go-diskfs for layout).\n", len(partitions))

		for i, p := range partitions {
			startBytes := int64(p.GetStart()) * 512
			fmt.Printf("Analyzing Partition %d (Start: %d bytes) with TSK...\n", i, startBytes)

			partReport := PartitionReport{
				Index: i,
				Type:  "TSK_Detected",
				Files: []FileHash{},
			}

			fs, err := tskImg.OpenFS(startBytes)
			if err != nil {
				log.Printf("  TSK failed to open FS at offset %d: %v", startBytes, err)
				report.Partitions = append(report.Partitions, partReport)
				continue
			}
			
			// Walk
			err = fs.Walk(func(file *tsk.File, path string) error {
				if file.IsDir() {
					return nil
				}
				// Skip . and ..
				if file.Name() == "." || file.Name() == ".." {
					return nil
				}

				fullPath := filepath.Join(path, file.Name())
				fmt.Printf("\r  Hashing: %s", fullPath)

				// Read and Hash
				h := sha256.New()
				
				// TSK file read in chunks
				buf := make([]byte, 32*1024)
				var offset int64 = 0
				for {
					n, rErr := file.ReadAt(buf, offset)
					if n > 0 {
						h.Write(buf[:n])
						offset += int64(n)
					}
					if rErr == io.EOF {
						break
					}
					if rErr != nil {
						log.Printf("    Read error %s: %v", fullPath, rErr)
						return nil // Skip file on error
					}
				}

				partReport.Files = append(partReport.Files, FileHash{
					Path:   fullPath,
					Size:   file.Size(),
					SHA256: fmt.Sprintf("%x", h.Sum(nil)),
				})

				return nil
			})
			
			fs.Close()
			fmt.Println()
			
			if err != nil {
				log.Printf("Error walking partition %d: %v", i, err)
			}
			
			report.Partitions = append(report.Partitions, partReport)
		}
	} else {
		fmt.Println("No partitions found via go-diskfs.")
	}

	return nil
}

func walkFS(fs filesystem.FileSystem, path string, walkFn filepath.WalkFunc) error {
	files, err := fs.ReadDir(path)
	if err != nil {
		return walkFn(path, nil, err)
	}

	for _, file := range files {
		fullPath := filepath.Join(path, file.Name())
		if err := walkFn(fullPath, file, nil); err != nil {
			return err
		}

		if file.IsDir() {
			if err := walkFS(fs, fullPath, walkFn); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVar(&analyzeInput, "in", "", "Input disk image path")
	analyzeCmd.Flags().StringVar(&analyzeDir, "dir", "", "Input directory path (e.g. mount point)")
	analyzeCmd.Flags().BoolVar(&useTSK, "tsk", false, "Use The Sleuth Kit (TSK) for filesystem analysis")
	// Remove required mark as we have logic to check one or the other
}
