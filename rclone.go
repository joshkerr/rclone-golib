package rclonelib

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// RcloneCommand represents the type of rclone operation
type RcloneCommand string

const (
	// RcloneCopy copies files from source to destination
	RcloneCopy RcloneCommand = "copy"
	// RcloneCopyTo copies a single file to a specific destination path
	RcloneCopyTo RcloneCommand = "copyto"
	// RcloneMove moves files from source to destination
	RcloneMove RcloneCommand = "move"
	// RcloneMoveTo moves a single file to a specific destination path
	RcloneMoveTo RcloneCommand = "moveto"
	// RcloneSync syncs source to destination, changing destination only
	RcloneSync RcloneCommand = "sync"
)

// RcloneOptions contains configuration for rclone operations
type RcloneOptions struct {
	// Command is the rclone command to execute (copy, copyto, move, etc.)
	Command RcloneCommand
	// Source is the source path
	Source string
	// Destination is the destination path
	Destination string
	// Flags are additional flags to pass to rclone
	Flags []string
	// StatsInterval is how often to update progress (e.g., "500ms", "1s")
	StatsInterval string
	// DryRun simulates the operation without making changes
	DryRun bool
	// Context allows cancellation of the operation
	Context context.Context
}

// Executor handles rclone command execution with progress tracking
type Executor struct {
	manager *Manager
}

// NewExecutor creates a new rclone executor
func NewExecutor(manager *Manager) *Executor {
	return &Executor{
		manager: manager,
	}
}

// Execute runs an rclone command and tracks its progress
func (e *Executor) Execute(transferID string, opts RcloneOptions) error {
	// Build command arguments
	args := []string{
		string(opts.Command),
		"-v", // Verbose: enables "Transferred:" progress lines to stderr
	}

	// Add stats interval
	statsInterval := opts.StatsInterval
	if statsInterval == "" {
		statsInterval = "500ms"
	}
	args = append(args, "--stats", statsInterval)

	// Add dry-run flag if requested
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	// Add custom flags
	args = append(args, opts.Flags...)

	// Add source and destination
	args = append(args, opts.Source, opts.Destination)

	// Create context if not provided
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Create command
	cmd := exec.CommandContext(ctx, "rclone", args...)

	// Create pipe for stderr (where "Transferred:" lines go with -v flag)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start rclone: %w", err)
	}

	// Parse stderr for progress in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		parseRcloneOutput(bufio.NewReader(stderr), transferID, e.manager)
	}()

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Wait for parsing to finish
	<-done

	return cmdErr
}

// parseRcloneOutput parses rclone output to extract progress information
func parseRcloneOutput(reader *bufio.Reader, transferID string, mgr *Manager) {
	// With -v flag, rclone outputs progress lines to stderr like:
	//   "Transferred:   100 MiB / 2.5 GiB, 4%, 45.2 MiB/s, ETA 50s"
	// Note: rclone uses \r (carriage return) to update progress in place

	scanner := bufio.NewScanner(reader)

	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Custom split function to handle both \r and \n
	// This is critical because rclone uses \r to update progress lines in place
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Look for \r or \n
		if i := strings.IndexAny(string(data), "\r\n"); i >= 0 {
			// Return the token before the delimiter
			token = data[0:i]

			// Skip the delimiter(s) - handle both \r\n and standalone \r or \n
			advance = i + 1
			if advance < len(data) && data[i] == '\r' && data[advance] == '\n' {
				advance++ // Skip the \n after \r
			}

			return advance, token, nil
		}

		// Request more data
		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	// Regex to match "Transferred:" lines with full details including speed and ETA
	// Example: "Transferred:   1.234 GiB / 5.678 GiB, 22%, 10 MiB/s, ETA 1m30s"
	statsRegex := regexp.MustCompile(`Transferred:\s+([0-9.]+)\s*([kKMGTP]i?[Bb]?)\s*/\s*([0-9.]+)\s*([kKMGTP]i?[Bb]?),\s*([0-9]+)%`)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Try to match progress line
		matches := statsRegex.FindStringSubmatch(line)
		if len(matches) >= 6 {
			// Parse percentage
			percentage, err := strconv.ParseFloat(matches[5], 64)
			if err == nil {
				// Parse bytes with proper unit handling
				copied := parseSize(matches[1], matches[2])
				total := parseSize(matches[3], matches[4])
				mgr.UpdateProgress(transferID, percentage, copied, total)
			}
		}
	}
}

// parseSize converts size string to bytes (e.g., "1.234" with unit "GiB")
func parseSize(value, unit string) int64 {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}

	// Normalize unit - handle both "MiB" and "MB" formats
	unit = strings.ToUpper(strings.TrimSpace(unit))
	unit = strings.TrimSuffix(unit, "B")
	unit = strings.TrimSuffix(unit, "I") // Handle MiB vs MB

	multiplier := int64(1)
	switch unit {
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "P":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	}

	return int64(val * float64(multiplier))
}
