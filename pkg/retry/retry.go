// Package retry provides exponential backoff retry logic for transient failures
//
// This implements a robust retry mechanism for handling transient AWS API
// failures with intelligent backoff and jitter. Key features:
//
// - Exponential backoff with configurable base delay and maximum attempts
// - Jitter to prevent thundering herd problems
// - Context-aware operations with timeout support
// - Retry predicate for selective retries based on error type
// - Metrics tracking for success rate and attempt distribution
//
// Example usage:
//
//	retryer := retry.NewExponentialBackoff(3, time.Second)
//	err := retryer.Do(ctx, func() error {
//	    return awsOperation()
//	})
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand" //nolint:gosec // math/rand used for retry jitter, not cryptography
	"strings"
	"time"
)

// ExponentialBackoff implements retry logic with exponential backoff and jitter
type ExponentialBackoff struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	jitterRatio float64 // 0.0-1.0, amount of randomness to add

	// Retry predicate (returns true if error is retryable)
	shouldRetry func(error) bool

	// Metrics
	totalOperations  int64
	successfulOps    int64
	failedOps        int64
	attemptHistogram map[int]int64 // attempts -> count
	totalRetries     int64
}

// RetryableError wraps an error with retry information
type RetryableError struct {
	Operation   string
	Attempt     int
	MaxAttempts int
	LastError   error
	TotalDelay  time.Duration
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf(
		"operation %s failed after %d/%d attempts (total delay: %v): %v",
		e.Operation, e.Attempt, e.MaxAttempts, e.TotalDelay.Round(time.Millisecond), e.LastError,
	)
}

func (e *RetryableError) Unwrap() error {
	return e.LastError
}

// NewExponentialBackoff creates a new exponential backoff retryer
//
// Parameters:
//   - maxAttempts: Maximum number of attempts (including initial try)
//   - baseDelay: Initial delay before first retry
//
// The backoff formula is: delay = baseDelay * (2 ^ (attempt - 1)) + jitter
// Maximum delay is capped at 30 seconds by default.
func NewExponentialBackoff(maxAttempts int, baseDelay time.Duration) *ExponentialBackoff {
	if maxAttempts <= 0 {
		maxAttempts = 3 // Default: 3 attempts
	}
	if baseDelay <= 0 {
		baseDelay = time.Second // Default: 1 second base delay
	}

	return &ExponentialBackoff{
		maxAttempts:      maxAttempts,
		baseDelay:        baseDelay,
		maxDelay:         30 * time.Second,
		jitterRatio:      0.2, // 20% jitter
		shouldRetry:      defaultRetryPredicate,
		attemptHistogram: make(map[int]int64),
	}
}

// WithMaxDelay sets the maximum delay between retries
func (r *ExponentialBackoff) WithMaxDelay(maxDelay time.Duration) *ExponentialBackoff {
	r.maxDelay = maxDelay
	return r
}

// WithJitter sets the jitter ratio (0.0-1.0)
//
// Jitter adds randomness to retry delays to prevent thundering herd
// problems when many clients retry simultaneously. A ratio of 0.2
// adds up to 20% random variation.
func (r *ExponentialBackoff) WithJitter(ratio float64) *ExponentialBackoff {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	r.jitterRatio = ratio
	return r
}

// WithRetryPredicate sets a custom retry predicate function
//
// The predicate receives an error and returns true if the error is
// retryable. By default, transient AWS errors are retried.
func (r *ExponentialBackoff) WithRetryPredicate(predicate func(error) bool) *ExponentialBackoff {
	r.shouldRetry = predicate
	return r
}

// Do executes an operation with retry logic
//
// The operation function is called up to maxAttempts times. If it returns
// an error that passes the retry predicate, it will be retried with
// exponential backoff. Context cancellation is checked before each attempt.
//
// Returns:
//   - nil if operation succeeds
//   - RetryableError if all attempts fail
//   - context error if context is canceled/timed out
func (r *ExponentialBackoff) Do(ctx context.Context, operation func() error) error {
	return r.DoNamed(ctx, "operation", operation)
}

// DoNamed is like Do but accepts an operation name for better error messages
func (r *ExponentialBackoff) DoNamed(ctx context.Context, operationName string, operation func() error) error {
	r.totalOperations++

	var lastErr error
	var totalDelay time.Duration

	for attempt := 1; attempt <= r.maxAttempts; attempt++ {
		// Check context before each attempt
		select {
		case <-ctx.Done():
			r.failedOps++
			return ctx.Err()
		default:
		}

		// Execute operation
		err := operation()
		if err == nil {
			// Success!
			r.successfulOps++
			r.attemptHistogram[attempt]++
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.shouldRetry(err) {
			// Non-retryable error, fail immediately
			r.failedOps++
			r.attemptHistogram[attempt]++
			return &RetryableError{
				Operation:   operationName,
				Attempt:     attempt,
				MaxAttempts: r.maxAttempts,
				LastError:   err,
				TotalDelay:  totalDelay,
			}
		}

		// If this was the last attempt, fail
		if attempt == r.maxAttempts {
			r.failedOps++
			r.attemptHistogram[attempt]++
			return &RetryableError{
				Operation:   operationName,
				Attempt:     attempt,
				MaxAttempts: r.maxAttempts,
				LastError:   err,
				TotalDelay:  totalDelay,
			}
		}

		// Calculate backoff delay
		delay := r.calculateBackoff(attempt)
		totalDelay += delay
		r.totalRetries++

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			r.failedOps++
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// Should never reach here, but handle gracefully
	r.failedOps++
	return &RetryableError{
		Operation:   operationName,
		Attempt:     r.maxAttempts,
		MaxAttempts: r.maxAttempts,
		LastError:   lastErr,
		TotalDelay:  totalDelay,
	}
}

// calculateBackoff calculates the backoff delay for a given attempt
// with exponential backoff and jitter
func (r *ExponentialBackoff) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^(attempt-1)
	// attempt 1 -> baseDelay * 1
	// attempt 2 -> baseDelay * 2
	// attempt 3 -> baseDelay * 4
	exponentialDelay := float64(r.baseDelay) * math.Pow(2, float64(attempt-1))

	// Cap at max delay
	if exponentialDelay > float64(r.maxDelay) {
		exponentialDelay = float64(r.maxDelay)
	}

	// Add jitter (random variation)
	jitter := exponentialDelay * r.jitterRatio * (rand.Float64() - 0.5) * 2
	finalDelay := exponentialDelay + jitter

	// Ensure positive delay
	if finalDelay < 0 {
		finalDelay = exponentialDelay
	}

	return time.Duration(finalDelay)
}

// defaultRetryPredicate determines if an error should be retried
//
// Retries transient errors like:
// - Network timeouts
// - Throttling errors
// - Service unavailable
// - Internal server errors
func isNetworkOrTimeoutError(s string) bool {
	return strings.Contains(s, "timeout") || strings.Contains(s, "timed out") ||
		strings.Contains(s, "connection reset") || strings.Contains(s, "connection refused") ||
		strings.Contains(s, "temporary failure")
}

func isThrottlingError(s string) bool {
	return strings.Contains(s, "throttl") || strings.Contains(s, "rate exceed") ||
		strings.Contains(s, "too many requests") || strings.Contains(s, "request limit exceeded")
}

func isServiceError(s string) bool {
	return strings.Contains(s, "service unavailable") || strings.Contains(s, "internal error") ||
		strings.Contains(s, "internal server error") || strings.Contains(s, "503") ||
		strings.Contains(s, "500")
}

func isCapacityError(s string) bool {
	return strings.Contains(s, "insufficientinstancecapacity") || strings.Contains(s, "capacity")
}

func defaultRetryPredicate(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return isNetworkOrTimeoutError(s) || isThrottlingError(s) || isServiceError(s) || isCapacityError(s)
}

// GetMetrics returns retry metrics
func (r *ExponentialBackoff) GetMetrics() Metrics {
	successRate := 0.0
	if r.totalOperations > 0 {
		successRate = float64(r.successfulOps) / float64(r.totalOperations)
	}

	avgRetries := 0.0
	if r.totalOperations > 0 {
		avgRetries = float64(r.totalRetries) / float64(r.totalOperations)
	}

	return Metrics{
		TotalOperations:  r.totalOperations,
		SuccessfulOps:    r.successfulOps,
		FailedOps:        r.failedOps,
		SuccessRate:      successRate,
		TotalRetries:     r.totalRetries,
		AvgRetries:       avgRetries,
		AttemptHistogram: r.copyHistogram(),
	}
}

func (r *ExponentialBackoff) copyHistogram() map[int]int64 {
	copy := make(map[int]int64, len(r.attemptHistogram))
	for k, v := range r.attemptHistogram {
		copy[k] = v
	}
	return copy
}

// ResetMetrics resets all metrics counters
func (r *ExponentialBackoff) ResetMetrics() {
	r.totalOperations = 0
	r.successfulOps = 0
	r.failedOps = 0
	r.totalRetries = 0
	r.attemptHistogram = make(map[int]int64)
}

// Metrics contains retry performance metrics
type Metrics struct {
	TotalOperations  int64         // Total operations attempted
	SuccessfulOps    int64         // Operations that succeeded
	FailedOps        int64         // Operations that failed after all retries
	SuccessRate      float64       // Success rate (0.0-1.0)
	TotalRetries     int64         // Total number of retries across all operations
	AvgRetries       float64       // Average retries per operation
	AttemptHistogram map[int]int64 // Attempts -> count distribution
}

// String returns a human-readable representation of metrics
func (m Metrics) String() string {
	return fmt.Sprintf(
		"Retry Metrics:\n"+
			"  Total Operations: %d\n"+
			"  Successful: %d (%.1f%%)\n"+
			"  Failed: %d\n"+
			"  Total Retries: %d\n"+
			"  Avg Retries/Op: %.2f",
		m.TotalOperations,
		m.SuccessfulOps, m.SuccessRate*100,
		m.FailedOps,
		m.TotalRetries,
		m.AvgRetries,
	)
}
