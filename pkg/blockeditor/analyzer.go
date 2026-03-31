package blockeditor

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Analyzer analyzes disk images and builds block maps
type Analyzer struct {
	imagePath string
	imageFile *os.File
	options   *AnalysisOptions
	diskMap   *DiskMap
}

// NewAnalyzer creates a new disk analyzer
func NewAnalyzer(imagePath string, options *AnalysisOptions) (*Analyzer, error) {
	if options == nil {
		options = DefaultAnalysisOptions()
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat image: %w", err)
	}

	totalBlocks := stat.Size() / int64(options.BlockSize)
	if stat.Size()%int64(options.BlockSize) != 0 {
		totalBlocks++
	}

	diskMap := &DiskMap{
		ImagePath:    imagePath,
		TotalSize:    stat.Size(),
		BlockSize:    options.BlockSize,
		TotalBlocks:  totalBlocks,
		Blocks:       make([]*Block, 0, totalBlocks),
		Files:        make(map[string]*FileInfo),
		Statistics:   &Statistics{},
		AnalysisDate: time.Now(),
	}

	return &Analyzer{
		imagePath: imagePath,
		imageFile: file,
		options:   options,
		diskMap:   diskMap,
	}, nil
}

// Close closes the analyzer
func (a *Analyzer) Close() error {
	if a.imageFile != nil {
		return a.imageFile.Close()
	}
	return nil
}

// Analyze performs the disk analysis
func (a *Analyzer) Analyze() (*DiskMap, error) {
	defer a.Close()

	fmt.Printf("Analyzing disk image: %s\n", a.imagePath)
	fmt.Printf("Total size: %d bytes (%d blocks)\n", a.diskMap.TotalSize, a.diskMap.TotalBlocks)

	// Analyze blocks
	if err := a.analyzeBlocks(); err != nil {
		return nil, fmt.Errorf("block analysis failed: %w", err)
	}

	// Compute statistics
	a.computeStatistics()

	fmt.Printf("Analysis complete: %d blocks analyzed\n", len(a.diskMap.Blocks))
	return a.diskMap, nil
}

// analyzeBlocks reads and analyzes all blocks
func (a *Analyzer) analyzeBlocks() error {
	buffer := make([]byte, a.options.BlockSize)
	blockIndex := int64(0)

	for {
		offset := blockIndex * int64(a.options.BlockSize)

		// Check if we should sample this block
		if a.options.SampleRate > 1 && blockIndex%int64(a.options.SampleRate) != 0 {
			blockIndex++
			if offset >= a.diskMap.TotalSize {
				break
			}
			continue
		}

		// Check max blocks limit
		if a.options.MaxBlocks > 0 && blockIndex >= a.options.MaxBlocks {
			break
		}

		// Read block
		n, err := a.imageFile.ReadAt(buffer, offset)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read block %d: %w", blockIndex, err)
		}

		if n == 0 {
			break
		}

		// Analyze this block
		block := a.analyzeBlock(blockIndex, offset, buffer[:n])
		a.diskMap.Blocks = append(a.diskMap.Blocks, block)

		// Progress reporting
		if blockIndex%10000 == 0 && blockIndex > 0 {
			progress := float64(blockIndex) / float64(a.diskMap.TotalBlocks) * 100
			fmt.Printf("Progress: %.1f%% (%d/%d blocks)\n", progress, blockIndex, a.diskMap.TotalBlocks)
		}

		blockIndex++

		if offset+int64(n) >= a.diskMap.TotalSize {
			break
		}
	}

	return nil
}

// analyzeBlock analyzes a single block
func (a *Analyzer) analyzeBlock(index, offset int64, data []byte) *Block {
	block := &Block{
		Index:  index,
		Offset: offset,
		Size:   len(data),
		Status: StatusUnallocated, // Default
		Type:   TypeUnknown,
	}

	// Check if zero block
	block.IsZero = isZeroBlock(data)
	if block.IsZero {
		return block
	}

	// Compute entropy if requested
	if a.options.ComputeEntropy {
		block.Entropy = computeEntropy(data)

		// High entropy suggests compression or encryption
		if block.Entropy > 7.5 {
			block.Encrypted = true
			block.Type = TypeEncrypted
		} else if block.Entropy > 6.5 {
			block.Compressed = true
			block.Type = TypeCompressed
		}
	}

	// Detect file signature if requested
	if a.options.IdentifySignatures {
		signature, fileType := detectSignature(data)
		if signature != "" {
			block.Signature = signature
			block.Type = fileType
			block.Status = StatusAllocated
		}
	}

	// Compute hash if requested
	if a.options.ComputeHashes {
		hash := sha256.Sum256(data)
		block.Hash = fmt.Sprintf("%x", hash)
	}

	return block
}

// computeStatistics computes aggregate statistics
func (a *Analyzer) computeStatistics() {
	stats := a.diskMap.Statistics

	for _, block := range a.diskMap.Blocks {
		switch block.Status {
		case StatusAllocated:
			stats.AllocatedBlocks++
			stats.UsedSpace += int64(block.Size)
		case StatusUnallocated:
			stats.UnallocatedBlocks++
			stats.FreeSpace += int64(block.Size)
		case StatusDeleted:
			stats.DeletedBlocks++
		case StatusSystem:
			stats.SystemBlocks++
		case StatusBad:
			stats.BadBlocks++
		}

		if block.IsZero {
			stats.ZeroBlocks++
		}
	}

	if a.diskMap.TotalSize > 0 {
		stats.Utilization = float64(stats.UsedSpace) / float64(a.diskMap.TotalSize) * 100
	}

	stats.FileCount = len(a.diskMap.Files)
	for _, file := range a.diskMap.Files {
		if file.Deleted {
			stats.DeletedFileCount++
		}
		if file.Fragmented {
			stats.FragmentedFiles++
		}
	}
}

// GetDiskMap returns the analyzed disk map
func (a *Analyzer) GetDiskMap() *DiskMap {
	return a.diskMap
}

// GetBlock returns a specific block by index
func (a *Analyzer) GetBlock(index int64) (*Block, error) {
	if index < 0 || index >= int64(len(a.diskMap.Blocks)) {
		return nil, fmt.Errorf("block index out of range: %d", index)
	}
	return a.diskMap.Blocks[index], nil
}

// GetBlockRange returns a range of blocks
func (a *Analyzer) GetBlockRange(start, end int64) ([]*Block, error) {
	if start < 0 || end > int64(len(a.diskMap.Blocks)) || start > end {
		return nil, fmt.Errorf("invalid block range: %d-%d", start, end)
	}
	return a.diskMap.Blocks[start:end], nil
}

// SearchBlocks searches for blocks matching the query
func (a *Analyzer) SearchBlocks(query *SearchQuery) []*Block {
	results := make([]*Block, 0)

	for _, block := range a.diskMap.Blocks {
		if matchesQuery(block, query) {
			results = append(results, block)
		}
	}

	return results
}

// matchesQuery checks if a block matches the search query
func matchesQuery(block *Block, query *SearchQuery) bool {
	if query == nil {
		return true
	}

	if query.FileID != "" && block.FileID != query.FileID {
		return false
	}

	if query.FileName != "" && block.FileName != query.FileName {
		return false
	}

	if query.Type != "" && block.Type != query.Type {
		return false
	}

	if query.Status != "" && block.Status != query.Status {
		return false
	}

	if query.MinEntropy > 0 && block.Entropy < query.MinEntropy {
		return false
	}

	if query.MaxEntropy > 0 && block.Entropy > query.MaxEntropy {
		return false
	}

	if query.IsZero != nil && block.IsZero != *query.IsZero {
		return false
	}

	if query.IsCompressed != nil && block.Compressed != *query.IsCompressed {
		return false
	}

	if query.IsEncrypted != nil && block.Encrypted != *query.IsEncrypted {
		return false
	}

	if query.Signature != "" && block.Signature != query.Signature {
		return false
	}

	if query.Range != nil {
		if block.Index < query.Range.Start || block.Index > query.Range.End {
			return false
		}
	}

	return true
}

// isZeroBlock checks if a block is all zeros
func isZeroBlock(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// computeEntropy calculates Shannon entropy of data
func computeEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	// Count byte frequencies
	frequencies := make(map[byte]int)
	for _, b := range data {
		frequencies[b]++
	}

	// Calculate entropy
	entropy := 0.0
	length := float64(len(data))

	for _, count := range frequencies {
		if count > 0 {
			probability := float64(count) / length
			entropy -= probability * math.Log2(probability)
		}
	}

	return entropy
}

// File signature database (magic bytes)
var signatures = map[string]struct {
	magic  []byte
	offset int
	ftype  BlockType
}{
	"PNG":  {[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0, TypeImage},
	"JPEG": {[]byte{0xFF, 0xD8, 0xFF}, 0, TypeImage},
	"GIF":  {[]byte{0x47, 0x49, 0x46, 0x38}, 0, TypeImage},
	"PDF":  {[]byte{0x25, 0x50, 0x44, 0x46}, 0, TypeDocument},
	"ZIP":  {[]byte{0x50, 0x4B, 0x03, 0x04}, 0, TypeArchive},
	"RAR":  {[]byte{0x52, 0x61, 0x72, 0x21}, 0, TypeArchive},
	"GZIP": {[]byte{0x1F, 0x8B}, 0, TypeArchive},
	"BZ2":  {[]byte{0x42, 0x5A, 0x68}, 0, TypeArchive},
	"EXE":  {[]byte{0x4D, 0x5A}, 0, TypeExecutable},
	"ELF":  {[]byte{0x7F, 0x45, 0x4C, 0x46}, 0, TypeExecutable},
	"MP3":  {[]byte{0xFF, 0xFB}, 0, TypeAudio},
	"OGG":  {[]byte{0x4F, 0x67, 0x67, 0x53}, 0, TypeAudio},
	"AVI":  {[]byte{0x52, 0x49, 0x46, 0x46}, 0, TypeVideo},
	"MP4":  {[]byte{0x66, 0x74, 0x79, 0x70}, 4, TypeVideo},
	"SQLITE": {[]byte{0x53, 0x51, 0x4C, 0x69, 0x74, 0x65}, 0, TypeDatabase},
}

// detectSignature detects file signature in data
func detectSignature(data []byte) (string, BlockType) {
	for name, sig := range signatures {
		if sig.offset+len(sig.magic) > len(data) {
			continue
		}

		match := true
		for i, b := range sig.magic {
			if data[sig.offset+i] != b {
				match = false
				break
			}
		}

		if match {
			return name, sig.ftype
		}
	}

	return "", TypeUnknown
}

// GetBlockSummaries returns lightweight block summaries for efficient transfer
func (a *Analyzer) GetBlockSummaries(start, end int64) ([]*BlockSummary, error) {
	if start < 0 || end > int64(len(a.diskMap.Blocks)) || start > end {
		return nil, fmt.Errorf("invalid range: %d-%d", start, end)
	}

	summaries := make([]*BlockSummary, 0, end-start)
	for i := start; i < end; i++ {
		block := a.diskMap.Blocks[i]
		summary := &BlockSummary{
			Index:   block.Index,
			Status:  block.Status,
			Type:    block.Type,
			FileID:  block.FileID,
			IsZero:  block.IsZero,
			Entropy: block.Entropy,
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// ExportAnalysis exports the analysis to a JSON file
func (a *Analyzer) ExportAnalysis(outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// For now, just create a placeholder
	// In a real implementation, this would serialize the DiskMap to JSON
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write basic info
	fmt.Fprintf(file, "Disk Analysis Report\n")
	fmt.Fprintf(file, "====================\n\n")
	fmt.Fprintf(file, "Image: %s\n", a.diskMap.ImagePath)
	fmt.Fprintf(file, "Total Size: %d bytes\n", a.diskMap.TotalSize)
	fmt.Fprintf(file, "Total Blocks: %d\n", a.diskMap.TotalBlocks)
	fmt.Fprintf(file, "Block Size: %d bytes\n", a.diskMap.BlockSize)
	fmt.Fprintf(file, "\nStatistics:\n")
	fmt.Fprintf(file, "  Allocated Blocks: %d\n", a.diskMap.Statistics.AllocatedBlocks)
	fmt.Fprintf(file, "  Unallocated Blocks: %d\n", a.diskMap.Statistics.UnallocatedBlocks)
	fmt.Fprintf(file, "  Zero Blocks: %d\n", a.diskMap.Statistics.ZeroBlocks)
	fmt.Fprintf(file, "  Utilization: %.2f%%\n", a.diskMap.Statistics.Utilization)

	return nil
}
