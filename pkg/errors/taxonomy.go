package errors

import (
	"fmt"
	"time"
)

// ErrorCategory represents the type of error
type ErrorCategory int

const (
	// ErrHardware indicates physical disk or hardware issues
	ErrHardware ErrorCategory = iota
	// ErrFormat indicates format parsing or structure issues
	ErrFormat
	// ErrStorage indicates issues with destination storage
	ErrStorage
	// ErrAuthentication indicates auth/permission issues
	ErrAuthentication
	// ErrValidation indicates data validation failures
	ErrValidation
	// ErrCancellation indicates user-initiated cancellation
	ErrCancellation
	// ErrNetwork indicates network/connectivity issues
	ErrNetwork
	// ErrFilesystem indicates filesystem-related errors
	ErrFilesystem
	// ErrConfiguration indicates configuration problems
	ErrConfiguration
	// ErrResource indicates resource exhaustion (memory, disk, etc)
	ErrResource
)

// String returns the string representation of error category
func (c ErrorCategory) String() string {
	switch c {
	case ErrHardware:
		return "Hardware"
	case ErrFormat:
		return "Format"
	case ErrStorage:
		return "Storage"
	case ErrAuthentication:
		return "Authentication"
	case ErrValidation:
		return "Validation"
	case ErrCancellation:
		return "Cancellation"
	case ErrNetwork:
		return "Network"
	case ErrFilesystem:
		return "Filesystem"
	case ErrConfiguration:
		return "Configuration"
	case ErrResource:
		return "Resource"
	default:
		return "Unknown"
	}
}

// ForensicError represents a structured error with forensic context
type ForensicError struct {
	Category    ErrorCategory          `json:"category"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Offset      int64                  `json:"offset,omitempty"`
	Recoverable bool                   `json:"recoverable"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Cause       error                  `json:"-"` // Underlying error
}

// Error implements the error interface
func (e *ForensicError) Error() string {
	if e.Offset >= 0 {
		return fmt.Sprintf("[%s:%s] %s (offset: %d)", e.Category, e.Code, e.Message, e.Offset)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Category, e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *ForensicError) Unwrap() error {
	return e.Cause
}

// IsRecoverable returns whether the error is recoverable
func (e *ForensicError) IsRecoverable() bool {
	return e.Recoverable
}

// WithContext adds context information to the error
func (e *ForensicError) WithContext(key string, value interface{}) *ForensicError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithOffset sets the offset where the error occurred
func (e *ForensicError) WithOffset(offset int64) *ForensicError {
	e.Offset = offset
	return e
}

// NewForensicError creates a new forensic error
func NewForensicError(category ErrorCategory, code, message string, recoverable bool) *ForensicError {
	return &ForensicError{
		Category:    category,
		Code:        code,
		Message:     message,
		Recoverable: recoverable,
		Timestamp:   time.Now(),
		Offset:      -1,
	}
}

// WrapForensicError wraps an existing error as a forensic error
func WrapForensicError(category ErrorCategory, code string, cause error, recoverable bool) *ForensicError {
	return &ForensicError{
		Category:    category,
		Code:        code,
		Message:     cause.Error(),
		Recoverable: recoverable,
		Timestamp:   time.Now(),
		Cause:       cause,
		Offset:      -1,
	}
}

// Common error codes and constructors

// Hardware Errors
func NewBadSectorError(offset int64, cause error) *ForensicError {
	return WrapForensicError(ErrHardware, "BAD_SECTOR", cause, true).
		WithOffset(offset).
		WithContext("sector_size", 512)
}

func NewDeviceNotReadyError(device string) *ForensicError {
	return NewForensicError(ErrHardware, "DEVICE_NOT_READY",
		fmt.Sprintf("device %s is not ready", device), false).
		WithContext("device", device)
}

func NewDeviceIOError(offset int64, cause error) *ForensicError {
	return WrapForensicError(ErrHardware, "IO_ERROR", cause, true).
		WithOffset(offset)
}

// Format Errors
func NewInvalidFormatError(format string, cause error) *ForensicError {
	return WrapForensicError(ErrFormat, "INVALID_FORMAT", cause, false).
		WithContext("format", format)
}

func NewCorruptedHeaderError(format string, offset int64) *ForensicError {
	return NewForensicError(ErrFormat, "CORRUPTED_HEADER",
		fmt.Sprintf("corrupted %s header", format), false).
		WithContext("format", format).
		WithOffset(offset)
}

func NewChecksumMismatchError(expected, actual string, offset int64) *ForensicError {
	return NewForensicError(ErrFormat, "CHECKSUM_MISMATCH",
		fmt.Sprintf("checksum mismatch: expected %s, got %s", expected, actual), false).
		WithContext("expected", expected).
		WithContext("actual", actual).
		WithOffset(offset)
}

// Storage Errors
func NewStorageFullError(path string) *ForensicError {
	return NewForensicError(ErrStorage, "STORAGE_FULL",
		fmt.Sprintf("storage full at %s", path), false).
		WithContext("path", path)
}

func NewStorageWriteError(path string, cause error) *ForensicError {
	return WrapForensicError(ErrStorage, "WRITE_ERROR", cause, true).
		WithContext("path", path)
}

func NewStoragePermissionError(path string, cause error) *ForensicError {
	return WrapForensicError(ErrStorage, "PERMISSION_DENIED", cause, false).
		WithContext("path", path)
}

// Authentication Errors
func NewAuthenticationFailedError(service string) *ForensicError {
	return NewForensicError(ErrAuthentication, "AUTH_FAILED",
		fmt.Sprintf("authentication failed for %s", service), false).
		WithContext("service", service)
}

func NewInvalidCredentialsError() *ForensicError {
	return NewForensicError(ErrAuthentication, "INVALID_CREDENTIALS",
		"invalid credentials provided", false)
}

func NewCertificateError(cause error) *ForensicError {
	return WrapForensicError(ErrAuthentication, "CERT_ERROR", cause, false)
}

// Network Errors
func NewNetworkTimeoutError(endpoint string) *ForensicError {
	return NewForensicError(ErrNetwork, "TIMEOUT",
		fmt.Sprintf("connection timeout to %s", endpoint), true).
		WithContext("endpoint", endpoint)
}

func NewNetworkConnectionError(endpoint string, cause error) *ForensicError {
	return WrapForensicError(ErrNetwork, "CONNECTION_ERROR", cause, true).
		WithContext("endpoint", endpoint)
}

// Filesystem Errors
func NewFilesystemNotSupportedError(fsType string) *ForensicError {
	return NewForensicError(ErrFilesystem, "FS_NOT_SUPPORTED",
		fmt.Sprintf("filesystem %s not supported", fsType), false).
		WithContext("fs_type", fsType)
}

func NewMountFailedError(path string, cause error) *ForensicError {
	return WrapForensicError(ErrFilesystem, "MOUNT_FAILED", cause, false).
		WithContext("path", path)
}

func NewFileNotFoundError(path string) *ForensicError {
	return NewForensicError(ErrFilesystem, "FILE_NOT_FOUND",
		fmt.Sprintf("file not found: %s", path), false).
		WithContext("path", path)
}

// Validation Errors
func NewHashMismatchError(expected, actual string) *ForensicError {
	return NewForensicError(ErrValidation, "HASH_MISMATCH",
		fmt.Sprintf("hash mismatch: expected %s, got %s", expected, actual), false).
		WithContext("expected", expected).
		WithContext("actual", actual)
}

func NewSizeMismatchError(expected, actual int64) *ForensicError {
	return NewForensicError(ErrValidation, "SIZE_MISMATCH",
		fmt.Sprintf("size mismatch: expected %d, got %d", expected, actual), false).
		WithContext("expected", expected).
		WithContext("actual", actual)
}

// Resource Errors
func NewOutOfMemoryError(requested int64) *ForensicError {
	return NewForensicError(ErrResource, "OUT_OF_MEMORY",
		fmt.Sprintf("out of memory (requested %d bytes)", requested), true).
		WithContext("requested_bytes", requested)
}

func NewTooManyOpenFilesError() *ForensicError {
	return NewForensicError(ErrResource, "TOO_MANY_FILES",
		"too many open files", true)
}

// Cancellation Errors
func NewUserCancelledError() *ForensicError {
	return NewForensicError(ErrCancellation, "USER_CANCELLED",
		"operation cancelled by user", false)
}

func NewTimeoutError(operation string, duration time.Duration) *ForensicError {
	return NewForensicError(ErrCancellation, "TIMEOUT",
		fmt.Sprintf("operation %s timed out after %v", operation, duration), false).
		WithContext("operation", operation).
		WithContext("duration", duration.String())
}

// Configuration Errors
func NewInvalidConfigError(field, reason string) *ForensicError {
	return NewForensicError(ErrConfiguration, "INVALID_CONFIG",
		fmt.Sprintf("invalid configuration for %s: %s", field, reason), false).
		WithContext("field", field).
		WithContext("reason", reason)
}

func NewMissingConfigError(field string) *ForensicError {
	return NewForensicError(ErrConfiguration, "MISSING_CONFIG",
		fmt.Sprintf("required configuration missing: %s", field), false).
		WithContext("field", field)
}
