//go:build !windows
// +build !windows

package rclonelib

import (
	"fmt"
	"syscall"
)

// getAvailableDiskSpace returns available disk space in bytes for Unix-like systems
func getAvailableDiskSpace(path string) (int64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	// Available blocks * block size
	available := int64(stat.Bavail) * int64(stat.Bsize)
	return available, nil
}
