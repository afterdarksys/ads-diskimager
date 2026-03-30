package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/afterdarksys/diskimager/pkg/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage diskimager daemon services",
	Long: `Manage diskimager daemon services using systemd.

This command provides convenient management of diskimager systemd services
without needing to use systemctl directly.

Available subcommands:
  install   - Install systemd service files
  uninstall - Remove systemd service files
  status    - Show service status
  start     - Start a service
  stop      - Stop a service
  restart   - Restart a service
  logs      - Show service logs

Services:
  serve - Collection server (mTLS)
  web   - Web UI server`,
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install systemd service files",
	Long:  `Install diskimager as systemd services. Requires root privileges.`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("Error: This command must be run as root (use sudo)")
			os.Exit(1)
		}

		// Find the install script
		scriptPath := findScript("install-daemon.sh")
		if scriptPath == "" {
			fmt.Println("Error: install-daemon.sh script not found")
			fmt.Println("Please run from the project directory or ensure scripts are in PATH")
			os.Exit(1)
		}

		// Run the install script
		installCmd := exec.Command("/bin/bash", scriptPath)
		installCmd.Stdin = os.Stdin
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr

		if err := installCmd.Run(); err != nil {
			fmt.Printf("Error: Installation failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove systemd service files",
	Long:  `Uninstall diskimager systemd services. Requires root privileges.`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("Error: This command must be run as root (use sudo)")
			os.Exit(1)
		}

		// Find the uninstall script
		scriptPath := findScript("uninstall-daemon.sh")
		if scriptPath == "" {
			fmt.Println("Error: uninstall-daemon.sh script not found")
			fmt.Println("Please run from the project directory or ensure scripts are in PATH")
			os.Exit(1)
		}

		// Run the uninstall script
		uninstallCmd := exec.Command("/bin/bash", scriptPath)
		uninstallCmd.Stdin = os.Stdin
		uninstallCmd.Stdout = os.Stdout
		uninstallCmd.Stderr = os.Stderr

		if err := uninstallCmd.Run(); err != nil {
			fmt.Printf("Error: Uninstallation failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status [serve|web]",
	Short: "Show service status",
	Long:  `Show the status of diskimager services.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		services := []string{"diskimager-serve", "diskimager-web"}

		if len(args) == 1 {
			service := getServiceName(args[0])
			if service == "" {
				fmt.Printf("Error: Invalid service '%s'. Use 'serve' or 'web'\n", args[0])
				os.Exit(1)
			}
			services = []string{service}
		}

		for _, service := range services {
			fmt.Printf("\n=== %s ===\n", service)
			statusCmd := exec.Command("systemctl", "status", service)
			statusCmd.Stdout = os.Stdout
			statusCmd.Stderr = os.Stderr
			statusCmd.Run() // Ignore error as status might be inactive
		}
	},
}

var daemonStartCmd = &cobra.Command{
	Use:   "start [serve|web]",
	Short: "Start a service",
	Long:  `Start one or both diskimager services.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("Error: This command must be run as root (use sudo)")
			os.Exit(1)
		}

		services := []string{"diskimager-serve", "diskimager-web"}

		if len(args) == 1 {
			service := getServiceName(args[0])
			if service == "" {
				fmt.Printf("Error: Invalid service '%s'. Use 'serve' or 'web'\n", args[0])
				os.Exit(1)
			}
			services = []string{service}
		}

		for _, service := range services {
			fmt.Printf("Starting %s...\n", service)
			startCmd := exec.Command("systemctl", "start", service)
			if err := startCmd.Run(); err != nil {
				fmt.Printf("Error: Failed to start %s: %v\n", service, err)
			} else {
				fmt.Printf("Successfully started %s\n", service)
			}
		}
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop [serve|web]",
	Short: "Stop a service",
	Long:  `Stop one or both diskimager services.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("Error: This command must be run as root (use sudo)")
			os.Exit(1)
		}

		services := []string{"diskimager-serve", "diskimager-web"}

		if len(args) == 1 {
			service := getServiceName(args[0])
			if service == "" {
				fmt.Printf("Error: Invalid service '%s'. Use 'serve' or 'web'\n", args[0])
				os.Exit(1)
			}
			services = []string{service}
		}

		for _, service := range services {
			fmt.Printf("Stopping %s...\n", service)
			stopCmd := exec.Command("systemctl", "stop", service)
			if err := stopCmd.Run(); err != nil {
				fmt.Printf("Error: Failed to stop %s: %v\n", service, err)
			} else {
				fmt.Printf("Successfully stopped %s\n", service)
			}
		}
	},
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart [serve|web]",
	Short: "Restart a service",
	Long:  `Restart one or both diskimager services.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("Error: This command must be run as root (use sudo)")
			os.Exit(1)
		}

		services := []string{"diskimager-serve", "diskimager-web"}

		if len(args) == 1 {
			service := getServiceName(args[0])
			if service == "" {
				fmt.Printf("Error: Invalid service '%s'. Use 'serve' or 'web'\n", args[0])
				os.Exit(1)
			}
			services = []string{service}
		}

		for _, service := range services {
			fmt.Printf("Restarting %s...\n", service)
			restartCmd := exec.Command("systemctl", "restart", service)
			if err := restartCmd.Run(); err != nil {
				fmt.Printf("Error: Failed to restart %s: %v\n", service, err)
			} else {
				fmt.Printf("Successfully restarted %s\n", service)
			}
		}
	},
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs [serve|web]",
	Short: "Show service logs",
	Long: `Show logs for diskimager services using journalctl.

Flags:
  -f, --follow    Follow log output
  -n, --lines     Number of lines to show (default: 50)`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		follow, _ := cmd.Flags().GetBool("follow")
		lines, _ := cmd.Flags().GetInt("lines")

		service := "diskimager-serve"
		if len(args) == 1 {
			service = getServiceName(args[0])
			if service == "" {
				fmt.Printf("Error: Invalid service '%s'. Use 'serve' or 'web'\n", args[0])
				os.Exit(1)
			}
		}

		// Build journalctl command
		cmdArgs := []string{"-u", service, "-n", fmt.Sprintf("%d", lines)}
		if follow {
			cmdArgs = append(cmdArgs, "-f")
		}

		logsCmd := exec.Command("journalctl", cmdArgs...)
		logsCmd.Stdout = os.Stdout
		logsCmd.Stderr = os.Stderr
		logsCmd.Stdin = os.Stdin

		if err := logsCmd.Run(); err != nil {
			fmt.Printf("Error: Failed to show logs: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)

	// Add subcommands
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
	daemonCmd.AddCommand(daemonLogsCmd)

	// Add flags
	daemonLogsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	daemonLogsCmd.Flags().IntP("lines", "n", 50, "Number of lines to show")
}

// getServiceName converts short name to full service name
func getServiceName(short string) string {
	switch strings.ToLower(short) {
	case "serve", "collection", "collector":
		return "diskimager-serve"
	case "web", "ui", "webui":
		return "diskimager-web"
	default:
		return ""
	}
}

// findScript finds a script in common locations
func findScript(name string) string {
	// Try current directory
	if _, err := os.Stat(name); err == nil {
		return name
	}

	// Try scripts subdirectory
	scriptPath := fmt.Sprintf("scripts/%s", name)
	if _, err := os.Stat(scriptPath); err == nil {
		return scriptPath
	}

	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		exeDir := strings.TrimSuffix(exe, "/diskimager")
		scriptPath = fmt.Sprintf("%s/scripts/%s", exeDir, name)
		if _, err := os.Stat(scriptPath); err == nil {
			return scriptPath
		}
	}

	// Try with daemon package
	scriptPath = daemon.GetDefaultPIDPath(name) // Reuse path logic
	if _, err := os.Stat(scriptPath); err == nil {
		return scriptPath
	}

	return ""
}
