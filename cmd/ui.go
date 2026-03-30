package cmd

import (
	"github.com/afterdarksys/diskimager/internal/gui/fyne"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the graphical user interface",
	Long: `Launch the enhanced Norton Ghost-style GUI with wizard-based workflows.

The GUI provides:
  - Imaging Wizard: 6-step guided disk imaging process
  - Restore Wizard: 5-step guided image restoration
  - Dashboard: Quick actions and system health monitoring
  - Recent Operations: Track your imaging history
  - Settings: Configure defaults and cloud credentials

Keyboard Shortcuts:
  - Ctrl+I: Start imaging wizard
  - Ctrl+R: Start restore wizard
  - Ctrl+Q: Quit application`,
	Run: func(cmd *cobra.Command, args []string) {
		startGUI()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func startGUI() {
	// Create and run the enhanced dashboard
	dashboard := fyne.NewDashboard()
	dashboard.Run()
}
