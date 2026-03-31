package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// ServerConfig holds API server configuration
type ServerConfig struct {
	BindAddress   string
	MaxWorkers    int
	TLSCert       string
	TLSKey        string
	TLSCA         string
	APIKeys       []string
	EnableCORS    bool
	AllowedOrigins []string
}

// Server represents the API server
type Server struct {
	config     ServerConfig
	jobQueue   *JobQueue
	httpServer *http.Server
	router     *gin.Engine
	startTime  time.Time
}

// NewServer creates a new API server
func NewServer(config ServerConfig) *Server {
	// Set Gin to release mode in production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	server := &Server{
		config:    config,
		jobQueue:  NewJobQueue(config.MaxWorkers),
		router:    router,
		startTime: time.Now(),
	}

	// Setup routes
	server.setupRoutes()

	// Configure HTTP server
	server.httpServer = &http.Server{
		Addr:           config.BindAddress,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Configure TLS if certificates provided
	if config.TLSCert != "" && config.TLSKey != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13,
		}

		// Setup mTLS if CA provided
		if config.TLSCA != "" {
			caCert, err := os.ReadFile(config.TLSCA)
			if err != nil {
				log.Fatalf("Failed to read CA cert: %v", err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.ClientCAs = caCertPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		server.httpServer.TLSConfig = tlsConfig
	}

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// CORS middleware
	if s.config.EnableCORS {
		s.router.Use(s.corsMiddleware())
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Public routes (no auth)
		v1.GET("/health", s.handleHealth)
		v1.GET("/version", s.handleVersion)

		// Protected routes (require auth)
		protected := v1.Group("")
		protected.Use(s.authMiddleware())
		{
			// Job management
			protected.POST("/jobs/image", s.handleCreateImageJob)
			protected.GET("/jobs/:jobId", s.handleGetJob)
			protected.DELETE("/jobs/:jobId", s.handleCancelJob)
			protected.GET("/jobs", s.handleListJobs)
			protected.GET("/jobs/:jobId/logs", s.handleGetJobLogs)
			protected.GET("/jobs/:jobId/artifacts", s.handleGetJobArtifacts)
			protected.GET("/jobs/:jobId/artifacts/:artifactId", s.handleDownloadArtifact)

			// Verification
			protected.POST("/verify", s.handleVerify)

			// Metadata
			protected.GET("/sources", s.handleListSources)
			protected.GET("/formats", s.handleListFormats)
		}

		// WebSocket endpoint (requires auth via query param or header)
		v1.GET("/jobs/:jobId/stream", s.handleJobStream)
	}
}

// Start starts the API server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.config.BindAddress)

	if s.config.TLSCert != "" && s.config.TLSKey != "" {
		log.Println("TLS enabled")
		if s.config.TLSCA != "" {
			log.Println("mTLS enabled (client certificate required)")
		}
		return s.httpServer.ListenAndServeTLS(s.config.TLSCert, s.config.TLSKey)
	}

	log.Println("WARNING: Running without TLS (not recommended for production)")
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API server...")

	// Shutdown job queue
	s.jobQueue.Shutdown()

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}

// authMiddleware validates API authentication
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			// Validate API key
			if s.isValidAPIKey(apiKey) {
				c.Next()
				return
			}
		}

		// Check for API key in query param (for WebSocket)
		apiKey = c.Query("api_key")
		if apiKey != "" {
			if s.isValidAPIKey(apiKey) {
				c.Next()
				return
			}
		}

		// Check for mTLS client certificate
		if c.Request.TLS != nil && len(c.Request.TLS.PeerCertificates) > 0 {
			// Client certificate verified by TLS layer
			c.Next()
			return
		}

		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Valid API key or client certificate required",
		})
		c.Abort()
	}
}

// isValidAPIKey checks if an API key is valid
func (s *Server) isValidAPIKey(key string) bool {
	// If no API keys configured, skip validation (mTLS only mode)
	if len(s.config.APIKeys) == 0 {
		return true
	}

	for _, validKey := range s.config.APIKeys {
		if key == validKey {
			return true
		}
	}
	return false
}

// corsMiddleware handles CORS
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if s.isAllowedOrigin(origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isAllowedOrigin checks if an origin is allowed for CORS
func (s *Server) isAllowedOrigin(origin string) bool {
	if len(s.config.AllowedOrigins) == 0 {
		return true
	}

	for _, allowed := range s.config.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    int64(time.Since(s.startTime).Seconds()),
	})
}

// handleVersion handles version requests
func (s *Server) handleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{
		APIVersion:        "1.0.0",
		DiskimagerVersion: "2.0.0", // TODO: Get from build metadata
		BuildDate:         "2024-01-15",
		GitCommit:         "abc123",
	})
}

// handleCreateImageJob handles job creation requests
func (s *Server) handleCreateImageJob(c *gin.Context) {
	var req CreateImageJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	// Validate request
	if err := s.validateImageJobRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Create job
	job, err := s.jobQueue.CreateJob(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create job",
			Details: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	// Get base URL for stream URL
	baseURL := fmt.Sprintf("http://%s", c.Request.Host)
	if c.Request.TLS != nil {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	c.JSON(http.StatusCreated, job.ToJobResponse(baseURL))
}

// handleGetJob handles get job requests
func (s *Server) handleGetJob(c *gin.Context) {
	jobID := c.Param("jobId")

	job, err := s.jobQueue.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("Job not found: %s", jobID),
		})
		return
	}

	baseURL := fmt.Sprintf("http://%s", c.Request.Host)
	if c.Request.TLS != nil {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	c.JSON(http.StatusOK, job.ToJobResponse(baseURL))
}

// handleCancelJob handles cancel job requests
func (s *Server) handleCancelJob(c *gin.Context) {
	jobID := c.Param("jobId")

	if err := s.jobQueue.CancelJob(jobID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == fmt.Sprintf("job not found: %s", jobID) {
			status = http.StatusNotFound
		} else if err.Error() == fmt.Sprintf("cannot cancel job in status: %s", StatusCompleted) ||
			err.Error() == fmt.Sprintf("cannot cancel job in status: %s", StatusFailed) {
			status = http.StatusConflict
		}

		c.JSON(status, ErrorResponse{
			Error:   "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job cancelled"})
}

// handleListJobs handles list jobs requests
func (s *Server) handleListJobs(c *gin.Context) {
	statusFilter := c.Query("status")
	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	jobs, total := s.jobQueue.ListJobs(statusFilter, limit, offset)

	// Convert to responses
	baseURL := fmt.Sprintf("http://%s", c.Request.Host)
	if c.Request.TLS != nil {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	responses := make([]JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = job.ToJobResponse(baseURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   responses,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// handleGetJobLogs handles get job logs requests
func (s *Server) handleGetJobLogs(c *gin.Context) {
	// TODO: Implement log retrieval
	c.JSON(http.StatusOK, gin.H{
		"job_id": c.Param("jobId"),
		"logs":   []interface{}{},
	})
}

// handleGetJobArtifacts handles get job artifacts requests
func (s *Server) handleGetJobArtifacts(c *gin.Context) {
	// TODO: Implement artifact listing
	c.JSON(http.StatusOK, gin.H{
		"artifacts": []interface{}{},
	})
}

// handleDownloadArtifact handles artifact download requests
func (s *Server) handleDownloadArtifact(c *gin.Context) {
	jobID := c.Param("jobId")
	artifactID := c.Param("artifactId")

	// TODO: Implement artifact download
	c.JSON(http.StatusNotImplemented, ErrorResponse{
		Error:   "not_implemented",
		Message: fmt.Sprintf("Artifact download not yet implemented: job=%s artifact=%s", jobID, artifactID),
	})
}

// handleVerify handles verification requests
func (s *Server) handleVerify(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
		return
	}

	// TODO: Implement verification
	c.JSON(http.StatusNotImplemented, ErrorResponse{
		Error:   "not_implemented",
		Message: "Verification endpoint not yet implemented",
	})
}

// handleListSources handles list sources requests
func (s *Server) handleListSources(c *gin.Context) {
	sources := []SourceType{
		{
			Type:        SourceTypeDisk,
			Name:        "Physical Disk",
			Description: "Physical disk devices (e.g., /dev/sda)",
			Capabilities: SourceCapabilities{
				Seekable:  true,
				SizeKnown: true,
				Resumable: true,
				Streaming: false,
			},
		},
		{
			Type:        SourceTypeFile,
			Name:        "File",
			Description: "Local filesystem file",
			Capabilities: SourceCapabilities{
				Seekable:  true,
				SizeKnown: true,
				Resumable: true,
				Streaming: false,
			},
		},
		{
			Type:        SourceTypeS3,
			Name:        "Amazon S3",
			Description: "Amazon S3 object storage",
			Capabilities: SourceCapabilities{
				Seekable:  true,
				SizeKnown: true,
				Resumable: true,
				Streaming: false,
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{"sources": sources})
}

// handleListFormats handles list formats requests
func (s *Server) handleListFormats(c *gin.Context) {
	formats := []string{"raw", "e01", "encrypted", "compressed"}
	c.JSON(http.StatusOK, gin.H{"formats": formats})
}

// validateImageJobRequest validates an image job request
func (s *Server) validateImageJobRequest(req *CreateImageJobRequest) error {
	if req.Source.Type == "" {
		return fmt.Errorf("source type is required")
	}
	if req.Destination.Type == "" {
		return fmt.Errorf("destination type is required")
	}
	return nil
}
