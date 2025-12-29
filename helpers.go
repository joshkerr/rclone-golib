package rclonelib

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ListFiles lists files in a remote or local path
func ListFiles(ctx context.Context, path string, recursive bool) ([]string, error) {
	args := []string{"lsf", path}
	if !recursive {
		args = append(args, "--max-depth", "1")
	}

	cmd := exec.CommandContext(ctx, "rclone", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, strings.TrimSpace(line))
		}
	}

	return files, nil
}

// ListRemotes lists all configured rclone remotes
func ListRemotes(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "rclone", "listremotes")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var remotes []string
	for _, line := range lines {
		if line != "" {
			// Remove trailing colon
			remote := strings.TrimSpace(line)
			remote = strings.TrimSuffix(remote, ":")
			remotes = append(remotes, remote)
		}
	}

	return remotes, nil
}

// CommandExists checks if a command exists in PATH
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GetRcloneVersion returns the rclone version string
func GetRcloneVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "rclone", "version", "--check=false")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get rclone version: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", fmt.Errorf("no version output from rclone")
}

// CheckDuplicates checks if files already exist at the destination
func CheckDuplicates(ctx context.Context, destination string, filenames []string) (map[string]bool, error) {
	if len(filenames) == 0 {
		return make(map[string]bool), nil
	}

	// List files at destination
	existingFiles, err := ListFiles(ctx, destination, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list destination: %w", err)
	}

	// Build map of existing files
	existing := make(map[string]bool)
	for _, file := range existingFiles {
		existing[file] = true
	}

	// Check which files already exist
	duplicates := make(map[string]bool)
	for _, filename := range filenames {
		if existing[filename] {
			duplicates[filename] = true
		}
	}

	return duplicates, nil
}

// IsRemotePath returns true if the path is an rclone remote path (contains :)
func IsRemotePath(path string) bool {
	return strings.Contains(path, ":")
}

// SplitRemotePath splits a remote path into remote name and path
// Example: "myremote:path/to/file" -> ("myremote", "path/to/file")
func SplitRemotePath(remotePath string) (remote, path string) {
	parts := strings.SplitN(remotePath, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", remotePath
}

// JoinRemotePath joins a remote name and path
func JoinRemotePath(remote, path string) string {
	if remote == "" {
		return path
	}
	return remote + ":" + path
}
