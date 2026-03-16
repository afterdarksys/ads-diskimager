package cmd

import (
	"fmt"
	"log"
	
	"github.com/spf13/cobra"
)

var forensickExtractOut string

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Recover deleted files and carve data from unallocated space",
	Run: func(cmd *cobra.Command, args []string) {
		requireImage(cmd)
		
		if forensickExtractOut == "" {
			log.Fatal("Error: Output directory (--out) is required for extraction")
		}

		fmt.Printf("Analyzing image %s for deleted files...\n", forensickImage)
		fmt.Printf("Extracted data will be written to: %s\n", forensickExtractOut)
		fmt.Println("[Not Implemented] Requires comprehensive TSK Data-layer bindings for deleted inode carving.")
	},
}

var analyzeTimelineCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Build timeline of MAC (Modified, Accessed, Changed) times",
	Run: func(cmd *cobra.Command, args []string) {
		requireImage(cmd)

		fmt.Printf("Extracting MAC timeline from %s...\n", forensickImage)
		fmt.Println("[Not Implemented] Requires TSK Metadata-layer bindings for inode timestamp retrieval.")
	},
}

func init() {
	forensickCmd.AddCommand(extractCmd)
	forensickCmd.AddCommand(analyzeTimelineCmd)
	
	extractCmd.Flags().StringVarP(&forensickExtractOut, "out", "o", "", "Directory to write extracted files")
}
