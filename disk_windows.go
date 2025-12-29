//go:build windows
// +build windows

package rclonelib

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

// getAvailableDiskSpace returns available disk space in bytes for Windows systems
func getAvailableDiskSpace(path string) (int64, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return 0, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get the root path (drive letter)
	root := filepath.VolumeName(absPath)
	if root == "" {
		root = absPath
	}
	root += "\\"

	// Convert to UTF16
	rootPtr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0, fmt.Errorf("failed to convert path: %w", err)
	}

	// Load kernel32.dll and get GetDiskFreeSpaceExW
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64

	// Call GetDiskFreeSpaceExW
	r1, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if r1 == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceExW failed: %w", err)
	}

	return int64(freeBytesAvailable), nil
}
