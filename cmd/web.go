package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/afterdarksys/diskimager/internal/api"
	"github.com/afterdarksys/diskimager/pkg/daemon"
	sdnotify "github.com/coreos/go-systemd/v22/daemon"
	"github.com/spf13/cobra"
)

var (
	webPort            string
	webDaemonMode      bool
	webPidFilePath     string
	webLogToSyslog     bool
	webShutdownTimeout int
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the web UI server",
	Long: `Start the web-based UI server with REST API and WebSocket support.

The web server provides:
  - REST API for job management
  - WebSocket for real-time progress updates
  - Modern React-based web interface
  - Remote management capabilities
  - Multi-user support (coming soon)

API Endpoints:
  GET  /api/disks          - List available disks
  POST /api/jobs/image     - Create imaging job
  POST /api/jobs/restore   - Create restore job
  GET  /api/jobs           - List all jobs
  GET  /api/jobs/:id       - Get job details
  DELETE /api/jobs/:id     - Cancel/delete job
  WS   /api/ws/jobs/:id    - Real-time progress updates

Daemon Mode:
  Use --daemon flag to run as a background service with systemd integration.
  Supports graceful shutdown on SIGTERM, SIGINT, and SIGHUP signals.

Access the web UI at: http://localhost:PORT`,
	Run: func(cmd *cobra.Command, args []string) {
		startWebServer()
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.Flags().StringVarP(&webPort, "port", "p", "8080", "Port to listen on")
	webCmd.Flags().BoolVar(&webDaemonMode, "daemon", false, "Run as daemon (enables systemd integration)")
	webCmd.Flags().StringVar(&webPidFilePath, "pid-file", "", "Path to PID file (default: /var/run/diskimager-web.pid)")
	webCmd.Flags().BoolVar(&webLogToSyslog, "syslog", false, "Log to syslog/journald instead of stderr")
	webCmd.Flags().IntVar(&webShutdownTimeout, "shutdown-timeout", 30, "Graceful shutdown timeout in seconds")
}

func startWebServer() {
	// Initialize logger
	var logger daemon.Logger
	var err error

	if webLogToSyslog {
		logger, err = daemon.NewLogger("diskimager-web", true, daemon.LogLevelInfo)
		if err != nil {
			logger = daemon.NewConsoleLogger("diskimager-web", daemon.LogLevelInfo)
			log.Printf("Warning: %v", err)
		}
	} else {
		logger = daemon.NewConsoleLogger("diskimager-web", daemon.LogLevelInfo)
	}
	defer logger.Close()

	logger.Info("Starting Diskimager Web UI Server")

	// Create PID file if daemon mode
	var pidFile *daemon.PIDFile
	if webDaemonMode {
		if webPidFilePath == "" {
			webPidFilePath = daemon.GetDefaultPIDPath("diskimager-web")
		}
		pidFile, err = daemon.CreatePIDFile(webPidFilePath)
		if err != nil {
			logger.Fatal(fmt.Sprintf("Failed to create PID file: %v", err))
		}
		logger.Info(fmt.Sprintf("Created PID file: %s (PID: %d)", webPidFilePath, pidFile.PID))
		defer daemon.RemovePIDFile(webPidFilePath)
	}

	// Create API server
	apiServer := api.NewAPIServer()
	httpServer, err := apiServer.RunWithServer(fmt.Sprintf(":%s", webPort))
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to create server: %v", err))
	}

	// Setup signal handling for graceful shutdown
	signalHandler := daemon.NewSignalHandler(
		&webServerAdapter{server: httpServer, apiServer: apiServer, logger: logger},
		time.Duration(webShutdownTimeout)*time.Second,
		logger,
	)
	if webPidFilePath != "" {
		signalHandler.SetPIDFile(webPidFilePath)
	}
	ctx := signalHandler.SetupSignalHandler()

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%s", webPort)
		logger.Info(fmt.Sprintf("Web UI Server listening on http://localhost%s", addr))
		logger.Info(fmt.Sprintf("API:    http://localhost%s/api", addr))
		logger.Info(fmt.Sprintf("Health: http://localhost%s/health", addr))

		// Notify systemd that we're ready (only works when running under systemd)
		if webDaemonMode {
			sdnotify.SdNotify(false, sdnotify.SdNotifyReady)
			logger.Info("Sent READY signal to systemd")
		}

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		logger.Info("Shutdown signal received")
	case err := <-serverErr:
		logger.Error(fmt.Sprintf("Server error: %v", err))
	}

	logger.Info("Server stopped")
}

// webServerAdapter adapts http.Server to daemon.DaemonServer interface
type webServerAdapter struct {
	server    *http.Server
	apiServer *api.APIServer
	logger    daemon.Logger
}

func (w *webServerAdapter) Shutdown(ctx context.Context) error {
	// Notify systemd we're stopping
	sdnotify.SdNotify(false, sdnotify.SdNotifyStopping)
	w.logger.Info("Initiating graceful shutdown...")

	// Shutdown API server (cancels jobs)
	if err := w.apiServer.Shutdown(ctx); err != nil {
		w.logger.Warn(fmt.Sprintf("API server shutdown warning: %v", err))
	}

	// Shutdown HTTP server
	w.logger.Info("Shutting down HTTP server...")
	return w.server.Shutdown(ctx)
}
