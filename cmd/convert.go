package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/afterdarksys/diskimager/pkg/format/virtual"
	"github.com/spf13/cobra"
)

var (
	convertInput  string
	convertOutput string
	convertFormat string
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert forensic images to virtual disk formats (VMDK, VHD)",
	Long: `Convert RAW/E01 forensic images to virtual machine disk formats.

Supported output formats:
  - vmdk: VMware VMDK (monolithic flat)
  - vhd:  Microsoft VHD (fixed disk)

This allows forensic images to be mounted directly in virtual machines
for analysis without modifying the original evidence.

Examples:
  # Convert RAW image to VMDK
  ./diskimager convert --in evidence.img --out evidence.vmdk --format vmdk

  # Convert RAW image to VHD
  ./diskimager convert --in evidence.img --out evidence.vhd --format vhd
`,
	Run: func(cmd *cobra.Command, args []string) {
		if convertInput == "" || convertOutput == "" {
			cmd.Usage()
			os.Exit(1)
		}

		// Validate format
		format := strings.ToLower(convertFormat)
		if format != "vmdk" && format != "vhd" {
			log.Fatalf("Unsupported format: %s (use vmdk or vhd)", convertFormat)
		}

		// Open input
		inFile, err := os.Open(convertInput)
		if err != nil {
			log.Fatalf("Failed to open input: %v", err)
		}
		defer inFile.Close()

		// Get input size
		stat, err := inFile.Stat()
		if err != nil {
			log.Fatalf("Failed to stat input: %v", err)
		}
		diskSize := stat.Size()

		fmt.Printf("Converting %s (%d bytes) to %s format...\n", convertInput, diskSize, strings.ToUpper(format))

		// Create output writer
		var outWriter io.WriteCloser
		switch format {
		case "vmdk":
			outWriter, err = virtual.NewVMDKWriter(convertOutput, diskSize)
		case "vhd":
			outWriter, err = virtual.NewVHDWriter(convertOutput, diskSize)
		}
		if err != nil {
			log.Fatalf("Failed to create output: %v", err)
		}
		defer outWriter.Close()

		// Copy data with progress
		start := time.Now()
		buf := make([]byte, 1024*1024) // 1MB buffer
		var totalCopied int64

		for {
			n, err := inFile.Read(buf)
			if n > 0 {
				if _, wErr := outWriter.Write(buf[:n]); wErr != nil {
					log.Fatalf("\nWrite error: %v", wErr)
				}
				totalCopied += int64(n)

				// Progress reporting
				if totalCopied%(100*1024*1024) == 0 { // Every 100MB
					fmt.Printf("\rConverted: %d / %d bytes (%.1f%%)",
						totalCopied, diskSize, float64(totalCopied)/float64(diskSize)*100)
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("\nRead error: %v", err)
			}
		}

		fmt.Printf("\n✅ Conversion completed in %v\n", time.Since(start))
		fmt.Printf("Output: %s\n", convertOutput)

		if format == "vmdk" {
			fmt.Printf("Note: VMDK consists of descriptor (.vmdk) and flat (-flat.vmdk) files\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVar(&convertInput, "in", "", "Input image file (required)")
	convertCmd.Flags().StringVar(&convertOutput, "out", "", "Output virtual disk path (required)")
	convertCmd.Flags().StringVar(&convertFormat, "format", "vmdk", "Output format (vmdk, vhd)")
}
