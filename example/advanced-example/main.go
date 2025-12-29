package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	rclone "github.com/joshkerr/rclone-golib"
)

func main() {
	fmt.Println("🚀 Advanced rclone-golib Example")
	fmt.Println()

	ctx := context.Background()

	// 1. Validation - check rclone is installed
	fmt.Println("✅ Validating rclone installation...")
	if err := rclone.ValidateRcloneInstalled(); err != nil {
		log.Fatalf("Rclone not installed: %v", err)
	}

	version, _ := rclone.GetRcloneVersion(ctx)
	fmt.Printf("   Found: %s\n", version)
	fmt.Println()

	// 2. Parse command line arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: advanced-example <source> <destination>")
		fmt.Println("Example: advanced-example /path/to/file remote:path/to/destination")
		fmt.Println()
		fmt.Println("Available remotes:")
		remotes, _ := rclone.ListRemotes(ctx)
		for _, remote := range remotes {
			fmt.Printf("  - %s\n", remote)
		}
		os.Exit(1)
	}

	source := os.Args[1]
	destination := os.Args[2]

	// 3. Validate paths
	fmt.Println("✅ Validating paths...")
	if err := rclone.ValidateSourcePath(source); err != nil {
		log.Fatalf("Invalid source: %v", err)
	}
	fmt.Printf("   Source: %s ✓\n", source)

	// If destination is a remote, validate it
	if rclone.IsRemotePath(destination) {
		remoteName, _ := rclone.SplitRemotePath(destination)
		if err := rclone.ValidateRemote(ctx, remoteName, 10*time.Second); err != nil {
			log.Fatalf("Invalid remote: %v", err)
		}
		fmt.Printf("   Destination: %s ✓\n", destination)
	}
	fmt.Println()

	// 4. Get file size and check disk space
	fmt.Println("✅ Checking disk space...")
	size, err := rclone.GetFileSize(ctx, source)
	if err != nil {
		log.Printf("Warning: Could not get file size: %v", err)
	} else {
		fmt.Printf("   File size: %s\n", rclone.FormattedBytes(size))
		
		// Check disk space if destination is local
		if !rclone.IsRemotePath(destination) {
			if err := rclone.CheckDiskSpace(destination, size); err != nil {
				log.Fatalf("Disk space check failed: %v", err)
			}
			fmt.Println("   Sufficient disk space ✓")
		}
	}
	fmt.Println()

	// 5. Check for duplicates
	fmt.Println("✅ Checking for duplicates...")
	duplicates, err := rclone.CheckDuplicates(ctx, destination, []string{source})
	if err != nil {
		log.Printf("Warning: Could not check duplicates: %v", err)
	} else if len(duplicates) > 0 {
		fmt.Println("   Warning: File may already exist at destination")
	} else {
		fmt.Println("   No duplicates found ✓")
	}
	fmt.Println()

	// 6. Create transfer manager
	manager := rclone.NewManager()
	transferID := "main_transfer"
	manager.Add(transferID, source, destination)

	// 7. Start the UI in a goroutine
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

	// 8. Execute the transfer with retry
	executor := rclone.NewExecutor(manager)
	manager.Start(transferID)

	// Build options with CommonFlags
	commonFlags := rclone.CommonFlags{
		Transfers: 4,
		Checkers:  8,
		Progress:  true,
	}

	opts := rclone.NewTransferOptions(source, destination).
		WithCommand(rclone.RcloneCopy).
		WithCommonFlags(commonFlags).
		WithStatsInterval(500 * time.Millisecond).
		Build()

	opts.Context = ctx

	// Configure retry with exponential backoff
	retryCfg := rclone.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 2 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}

	// Execute with retry
	err = executor.ExecuteWithRetry(transferID, opts, retryCfg)
	
	if err != nil {
		manager.Fail(transferID, err)
		
		// Classify the error
		classified := rclone.ClassifyError(err)
		fmt.Printf("\n❌ Transfer failed: %v\n", err)
		fmt.Printf("   Error type: %s\n", classified.Type)
		fmt.Printf("   Retryable: %v\n", classified.Retryable)
		fmt.Printf("   Temporary: %v\n", classified.Temporary)
	} else {
		manager.Complete(transferID)
		fmt.Println("\n✅ Transfer completed successfully!")
	}

	// Wait for UI to finish
	time.Sleep(500 * time.Millisecond)
	wg.Wait()
}
