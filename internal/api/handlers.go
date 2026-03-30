package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/afterdarksys/diskimager/imager"
	"github.com/afterdarksys/diskimager/pkg/format/e01"
	"github.com/afterdarksys/diskimager/pkg/format/raw"
	"github.com/afterdarksys/diskimager/pkg/progress"
	"github.com/afterdarksys/diskimager/pkg/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DiskInfo represents disk information
type DiskInfo struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Model  string `json:"model"`
	Serial string `json:"serial"`
	Health string `json:"health"`
	Type   string `json:"type"` // HDD, SSD, USB
}

// JobRequest represents a job creation request
type JobRequest struct {
	Type        string            `json:"type"` // image, restore
	SourcePath  string            `json:"source_path"`
	DestPath    string            `json:"dest_path"`
	Format      string            `json:"format"`
	Compression int               `json:"compression"`
	Metadata    map[string]string `json:"metadata"`
}

// Job represents an imaging/restore job
type Job struct {
	ID          string                `json:"id"`
	Type        string                `json:"type"`
	Status      progress.OperationStatus `json:"status"`
	Phase       progress.Phase          `json:"phase"`
	Progress    float64               `json:"progress"`
	Speed       int64                 `json:"speed"`
	ETA         string                `json:"eta"`
	BytesTotal  int64                 `json:"bytes_total"`
	BytesDone   int64                 `json:"bytes_done"`
	BadSectors  int                   `json:"bad_sectors"`
	Errors      []string              `json:"errors"`
	CreatedAt   time.Time             `json:"created_at"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	SourcePath  string                `json:"source_path"`
	DestPath    string                `json:"dest_path"`
	Format      string                `json:"format"`
	Hash        string                `json:"hash,omitempty"`

	// Internal
	tracker *progress.Tracker
	cancel  context.CancelFunc
}

// APIServer manages the REST API and jobs
type APIServer struct {
	router *gin.Engine
	jobs   map[string]*Job
	jobsMu sync.RWMutex
	ws     *WebSocketServer
}

// NewAPIServer creates a new API server
func NewAPIServer() *APIServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	server := &APIServer{
		router: router,
		jobs:   make(map[string]*Job),
		ws:     NewWebSocketServer(),
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures API routes
func (s *APIServer) setupRoutes() {
	api := s.router.Group("/api")
	{
		// Disk operations
		api.GET("/disks", s.listDisks)

		// Job operations
		api.POST("/jobs/image", s.createImageJob)
		api.POST("/jobs/restore", s.createRestoreJob)
		api.GET("/jobs", s.listJobs)
		api.GET("/jobs/:id", s.getJob)
		api.DELETE("/jobs/:id", s.deleteJob)

		// WebSocket endpoint
		api.GET("/ws/jobs/:id", s.ws.HandleConnection)
	}

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now(),
		})
	})
}

// Run starts the API server
func (s *APIServer) Run(addr string) error {
	return s.router.Run(addr)
}

// RunWithServer starts the API server and returns the underlying http.Server
func (s *APIServer) RunWithServer(addr string) (*http.Server, error) {
	server := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	return server, nil
}

// Shutdown gracefully shuts down the API server
func (s *APIServer) Shutdown(ctx context.Context) error {
	// Cancel all running jobs
	s.jobsMu.Lock()
	activeJobs := 0
	for _, job := range s.jobs {
		if job.Status == progress.StatusRunning || job.Status == progress.StatusPending {
			activeJobs++
			if job.cancel != nil {
				job.cancel()
			}
			if job.tracker != nil {
				job.tracker.Fail(fmt.Errorf("server shutdown"))
			}
		}
	}
	s.jobsMu.Unlock()

	if activeJobs > 0 {
		// Give jobs a moment to clean up
		time.Sleep(2 * time.Second)
	}

	return nil
}

// listDisks returns available disks
func (s *APIServer) listDisks(c *gin.Context) {
	// Mock disk list - would integrate with actual disk detection
	disks := []DiskInfo{
		{
			Path:   "/dev/disk0",
			Name:   "System SSD",
			Size:   500 * 1024 * 1024 * 1024, // 500GB
			Model:  "Apple SSD Controller",
			Serial: "SSD-12345",
			Health: "Healthy",
			Type:   "SSD",
		},
		{
			Path:   "/dev/disk1",
			Name:   "Data HDD",
			Size:   2 * 1024 * 1024 * 1024 * 1024, // 2TB
			Model:  "WD Blue 2TB",
			Serial: "WD-67890",
			Health: "Healthy",
			Type:   "HDD",
		},
		{
			Path:   "/dev/disk2",
			Name:   "USB Drive",
			Size:   1 * 1024 * 1024 * 1024 * 1024, // 1TB
			Model:  "SanDisk Ultra",
			Serial: "USB-54321",
			Health: "Healthy",
			Type:   "USB",
		},
	}

	c.JSON(http.StatusOK, disks)
}

// createImageJob creates a new imaging job
func (s *APIServer) createImageJob(c *gin.Context) {
	var req JobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request
	if req.SourcePath == "" || req.DestPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_path and dest_path are required"})
		return
	}

	// Get source size
	stat, err := os.Stat(req.SourcePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot access source: %v", err)})
		return
	}

	// Create job
	jobID := uuid.New().String()
	job := &Job{
		ID:         jobID,
		Type:       "image",
		Status:     progress.StatusPending,
		Phase:      progress.PhaseInitializing,
		BytesTotal: stat.Size(),
		CreatedAt:  time.Now(),
		SourcePath: req.SourcePath,
		DestPath:   req.DestPath,
		Format:     req.Format,
	}

	s.jobsMu.Lock()
	s.jobs[jobID] = job
	s.jobsMu.Unlock()

	// Start job in background
	go s.executeImageJob(job, req)

	c.JSON(http.StatusCreated, gin.H{
		"id":     jobID,
		"status": "created",
	})
}

// executeImageJob executes an imaging job
func (s *APIServer) executeImageJob(job *Job, req JobRequest) {
	_, cancel := context.WithCancel(context.Background())
	job.cancel = cancel
	defer cancel()

	// Update status
	s.updateJobStatus(job.ID, progress.StatusRunning, progress.PhaseReading)

	// Open source
	source, err := os.Open(req.SourcePath)
	if err != nil {
		s.updateJobError(job.ID, fmt.Errorf("failed to open source: %v", err))
		return
	}
	defer source.Close()

	// Open destination
	outTarget, err := storage.OpenDestination(req.DestPath, false)
	if err != nil {
		s.updateJobError(job.ID, fmt.Errorf("failed to open destination: %v", err))
		return
	}
	defer outTarget.Close()

	// Create format writer
	var out io.WriteCloser
	metadata := imager.Metadata{
		CaseNumber:  req.Metadata["case_number"],
		Examiner:    req.Metadata["examiner"],
		EvidenceNum: req.Metadata["evidence"],
		Description: req.Metadata["description"],
	}

	switch req.Format {
	case "E01":
		out, err = e01.NewWriter(outTarget, false, metadata)
	default:
		out, err = raw.NewWriter(outTarget)
	}

	if err != nil {
		s.updateJobError(job.ID, fmt.Errorf("failed to create writer: %v", err))
		return
	}
	defer out.Close()

	// Create progress tracker
	tracker := progress.NewTracker(job.BytesTotal, 500*time.Millisecond)
	job.tracker = tracker

	// Monitor progress and broadcast to WebSocket clients
	go func() {
		for prog := range tracker.Progress() {
			s.jobsMu.Lock()
			if j, ok := s.jobs[job.ID]; ok {
				j.Phase = prog.Phase
				j.Progress = prog.Percentage
				j.Speed = prog.Speed
				j.BytesDone = prog.BytesProcessed
				j.BadSectors = prog.BadSectors
				j.Errors = prog.Errors
				if prog.ETA > 0 {
					j.ETA = prog.ETA.Round(time.Second).String()
				}
			}
			s.jobsMu.Unlock()

			// Broadcast to WebSocket clients
			s.ws.BroadcastProgress(job.ID, prog)
		}
	}()

	// Perform imaging
	cfg := imager.Config{
		Source:      io.TeeReader(source, &progressWriter{tracker: tracker}),
		Destination: out,
		HashAlgo:    "sha256",
		Metadata:    metadata,
	}

	result, imgErr := imager.Image(cfg)

	if imgErr != nil {
		s.updateJobError(job.ID, imgErr)
		tracker.Fail(imgErr)
	} else {
		s.jobsMu.Lock()
		if j, ok := s.jobs[job.ID]; ok {
			j.Status = progress.StatusCompleted
			j.Phase = progress.PhaseCompleted
			j.Progress = 100.0
			j.Hash = result.Hash
			now := time.Now()
			j.CompletedAt = &now
		}
		s.jobsMu.Unlock()
		tracker.Complete()
	}

	tracker.Wait()
}

// createRestoreJob creates a new restore job
func (s *APIServer) createRestoreJob(c *gin.Context) {
	var req JobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Similar to createImageJob but for restore
	jobID := uuid.New().String()
	job := &Job{
		ID:         jobID,
		Type:       "restore",
		Status:     progress.StatusPending,
		Phase:      progress.PhaseInitializing,
		CreatedAt:  time.Now(),
		SourcePath: req.SourcePath,
		DestPath:   req.DestPath,
	}

	s.jobsMu.Lock()
	s.jobs[jobID] = job
	s.jobsMu.Unlock()

	c.JSON(http.StatusCreated, gin.H{
		"id":     jobID,
		"status": "created",
	})
}

// listJobs returns all jobs
func (s *APIServer) listJobs(c *gin.Context) {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, jobs)
}

// getJob returns a specific job
func (s *APIServer) getJob(c *gin.Context) {
	jobID := c.Param("id")

	s.jobsMu.RLock()
	job, ok := s.jobs[jobID]
	s.jobsMu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// deleteJob cancels and deletes a job
func (s *APIServer) deleteJob(c *gin.Context) {
	jobID := c.Param("id")

	s.jobsMu.Lock()
	job, ok := s.jobs[jobID]
	if ok {
		if job.cancel != nil {
			job.cancel()
		}
		delete(s.jobs, jobID)
	}
	s.jobsMu.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// updateJobStatus updates job status
func (s *APIServer) updateJobStatus(jobID string, status progress.OperationStatus, phase progress.Phase) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	if job, ok := s.jobs[jobID]; ok {
		job.Status = status
		job.Phase = phase
	}
}

// updateJobError updates job with error
func (s *APIServer) updateJobError(jobID string, err error) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	if job, ok := s.jobs[jobID]; ok {
		job.Status = progress.StatusFailed
		job.Phase = progress.PhaseFailed
		if err != nil {
			job.Errors = append(job.Errors, err.Error())
		}
		now := time.Now()
		job.CompletedAt = &now
	}
}

// progressWriter tracks progress
type progressWriter struct {
	tracker *progress.Tracker
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.tracker.AddBytes(int64(n))
	return n, nil
}

// MarshalJSON customizes Job JSON serialization
func (j *Job) MarshalJSON() ([]byte, error) {
	type Alias Job
	return json.Marshal(&struct {
		*Alias
		StatusStr string `json:"status_str"`
	}{
		Alias:     (*Alias)(j),
		StatusStr: j.Status.String(),
	})
}
