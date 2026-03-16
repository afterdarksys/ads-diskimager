package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
)

// OpenDestination returns an io.WriteCloser for the given target path.
// It supports local files and cloud URLs (s3://, gs://, azblob://, minio://).
//
// Supported formats:
//   - Local file: /path/to/file.img
//   - AWS S3: s3://bucket/path/to/file.img
//   - S3 with custom endpoint: s3://bucket/path/to/file.img?endpoint=https://s3.example.com
//   - MinIO (shorthand): minio://endpoint/bucket/path/to/file.img
//   - Google Cloud Storage: gs://bucket/path/to/file.img
//   - Azure Blob: azblob://container/path/to/file.img
//
// Environment variables for authentication:
//   - AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY (for S3/MinIO)
//   - MINIO_ACCESS_KEY, MINIO_SECRET_KEY (alternative for MinIO)
//   - GOOGLE_APPLICATION_CREDENTIALS (for GCS)
//   - AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_KEY (for Azure)
func OpenDestination(target string, appendMode bool) (io.WriteCloser, error) {
	// Simple heuristic: if it contains "://", it's likely a URL scheme.
	// Otherwise, treat as a local file.
	if !strings.Contains(target, "://") {
		// Local file
		flags := os.O_CREATE | os.O_WRONLY
		if appendMode {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		return os.OpenFile(target, flags, 0644)
	}

	// Cloud storage URL detected
	if appendMode {
		return nil, fmt.Errorf("append/resume mode is not supported for cloud storage targets")
	}

	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid storage URL: %w", err)
	}

	// Handle MinIO shorthand: minio://endpoint/bucket/path/to/file.img
	// Convert to s3://bucket/path?endpoint=https://endpoint
	if u.Scheme == "minio" {
		endpoint := u.Host
		pathParts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
		if len(pathParts) < 2 {
			return nil, fmt.Errorf("MinIO URL format: minio://endpoint/bucket/path/to/file.img")
		}
		bucket := pathParts[0]
		objectPath := pathParts[1]

		// Reconstruct as S3 URL with endpoint parameter
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}
		return openMinIOBucket(endpoint, bucket, objectPath)
	}

	// Check for custom endpoint in query parameters (for S3-compatible services)
	queryParams := u.Query()
	customEndpoint := queryParams.Get("endpoint")

	if u.Scheme == "s3" && customEndpoint != "" {
		// S3 with custom endpoint (like MinIO or other S3-compatible services)
		bucket := u.Host
		objectKey := strings.TrimPrefix(u.Path, "/")

		if objectKey == "" {
			return nil, fmt.Errorf("no object key (file path) specified in URL")
		}

		return openMinIOBucket(customEndpoint, bucket, objectKey)
	}

	// Standard cloud storage (S3, GCS, Azure)
	// gocloud.dev/blob expects URLs like "s3://bucket", we need to extract the bucket URL and the object key.
	// target: s3://my-bucket/path/to/image.dd
	// bucket URL: s3://my-bucket
	// object key: path/to/image.dd

	bucketURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	objectKey := strings.TrimPrefix(u.Path, "/")

	if objectKey == "" {
		return nil, fmt.Errorf("no object key (file path) specified in URL")
	}

	ctx := context.Background()
	b, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open bucket %s: %w", bucketURL, err)
	}

	// To keep this io.WriteCloser compatible, we wrap the blob.Writer so that
	// calling Close() closes the writer and then the bucket.
	writer, err := b.NewWriter(ctx, objectKey, nil)
	if err != nil {
		b.Close()
		return nil, fmt.Errorf("failed to create object writer for %s: %w", objectKey, err)
	}

	return &cloudWriter{
		writer: writer,
		bucket: b,
	}, nil
}

// openMinIOBucket opens a MinIO or S3-compatible bucket with a custom endpoint.
func openMinIOBucket(endpoint, bucket, objectKey string) (io.WriteCloser, error) {
	ctx := context.Background()

	// Get credentials from environment variables
	// Try MinIO-specific env vars first, fall back to AWS env vars
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")

	if accessKey == "" {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if secretKey == "" {
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("MinIO/S3 credentials not found. Set MINIO_ACCESS_KEY/MINIO_SECRET_KEY or AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY")
	}

	// Create custom endpoint resolver for MinIO
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     "us-east-1",
			HostnameImmutable: true,
		}, nil
	})

	// Load AWS config with custom endpoint and credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO config: %w", err)
	}

	// Create S3 client with path-style addressing (required for MinIO)
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Open bucket using gocloud s3blob with custom client
	b, err := s3blob.OpenBucket(ctx, s3Client, bucket, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open MinIO bucket %s at %s: %w", bucket, endpoint, err)
	}

	// Create writer
	writer, err := b.NewWriter(ctx, objectKey, nil)
	if err != nil {
		b.Close()
		return nil, fmt.Errorf("failed to create object writer for %s: %w", objectKey, err)
	}

	return &cloudWriter{
		writer: writer,
		bucket: b,
	}, nil
}

type cloudWriter struct {
	writer *blob.Writer
	bucket *blob.Bucket
}

func (cw *cloudWriter) Write(p []byte) (n int, err error) {
	return cw.writer.Write(p)
}

func (cw *cloudWriter) Close() error {
	wErr := cw.writer.Close()
	bErr := cw.bucket.Close()
	if wErr != nil {
		return wErr
	}
	return bErr
}
