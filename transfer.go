// Package rclonelib provides a reusable library for rclone file transfers with progress tracking.
package rclonelib

import (
	"fmt"
	"sync"
	"time"
)

// Status represents the current status of a transfer
type Status string

// Status constants for transfer states
const (
	StatusPending    Status = "pending"     // Transfer is queued and waiting to start
	StatusInProgress Status = "in_progress" // Transfer is currently running
	StatusCompleted  Status = "completed"   // Transfer finished successfully
	StatusFailed     Status = "failed"      // Transfer failed with an error
)

// Transfer represents a single file transfer operation
type Transfer struct {
	ID          string
	Source      string
	Destination string
	Status      Status
	Progress    float64 // 0-100
	BytesTotal  int64
	BytesCopied int64
	StartTime   time.Time
	EndTime     time.Time
	Error       error
}

// Manager tracks multiple file transfers
type Manager struct {
	mu        sync.RWMutex
	transfers map[string]*Transfer
	order     []string // Maintains insertion order
}

// NewManager creates a new transfer manager
func NewManager() *Manager {
	return &Manager{
		transfers: make(map[string]*Transfer),
		order:     make([]string, 0),
	}
}

// Add adds a new transfer to the manager
func (m *Manager) Add(id, source, destination string) *Transfer {
	m.mu.Lock()
	defer m.mu.Unlock()

	t := &Transfer{
		ID:          id,
		Source:      source,
		Destination: destination,
		Status:      StatusPending,
		Progress:    0,
	}

	m.transfers[id] = t
	m.order = append(m.order, id)
	return t
}

// Start marks a transfer as in progress
func (m *Manager) Start(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.transfers[id]; exists {
		t.Status = StatusInProgress
		t.StartTime = time.Now()
	}
}

// UpdateProgress updates the progress of a transfer
func (m *Manager) UpdateProgress(id string, progress float64, bytesCopied, bytesTotal int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.transfers[id]; exists {
		t.Progress = progress
		t.BytesCopied = bytesCopied
		t.BytesTotal = bytesTotal
	}
}

// Complete marks a transfer as completed successfully
func (m *Manager) Complete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.transfers[id]; exists {
		t.Status = StatusCompleted
		t.Progress = 100
		t.EndTime = time.Now()
	}
}

// Fail marks a transfer as failed with an error
func (m *Manager) Fail(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.transfers[id]; exists {
		t.Status = StatusFailed
		t.EndTime = time.Now()
		t.Error = err
	}
}

// Get retrieves a transfer by ID
func (m *Manager) Get(id string) (*Transfer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, exists := m.transfers[id]
	return t, exists
}

// GetAll returns all transfers in insertion order
func (m *Manager) GetAll() []*Transfer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Transfer, 0, len(m.order))
	for _, id := range m.order {
		if t, exists := m.transfers[id]; exists {
			result = append(result, t)
		}
	}
	return result
}

// Stats returns counts for each status
func (m *Manager) Stats() (pending, inProgress, completed, failed int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.transfers {
		switch t.Status {
		case StatusPending:
			pending++
		case StatusInProgress:
			inProgress++
		case StatusCompleted:
			completed++
		case StatusFailed:
			failed++
		}
	}
	return
}

// Duration returns the elapsed time for a transfer
func (t *Transfer) Duration() time.Duration {
	if t.StartTime.IsZero() {
		return 0
	}
	if t.EndTime.IsZero() {
		return time.Since(t.StartTime)
	}
	return t.EndTime.Sub(t.StartTime)
}

// FormattedBytes returns a human-readable byte size
func FormattedBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Speed calculates transfer speed in bytes per second
func (t *Transfer) Speed() float64 {
	if t.StartTime.IsZero() {
		return 0
	}
	elapsed := t.Duration().Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(t.BytesCopied) / elapsed
}

// FormattedSpeed returns human-readable transfer speed
func (t *Transfer) FormattedSpeed() string {
	speed := t.Speed()
	if speed == 0 {
		return "0 B/s"
	}
	return FormattedBytes(int64(speed)) + "/s"
}
