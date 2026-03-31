package blockeditor

import (
	"time"
)

// BlockSize represents the size of each block in the visualization
const DefaultBlockSize = 4096 // 4KB blocks

// BlockStatus represents the allocation status of a block
type BlockStatus string

const (
	StatusAllocated   BlockStatus = "allocated"   // Block is allocated to a file
	StatusUnallocated BlockStatus = "unallocated" // Block is free
	StatusSlack       BlockStatus = "slack"       // Slack space
	StatusDeleted     BlockStatus = "deleted"     // Deleted file remnant
	StatusSystem      BlockStatus = "system"      // System/metadata
	StatusBad         BlockStatus = "bad"         // Bad sector
)

// BlockType represents the type of data in a block
type BlockType string

const (
	TypeUnknown     BlockType = "unknown"
	TypeDirectory   BlockType = "directory"
	TypeFile        BlockType = "file"
	TypeMetadata    BlockType = "metadata"
	TypeBootSector  BlockType = "boot"
	TypePartition   BlockType = "partition"
	TypeJournal     BlockType = "journal"
	TypeImage       BlockType = "image"
	TypeVideo       BlockType = "video"
	TypeAudio       BlockType = "audio"
	TypeDocument    BlockType = "document"
	TypeArchive     BlockType = "archive"
	TypeExecutable  BlockType = "executable"
	TypeDatabase    BlockType = "database"
	TypeEncrypted   BlockType = "encrypted"
	TypeCompressed  BlockType = "compressed"
)

// Block represents a single block on the disk
type Block struct {
	Index      int64       `json:"index"`       // Block index
	Offset     int64       `json:"offset"`      // Byte offset on disk
	Size       int         `json:"size"`        // Block size in bytes
	Status     BlockStatus `json:"status"`      // Allocation status
	Type       BlockType   `json:"type"`        // Data type
	FileID     string      `json:"file_id"`     // Associated file ID (if any)
	FileName   string      `json:"file_name"`   // File name (if known)
	FilePath   string      `json:"file_path"`   // Full file path
	Signature  string      `json:"signature"`   // Magic bytes signature
	Hash       string      `json:"hash"`        // Block hash (SHA256)
	IsZero     bool        `json:"is_zero"`     // Is block all zeros
	Modified   time.Time   `json:"modified"`    // Last modified time (if known)
	Entropy    float64     `json:"entropy"`     // Data entropy (0-8 bits)
	Compressed bool        `json:"compressed"`  // Appears compressed
	Encrypted  bool        `json:"encrypted"`   // Appears encrypted
}

// FileInfo represents information about a file on disk
type FileInfo struct {
	ID         string      `json:"id"`          // Unique file ID
	Name       string      `json:"name"`        // File name
	Path       string      `json:"path"`        // Full path
	Size       int64       `json:"size"`        // File size in bytes
	BlockCount int         `json:"block_count"` // Number of blocks
	Blocks     []int64     `json:"blocks"`      // Block indices
	Type       BlockType   `json:"type"`        // File type
	Status     BlockStatus `json:"status"`      // File status
	Created    time.Time   `json:"created"`     // Creation time
	Modified   time.Time   `json:"modified"`    // Modification time
	Accessed   time.Time   `json:"accessed"`    // Access time
	Deleted    bool        `json:"deleted"`     // Is deleted
	Fragmented bool        `json:"fragmented"`  // Is fragmented
	Signature  string      `json:"signature"`   // File signature
	MIMEType   string      `json:"mime_type"`   // MIME type
	Hash       string      `json:"hash"`        // File hash
}

// DiskMap represents the entire disk block map
type DiskMap struct {
	ImagePath      string               `json:"image_path"`
	TotalSize      int64                `json:"total_size"`
	BlockSize      int                  `json:"block_size"`
	TotalBlocks    int64                `json:"total_blocks"`
	Blocks         []*Block             `json:"-"` // Full block data (not sent to client)
	Files          map[string]*FileInfo `json:"files"`
	Filesystem     string               `json:"filesystem"`
	VolumeLabel    string               `json:"volume_label"`
	Created        time.Time            `json:"created"`
	LastMounted    time.Time            `json:"last_mounted"`
	Statistics     *Statistics          `json:"statistics"`
	AnalysisDate   time.Time            `json:"analysis_date"`
}

// Statistics holds aggregate statistics about the disk
type Statistics struct {
	AllocatedBlocks   int64   `json:"allocated_blocks"`
	UnallocatedBlocks int64   `json:"unallocated_blocks"`
	DeletedBlocks     int64   `json:"deleted_blocks"`
	SystemBlocks      int64   `json:"system_blocks"`
	BadBlocks         int64   `json:"bad_blocks"`
	ZeroBlocks        int64   `json:"zero_blocks"`
	FileCount         int     `json:"file_count"`
	DeletedFileCount  int     `json:"deleted_file_count"`
	FragmentedFiles   int     `json:"fragmented_files"`
	UsedSpace         int64   `json:"used_space"`
	FreeSpace         int64   `json:"free_space"`
	WastedSpace       int64   `json:"wasted_space"` // Slack space
	Utilization       float64 `json:"utilization"`  // Percentage
}

// BlockRange represents a range of blocks for efficient queries
type BlockRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
	Count int64 `json:"count"`
}

// SearchQuery represents a search query for blocks
type SearchQuery struct {
	FileID       string      `json:"file_id,omitempty"`
	FileName     string      `json:"file_name,omitempty"`
	Type         BlockType   `json:"type,omitempty"`
	Status       BlockStatus `json:"status,omitempty"`
	MinEntropy   float64     `json:"min_entropy,omitempty"`
	MaxEntropy   float64     `json:"max_entropy,omitempty"`
	IsZero       *bool       `json:"is_zero,omitempty"`
	IsCompressed *bool       `json:"is_compressed,omitempty"`
	IsEncrypted  *bool       `json:"is_encrypted,omitempty"`
	IsDeleted    *bool       `json:"is_deleted,omitempty"`
	Range        *BlockRange `json:"range,omitempty"`
	Signature    string      `json:"signature,omitempty"`
}

// BlockSummary is a lightweight version of Block for efficient transfer
type BlockSummary struct {
	Index    int64       `json:"index"`
	Status   BlockStatus `json:"status"`
	Type     BlockType   `json:"type"`
	FileID   string      `json:"file_id,omitempty"`
	IsZero   bool        `json:"is_zero,omitempty"`
	Entropy  float64     `json:"entropy,omitempty"`
}

// ColorScheme defines colors for different block types and statuses
type ColorScheme struct {
	Allocated   string `json:"allocated"`   // #4CAF50
	Unallocated string `json:"unallocated"` // #9E9E9E
	Deleted     string `json:"deleted"`     // #F44336
	System      string `json:"system"`      // #2196F3
	Bad         string `json:"bad"`         // #000000
	Image       string `json:"image"`       // #FF9800
	Video       string `json:"video"`       // #9C27B0
	Audio       string `json:"audio"`       // #00BCD4
	Document    string `json:"document"`    // #FFC107
	Executable  string `json:"executable"`  // #E91E63
	Archive     string `json:"archive"`     // #607D8B
	Encrypted   string `json:"encrypted"`   // #3F51B5
	Compressed  string `json:"compressed"`  // #8BC34A
}

// DefaultColorScheme returns the default color scheme
func DefaultColorScheme() *ColorScheme {
	return &ColorScheme{
		Allocated:   "#4CAF50",
		Unallocated: "#9E9E9E",
		Deleted:     "#F44336",
		System:      "#2196F3",
		Bad:         "#000000",
		Image:       "#FF9800",
		Video:       "#9C27B0",
		Audio:       "#00BCD4",
		Document:    "#FFC107",
		Executable:  "#E91E63",
		Archive:     "#607D8B",
		Encrypted:   "#3F51B5",
		Compressed:  "#8BC34A",
	}
}

// AnalysisOptions defines options for disk analysis
type AnalysisOptions struct {
	BlockSize           int    `json:"block_size"`
	ComputeHashes       bool   `json:"compute_hashes"`
	ComputeEntropy      bool   `json:"compute_entropy"`
	DetectFileTypes     bool   `json:"detect_file_types"`
	ParseFilesystem     bool   `json:"parse_filesystem"`
	IdentifySignatures  bool   `json:"identify_signatures"`
	MaxBlocks           int64  `json:"max_blocks"` // Limit for large disks
	SampleRate          int    `json:"sample_rate"` // Sample every N blocks for large disks
}

// DefaultAnalysisOptions returns default analysis options
func DefaultAnalysisOptions() *AnalysisOptions {
	return &AnalysisOptions{
		BlockSize:          DefaultBlockSize,
		ComputeHashes:      false, // Expensive
		ComputeEntropy:     true,
		DetectFileTypes:    true,
		ParseFilesystem:    true,
		IdentifySignatures: true,
		MaxBlocks:          0, // No limit
		SampleRate:         1, // Analyze every block
	}
}
