package cmd

import (
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

	"github.com/afterdarksys/diskimager/config"
	"github.com/afterdarksys/diskimager/imager"
	"github.com/spf13/cobra"
)

var (
	configFile string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the diskimager collection server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		if err := os.MkdirAll(cfg.Server.StoragePath, 0755); err != nil {
			log.Fatalf("Failed to create storage directory: %v", err)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
			clientID := "unknown"
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				clientID = r.TLS.PeerCertificates[0].Subject.CommonName
			}

			// We need a stable identifier if we're resuming. Let's use an X-Upload-ID header,
			// or default to generating a new timestamp-based one.
			uploadID := r.Header.Get("X-Upload-ID")
			if uploadID == "" {
				uploadID = fmt.Sprintf("%s_%s", clientID, time.Now().Format("20060102_150405"))
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
			os.WriteFile(logFile, logBytes, 0644)

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

		fmt.Printf("Server listening on %s (mTLS enabled)\n", cfg.Server.BindAddress)
		log.Fatal(server.ListenAndServeTLS(cfg.Server.TLSCert, cfg.Server.TLSKey))
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVar(&configFile, "config", "config.json", "Path to configuration file")
}
