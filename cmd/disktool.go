package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var disktoolDevice string

var disktoolCmd = &cobra.Command{
	Use:   "disktool",
	Short: "Advanced filesystem and disk recovery utilities",
	Long: `disktool provides low-level forensics and recovery tools operating directly 
on raw block devices, physical disks, and disk images.
Powered by The Sleuth Kit (TSK) and go-diskfs.

Available commands allow scanning, fixing superblocks, recovering deleted files,
dumping bootloaders, and mapping paths to physical LBA sectors.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(disktoolCmd)
	// Global flag for disktool subcommands
	disktoolCmd.PersistentFlags().StringVarP(&disktoolDevice, "device", "d", "", "Target block device or image file (e.g. /dev/sda, image.dd)")
}

// requireDevice is a helper for subcommands to ensure device is provided
func requireDevice(cmd *cobra.Command) {
	if disktoolDevice == "" {
		fmt.Println("Error: required flag(s) \"device\" not set")
		cmd.Usage()
		os.Exit(1)
	}
}
