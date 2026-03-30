package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sync"

	"github.com/afterdarksys/diskimager/config"
	"github.com/afterdarksys/diskimager/imager"
	"github.com/afterdarksys/diskimager/pkg/daemon"
	sdnotify "github.com/coreos/go-systemd/v22/daemon"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	configFile     string
	daemonMode     bool
	pidFilePath    string
	logToSyslog    bool
	shutdownTimout int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the diskimager collection server",
	Long: `Start the forensic disk image collection server with mTLS authentication.

The server accepts disk images from remote clients over HTTPS with mutual TLS
authentication. All uploads are verified with cryptographic hashes.

Features:
  - Mutual TLS (mTLS) authentication
  - Resumable uploads with UUID-based tracking
  - Cryptographic verification (SHA256)
  - Forensic metadata capture
  - Server-side audit logging
  - Graceful shutdown with in-progress upload handling

Daemon Mode:
  Use --daemon flag to run as a background service with systemd integration.
  Supports graceful shutdown on SIGTERM, SIGINT, and SIGHUP signals.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize logger
		var logger daemon.Logger
		var err error

		if logToSyslog {
			logger, err = daemon.NewLogger("diskimager-serve", true, daemon.LogLevelInfo)
			if err != nil {
				logger = daemon.NewConsoleLogger("diskimager-serve", daemon.LogLevelInfo)
				log.Printf("Warning: %v", err)
			}
		} else {
			logger = daemon.NewConsoleLogger("diskimager-serve", daemon.LogLevelInfo)
		}
		defer logger.Close()

		logger.Info("Starting Diskimager Collection Server")

		// Load configuration
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			logger.Fatal(fmt.Sprintf("Failed to load config: %v", err))
		}

		// Create storage directory
		if err := os.MkdirAll(cfg.Server.StoragePath, 0755); err != nil {
			logger.Fatal(fmt.Sprintf("Failed to create storage directory: %v", err))
		}

		// Create PID file if daemon mode
		var pidFile *daemon.PIDFile
		if daemonMode {
			if pidFilePath == "" {
				pidFilePath = daemon.GetDefaultPIDPath("diskimager-serve")
			}
			pidFile, err = daemon.CreatePIDFile(pidFilePath)
			if err != nil {
				logger.Fatal(fmt.Sprintf("Failed to create PID file: %v", err))
			}
			logger.Info(fmt.Sprintf("Created PID file: %s (PID: %d)", pidFilePath, pidFile.PID))
			defer daemon.RemovePIDFile(pidFilePath)
		}

		// Track active uploads for graceful shutdown
		activeUploads := &activeUploadTracker{
			count: 0,
			mu:    &sync.Mutex{},
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
			activeUploads.increment()
			defer activeUploads.decrement()
			clientID := "unknown"
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				clientID = r.TLS.PeerCertificates[0].Subject.CommonName
			}

			// Use UUID for upload ID to prevent collisions
			// Client can provide X-Upload-ID header for resume, otherwise generate new UUID
			uploadID := r.Header.Get("X-Upload-ID")
			if uploadID == "" {
				// Generate UUID v4 for guaranteed uniqueness
				newUUID, err := uuid.NewRandom()
				if err != nil {
					http.Error(w, "Failed to generate upload ID", http.StatusInternalServerError)
					log.Printf("UUID generation failed: %v", err)
					return
				}
				uploadID = fmt.Sprintf("%s_%s", clientID, newUUID.String())
			}

			filename := uploadID + ".img"
			filePath := filepath.Join(cfg.Server.StoragePath, filename)
			logFile := filePath + ".log"

			// Handle HEAD request for checking current size
			if r.Method == http.MethodHead {
				stat, err := os.Stat(filePath)
				if err == nil {
					w.Header().Set("X-Current-Size", fmt.Sprintf("%d", stat.Size()))
					w.WriteHeader(http.StatusOK)
				} else {
					w.Header().Set("X-Current-Size", "0")
					w.WriteHeader(http.StatusNotFound)
				}
				return
			}

			if r.Method != http.MethodPost && r.Method != http.MethodPatch {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			isResume := r.Header.Get("Content-Range") != ""
			
			flags := os.O_CREATE | os.O_WRONLY
			if isResume {
				flags |= os.O_APPEND
			} else {
				flags |= os.O_TRUNC
			}

			out, err := os.OpenFile(filePath, flags, 0644)
			if err != nil {
				http.Error(w, "Failed to create/open file", http.StatusInternalServerError)
				return
			}
			defer out.Close()

			if !isResume {
				fmt.Printf("Incoming NEW stream from %s -> %s\n", clientID, filePath)
			} else {
				fmt.Printf("Incoming RESUME stream from %s -> %s\n", clientID, filePath)
			}

			meta := imager.Metadata{
				CaseNumber:  r.Header.Get("X-Forensic-Case"),
				EvidenceNum: r.Header.Get("X-Forensic-Evidence"),
				Examiner:    r.Header.Get("X-Forensic-Examiner"),
				Description: r.Header.Get("X-Forensic-Desc"),
				Notes:       r.Header.Get("X-Forensic-Notes"),
			}

			serverCfg := imager.Config{
				Source:      r.Body,
				Destination: out,
				BlockSize:   64 * 1024,
				HashAlgo:    "sha256", // Force SHA256 for now, or read header
				Metadata:    meta,
			}

			res, err := imager.Image(serverCfg)
			
			// Adjust bytes copied if resuming
			if isResume {
				crange := r.Header.Get("Content-Range")
				// Simple parsing "bytes start-end/total"
				// e.g., "bytes 1048576-"
				parts := strings.Split(crange, " ")
				if len(parts) == 2 && parts[0] == "bytes" {
					rangeParts := strings.Split(parts[1], "-")
					if len(rangeParts) >= 1 {
						offset, _ := strconv.ParseInt(rangeParts[0], 10, 64)
						res.BytesCopied += offset
					}
				}
			}

			// Write Server-Side Audit Log
			logEntry := struct {
				Source      string
				Destination string
				UploadID    string
				ClientID    string
				Config      imager.Config
				Result      *imager.Result
				Error       string `json:",omitempty"`
			}{
				Source:      "NetworkStream",
				Destination: filePath,
				UploadID:    uploadID,
				ClientID:    clientID,
				Config:      serverCfg,
				Result:      res,
			}
			logEntry.Config.Source = nil
			logEntry.Config.Destination = nil
			if err != nil {
				logEntry.Error = err.Error()
			}

			logBytes, _ := json.MarshalIndent(logEntry, "", "  ")
			os.WriteFile(logFile, logBytes, 0600) // Secure permissions

			if err != nil {
				log.Printf("Error receiving stream: %v", err)
				http.Error(w, "Stream error", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "%s", res.Hash)
			fmt.Printf("Finished receiving %s. Hash: %s\n", filename, res.Hash)
		})

		// TLS Config
		tlsConfig := &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			MinVersion: tls.VersionTLS13,
		}

		// Load CA
		caCert, err := os.ReadFile(cfg.Server.TLSCA)
		if err != nil {
			log.Fatalf("Failed to read CA cert: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.ClientCAs = caCertPool

		server := &http.Server{
			Addr:      cfg.Server.BindAddress,
			Handler:   mux,
			TLSConfig: tlsConfig,
		}

		// Setup signal handling for graceful shutdown
		signalHandler := daemon.NewSignalHandler(
			&httpServerAdapter{server: server, activeUploads: activeUploads, logger: logger},
			time.Duration(shutdownTimout)*time.Second,
			logger,
		)
		if pidFilePath != "" {
			signalHandler.SetPIDFile(pidFilePath)
		}
		ctx := signalHandler.SetupSignalHandler()

		// Start server in goroutine
		serverErr := make(chan error, 1)
		go func() {
			logger.Info(fmt.Sprintf("Server listening on %s (mTLS enabled)", cfg.Server.BindAddress))
			logger.Info(fmt.Sprintf("Storage path: %s", cfg.Server.StoragePath))

			// Notify systemd that we're ready (only works when running under systemd)
			if daemonMode {
				sdnotify.SdNotify(false, sdnotify.SdNotifyReady)
				logger.Info("Sent READY signal to systemd")
			}

			if err := server.ListenAndServeTLS(cfg.Server.TLSCert, cfg.Server.TLSKey); err != nil && err != http.ErrServerClosed {
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
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVar(&configFile, "config", "config.json", "Path to configuration file")
	serveCmd.Flags().BoolVar(&daemonMode, "daemon", false, "Run as daemon (enables systemd integration)")
	serveCmd.Flags().StringVar(&pidFilePath, "pid-file", "", "Path to PID file (default: /var/run/diskimager-serve.pid)")
	serveCmd.Flags().BoolVar(&logToSyslog, "syslog", false, "Log to syslog/journald instead of stderr")
	serveCmd.Flags().IntVar(&shutdownTimout, "shutdown-timeout", 30, "Graceful shutdown timeout in seconds")
}

// activeUploadTracker tracks the number of active uploads
type activeUploadTracker struct {
	count int
	mu    *sync.Mutex
}

func (a *activeUploadTracker) increment() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.count++
}

func (a *activeUploadTracker) decrement() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.count--
}

func (a *activeUploadTracker) getCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.count
}

// httpServerAdapter adapts http.Server to daemon.DaemonServer interface
type httpServerAdapter struct {
	server        *http.Server
	activeUploads *activeUploadTracker
	logger        daemon.Logger
}

func (h *httpServerAdapter) Shutdown(ctx context.Context) error {
	// Notify systemd we're stopping
	sdnotify.SdNotify(false, sdnotify.SdNotifyStopping)
	h.logger.Info("Initiating graceful shutdown...")

	// Wait for active uploads to complete or timeout
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			activeCount := h.activeUploads.getCount()
			if activeCount > 0 {
				h.logger.Warn(fmt.Sprintf("Shutdown timeout reached with %d active uploads", activeCount))
			}
			// Force shutdown
			return h.server.Shutdown(context.Background())
		case <-ticker.C:
			activeCount := h.activeUploads.getCount()
			if activeCount == 0 {
				h.logger.Info("All uploads completed, shutting down")
				return h.server.Shutdown(ctx)
			}
			h.logger.Info(fmt.Sprintf("Waiting for %d active uploads to complete...", activeCount))
		}
	}
}
