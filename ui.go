package rclonelib

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the UI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	pendingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	inProgressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	completedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	failedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	statsStyle = lipgloss.NewStyle().
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)
)

// Model represents the Bubble Tea model for the transfer UI
type Model struct {
	manager  *Manager
	progress map[string]progress.Model
	width    int
	height   int
	done     bool
}

// NewModel creates a new transfer UI model
func NewModel(manager *Manager) Model {
	return Model{
		manager:  manager,
		progress: make(map[string]progress.Model),
		width:    80,
		height:   24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.EnterAltScreen)
}

// tickMsg is sent periodically to update the UI
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// doneMsg is sent when all transfers are complete
type doneMsg struct{}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update all progress bar widths
		for id := range m.progress {
			prog := m.progress[id]
			prog.Width = m.width - 20 // Leave space for labels
			m.progress[id] = prog
		}

	case tickMsg:
		// Check if all transfers are done
		pending, inProgress, _, _ := m.manager.Stats()
		if pending == 0 && inProgress == 0 && !m.done {
			m.done = true
			return m, tea.Sequence(
				tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return doneMsg{}
				}),
			)
		}

		// Update progress bars
		cmds := make([]tea.Cmd, 0)
		for _, t := range m.manager.GetAll() {
			if t.Status == StatusInProgress {
				if _, exists := m.progress[t.ID]; !exists {
					prog := progress.New(
						progress.WithDefaultGradient(),
						progress.WithWidth(m.width-20),
					)
					m.progress[t.ID] = prog
				}
			}
		}

		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case doneMsg:
		return m, tea.Quit
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.manager == nil {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("File Transfer Progress"))
	b.WriteString("\n\n")

	// Stats
	pending, inProgress, completed, failed := m.manager.Stats()
	stats := fmt.Sprintf(
		"Pending: %d | In Progress: %d | Completed: %d | Failed: %d",
		pending, inProgress, completed, failed,
	)
	b.WriteString(statsStyle.Render(stats))
	b.WriteString("\n")

	// List transfers
	transfers := m.manager.GetAll()

	// Show in-progress first
	for _, t := range transfers {
		if t.Status == StatusInProgress {
			b.WriteString(m.renderTransfer(t))
		}
	}

	// Then pending
	for _, t := range transfers {
		if t.Status == StatusPending {
			b.WriteString(m.renderTransfer(t))
		}
	}

	// Then completed
	for _, t := range transfers {
		if t.Status == StatusCompleted {
			b.WriteString(m.renderTransfer(t))
		}
	}

	// Finally failed
	for _, t := range transfers {
		if t.Status == StatusFailed {
			b.WriteString(m.renderTransfer(t))
		}
	}

	// Footer
	if m.done {
		b.WriteString("\n")
		b.WriteString(completedStyle.Render("All transfers complete! Exiting in 2 seconds..."))
	} else {
		b.WriteString("\n")
		b.WriteString(pendingStyle.Render("Press q to quit"))
	}

	return b.String()
}

func (m Model) renderTransfer(t *Transfer) string {
	var b strings.Builder

	// Status prefix
	prefix := " "
	style := pendingStyle
	switch t.Status {
	case StatusPending:
		prefix = "[PENDING]"
		style = pendingStyle
	case StatusInProgress:
		prefix = "[ACTIVE] "
		style = inProgressStyle
	case StatusCompleted:
		prefix = "[DONE]   "
		style = completedStyle
	case StatusFailed:
		prefix = "[FAILED] "
		style = failedStyle
	}

	// First line: status prefix, filename, destination
	filename := filepath.Base(t.Source)
	if len(filename) > 40 {
		filename = filename[:37] + "..."
	}

	dest := t.Destination
	if len(dest) > 30 {
		dest = "..." + dest[len(dest)-27:]
	}

	statusLine := fmt.Sprintf("%s %s -> %s", prefix, filename, dest)
	b.WriteString(itemStyle.Render(style.Render(statusLine)))
	b.WriteString("\n")

	// Second line: progress bar (if in progress)
	if t.Status == StatusInProgress {
		if prog, exists := m.progress[t.ID]; exists {
			// Show progress bar even if we don't have percentage yet
			if t.Progress > 0 {
				progressBar := prog.ViewAs(t.Progress / 100.0)
				b.WriteString(itemStyle.Render(progressBar))
				b.WriteString("\n")

				// Add stats if we have them
				if t.BytesTotal > 0 {
					stats := fmt.Sprintf("  %s / %s (%s) %.0f%%",
						FormattedBytes(t.BytesCopied),
						FormattedBytes(t.BytesTotal),
						t.FormattedSpeed(),
						t.Progress,
					)
					b.WriteString(itemStyle.Render(pendingStyle.Render(stats)))
					b.WriteString("\n")
				}
			} else {
				// No progress yet, show waiting message
				waiting := "  Initializing transfer..."
				b.WriteString(itemStyle.Render(pendingStyle.Render(waiting)))
				b.WriteString("\n")
			}
		}
	}

	// Error message (if failed)
	if t.Status == StatusFailed && t.Error != nil {
		errorMsg := fmt.Sprintf("  Error: %v", t.Error)
		b.WriteString(itemStyle.Render(failedStyle.Render(errorMsg)))
		b.WriteString("\n")
	}

	// Duration (if completed or failed)
	if t.Status == StatusCompleted || t.Status == StatusFailed {
		duration := t.Duration()
		timeMsg := fmt.Sprintf("  Completed in %v", duration.Round(time.Millisecond))
		b.WriteString(itemStyle.Render(pendingStyle.Render(timeMsg)))
		b.WriteString("\n")
	}

	return b.String()
}

// Run starts the transfer UI
func Run(manager *Manager) error {
	p := tea.NewProgram(NewModel(manager))
	_, err := p.Run()
	return err
}
