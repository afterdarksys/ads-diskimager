package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	Version = "2.1.0"
	BuildDate = "2024-03-30"
)

var (
	showVersion bool
)

var rootCmd = &cobra.Command{
	Use:   "diskimager",
	Short: "A forensics-grade disk imaging and analysis tool",
	Long: `Diskimager - Professional Forensic Disk Imaging Tool

A comprehensive forensics-grade disk imaging and analysis tool with:
- Multi-hash validation (MD5, SHA1, SHA256 simultaneously)
- Cloud storage support (S3, MinIO, GCS, Azure)
- Intelligent error recovery (ddrescue-style)
- Bandwidth throttling and compression
- Sparse file support for space savings
- Chain of custody metadata tracking`,
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Printf("Diskimager v%s (built %s)\n", Version, BuildDate)
			fmt.Println("\nFeatures:")
			fmt.Println("  ✓ Parallel multi-hash computing")
			fmt.Println("  ✓ Intelligent error recovery")
			fmt.Println("  ✓ Bandwidth throttling")
			fmt.Println("  ✓ Compression support (gzip, zstd)")
			fmt.Println("  ✓ Sparse file optimization")
			fmt.Println("  ✓ Cloud storage integration")
			fmt.Println("  ✓ Chain of custody tracking")
			os.Exit(0)
		}
		cmd.Help()
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
