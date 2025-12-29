package rclonelib

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateRcloneInstalled checks if rclone is installed and accessible
func ValidateRcloneInstalled() error {
	_, err := exec.LookPath("rclone")
	if err != nil {
		return fmt.Errorf("rclone not found in PATH: %w", err)
	}
	return nil
}

// ValidateRcloneVersion checks if rclone version meets minimum requirements
func ValidateRcloneVersion(minVersion string) error {
	cmd := exec.Command("rclone", "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get rclone version: %w", err)
	}

	version := string(output)
	if !strings.Contains(version, "rclone") {
		return fmt.Errorf("unexpected rclone version output: %s", version)
	}

	// Basic version check (can be enhanced with proper semver comparison)
	if minVersion != "" && !strings.Contains(version, minVersion) {
		return fmt.Errorf("rclone version check failed: want %s, got %s", minVersion, version)
	}

	return nil
}

// ValidateSourcePath checks if source path exists
func ValidateSourcePath(path string) error {
	if path == "" {
		return &ValidationError{Field: "source", Message: "source path cannot be empty"}
	}

	// Skip validation for remote paths (contain :)
	if strings.Contains(path, ":") {
		return nil
	}

	// Check local path exists
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ValidationError{Field: "source", Message: fmt.Sprintf("path does not exist: %s", path)}
		}
		return &ValidationError{Field: "source", Message: fmt.Sprintf("cannot access path: %v", err)}
	}

	return nil
}

// ValidateDestinationPath checks if destination path is accessible
func ValidateDestinationPath(path string) error {
	if path == "" {
		return &ValidationError{Field: "destination", Message: "destination path cannot be empty"}
	}

	// For remote paths, we can't easily validate without rclone
	if strings.Contains(path, ":") {
		return nil
	}

	// For local paths, ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		_, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return &ValidationError{Field: "destination", Message: fmt.Sprintf("parent directory does not exist: %s", dir)}
			}
			return &ValidationError{Field: "destination", Message: fmt.Sprintf("cannot access parent directory: %v", err)}
		}
	}

	return nil
}

// ValidateRemote checks if an rclone remote is accessible
func ValidateRemote(ctx context.Context, remoteName string, timeout time.Duration) error {
	if remoteName == "" {
		return &ValidationError{Field: "remote", Message: "remote name cannot be empty"}
	}

	// Strip trailing colon if present
	remoteName = strings.TrimSuffix(remoteName, ":")

	// Create context with timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to list the remote to verify it's accessible
	cmd := exec.CommandContext(timeoutCtx, "rclone", "lsf", remoteName+":", "--max-depth", "1")
	if err := cmd.Run(); err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return &ValidationError{Field: "remote", Message: fmt.Sprintf("timeout validating remote: %s", remoteName)}
		}
		return &ValidationError{Field: "remote", Message: fmt.Sprintf("remote not accessible: %s (%v)", remoteName, err)}
	}

	return nil
}

// CheckDiskSpace checks if there's enough disk space for a transfer
func CheckDiskSpace(path string, requiredBytes int64) error {
	// For remote paths, skip disk check
	if strings.Contains(path, ":") {
		return nil
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path exists, use parent directory if not
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			absPath = filepath.Dir(absPath)
		} else {
			return fmt.Errorf("failed to stat path: %w", err)
		}
	} else if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	available, err := getAvailableDiskSpace(absPath)
	if err != nil {
		return fmt.Errorf("failed to get disk space: %w", err)
	}

	if available < requiredBytes {
		return fmt.Errorf("insufficient disk space: need %s, have %s",
			FormattedBytes(requiredBytes),
			FormattedBytes(available))
	}

	return nil
}

// HasPartialFiles checks if a directory contains partial/incomplete downloads
func HasPartialFiles(dir string) (bool, error) {
	if dir == "" {
		return false, nil
	}

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	// Look for partial file extensions
	partialExts := []string{".partial", ".rclonepart", ".tmp", ".crdownload", ".part"}

	var hasPartial bool
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		for _, ext := range partialExts {
			if strings.HasSuffix(strings.ToLower(info.Name()), ext) {
				hasPartial = true
				return filepath.SkipAll // Found one, stop walking
			}
		}

		return nil
	})

	return hasPartial, err
}

// GetFileSize returns the size of a local or remote file
func GetFileSize(ctx context.Context, path string) (int64, error) {
	// For local files
	if !strings.Contains(path, ":") {
		info, err := os.Stat(path)
		if err != nil {
			return 0, err
		}
		return info.Size(), nil
	}

	// For remote files, use rclone size command
	cmd := exec.CommandContext(ctx, "rclone", "size", path, "--json")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get remote file size: %w", err)
	}

	// Parse JSON output (basic parsing, can be enhanced)
	// Output format: {"count":1,"bytes":12345}
	sizeStr := string(output)
	if strings.Contains(sizeStr, `"bytes":`) {
		parts := strings.Split(sizeStr, `"bytes":`)
		if len(parts) > 1 {
			bytesStr := strings.TrimSpace(parts[1])
			bytesStr = strings.TrimPrefix(bytesStr, "")
			bytesStr = strings.TrimSuffix(bytesStr, "}")
			var size int64
			_, err := fmt.Sscanf(bytesStr, "%d", &size)
			if err == nil {
				return size, nil
			}
		}
	}

	return 0, fmt.Errorf("failed to parse size from rclone output")
}
