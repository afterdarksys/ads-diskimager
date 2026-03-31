package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/afterdarksys/diskimager/pkg/api"
	"github.com/spf13/cobra"
)

var (
	apiBindAddress    string
	apiMaxWorkers     int
	apiTLSCert        string
	apiTLSKey         string
	apiTLSCA          string
	apiAPIKeys        []string
	apiEnableCORS     bool
	apiAllowedOrigins []string
)

var apiServerCmd = &cobra.Command{
	Use:   "api-server",
	Short: "Start the forensic imaging API server",
	Long: `Start the RESTful API server for forensic disk imaging operations.

The API server provides:
- Asynchronous job processing with real-time WebSocket progress
- Support for multiple source types (disk, VM, cloud storage)
- Compression, encryption, and multi-hash verification
- Block-level hashing and sparse block detection
- Chain of custody metadata tracking

Security:
- API key authentication via X-API-Key header
- Optional mTLS (mutual TLS) authentication
- CORS support for web applications

Example:
  # Start API server with API key authentication
  diskimager api-server --bind-address :8080 --api-keys secret-key-1,secret-key-2

  # Start with mTLS authentication
  diskimager api-server --bind-address :8443 \
    --tls-cert server.crt --tls-key server.key --tls-ca ca.crt

  # Start with both API keys and mTLS
  diskimager api-server --bind-address :8443 \
    --tls-cert server.crt --tls-key server.key --tls-ca ca.crt \
    --api-keys key1,key2 \
    --enable-cors --allowed-origins https://forensics.local

API Documentation:
  OpenAPI specification available at: api/openapi.yaml
  Interactive docs: http://localhost:8080/api/v1/health`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create server configuration
		config := api.ServerConfig{
			BindAddress:    apiBindAddress,
			MaxWorkers:     apiMaxWorkers,
			TLSCert:        apiTLSCert,
			TLSKey:         apiTLSKey,
			TLSCA:          apiTLSCA,
			APIKeys:        apiAPIKeys,
			EnableCORS:     apiEnableCORS,
			AllowedOrigins: apiAllowedOrigins,
		}

		// Create API server
		server := api.NewServer(config)

		// Start server in goroutine
		serverErr := make(chan error, 1)
		go func() {
			log.Println("=== DiskImager API Server ===")
			log.Printf("Version: 2.0.0")
			log.Printf("Bind Address: %s", config.BindAddress)
			log.Printf("Max Workers: %d", config.MaxWorkers)

			if config.TLSCert != "" {
				log.Printf("TLS: Enabled")
				if config.TLSCA != "" {
					log.Println("mTLS: Enabled (client certificate required)")
				}
			} else {
				log.Println("WARNING: TLS not configured - not recommended for production")
			}

			if len(config.APIKeys) > 0 {
				log.Printf("API Keys: %d configured", len(config.APIKeys))
			} else {
				if config.TLSCA == "" {
					log.Println("WARNING: No authentication configured (no API keys or mTLS)")
				}
			}

			if config.EnableCORS {
				log.Printf("CORS: Enabled (origins: %v)", config.AllowedOrigins)
			}

			log.Println("=============================")
			log.Println("Server starting...")

			if err := server.Start(); err != nil {
				serverErr <- err
			}
		}()

		// Wait for interrupt signal or server error
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		select {
		case err := <-serverErr:
			log.Fatalf("Server error: %v", err)
		case sig := <-sigChan:
			log.Printf("Received signal: %v", sig)
			log.Println("Initiating graceful shutdown...")

			// Create shutdown context with timeout
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()

			// Shutdown server
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error during shutdown: %v", err)
			} else {
				log.Println("Server shutdown complete")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(apiServerCmd)

	// Server configuration
	apiServerCmd.Flags().StringVar(&apiBindAddress, "bind-address", ":8080", "Address to bind the API server")
	apiServerCmd.Flags().IntVar(&apiMaxWorkers, "max-workers", 10, "Maximum number of concurrent imaging jobs")

	// TLS configuration
	apiServerCmd.Flags().StringVar(&apiTLSCert, "tls-cert", "", "Path to TLS certificate file")
	apiServerCmd.Flags().StringVar(&apiTLSKey, "tls-key", "", "Path to TLS private key file")
	apiServerCmd.Flags().StringVar(&apiTLSCA, "tls-ca", "", "Path to CA certificate for mTLS (client cert verification)")

	// Authentication
	apiServerCmd.Flags().StringSliceVar(&apiAPIKeys, "api-keys", []string{}, "Comma-separated list of valid API keys")

	// CORS configuration
	apiServerCmd.Flags().BoolVar(&apiEnableCORS, "enable-cors", false, "Enable CORS support")
	apiServerCmd.Flags().StringSliceVar(&apiAllowedOrigins, "allowed-origins", []string{"*"}, "Comma-separated list of allowed CORS origins")

	// Environment variable support
	if keys := os.Getenv("DISKIMAGER_API_KEYS"); keys != "" {
		// Parse comma-separated keys from environment
		fmt.Println("Loading API keys from environment variable DISKIMAGER_API_KEYS")
	}
}
