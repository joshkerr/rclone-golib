package rclonelib

import "time"

// CommonFlags provides commonly used rclone flags
type CommonFlags struct {
	// Transfers sets the number of file transfers to run in parallel
	Transfers int
	// Checkers sets the number of checkers to run in parallel
	Checkers int
	// Bandwidth limits bandwidth in kBytes/s (0 = unlimited)
	Bandwidth int
	// IgnoreChecksum skips checksum verification for faster transfers
	IgnoreChecksum bool
	// NoTraverse disables directory traversal optimization
	NoTraverse bool
	// Progress shows progress during transfer (-P)
	Progress bool
	// Verbose enables verbose output (-v)
	Verbose bool
	// Exclude patterns to exclude from transfer
	Exclude []string
	// Include patterns to include in transfer
	Include []string
	// MinAge only transfer files older than this
	MinAge string
	// MaxAge only transfer files younger than this
	MaxAge string
}

// ToFlags converts CommonFlags to rclone command-line flags
func (f CommonFlags) ToFlags() []string {
	var flags []string

	if f.Transfers > 0 {
		flags = append(flags, "--transfers", formatInt(f.Transfers))
	}
	if f.Checkers > 0 {
		flags = append(flags, "--checkers", formatInt(f.Checkers))
	}
	if f.Bandwidth > 0 {
		flags = append(flags, "--bwlimit", formatInt(f.Bandwidth)+"k")
	}
	if f.IgnoreChecksum {
		flags = append(flags, "--ignore-checksum")
	}
	if f.NoTraverse {
		flags = append(flags, "--no-traverse")
	}
	if f.Progress {
		flags = append(flags, "-P")
	}
	if f.Verbose {
		flags = append(flags, "-v")
	}

	for _, pattern := range f.Exclude {
		flags = append(flags, "--exclude", pattern)
	}
	for _, pattern := range f.Include {
		flags = append(flags, "--include", pattern)
	}

	if f.MinAge != "" {
		flags = append(flags, "--min-age", f.MinAge)
	}
	if f.MaxAge != "" {
		flags = append(flags, "--max-age", f.MaxAge)
	}

	return flags
}

// TransferOptions provides a builder-pattern for configuring transfers
type TransferOptions struct {
	opts RcloneOptions
}

// NewTransferOptions creates a new TransferOptions builder
func NewTransferOptions(source, destination string) *TransferOptions {
	return &TransferOptions{
		opts: RcloneOptions{
			Command:       RcloneCopy,
			Source:        source,
			Destination:   destination,
			Flags:         []string{},
			StatsInterval: "500ms",
		},
	}
}

// WithCommand sets the rclone command
func (t *TransferOptions) WithCommand(cmd RcloneCommand) *TransferOptions {
	t.opts.Command = cmd
	return t
}

// WithFlags adds custom flags
func (t *TransferOptions) WithFlags(flags ...string) *TransferOptions {
	t.opts.Flags = append(t.opts.Flags, flags...)
	return t
}

// WithCommonFlags adds common flags
func (t *TransferOptions) WithCommonFlags(common CommonFlags) *TransferOptions {
	t.opts.Flags = append(t.opts.Flags, common.ToFlags()...)
	return t
}

// WithStatsInterval sets the stats update interval
func (t *TransferOptions) WithStatsInterval(interval time.Duration) *TransferOptions {
	t.opts.StatsInterval = interval.String()
	return t
}

// WithDryRun enables dry-run mode
func (t *TransferOptions) WithDryRun() *TransferOptions {
	t.opts.DryRun = true
	return t
}

// Build returns the configured RcloneOptions
func (t *TransferOptions) Build() RcloneOptions {
	return t.opts
}

// Helper to format int as string
func formatInt(i int) string {
	if i < 10 {
		return string(rune(i + '0'))
	}
	// For larger numbers, use proper conversion
	var result []byte
	for i > 0 {
		result = append([]byte{byte(i%10 + '0')}, result...)
		i /= 10
	}
	return string(result)
}
