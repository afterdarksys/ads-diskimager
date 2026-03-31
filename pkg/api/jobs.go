package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Job represents an imaging job
type Job struct {
	ID          string
	Request     CreateImageJobRequest
	Status      string
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Progress    *JobProgress
	Result      *JobResult
	Error       error
	CancelFunc  context.CancelFunc

	// Progress channel for real-time updates
	progressChan chan *JobProgress
	subscribers  []chan *JobProgress
	subMutex     sync.RWMutex
}

// JobQueue manages asynchronous imaging jobs
type JobQueue struct {
	jobs       map[string]*Job
	jobsMutex  sync.RWMutex
	workerPool chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewJobQueue creates a new job queue
func NewJobQueue(maxWorkers int) *JobQueue {
	ctx, cancel := context.WithCancel(context.Background())

	return &JobQueue{
		jobs:       make(map[string]*Job),
		workerPool: make(chan struct{}, maxWorkers),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// CreateJob creates a new job and adds it to the queue
func (jq *JobQueue) CreateJob(req CreateImageJobRequest) (*Job, error) {
	jobID := uuid.New().String()

	job := &Job{
		ID:           jobID,
		Request:      req,
		Status:       StatusQueued,
		CreatedAt:    time.Now(),
		progressChan: make(chan *JobProgress, 10),
		subscribers:  make([]chan *JobProgress, 0),
	}

	jq.jobsMutex.Lock()
	jq.jobs[jobID] = job
	jq.jobsMutex.Unlock()

	// Start the job asynchronously
	go jq.executeJob(job)

	return job, nil
}

// GetJob retrieves a job by ID
func (jq *JobQueue) GetJob(jobID string) (*Job, error) {
	jq.jobsMutex.RLock()
	defer jq.jobsMutex.RUnlock()

	job, exists := jq.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// ListJobs lists all jobs with optional status filter
func (jq *JobQueue) ListJobs(statusFilter string, limit, offset int) ([]*Job, int) {
	jq.jobsMutex.RLock()
	defer jq.jobsMutex.RUnlock()

	// Collect matching jobs
	var filteredJobs []*Job
	for _, job := range jq.jobs {
		if statusFilter == "" || job.Status == statusFilter {
			filteredJobs = append(filteredJobs, job)
		}
	}

	total := len(filteredJobs)

	// Apply pagination
	start := offset
	if start > total {
		start = total
	}

	end := start + limit
	if end > total {
		end = total
	}

	return filteredJobs[start:end], total
}

// CancelJob cancels a running or queued job
func (jq *JobQueue) CancelJob(jobID string) error {
	jq.jobsMutex.Lock()
	defer jq.jobsMutex.Unlock()

	job, exists := jq.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status == StatusCompleted || job.Status == StatusFailed {
		return fmt.Errorf("cannot cancel job in status: %s", job.Status)
	}

	// Cancel the job context
	if job.CancelFunc != nil {
		job.CancelFunc()
	}

	job.Status = StatusCancelled
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// Subscribe creates a subscription to job progress updates
func (job *Job) Subscribe() <-chan *JobProgress {
	job.subMutex.Lock()
	defer job.subMutex.Unlock()

	ch := make(chan *JobProgress, 10)
	job.subscribers = append(job.subscribers, ch)

	// Send current progress immediately if available
	if job.Progress != nil {
		select {
		case ch <- job.Progress:
		default:
		}
	}

	return ch
}

// Unsubscribe removes a progress subscription
func (job *Job) Unsubscribe(ch <-chan *JobProgress) {
	job.subMutex.Lock()
	defer job.subMutex.Unlock()

	for i, subscriber := range job.subscribers {
		if subscriber == ch {
			close(subscriber)
			job.subscribers = append(job.subscribers[:i], job.subscribers[i+1:]...)
			break
		}
	}
}

// updateProgress broadcasts progress to all subscribers
func (job *Job) updateProgress(progress *JobProgress) {
	job.Progress = progress

	job.subMutex.RLock()
	defer job.subMutex.RUnlock()

	for _, subscriber := range job.subscribers {
		select {
		case subscriber <- progress:
		default:
			// Skip if subscriber channel is full
		}
	}
}

// executeJob executes an imaging job
func (jq *JobQueue) executeJob(job *Job) {
	// Wait for worker slot
	jq.workerPool <- struct{}{}
	defer func() { <-jq.workerPool }()

	// Create cancellable context
	ctx, cancel := context.WithCancel(jq.ctx)
	job.CancelFunc = cancel
	defer cancel()

	// Update status to running
	job.Status = StatusRunning
	now := time.Now()
	job.StartedAt = &now

	// Send initial progress
	job.updateProgress(&JobProgress{
		Phase:      PhaseInitializing,
		Percentage: 0,
	})

	// Execute the imaging operation
	// TODO: This will call the actual imaging logic
	result, err := jq.performImaging(ctx, job)

	// Update final status
	completedAt := time.Now()
	job.CompletedAt = &completedAt

	if err != nil {
		if ctx.Err() == context.Canceled {
			job.Status = StatusCancelled
		} else {
			job.Status = StatusFailed
			job.Error = err
		}
	} else {
		job.Status = StatusCompleted
		job.Result = result

		// Send final progress
		job.updateProgress(&JobProgress{
			Phase:      PhaseCompleted,
			Percentage: 100,
		})
	}

	// Close all subscriber channels
	job.subMutex.Lock()
	for _, subscriber := range job.subscribers {
		close(subscriber)
	}
	job.subscribers = nil
	job.subMutex.Unlock()
}

// performImaging performs the actual imaging operation
func (jq *JobQueue) performImaging(ctx context.Context, job *Job) (*JobResult, error) {
	// Create imaging engine
	engine := newImagingEngine(ctx, job)

	// Execute the imaging operation
	result, err := engine.performActualImaging()
	if err != nil {
		return nil, fmt.Errorf("imaging failed: %w", err)
	}

	return result, nil
}

// Shutdown gracefully shuts down the job queue
func (jq *JobQueue) Shutdown() {
	jq.cancel()

	// Wait for all workers to complete
	for i := 0; i < cap(jq.workerPool); i++ {
		jq.workerPool <- struct{}{}
	}
}

// ToJobResponse converts a Job to a JobResponse
func (job *Job) ToJobResponse(baseURL string) JobResponse {
	resp := JobResponse{
		JobID:       job.ID,
		Status:      job.Status,
		CreatedAt:   job.CreatedAt,
		StartedAt:   job.StartedAt,
		CompletedAt: job.CompletedAt,
		Source:      job.Request.Source,
		Destination: job.Request.Destination,
		Options:     job.Request.Options,
		Metadata:    job.Request.Metadata,
		Progress:    job.Progress,
		Result:      job.Result,
		StreamURL:   fmt.Sprintf("%s/api/v1/jobs/%s/stream", baseURL, job.ID),
	}

	if job.Error != nil {
		resp.Error = job.Error.Error()
	}

	return resp
}
