// Package aws provides retry logic with exponential backoff for transient AWS failures
package aws

import (
	"context"
	"fmt"
	"math"

	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- retry jitter, not cryptography.
	"math/rand" //nolint:gosec // math/rand used for retry jitter, not cryptography
	"net"
	"strings"
	"time"
)

// RetryConfig configures retry behavior for AWS operations
type RetryConfig struct {
	MaxAttempts     int           // Maximum number of retry attempts (default: 3)
	InitialDelay    time.Duration // Initial delay before first retry (default: 1s)
	MaxDelay        time.Duration // Maximum delay between retries (default: 30s)
	Multiplier      float64       // Exponential backoff multiplier (default: 2.0)
	JitterFraction  float64       // Jitter as fraction of delay (default: 0.1 = 10%)
	RetryableErrors []string      // Additional retryable error patterns
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:    5, // Issue #108: Max 5 retries for transient AWS failures
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0.1,
		RetryableErrors: []string{
			// AWS throttling errors
			"Throttling",
			"ThrottlingException",
			"TooManyRequestsException",
			"RequestLimitExceeded",
			"ProvisionedThroughputExceededException",
			"RequestThrottled",

			// AWS service errors
			"ServiceUnavailable",
			"InternalError",
			"InternalFailure",
			"InternalServerError",
			"SlowDown",

			// AWS capacity errors (Issue #108)
			"InsufficientInstanceCapacity",
			"InsufficientCapacity",
			"InstanceLimitExceeded",

			// Network errors
			"RequestTimeout",
			"connection reset",
			"connection refused",
			"no such host",
			"timeout",
			"TLS handshake timeout",

			// Transient errors
			"Unavailable",
			"temporarily unavailable",
			"try again",
		},
	}
}

// RetryableOperation is a function that can be retried
type RetryableOperation func(ctx context.Context) error

// WithRetry executes an operation with exponential backoff retry logic
func WithRetry(ctx context.Context, config *RetryConfig, operation RetryableOperation) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute operation
		err := operation(ctx)
		if err == nil {
			// Success
			if attempt > 1 {
				// Log successful retry
				fmt.Printf("✅ Operation succeeded after %d attempt(s)\n", attempt)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err, config) {
			// Non-retryable error, fail immediately
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Check if we've exhausted retries
		if attempt >= config.MaxAttempts {
			return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
		}

		// Check context cancellation before sleeping
		if ctx.Err() != nil {
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		}

		// Log retry attempt
		fmt.Printf("⚠️  Retryable error on attempt %d/%d: %v\n", attempt, config.MaxAttempts, err)
		fmt.Printf("   Retrying in %s...\n", delay)

		// Sleep with exponential backoff and jitter
		select {
		case <-time.After(delay):
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * config.Multiplier)

			// Apply jitter (random ±10% by default)
			jitter := time.Duration(float64(delay) * config.JitterFraction * (2*rand.Float64() - 1))
			delay += jitter

			// Cap at max delay
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled during retry: %w", ctx.Err())
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error, config *RetryConfig) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Check against configured retryable error patterns
	for _, pattern := range config.RetryableErrors {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Check for network errors (always retryable)
	if isNetworkError(err) {
		return true
	}

	// Check for timeout errors
	if isTimeoutError(err) {
		return true
	}

	return false
}

// isNetworkError checks if an error is a network-related error
func isNetworkError(err error) bool {
	// Check for net.Error interface (timeout errors)
	// Note: We no longer check netErr.Temporary() as it was deprecated in Go 1.18
	// Most temporary errors are actually timeouts, which we handle explicitly
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Check for DNS errors
	if _, ok := err.(*net.DNSError); ok {
		return true
	}

	// Check for operation errors
	if _, ok := err.(*net.OpError); ok {
		return true
	}

	return false
}

// isTimeoutError checks if an error is a timeout
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "deadline exceeded")
}

// RetryMetrics tracks retry statistics
type RetryMetrics struct {
	TotalOperations   int
	SuccessfulRetries int
	FailedRetries     int
	AverageAttempts   float64
}

// RetryTracker tracks retry metrics for monitoring
type RetryTracker struct {
	metrics RetryMetrics
}

// NewRetryTracker creates a new retry tracker
func NewRetryTracker() *RetryTracker {
	return &RetryTracker{}
}

// RecordSuccess records a successful operation (with retry count)
func (t *RetryTracker) RecordSuccess(attempts int) {
	t.metrics.TotalOperations++
	if attempts > 1 {
		t.metrics.SuccessfulRetries++
	}
	t.updateAverage(attempts)
}

// RecordFailure records a failed operation (after all retries exhausted)
func (t *RetryTracker) RecordFailure(attempts int) {
	t.metrics.TotalOperations++
	t.metrics.FailedRetries++
	t.updateAverage(attempts)
}

func (t *RetryTracker) updateAverage(attempts int) {
	// Update running average
	total := t.metrics.TotalOperations
	prevTotal := float64(total - 1)
	t.metrics.AverageAttempts = (t.metrics.AverageAttempts*prevTotal + float64(attempts)) / float64(total)
}

// GetMetrics returns current retry metrics
func (t *RetryTracker) GetMetrics() RetryMetrics {
	return t.metrics
}

// WithRetryAndTracking executes an operation with retry logic and tracks metrics
func WithRetryAndTracking(ctx context.Context, config *RetryConfig, tracker *RetryTracker, operation RetryableOperation) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	attempts := 0
	err := WithRetry(ctx, config, func(ctx context.Context) error {
		attempts++
		return operation(ctx)
	})

	// Record metrics
	if tracker != nil {
		if err == nil {
			tracker.RecordSuccess(attempts)
		} else {
			tracker.RecordFailure(attempts)
		}
	}

	return err
}

// ExponentialBackoff calculates the delay for a given attempt
func ExponentialBackoff(attempt int, initialDelay, maxDelay time.Duration, multiplier float64) time.Duration {
	// Calculate exponential delay: initialDelay * multiplier^(attempt-1)
	delay := time.Duration(float64(initialDelay) * math.Pow(multiplier, float64(attempt-1)))

	// Cap at max delay
	if delay > maxDelay {
		return maxDelay
	}

	return delay
}
