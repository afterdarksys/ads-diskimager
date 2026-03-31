package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/afterdarksys/diskimager/pkg/blockeditor"
	"github.com/spf13/cobra"
)

var (
	diskEditorPort       int
	diskEditorBlockSize  int
	diskEditorMaxBlocks  int64
	diskEditorComputeHash bool
	diskEditorEntropy    bool
	diskEditorOpenBrowser bool
)

var diskEditorCmd = &cobra.Command{
	Use:   "disk-editor",
	Short: "Interactive disk block editor and visualizer",
	Long: `Launch an interactive web-based disk block editor that visualizes disk blocks
as a grid of colored squares. Features include:

  • Visual block map with color-coded allocation and file types
  • Interactive mouse hover for block information
  • Click to select blocks and view detailed information
  • Search and filter by file name, type, or status
  • File identification and correlation across blocks
  • Zoom and pan for detailed inspection
  • Real-time entropy and signature detection

Perfect for:
  - Forensic analysis and investigation
  - Understanding disk structure and fragmentation
  - Identifying file types and locations
  - Finding deleted or unallocated data
  - Visualizing disk utilization

Example:
  # Analyze and visualize a disk image
  diskimager disk-editor --in /evidence/disk001.img

  # Quick analysis with sampling (for large disks)
  diskimager disk-editor --in /dev/sda --max-blocks 100000

  # Full analysis with hash computation
  diskimager disk-editor --in image.dd --compute-hash

The disk editor will start a local web server and open your browser
automatically. Use your mouse to explore the disk visually!`,
	Run: runDiskEditor,
}

func init() {
	rootCmd.AddCommand(diskEditorCmd)

	diskEditorCmd.Flags().StringVarP(&inputFile, "in", "i", "", "Input disk image or device (required)")
	diskEditorCmd.MarkFlagRequired("in")

	diskEditorCmd.Flags().IntVar(&diskEditorPort, "port", 9090, "Web server port")
	diskEditorCmd.Flags().IntVar(&diskEditorBlockSize, "block-size", 4096, "Block size in bytes (default: 4KB)")
	diskEditorCmd.Flags().Int64Var(&diskEditorMaxBlocks, "max-blocks", 0, "Maximum blocks to analyze (0 = all)")
	diskEditorCmd.Flags().BoolVar(&diskEditorComputeHash, "compute-hash", false, "Compute SHA256 hash for each block (slower)")
	diskEditorCmd.Flags().BoolVar(&diskEditorEntropy, "entropy", true, "Compute entropy for each block")
	diskEditorCmd.Flags().BoolVar(&diskEditorOpenBrowser, "open-browser", true, "Automatically open browser")
}

func runDiskEditor(cmd *cobra.Command, args []string) {
	log.Println("=== Diskimager Block Editor ===")
	log.Printf("Input: %s", inputFile)

	// Check if input exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		log.Fatalf("Error: Input file does not exist: %s", inputFile)
	}

	// Configure analysis options
	options := &blockeditor.AnalysisOptions{
		BlockSize:          diskEditorBlockSize,
		ComputeHashes:      diskEditorComputeHash,
		ComputeEntropy:     diskEditorEntropy,
		DetectFileTypes:    true,
		ParseFilesystem:    false, // Future feature
		IdentifySignatures: true,
		MaxBlocks:          diskEditorMaxBlocks,
		SampleRate:         1, // Analyze every block
	}

	// If max blocks is set, enable sampling for efficiency
	if diskEditorMaxBlocks > 0 {
		log.Printf("Limiting analysis to first %d blocks", diskEditorMaxBlocks)
	}

	log.Println("Analyzing disk image...")
	log.Printf("Block size: %d bytes", options.BlockSize)
	if options.ComputeEntropy {
		log.Println("Computing entropy for each block")
	}
	if options.ComputeHashes {
		log.Println("Computing SHA256 hashes (this will be slower)")
	}

	// Create analyzer
	analyzer, err := blockeditor.NewAnalyzer(inputFile, options)
	if err != nil {
		log.Fatalf("Failed to create analyzer: %v", err)
	}

	// Perform analysis
	diskMap, err := analyzer.Analyze()
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	log.Println("✓ Analysis complete!")
	log.Printf("  Total blocks: %d", diskMap.TotalBlocks)
	log.Printf("  Total size: %d bytes (%.2f GB)", diskMap.TotalSize, float64(diskMap.TotalSize)/1024/1024/1024)
	log.Printf("  Allocated: %d blocks", diskMap.Statistics.AllocatedBlocks)
	log.Printf("  Unallocated: %d blocks", diskMap.Statistics.UnallocatedBlocks)
	log.Printf("  Zero blocks: %d blocks", diskMap.Statistics.ZeroBlocks)
	log.Printf("  Utilization: %.2f%%", diskMap.Statistics.Utilization)

	// Create and start web server
	server := blockeditor.NewServer(analyzer, diskEditorPort)

	// Open browser if requested
	if diskEditorOpenBrowser {
		url := fmt.Sprintf("http://localhost:%d", diskEditorPort)
		log.Printf("Opening browser: %s", url)
		go openBrowser(url)
	}

	log.Println("Starting web server...")
	log.Printf("Press Ctrl+C to stop")

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// openBrowser opens the default browser to the given URL
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		log.Println("Cannot open browser automatically on this platform")
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Warning: Failed to open browser: %v", err)
		log.Printf("Please open your browser manually to: %s", url)
	}
}
