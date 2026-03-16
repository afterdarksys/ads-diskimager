package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/afterdarksys/diskimager/imager"
	"github.com/spf13/cobra"
)

var (
	streamTarget string
	streamCert   string
	streamKey    string
	streamCA     string
	streamInput  string
	streamBS     int
	streamResume bool
	streamID     string

	// Stream Metadata flags
	streamCaseNum  string
	streamEvidence string
	streamExaminer string
	streamDesc     string
	streamNotes    string
)

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Stream a local disk image to a remote server securely",
	Run: func(cmd *cobra.Command, args []string) {
		if streamTarget == "" || streamInput == "" || streamCert == "" || streamKey == "" || streamCA == "" {
			cmd.Usage()
			os.Exit(1)
		}

		// Load Client Certs
		cert, err := tls.LoadX509KeyPair(streamCert, streamKey)
		if err != nil {
			log.Fatalf("Failed to load client cert/key: %v", err)
		}

		// Load CA
		caCert, err := os.ReadFile(streamCA)
		if err != nil {
			log.Fatalf("Failed to read CA cert: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Setup HTTP Client with mTLS
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
					RootCAs:      caCertPool,
				},
			},
			Timeout: 0, // No timeout for long streaming
		}

		// Open Input
		inFile, err := os.Open(streamInput)
		if err != nil {
			log.Fatalf("Failed to open input file: %v", err)
		}
		defer inFile.Close()

		var startOffset int64 = 0
		if streamResume {
			// Do HEAD to get current size
			headReq, err := http.NewRequest(http.MethodHead, streamTarget+"/upload", nil)
			if err != nil {
				log.Fatalf("Failed to create HEAD request: %v", err)
			}
			if streamID != "" {
				headReq.Header.Set("X-Upload-ID", streamID)
			}
			
			headResp, err := client.Do(headReq)
			if err == nil && headResp.StatusCode == http.StatusOK {
				sizeStr := headResp.Header.Get("X-Current-Size")
				if sizeStr != "" {
					startOffset, _ = strconv.ParseInt(sizeStr, 10, 64)
					fmt.Printf("Server reports existing file size: %d bytes. Resuming...\n", startOffset)
				}
			} else {
				fmt.Println("Server could not provide resume point. Starting from beginning.")
			}
			
			if headResp != nil {
				headResp.Body.Close()
			}
		}

		if startOffset > 0 {
			if _, err := inFile.Seek(startOffset, io.SeekStart); err != nil {
				log.Fatalf("Failed to seek input file: %v", err)
			}
		}

		stat, err := inFile.Stat()
		if err == nil {
			fmt.Printf("Streaming %s (Total: %d bytes) to %s\n", streamInput, stat.Size(), streamTarget)
		}

		pr, pw := io.Pipe()

		errChan := make(chan error, 1)
		
		go func() {
			method := http.MethodPost
			if startOffset > 0 {
				method = http.MethodPatch // or POST, depending on server. Server accepts both now.
			}
			
			req, err := http.NewRequest(method, streamTarget+"/upload", pr)
			if err != nil {
				errChan <- fmt.Errorf("failed to create request: %v", err)
				return
			}
			
			// Chain of custody headers
			req.Header.Set("X-Forensic-Case", streamCaseNum)
			req.Header.Set("X-Forensic-Evidence", streamEvidence)
			req.Header.Set("X-Forensic-Examiner", streamExaminer)
			req.Header.Set("X-Forensic-Desc", streamDesc)
			req.Header.Set("X-Forensic-Notes", streamNotes)
			
			if streamID != "" {
				req.Header.Set("X-Upload-ID", streamID)
			}

			if startOffset > 0 {
				req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-", startOffset))
			}
			
			resp, err := client.Do(req)
			if err != nil {
				errChan <- fmt.Errorf("request failed: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				errChan <- fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
				return
			}

			serverHash, _ := io.ReadAll(resp.Body)
			fmt.Printf("\nServer confirmed upload. Server Hash: %s\n", string(serverHash))
			errChan <- nil
		}()

		fmt.Println("Starting stream...")
		start := time.Now()
		
		wrappedReader := &ProgressReader{
			Reader:    inFile,
			BytesRead: startOffset,
		}

		cfg := imager.Config{
			Source:      wrappedReader,
			Destination: pw,
			BlockSize:   streamBS,
			HashAlgo:    "sha256",
		}

		res, imgErr := imager.Image(cfg)
		pw.Close()

		if imgErr != nil {
			log.Fatalf("Imaging failed: %v", imgErr)
		}

		if reqErr := <-errChan; reqErr != nil {
			log.Fatalf("Streaming failed: %v", reqErr)
		}

		fmt.Printf("Streaming completed in %v.\n", time.Since(start))
		
		if res != nil {
			fmt.Printf("Client Bytes Processed this session: %d\n", res.BytesCopied)
			fmt.Printf("Client Hash: %s\n", res.Hash)
		}
	},
}

func init() {
	rootCmd.AddCommand(streamCmd)
	streamCmd.Flags().StringVar(&streamTarget, "target", "", "Target Server URL (e.g. https://localhost:8080)")
	streamCmd.Flags().StringVar(&streamInput, "in", "", "Input device or file")
	streamCmd.Flags().StringVar(&streamCert, "cert", "client.crt", "Client Certificate file")
	streamCmd.Flags().StringVar(&streamKey, "key", "client.key", "Client Key file")
	streamCmd.Flags().StringVar(&streamCA, "ca", "ca.crt", "CA Certificate file")
	streamCmd.Flags().IntVar(&streamBS, "bs", 64*1024, "Block size")
	
	streamCmd.Flags().BoolVar(&streamResume, "resume", false, "Attempt to resume stream")
	streamCmd.Flags().StringVar(&streamID, "id", "", "Unique Upload ID for resuming")

	streamCmd.Flags().StringVar(&streamCaseNum, "case", "", "Case Number")
	streamCmd.Flags().StringVar(&streamEvidence, "evidence", "", "Evidence Number")
	streamCmd.Flags().StringVar(&streamExaminer, "examiner", "", "Examiner Name")
	streamCmd.Flags().StringVar(&streamDesc, "desc", "", "Description of evidence")
	streamCmd.Flags().StringVar(&streamNotes, "notes", "", "Additional notes")
}
