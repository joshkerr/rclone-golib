package rclonelib

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeTimeout represents timeout errors
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeAuth represents authentication errors
	ErrorTypeAuth ErrorType = "auth"
	// ErrorTypeNotFound represents file/directory not found errors
	ErrorTypeNotFound ErrorType = "not_found"
	// ErrorTypeFileSystem represents filesystem errors
	ErrorTypeFileSystem ErrorType = "filesystem"
	// ErrorTypeInvalidInput represents invalid input errors
	ErrorTypeInvalidInput ErrorType = "invalid_input"
	// ErrorTypeInsufficientSpace represents insufficient disk space errors
	ErrorTypeInsufficientSpace ErrorType = "insufficient_space"
	// ErrorTypeUnknown represents unknown errors
	ErrorTypeUnknown ErrorType = "unknown"
)

// ClassifiedError wraps an error with classification information
type ClassifiedError struct {
	Type      ErrorType
	Err       error
	Retryable bool
	Temporary bool
}

func (e *ClassifiedError) Error() string {
	return fmt.Sprintf("[%s] %v", e.Type, e.Err)
}

func (e *ClassifiedError) Unwrap() error {
	return e.Err
}

// ClassifyError attempts to classify an error
func ClassifyError(err error) *ClassifiedError {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())

	// Check for validation errors first
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return &ClassifiedError{
			Type:      ErrorTypeInvalidInput,
			Err:       err,
			Retryable: false,
			Temporary: false,
		}
	}

	// Network errors
	if strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "dial") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "host is down") {
		return &ClassifiedError{
			Type:      ErrorTypeNetwork,
			Err:       err,
			Retryable: true,
			Temporary: true,
		}
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "i/o timeout") {
		return &ClassifiedError{
			Type:      ErrorTypeTimeout,
			Err:       err,
			Retryable: true,
			Temporary: true,
		}
	}

	// Authentication errors
	if strings.Contains(errStr, "auth") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "access denied") {
		return &ClassifiedError{
			Type:      ErrorTypeAuth,
			Err:       err,
			Retryable: false,
			Temporary: false,
		}
	}

	// Not found errors
	if strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "no such file") ||
		strings.Contains(errStr, "does not exist") ||
		strings.Contains(errStr, "404") {
		return &ClassifiedError{
			Type:      ErrorTypeNotFound,
			Err:       err,
			Retryable: false,
			Temporary: false,
		}
	}

	// Disk space errors
	if strings.Contains(errStr, "no space left") ||
		strings.Contains(errStr, "insufficient space") ||
		strings.Contains(errStr, "disk full") ||
		strings.Contains(errStr, "quota exceeded") {
		return &ClassifiedError{
			Type:      ErrorTypeInsufficientSpace,
			Err:       err,
			Retryable: false,
			Temporary: false,
		}
	}

	// Filesystem errors
	if strings.Contains(errStr, "filesystem") ||
		strings.Contains(errStr, "i/o error") ||
		strings.Contains(errStr, "read-only") {
		return &ClassifiedError{
			Type:      ErrorTypeFileSystem,
			Err:       err,
			Retryable: false,
			Temporary: false,
		}
	}

	// Unknown error
	return &ClassifiedError{
		Type:      ErrorTypeUnknown,
		Err:       err,
		Retryable: true, // Conservative: allow retry for unknown errors
		Temporary: false,
	}
}

// IsRetryable returns true if the error should be retried
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var classified *ClassifiedError
	if errors.As(err, &classified) {
		return classified.Retryable
	}

	// Classify and check
	classified = ClassifyError(err)
	return classified.Retryable
}

// IsTemporary returns true if the error is temporary
func IsTemporary(err error) bool {
	if err == nil {
		return false
	}

	var classified *ClassifiedError
	if errors.As(err, &classified) {
		return classified.Temporary
	}

	// Classify and check
	classified = ClassifyError(err)
	return classified.Temporary
}

// GetErrorType returns the error type
func GetErrorType(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	var classified *ClassifiedError
	if errors.As(err, &classified) {
		return classified.Type
	}

	classified = ClassifyError(err)
	return classified.Type
}
