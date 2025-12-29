package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	rclone "github.com/joshkerr/rclone-golib"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: example <source> <destination>")
		fmt.Println("Example: example /path/to/file remote:path/to/destination")
		os.Exit(1)
	}

	source := os.Args[1]
	destination := os.Args[2]

	// Create a transfer manager
	manager := rclone.NewManager()

	// Add the transfer
	transferID := "example_transfer"
	manager.Add(transferID, source, destination)

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

	// Execute the transfer
	executor := rclone.NewExecutor(manager)
	manager.Start(transferID)

	opts := rclone.RcloneOptions{
		Command:       rclone.RcloneCopy,
		Source:        source,
		Destination:   destination,
		StatsInterval: "500ms",
		Context:       context.Background(),
	}

	if err := executor.Execute(transferID, opts); err != nil {
		manager.Fail(transferID, err)
		fmt.Printf("\nTransfer failed: %v\n", err)
	} else {
		manager.Complete(transferID)
		fmt.Println("\nTransfer completed successfully!")
	}

	// Wait for UI to finish
	time.Sleep(500 * time.Millisecond)
	wg.Wait()
}
