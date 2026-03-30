package progress

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Phase represents the current phase of an operation
type Phase string

const (
	PhaseInitializing Phase = "initializing"
	PhaseReading      Phase = "reading"
	PhaseHashing      Phase = "hashing"
	PhaseCompressing  Phase = "compressing"
	PhaseWriting      Phase = "writing"
	PhaseVerifying    Phase = "verifying"
	PhaseCompleting   Phase = "completing"
	PhaseCancelling   Phase = "cancelling"
	PhaseCompleted    Phase = "completed"
	PhaseFailed       Phase = "failed"
)

// OperationStatus represents the current status of an operation
type OperationStatus int

const (
	StatusPending OperationStatus = iota
	StatusRunning
	StatusPaused
	StatusCompleted
	StatusFailed
	StatusCancelled
)

// String returns the string representation of status
func (s OperationStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusPaused:
		return "paused"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Progress represents the current progress of an operation
type Progress struct {
	BytesProcessed int64         `json:"bytes_processed"`
	TotalBytes     int64         `json:"total_bytes"`
	Phase          Phase         `json:"phase"`
	Message        string        `json:"message"`
	Speed          int64         `json:"speed"` // bytes per second
	ETA            time.Duration `json:"eta"`
	Timestamp      time.Time     `json:"timestamp"`
	Percentage     float64       `json:"percentage"`
	BadSectors     int           `json:"bad_sectors"`
	Errors         []string      `json:"errors,omitempty"`
}

// Operation represents a long-running operation with progress tracking
type Operation interface {
	Start(ctx context.Context) error
	Progress() <-chan Progress
	Cancel() error
	Status() OperationStatus
	Wait() error
}

// Tracker tracks progress for an operation
type Tracker struct {
	totalBytes     int64
	bytesProcessed int64
	badSectors     int64
	startTime      time.Time
	phase          Phase
	message        string
	status         OperationStatus
	errors         []string

	progressChan chan Progress
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex

	updateInterval time.Duration
	lastUpdate     time.Time
	lastBytes      int64
}

// NewTracker creates a new progress tracker
func NewTracker(totalBytes int64, updateInterval time.Duration) *Tracker {
	if updateInterval <= 0 {
		updateInterval = 500 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(context.Background())

	t := &Tracker{
		totalBytes:     totalBytes,
		progressChan:   make(chan Progress, 10),
		ctx:            ctx,
		cancel:         cancel,
		updateInterval: updateInterval,
		status:         StatusPending,
		phase:          PhaseInitializing,
		startTime:      time.Now(),
		lastUpdate:     time.Now(),
	}

	// Start progress reporter
	t.wg.Add(1)
	go t.progressReporter()

	return t
}

// progressReporter periodically sends progress updates
func (t *Tracker) progressReporter() {
	defer t.wg.Done()
	defer close(t.progressChan)

	ticker := time.NewTicker(t.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			// Send final progress
			t.sendProgress()
			return
		case <-ticker.C:
			t.sendProgress()
		}
	}
}

// sendProgress sends a progress update
func (t *Tracker) sendProgress() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	bytesProcessed := atomic.LoadInt64(&t.bytesProcessed)
	elapsed := now.Sub(t.lastUpdate).Seconds()

	// Calculate speed
	var speed int64
	if elapsed > 0 {
		bytesDelta := bytesProcessed - t.lastBytes
		speed = int64(float64(bytesDelta) / elapsed)
	}

	// Calculate ETA
	var eta time.Duration
	if speed > 0 && t.totalBytes > 0 {
		remaining := t.totalBytes - bytesProcessed
		etaSeconds := float64(remaining) / float64(speed)
		eta = time.Duration(etaSeconds) * time.Second
	}

	// Calculate percentage
	var percentage float64
	if t.totalBytes > 0 {
		percentage = float64(bytesProcessed) / float64(t.totalBytes) * 100.0
	}

	progress := Progress{
		BytesProcessed: bytesProcessed,
		TotalBytes:     t.totalBytes,
		Phase:          t.phase,
		Message:        t.message,
		Speed:          speed,
		ETA:            eta,
		Timestamp:      now,
		Percentage:     percentage,
		BadSectors:     int(atomic.LoadInt64(&t.badSectors)),
		Errors:         append([]string{}, t.errors...), // Copy errors
	}

	select {
	case t.progressChan <- progress:
	default:
		// Channel full, skip this update
	}

	t.lastUpdate = now
	t.lastBytes = bytesProcessed
}

// Progress returns the progress channel
func (t *Tracker) Progress() <-chan Progress {
	return t.progressChan
}

// AddBytes atomically adds to bytes processed
func (t *Tracker) AddBytes(n int64) {
	atomic.AddInt64(&t.bytesProcessed, n)
}

// SetBytesProcessed sets the current bytes processed
func (t *Tracker) SetBytesProcessed(n int64) {
	atomic.StoreInt64(&t.bytesProcessed, n)
}

// GetBytesProcessed returns the current bytes processed
func (t *Tracker) GetBytesProcessed() int64 {
	return atomic.LoadInt64(&t.bytesProcessed)
}

// AddBadSector increments bad sector count
func (t *Tracker) AddBadSector() {
	atomic.AddInt64(&t.badSectors, 1)
}

// SetPhase sets the current phase
func (t *Tracker) SetPhase(phase Phase) {
	t.mu.Lock()
	t.phase = phase
	t.mu.Unlock()
	t.sendProgress()
}

// SetMessage sets the current message
func (t *Tracker) SetMessage(message string) {
	t.mu.Lock()
	t.message = message
	t.mu.Unlock()
}

// SetStatus sets the operation status
func (t *Tracker) SetStatus(status OperationStatus) {
	t.mu.Lock()
	t.status = status
	t.mu.Unlock()
	t.sendProgress()
}

// Status returns the current status
func (t *Tracker) Status() OperationStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// AddError adds an error to the tracker
func (t *Tracker) AddError(err error) {
	t.mu.Lock()
	if len(t.errors) < 100 { // Limit error history
		t.errors = append(t.errors, err.Error())
	}
	t.mu.Unlock()
}

// Cancel cancels the operation
func (t *Tracker) Cancel() error {
	t.mu.Lock()
	t.status = StatusCancelled
	t.phase = PhaseCancelling
	t.mu.Unlock()

	t.cancel()
	return nil
}

// Complete marks the operation as completed
func (t *Tracker) Complete() {
	t.mu.Lock()
	t.status = StatusCompleted
	t.phase = PhaseCompleted
	t.mu.Unlock()

	t.sendProgress()
	t.cancel()
}

// Fail marks the operation as failed
func (t *Tracker) Fail(err error) {
	t.mu.Lock()
	t.status = StatusFailed
	t.phase = PhaseFailed
	if err != nil {
		t.errors = append(t.errors, err.Error())
	}
	t.mu.Unlock()

	t.sendProgress()
	t.cancel()
}

// Wait waits for the progress reporter to finish
func (t *Tracker) Wait() {
	t.wg.Wait()
}

// GetElapsed returns the elapsed time since start
func (t *Tracker) GetElapsed() time.Duration {
	return time.Since(t.startTime)
}

// MultiTracker tracks multiple concurrent operations
type MultiTracker struct {
	trackers map[string]*Tracker
	mu       sync.RWMutex
}

// NewMultiTracker creates a new multi-tracker
func NewMultiTracker() *MultiTracker {
	return &MultiTracker{
		trackers: make(map[string]*Tracker),
	}
}

// Add adds a tracker with the given ID
func (mt *MultiTracker) Add(id string, tracker *Tracker) {
	mt.mu.Lock()
	mt.trackers[id] = tracker
	mt.mu.Unlock()
}

// Get retrieves a tracker by ID
func (mt *MultiTracker) Get(id string) *Tracker {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.trackers[id]
}

// Remove removes a tracker by ID
func (mt *MultiTracker) Remove(id string) {
	mt.mu.Lock()
	delete(mt.trackers, id)
	mt.mu.Unlock()
}

// GetAll returns all trackers
func (mt *MultiTracker) GetAll() map[string]*Tracker {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	result := make(map[string]*Tracker, len(mt.trackers))
	for k, v := range mt.trackers {
		result[k] = v
	}
	return result
}

// CancelAll cancels all tracked operations
func (mt *MultiTracker) CancelAll() {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	for _, tracker := range mt.trackers {
		tracker.Cancel()
	}
}
