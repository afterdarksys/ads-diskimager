package storage

import (
	"fmt"
	"io"
	"os"
)

// OpenCloudSource opens a cloud storage source for reading
// Returns a ReadCloser and the total size
func OpenCloudSource(sourceType, bucket, key string, credentials map[string]interface{}) (io.ReadCloser, int64, error) {
	// TODO: Implement cloud source reading
	// For now, this is a placeholder
	return nil, 0, fmt.Errorf("cloud source reading not yet implemented for type: %s", sourceType)
}

// OpenCloudDestination opens a cloud storage destination for writing
func OpenCloudDestination(destType, bucket, key string) (io.WriteCloser, error) {
	// Build URL based on destination type
	var url string
	switch destType {
	case "s3":
		url = fmt.Sprintf("s3://%s/%s", bucket, key)
	case "azure-blob":
		url = fmt.Sprintf("azblob://%s/%s", bucket, key)
	case "gcs":
		url = fmt.Sprintf("gs://%s/%s", bucket, key)
	default:
		return nil, fmt.Errorf("unsupported destination type: %s", destType)
	}

	// Use existing OpenDestination function
	return OpenDestination(url, false)
}

// GetLocalFileSize returns the size of a local file
func GetLocalFileSize(path string) (int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}
	return stat.Size(), nil
}
