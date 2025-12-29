package rclonelib

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig holds retry configuration for rclone operations
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts (default: 3)
	MaxAttempts int
	// InitialDelay is the initial delay between retries (default: 2s)
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries (default: 30s)
	MaxDelay time.Duration
	// Multiplier is the multiplier for exponential backoff (default: 2.0)
	Multiplier float64
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 2 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// ExecuteWithRetry executes an rclone command with retry logic and exponential backoff
func (e *Executor) ExecuteWithRetry(transferID string, opts RcloneOptions, retryCfg RetryConfig) error {
	if retryCfg.MaxAttempts <= 0 {
		retryCfg.MaxAttempts = 1
	}
	if retryCfg.InitialDelay <= 0 {
		retryCfg.InitialDelay = 2 * time.Second
	}
	if retryCfg.MaxDelay <= 0 {
		retryCfg.MaxDelay = 30 * time.Second
	}
	if retryCfg.Multiplier <= 0 {
		retryCfg.Multiplier = 2.0
	}

	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	var lastErr error
	delay := retryCfg.InitialDelay

	for attempt := 1; attempt <= retryCfg.MaxAttempts; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("context cancelled after %d attempts: %w", attempt-1, lastErr)
			}
			return ctx.Err()
		default:
		}

		// Execute the transfer
		err := e.Execute(transferID, opts)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Don't sleep after last attempt
		if attempt == retryCfg.MaxAttempts {
			break
		}

		// Calculate next delay with exponential backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled after %d attempts: %w", attempt, lastErr)
		case <-time.After(delay):
			delay = time.Duration(math.Min(
				float64(delay)*retryCfg.Multiplier,
				float64(retryCfg.MaxDelay),
			))
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", retryCfg.MaxAttempts, lastErr)
}
