# rclone-golib

A reusable Go library for rclone file transfers with beautiful progress bars using Charm's Bubble Tea.

## Features

- 🚀 **Easy rclone integration** - Simple API for executing rclone commands
- 📊 **Real-time progress tracking** - Parse rclone output for live transfer status using proper `\r` carriage return handling
- 🎨 **Beautiful UI** - Charmbracelet Bubble Tea progress bars and styled output
- 🔄 **Multiple transfers** - Track and display multiple concurrent transfers
- 📈 **Transfer stats** - Speed, ETA, bytes transferred, and more
- 🎯 **Thread-safe** - Safely update transfers from multiple goroutines
- ⚡ **Accurate parsing** - Custom scanner split function handles rclone's in-place progress updates
- 🔁 **Retry logic** - Exponential backoff retry for transient failures
- ✅ **Validation** - Pre-flight checks for paths, remotes, disk space
- 🏷️ **Error classification** - Categorize errors as retryable, temporary, etc.
- 🛠️ **Helper utilities** - Remote listing, duplicate checking, path helpers

## Installation

```bash
go get github.com/joshkerr/rclone-golib
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	rclone "github.com/joshkerr/rclone-golib"
)

func main() {
	// Create a transfer manager
	manager := rclone.NewManager()
	
	// Add transfers
	manager.Add("transfer1", "/path/to/source", "remote:destination")
	manager.Add("transfer2", "/path/to/other", "remote:other")
	
	// Start the UI in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p := tea.NewProgram(rclone.NewModel(manager))
		if _, err := p.Run(); err != nil {
			fmt.Printf("UI error: %v\n", err)
		}
	}()
	
	// Small delay to let UI start
	time.Sleep(100 * time.Millisecond)
	
	// Execute transfers
	executor := rclone.NewExecutor(manager)
	
	for _, t := range manager.GetAll() {
		manager.Start(t.ID)
		
		opts := rclone.RcloneOptions{
			Command:       rclone.RcloneCopy,
			Source:        t.Source,
			Destination:   t.Destination,
			StatsInterval: "500ms",
			Context:       context.Background(),
		}
		
		if err := executor.Execute(t.ID, opts); err != nil {
			manager.Fail(t.ID, err)
		} else {
			manager.Complete(t.ID)
		}
	}
	
	// Wait for UI to finish
	wg.Wait()
}
```

## API Reference

### Manager

The `Manager` tracks multiple file transfers and their progress.

```go
// Create a new manager
manager := rclone.NewManager()

// Add a transfer
transfer := manager.Add("id", "/source", "/destination")

// Update transfer status
manager.Start("id")
manager.UpdateProgress("id", percentage, bytesCopied, bytesTotal)
manager.Complete("id")
manager.Fail("id", err)

// Get transfer info
transfer, exists := manager.Get("id")
allTransfers := manager.GetAll()
pending, inProgress, completed, failed := manager.Stats()
```

### Executor

The `Executor` runs rclone commands and tracks their progress.

```go
executor := rclone.NewExecutor(manager)

opts := rclone.RcloneOptions{
	Command:       rclone.RcloneCopy,  // or RcloneCopyTo, RcloneMove, etc.
	Source:        "/source/path",
	Destination:   "remote:destination",
	Flags:         []string{"--transfers=4"},
	StatsInterval: "500ms",
	DryRun:        false,
	Context:       ctx,
}

err := executor.Execute("transfer_id", opts)
```

### UI

The UI provides a beautiful Bubble Tea interface for tracking transfers.

```go
// Create a model
model := rclone.NewModel(manager)

// Run the UI (blocking)
err := rclone.Run(manager)

// Or use with tea.NewProgram for more control
p := tea.NewProgram(model)
_, err := p.Run()
```

## Transfer States

- **Pending**: Transfer is queued and waiting to start
- **In Progress**: Transfer is currently running
- **Completed**: Transfer finished successfully
- **Failed**: Transfer failed with an error

## Helper Functions

```go
// Format bytes to human-readable string
size := rclone.FormattedBytes(1536000) // "1.5 MB"

// Get transfer duration
duration := transfer.Duration()

// Get transfer speed
speed := transfer.Speed() // bytes per second
formattedSpeed := transfer.FormattedSpeed() // e.g., "45.2 MB/s"
```

## Examples

### Single File Transfer

```go
manager := rclone.NewManager()
manager.Add("file1", "/path/to/file.mkv", "remote:movies/")

executor := rclone.NewExecutor(manager)
manager.Start("file1")

opts := rclone.RcloneOptions{
	Command:     rclone.RcloneCopyTo,
	Source:      "/path/to/file.mkv",
	Destination: "remote:movies/file.mkv",
}

if err := executor.Execute("file1", opts); err != nil {
	manager.Fail("file1", err)
} else {
	manager.Complete("file1")
}
```

### Directory Transfer

```go
opts := rclone.RcloneOptions{
	Command:     rclone.RcloneCopy,
	Source:      "/path/to/directory",
	Destination: "remote:backup/",
	Flags:       []string{"--transfers=8", "--checkers=16"},
}

err := executor.Execute("dir1", opts)
```

### Transfer with Custom Flags

```go
opts := rclone.RcloneOptions{
	Command:     rclone.RcloneSync,
	Source:      "/local/photos",
	Destination: "remote:photos",
	Flags: []string{
		"--transfers=4",
		"--checkers=8",
		"--exclude=*.tmp",
		"--max-age=30d",
	},
	StatsInterval: "1s",
}
```

### Transfer with Retry

```go
executor := rclone.NewExecutor(manager)
manager.Start("file1")

opts := rclone.RcloneOptions{
	Command:     rclone.RcloneCopy,
	Source:      "/path/to/file",
	Destination: "remote:backup/",
	Context:     context.Background(),
}

retryCfg := rclone.RetryConfig{
	MaxAttempts:  5,
	InitialDelay: 2 * time.Second,
	MaxDelay:     30 * time.Second,
	Multiplier:   2.0,
}

// Execute with exponential backoff retry
if err := executor.ExecuteWithRetry("file1", opts, retryCfg); err != nil {
	manager.Fail("file1", err)
} else {
	manager.Complete("file1")
}
```

### Validation and Pre-flight Checks

```go
// Check if rclone is installed
if err := rclone.ValidateRcloneInstalled(); err != nil {
	log.Fatal(err)
}

// Validate source path exists
if err := rclone.ValidateSourcePath("/path/to/source"); err != nil {
	log.Fatal(err)
}

// Validate remote is accessible
ctx := context.Background()
if err := rclone.ValidateRemote(ctx, "myremote", 10*time.Second); err != nil {
	log.Fatal(err)
}

// Check disk space before transfer
if err := rclone.CheckDiskSpace("/destination", 10*1024*1024*1024); err != nil {
	log.Fatal(err) // Not enough space for 10GB
}

// Check for partial downloads
hasPartial, _ := rclone.HasPartialFiles("/downloads")
if hasPartial {
	fmt.Println("Found incomplete downloads that can be resumed")
}
```

### Using CommonFlags Builder

```go
commonFlags := rclone.CommonFlags{
	Transfers:      8,
	Checkers:       16,
	Bandwidth:      10000, // 10MB/s
	IgnoreChecksum: true,
	Exclude:        []string{"*.tmp", "*.partial"},
	MaxAge:         "30d",
}

opts := rclone.NewTransferOptions("/source", "remote:dest").
	WithCommand(rclone.RcloneCopy).
	WithCommonFlags(commonFlags).
	WithStatsInterval(500 * time.Millisecond).
	Build()

executor.Execute("transfer1", opts)
```

### Error Classification

```go
err := executor.Execute("transfer1", opts)
if err != nil {
	// Classify the error
	classified := rclone.ClassifyError(err)
	
	switch classified.Type {
	case rclone.ErrorTypeNetwork:
		fmt.Println("Network error - retrying...")
		if classified.Retryable {
			// Retry the operation
		}
	case rclone.ErrorTypeAuth:
		fmt.Println("Authentication failed - check credentials")
	case rclone.ErrorTypeInsufficientSpace:
		fmt.Println("Not enough disk space")
	}
	
	// Or use helper functions
	if rclone.IsRetryable(err) {
		fmt.Println("This error can be retried")
	}
}
```

### Helper Utilities

```go
ctx := context.Background()

// List all configured remotes
remotes, _ := rclone.ListRemotes(ctx)
for _, remote := range remotes {
	fmt.Println("Remote:", remote)
}

// List files in a path
files, _ := rclone.ListFiles(ctx, "myremote:path", false)
for _, file := range files {
	fmt.Println("File:", file)
}

// Check for duplicates before transfer
duplicates, _ := rclone.CheckDuplicates(ctx, "remote:dest", []string{"file1.txt", "file2.txt"})
for file := range duplicates {
	fmt.Printf("%s already exists\n", file)
}

// Get file size
size, _ := rclone.GetFileSize(ctx, "remote:path/file.mkv")
fmt.Printf("File size: %s\n", rclone.FormattedBytes(size))

// Path helpers
remote, path := rclone.SplitRemotePath("myremote:path/to/file")
// remote = "myremote", path = "path/to/file"

fullPath := rclone.JoinRemotePath("myremote", "path/to/file")
// fullPath = "myremote:path/to/file"
```

## How It Works

### Progress Parsing

The library properly handles rclone's progress output by using a **custom scanner split function** that handles both `\r` (carriage return) and `\n` (newline) characters. This is critical because:

- Rclone uses `\r` to update progress lines **in place** on the terminal
- Standard `bufio.Scanner` only splits on `\n` by default
- Without this, progress updates would be missed or garbled

The regex pattern matches rclone's "Transferred:" lines:
```
Transferred:   1.234 GiB / 5.678 GiB, 22%, 10 MiB/s, ETA 1m30s
```

And extracts:
- **Bytes copied**: `1.234 GiB`
- **Total bytes**: `5.678 GiB`  
- **Percentage**: `22%`
- **Speed**: `10 MiB/s` (captured but not yet used)
- **ETA**: `1m30s` (captured but not yet used)

## Used By

This library is used by:
- [qbt-dl](https://github.com/joshkerr/qbt-dl) - qBittorrent download manager
- [rclonecp](https://github.com/joshkerr/rclonecp) - Enhanced rclone copy with TMDB cover art

## License

MIT License - see LICENSE file for details
