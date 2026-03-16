package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var forensickImage string

var forensickCmd = &cobra.Command{
	Use:   "forensick",
	Short: "Non-destructive forensic analysis of disk images",
	Long: `forensick provides analytical tools meant exclusively for disk images (e.g., E01, Raw DD).
It uses The Sleuth Kit (TSK) to extract deleted files, carve data, and build forensic timelines 
without ever chancing writes to a physical disk.

This component is purely read-only and designed for post-acquisition analysis.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(forensickCmd)
	forensickCmd.PersistentFlags().StringVarP(&forensickImage, "image", "i", "", "Target disk image file (e.g. evidence.e01, sda.dd)")
}

func requireImage(cmd *cobra.Command) {
	if forensickImage == "" {
		fmt.Println("Error: required flag(s) \"image\" not set")
		cmd.Usage()
		os.Exit(1)
	}
}
