package api

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/afterdarksys/diskimager/imager"
	"github.com/afterdarksys/diskimager/pkg/compression"
	"github.com/afterdarksys/diskimager/pkg/format/e01"
	"github.com/afterdarksys/diskimager/pkg/format/raw"
	"github.com/afterdarksys/diskimager/pkg/hash"
	"github.com/afterdarksys/diskimager/pkg/sparse"
	"github.com/afterdarksys/diskimager/pkg/storage"
	"github.com/afterdarksys/diskimager/pkg/throttle"
)

// imagingEngine handles the actual imaging operations
type imagingEngine struct {
	job *Job
	ctx context.Context
}

// newImagingEngine creates a new imaging engine for a job
func newImagingEngine(ctx context.Context, job *Job) *imagingEngine {
	return &imagingEngine{
		job: job,
		ctx: ctx,
	}
}

// performActualImaging executes the real imaging operation
func (ie *imagingEngine) performActualImaging() (*JobResult, error) {
	startTime := time.Now()

	// Phase 1: Initialize source
	ie.job.updateProgress(&JobProgress{
		Phase:      PhaseInitializing,
		Percentage: 0,
	})

	source, sourceSize, err := ie.openSource()
	if err != nil {
		return nil, fmt.Errorf("failed to open source: %w", err)
	}
	defer source.Close()

	// Phase 2: Initialize destination
	destination, err := ie.openDestination()
	if err != nil {
		return nil, fmt.Errorf("failed to open destination: %w", err)
	}
	defer destination.Close()

	// Phase 3: Setup writer stack (format -> sparse -> compression -> throttle)
	writer, writerClosers, err := ie.buildWriterStack(destination)
	if err != nil {
		return nil, fmt.Errorf("failed to build writer stack: %w", err)
	}
	defer func() {
		for i := len(writerClosers) - 1; i >= 0; i-- {
			writerClosers[i].Close()
		}
	}()

	// Phase 4: Setup hasher
	var multiHasher *hash.MultiHasher

	if len(ie.job.Request.Options.HashAlgorithms) > 0 {
		multiHasher = hash.NewMultiHasher(ie.job.Request.Options.HashAlgorithms...)
	}

	// Phase 5: Setup progress tracking
	progressReader := &progressTrackingReader{
		reader:     source,
		totalBytes: sourceSize,
		job:        ie.job,
	}

	// Phase 6: Execute imaging
	ie.job.updateProgress(&JobProgress{
		Phase:          PhaseReading,
		BytesProcessed: 0,
		TotalBytes:     sourceSize,
		Percentage:     0,
	})

	// Create imager config
	config := imager.Config{
		Source:         progressReader,
		Destination:    writer,
		BlockSize:      ie.job.Request.Options.BlockSize,
		HashAlgo:       "sha256", // Default
		Metadata:       ie.job.Request.Metadata.ToImagerMetadata(),
	}

	if multiHasher != nil {
		config.MultiHasher = multiHasher
		config.HashAlgorithms = ie.job.Request.Options.HashAlgorithms
	}

	// Run the imager
	result, err := imager.Image(config)
	if err != nil {
		return nil, fmt.Errorf("imaging failed: %w", err)
	}

	// Phase 7: Extract hashes if using MultiHasher
	hashes := result.Hashes
	if multiHasher != nil {
		hashResult := multiHasher.Sum()
		hashes = make(map[string]string)
		if hashResult.MD5 != "" {
			hashes["md5"] = hashResult.MD5
		}
		if hashResult.SHA1 != "" {
			hashes["sha1"] = hashResult.SHA1
		}
		if hashResult.SHA256 != "" {
			hashes["sha256"] = hashResult.SHA256
		}
	}

	// Phase 8: Finalize
	ie.job.updateProgress(&JobProgress{
		Phase:          PhaseCompleted,
		BytesProcessed: result.BytesCopied,
		TotalBytes:     sourceSize,
		Percentage:     100,
	})

	duration := int(time.Since(startTime).Seconds())

	return &JobResult{
		BytesCopied: result.BytesCopied,
		Hashes:      hashes,
		BadSectors:  result.BadSectors,
		Duration:    duration,
	}, nil
}

// openSource opens the source based on source type
func (ie *imagingEngine) openSource() (io.ReadCloser, int64, error) {
	src := ie.job.Request.Source

	switch src.Type {
	case SourceTypeDisk:
		// Open physical disk device
		file, err := os.Open(src.Device)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to open disk %s: %w", src.Device, err)
		}

		// Get disk size
		stat, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, 0, fmt.Errorf("failed to stat disk: %w", err)
		}

		size := stat.Size()
		if size == 0 {
			// For block devices, size might be 0, try to seek to end
			size, err = file.Seek(0, io.SeekEnd)
			if err != nil {
				file.Close()
				return nil, 0, fmt.Errorf("failed to get disk size: %w", err)
			}
			_, err = file.Seek(0, io.SeekStart)
			if err != nil {
				file.Close()
				return nil, 0, fmt.Errorf("failed to seek to start: %w", err)
			}
		}

		return file, size, nil

	case SourceTypeFile:
		// Open file
		file, err := os.Open(src.Path)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to open file %s: %w", src.Path, err)
		}

		stat, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, 0, fmt.Errorf("failed to stat file: %w", err)
		}

		return file, stat.Size(), nil

	case SourceTypeS3, SourceTypeAzure, SourceTypeGCS:
		// Use cloud storage
		reader, size, err := storage.OpenCloudSource(src.Type, src.Bucket, src.Key, src.Credentials)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to open cloud source: %w", err)
		}
		return reader, size, nil

	default:
		return nil, 0, fmt.Errorf("unsupported source type: %s", src.Type)
	}
}

// openDestination opens the destination based on destination type
func (ie *imagingEngine) openDestination() (io.WriteCloser, error) {
	dst := ie.job.Request.Destination

	switch dst.Type {
	case DestinationTypeFile:
		// Create file
		file, err := os.Create(dst.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to create file %s: %w", dst.Path, err)
		}
		return file, nil

	case DestinationTypeS3, DestinationTypeAzure, DestinationTypeGCS:
		// Use cloud storage
		writer, err := storage.OpenCloudDestination(dst.Type, dst.Bucket, dst.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to open cloud destination: %w", err)
		}
		return writer, nil

	default:
		return nil, fmt.Errorf("unsupported destination type: %s", dst.Type)
	}
}

// buildWriterStack builds the writer stack with format, compression, sparse, and throttling
func (ie *imagingEngine) buildWriterStack(base io.WriteCloser) (io.Writer, []io.Closer, error) {
	var closers []io.Closer
	var currentWriter io.Writer = base
	closers = append(closers, base)

	// Layer 1: Format writer (RAW or E01)
	format := ie.job.Request.Destination.Format
	if format == "" {
		format = "raw"
	}

	switch format {
	case "raw":
		rawWriter, err := raw.NewWriter(base)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create raw writer: %w", err)
		}
		currentWriter = rawWriter
		closers = append(closers, rawWriter)

	case "e01":
		metadata := ie.job.Request.Metadata.ToImagerMetadata()
		e01Writer, err := e01.NewWriter(base, false, metadata)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create e01 writer: %w", err)
		}
		currentWriter = e01Writer
		closers = append(closers, e01Writer)

	default:
		return nil, nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Layer 2: Sparse file support
	if ie.job.Request.Options.DetectSparse {
		blockSize := ie.job.Request.Options.BlockSize
		if blockSize == 0 {
			blockSize = 65536
		}
		sparseWriter := sparse.NewWriter(currentWriter, blockSize, true)
		currentWriter = sparseWriter
		closers = append(closers, sparseWriter)
	}

	// Layer 3: Compression
	if ie.job.Request.Options.Compression != "" && ie.job.Request.Options.Compression != "none" {
		var algo compression.Algorithm
		switch ie.job.Request.Options.Compression {
		case "gzip":
			algo = compression.AlgorithmGzip
		case "zstd":
			algo = compression.AlgorithmZstd
		default:
			return nil, nil, fmt.Errorf("unsupported compression: %s", ie.job.Request.Options.Compression)
		}

		level := ie.job.Request.Options.CompressionLevel
		if level == 0 {
			level = 5
		}

		compWriter, err := compression.NewWriter(currentWriter, algo, compression.Level(level))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create compression writer: %w", err)
		}
		currentWriter = compWriter
		closers = append(closers, compWriter)
	}

	// Layer 4: Bandwidth throttling
	if ie.job.Request.Options.RateLimit > 0 {
		throttleWriter := throttle.NewWriter(currentWriter, ie.job.Request.Options.RateLimit)
		currentWriter = throttleWriter
		closers = append(closers, throttleWriter)
	}

	return currentWriter, closers, nil
}

// progressTrackingReader wraps a reader to track progress
type progressTrackingReader struct {
	reader       io.Reader
	totalBytes   int64
	bytesRead    int64
	job          *Job
	lastUpdate   time.Time
	startTime    time.Time
}

func (ptr *progressTrackingReader) Read(p []byte) (int, error) {
	if ptr.startTime.IsZero() {
		ptr.startTime = time.Now()
		ptr.lastUpdate = ptr.startTime
	}

	n, err := ptr.reader.Read(p)
	ptr.bytesRead += int64(n)

	// Update progress every 500ms
	if time.Since(ptr.lastUpdate) >= 500*time.Millisecond {
		elapsed := time.Since(ptr.startTime).Seconds()
		speed := int64(float64(ptr.bytesRead) / elapsed)

		percentage := 0.0
		eta := 0
		if ptr.totalBytes > 0 {
			percentage = float64(ptr.bytesRead) / float64(ptr.totalBytes) * 100
			if speed > 0 {
				remaining := ptr.totalBytes - ptr.bytesRead
				eta = int(float64(remaining) / float64(speed))
			}
		}

		ptr.job.updateProgress(&JobProgress{
			Phase:          PhaseReading,
			BytesProcessed: ptr.bytesRead,
			TotalBytes:     ptr.totalBytes,
			Percentage:     percentage,
			Speed:          speed,
			ETA:            eta,
		})

		ptr.lastUpdate = time.Now()
	}

	return n, err
}

func (ptr *progressTrackingReader) Close() error {
	if closer, ok := ptr.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
