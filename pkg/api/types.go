package api

import (
	"time"

	"github.com/afterdarksys/diskimager/imager"
)

// Job statuses
const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Progress phases
const (
	PhaseInitializing = "initializing"
	PhaseReading      = "reading"
	PhaseHashing      = "hashing"
	PhaseCompressing  = "compressing"
	PhaseEncrypting   = "encrypting"
	PhaseWriting      = "writing"
	PhaseVerifying    = "verifying"
	PhaseCompleted    = "completed"
)

// Source types
const (
	SourceTypeDisk      = "disk"
	SourceTypeFile      = "file"
	SourceTypeS3        = "s3"
	SourceTypeAzure     = "azure-blob"
	SourceTypeGCS       = "gcs"
	SourceTypeHTTP      = "http"
	SourceTypeVMDK      = "vm-vmdk"
	SourceTypeVHD       = "vm-vhd"
)

// Destination types
const (
	DestinationTypeFile  = "file"
	DestinationTypeS3    = "s3"
	DestinationTypeAzure = "azure-blob"
	DestinationTypeGCS   = "gcs"
	DestinationTypeStream = "stream"
)

// CreateImageJobRequest represents a request to create an imaging job
type CreateImageJobRequest struct {
	Source      Source       `json:"source"`
	Destination Destination  `json:"destination"`
	Options     ImageOptions `json:"options"`
	Metadata    Metadata     `json:"metadata"`
}

// Source represents an imaging source
type Source struct {
	Type        string                 `json:"type"`
	Path        string                 `json:"path,omitempty"`
	Device      string                 `json:"device,omitempty"`
	Bucket      string                 `json:"bucket,omitempty"`
	Container   string                 `json:"container,omitempty"`
	Key         string                 `json:"key,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Credentials map[string]interface{} `json:"credentials,omitempty"`
}

// Destination represents an imaging destination
type Destination struct {
	Type      string `json:"type"`
	Path      string `json:"path,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
	Container string `json:"container,omitempty"`
	Key       string `json:"key,omitempty"`
	Format    string `json:"format,omitempty"`
}

// ImageOptions represents imaging options
type ImageOptions struct {
	BlockSize         int      `json:"block_size"`
	Compression       string   `json:"compression"`
	CompressionLevel  int      `json:"compression_level"`
	Encryption        bool     `json:"encryption"`
	EncryptionKey     string   `json:"encryption_key,omitempty"`
	HashAlgorithms    []string `json:"hash_algorithms"`
	BlockHash         bool     `json:"block_hash"`
	BlockHashSize     int64    `json:"block_hash_size"`
	DetectSparse      bool     `json:"detect_sparse"`
	RateLimit         int64    `json:"rate_limit"`
	ChunkSize         int64    `json:"chunk_size"`
}

// Metadata represents chain-of-custody metadata
type Metadata struct {
	CaseNumber     string                 `json:"case_number,omitempty"`
	EvidenceNumber string                 `json:"evidence_number,omitempty"`
	Examiner       string                 `json:"examiner,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Notes          string                 `json:"notes,omitempty"`
	Custom         map[string]interface{} `json:"custom,omitempty"`
}

// JobResponse represents a job's current state
type JobResponse struct {
	JobID       string       `json:"job_id"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	StartedAt   *time.Time   `json:"started_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	Source      Source       `json:"source"`
	Destination Destination  `json:"destination"`
	Options     ImageOptions `json:"options"`
	Metadata    Metadata     `json:"metadata"`
	Progress    *JobProgress `json:"progress,omitempty"`
	Result      *JobResult   `json:"result,omitempty"`
	Error       string       `json:"error,omitempty"`
	StreamURL   string       `json:"stream_url,omitempty"`
}

// JobProgress represents current progress of a job
type JobProgress struct {
	Phase          string  `json:"phase"`
	BytesProcessed int64   `json:"bytes_processed"`
	TotalBytes     int64   `json:"total_bytes"`
	Percentage     float64 `json:"percentage"`
	Speed          int64   `json:"speed"`
	ETA            int     `json:"eta"`
}

// JobResult represents the result of a completed job
type JobResult struct {
	BytesCopied int64                  `json:"bytes_copied"`
	Hashes      map[string]string      `json:"hashes"`
	BadSectors  []imager.BadSector     `json:"bad_sectors,omitempty"`
	Duration    int                    `json:"duration"`
	Artifacts   []string               `json:"artifacts,omitempty"`
}

// VerifyRequest represents a verification request
type VerifyRequest struct {
	Source              Source `json:"source"`
	ExpectedHash        string `json:"expected_hash,omitempty"`
	HashAlgorithm       string `json:"hash_algorithm"`
	BlockHashManifest   string `json:"block_hash_manifest,omitempty"`
}

// VerifyResponse represents a verification response
type VerifyResponse struct {
	Verified          bool                  `json:"verified"`
	ComputedHash      string                `json:"computed_hash"`
	ExpectedHash      string                `json:"expected_hash,omitempty"`
	Algorithm         string                `json:"algorithm"`
	BytesVerified     int64                 `json:"bytes_verified"`
	Duration          int                   `json:"duration"`
	BlockVerification *BlockVerification    `json:"block_verification,omitempty"`
}

// BlockVerification represents block-level verification results
type BlockVerification struct {
	TotalBlocks      int   `json:"total_blocks"`
	VerifiedBlocks   int   `json:"verified_blocks"`
	CorruptedBlocks  []int `json:"corrupted_blocks,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    int64     `json:"uptime"`
}

// VersionResponse represents version information
type VersionResponse struct {
	APIVersion         string `json:"api_version"`
	DiskimagerVersion  string `json:"diskimager_version"`
	BuildDate          string `json:"build_date"`
	GitCommit          string `json:"git_commit"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Artifact represents a downloadable job artifact
type Artifact struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	DownloadURL string    `json:"download_url"`
}

// SourceType represents a source type definition
type SourceType struct {
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Capabilities SourceCapabilities `json:"capabilities"`
}

// SourceCapabilities represents source capabilities
type SourceCapabilities struct {
	Seekable  bool `json:"seekable"`
	SizeKnown bool `json:"size_known"`
	Resumable bool `json:"resumable"`
	Streaming bool `json:"streaming"`
}

// ToImagerMetadata converts API metadata to imager metadata
func (m Metadata) ToImagerMetadata() imager.Metadata {
	return imager.Metadata{
		CaseNumber:  m.CaseNumber,
		EvidenceNum: m.EvidenceNumber,
		Examiner:    m.Examiner,
		Description: m.Description,
		Notes:       m.Notes,
	}
}
