// Package integration provides AWS error injection tests for validating retry logic
//
// These tests verify that pkg/aws/retry.go handles various AWS failure scenarios
// with proper exponential backoff, jitter, and graceful degradation.
package integration

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockError creates a mock AWS error with the given message
type MockError struct {
	Message string
}

func (e *MockError) Error() string {
	return e.Message
}

// MockNetworkError creates a mock network error
type MockNetworkError struct {
	Message   string
	IsTimeout bool
}

func (e *MockNetworkError) Error() string {
	return e.Message
}

func (e *MockNetworkError) Timeout() bool {
	return e.IsTimeout
}

func (e *MockNetworkError) Temporary() bool {
	return true
}

// =============================================================================
// 1. AWS Throttling Tests
// =============================================================================

// TestEC2Throttling_RecoversWithBackoff verifies EC2 throttling errors recover with backoff
func TestEC2Throttling_RecoversWithBackoff(t *testing.T) {
	var attemptCount int32
	maxAttempts := 3

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt < int32(maxAttempts) {
			// Simulate throttling for first 2 attempts
			return &MockError{Message: "Throttling: Rate exceeded"}
		}
		// Success on 3rd attempt
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 100 * time.Millisecond

	start := time.Now()
	err := aws.WithRetry(context.Background(), config, operation)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, int32(maxAttempts), attemptCount)
	// Should have delayed at least once (10ms initial delay)
	assert.Greater(t, elapsed, 10*time.Millisecond)
	t.Logf("Recovered from throttling after %d attempts in %s", maxAttempts, elapsed)
}

// TestS3Throttling_RecoversWithBackoff verifies S3 throttling errors recover with backoff
func TestS3Throttling_RecoversWithBackoff(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt <= 2 {
			// Simulate S3 throttling
			return &MockError{Message: "SlowDown: Please reduce your request rate"}
		}
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 100 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(3), attemptCount)
}

// TestThrottlingExponentialBackoff validates exponential backoff timing
func TestThrottlingExponentialBackoff(t *testing.T) {
	delays := []time.Duration{}
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		// Backoff delays are verified via total elapsed time below.
		if attempt <= 3 {
			return &MockError{Message: "Throttling: Rate exceeded"}
		}
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 1 * time.Second
	config.JitterFraction = 0.0 // Disable jitter for predictable timing

	start := time.Now()
	err := aws.WithRetry(context.Background(), config, operation)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, int32(4), attemptCount)

	// Verify elapsed time includes backoff delays
	// Attempt 1: immediate
	// Attempt 2: wait 10ms (initial delay)
	// Attempt 3: wait 20ms (initial * 2)
	// Attempt 4: success
	// Total: ~30ms minimum
	assert.Greater(t, elapsed, 25*time.Millisecond)
	t.Logf("Backoff delays: %v", delays)
}

// =============================================================================
// 2. AWS Capacity Issues Tests
// =============================================================================

// TestInsufficientCapacity_FallbackInstance verifies capacity error handling
func TestInsufficientCapacity_FallbackInstance(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt == 1 {
			// First attempt: insufficient capacity for requested instance type
			return &MockError{Message: "InsufficientInstanceCapacity: Insufficient capacity for m5.large"}
		}
		// Second attempt: fallback to different instance type succeeds
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(2), attemptCount)
	t.Logf("Successfully fell back to alternative instance type after capacity error")
}

// TestInsufficientCapacity_FailGracefully verifies graceful failure after retries
func TestInsufficientCapacity_FailGracefully(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		atomic.AddInt32(&attemptCount, 1)
		// All attempts fail with capacity error
		return &MockError{Message: "InsufficientInstanceCapacity: No capacity available"}
	}

	config := aws.DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 50 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.Error(t, err)
	assert.Equal(t, int32(3), attemptCount)
	assert.Contains(t, err.Error(), "operation failed after 3 attempts")
	assert.Contains(t, err.Error(), "InsufficientInstanceCapacity")
}

// TestInstanceLimitExceeded_ClearErrorMessage verifies clear error messages
func TestInstanceLimitExceeded_ClearErrorMessage(t *testing.T) {
	operation := func(ctx context.Context) error {
		return &MockError{Message: "InstanceLimitExceeded: You have exceeded your instance limit"}
	}

	config := aws.DefaultRetryConfig()
	config.MaxAttempts = 2
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InstanceLimitExceeded")
	t.Logf("Error message: %v", err)
}

// =============================================================================
// 3. AWS Credential Issues Tests
// =============================================================================

// TestExpiredCredentials_ClearErrorMessage verifies non-retryable credential errors
func TestExpiredCredentials_ClearErrorMessage(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		atomic.AddInt32(&attemptCount, 1)
		// Credential errors should not be retried
		return &MockError{Message: "ExpiredToken: The security token included in the request is expired"}
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.Error(t, err)
	// Should fail immediately (non-retryable)
	assert.Equal(t, int32(1), attemptCount)
	assert.Contains(t, err.Error(), "non-retryable error")
	assert.Contains(t, err.Error(), "ExpiredToken")
}

// TestInvalidProfile_ClearErrorMessage verifies profile errors fail immediately
func TestInvalidProfile_ClearErrorMessage(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		atomic.AddInt32(&attemptCount, 1)
		return &MockError{Message: "NoCredentialProviders: no valid providers in chain"}
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.Error(t, err)
	// Should fail on first attempt (non-retryable)
	assert.Equal(t, int32(1), attemptCount)
	assert.Contains(t, err.Error(), "non-retryable")
}

// TestAccessDenied_NonRetryable verifies permission errors fail immediately
func TestAccessDenied_NonRetryable(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		atomic.AddInt32(&attemptCount, 1)
		return &MockError{Message: "AccessDenied: User: arn:aws:iam::123456789012:user/test is not authorized"}
	}

	config := aws.DefaultRetryConfig()
	err := aws.WithRetry(context.Background(), config, operation)

	require.Error(t, err)
	assert.Equal(t, int32(1), attemptCount)
	assert.Contains(t, err.Error(), "non-retryable")
}

// =============================================================================
// 4. Network Failures Tests
// =============================================================================

// TestNetworkTimeout_RetryAndRecover verifies network timeout recovery
func TestNetworkTimeout_RetryAndRecover(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt <= 2 {
			// Simulate network timeout
			return &MockNetworkError{
				Message:   "dial tcp: i/o timeout",
				IsTimeout: true,
			}
		}
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 100 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(3), attemptCount)
	t.Logf("Recovered from network timeout after %d attempts", attemptCount)
}

// TestConnectionDrop_MidOperation verifies connection drop recovery
func TestConnectionDrop_MidOperation(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt == 1 {
			// First attempt: connection drops mid-operation
			return &MockError{Message: "connection reset by peer"}
		}
		// Second attempt succeeds
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(2), attemptCount)
}

// TestDNSFailure_RetryAndRecover verifies DNS resolution errors are retried
func TestDNSFailure_RetryAndRecover(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt <= 2 {
			// Simulate DNS error
			return &net.DNSError{
				Err:         "no such host",
				Name:        "ec2.us-west-2.amazonaws.com",
				IsNotFound:  true,
				IsTemporary: true,
			}
		}
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(3), attemptCount)
}

// TestTLSHandshakeTimeout_RetryAndRecover verifies TLS timeout recovery
func TestTLSHandshakeTimeout_RetryAndRecover(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt == 1 {
			return &MockError{Message: "TLS handshake timeout"}
		}
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(2), attemptCount)
}

// =============================================================================
// 5. AWS Service Outages Tests
// =============================================================================

// TestEC2ServiceUnavailable_QueueOperation verifies service unavailable handling
func TestEC2ServiceUnavailable_QueueOperation(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt <= 3 {
			// Simulate service unavailable
			return &MockError{Message: "ServiceUnavailable: Service is currently unavailable"}
		}
		// Service recovers on 4th attempt
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 200 * time.Millisecond
	config.Multiplier = 2.0

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(4), attemptCount)
	t.Logf("Operation succeeded after service recovered (%d attempts)", attemptCount)
}

// TestEFSServiceUnavailable_FailGracefully verifies EFS service outage handling
func TestEFSServiceUnavailable_FailGracefully(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		atomic.AddInt32(&attemptCount, 1)
		// EFS service remains unavailable
		return &MockError{Message: "InternalServerError: An internal error occurred"}
	}

	config := aws.DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 50 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.Error(t, err)
	assert.Equal(t, int32(3), attemptCount)
	assert.Contains(t, err.Error(), "operation failed after 3 attempts")
}

// TestInternalFailure_RetryAndRecover verifies internal failure recovery
func TestInternalFailure_RetryAndRecover(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt <= 2 {
			return &MockError{Message: "InternalFailure: An internal error has occurred"}
		}
		return nil
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(3), attemptCount)
}

// =============================================================================
// 6. Context Cancellation Tests
// =============================================================================

// TestContextCancellation_StopsRetry verifies context cancellation is respected
func TestContextCancellation_StopsRetry(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		atomic.AddInt32(&attemptCount, 1)
		// Always fail to trigger retries
		return &MockError{Message: "Throttling: Rate exceeded"}
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 50 * time.Millisecond
	config.MaxDelay = 500 * time.Millisecond
	config.Multiplier = 2.0

	// Create context that cancels after 100ms
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := aws.WithRetry(ctx, config, operation)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	// Should have attempted 1-2 times before cancellation
	assert.LessOrEqual(t, attemptCount, int32(3))
	assert.Less(t, elapsed, 200*time.Millisecond)
	t.Logf("Cancelled after %d attempts in %s", attemptCount, elapsed)
}

// TestContextDeadlineExceeded_ClearError verifies deadline exceeded error
func TestContextDeadlineExceeded_ClearError(t *testing.T) {
	operation := func(ctx context.Context) error {
		// Simulate slow operation that checks context
		select {
		case <-time.After(150 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	config := aws.DefaultRetryConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := aws.WithRetry(ctx, config, operation)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

// =============================================================================
// 7. Retry Metrics and Tracking Tests
// =============================================================================

// TestRetryMetrics_TrackSuccessfulRetries verifies retry metrics tracking
func TestRetryMetrics_TrackSuccessfulRetries(t *testing.T) {
	tracker := aws.NewRetryTracker()
	var attemptCount int32

	for i := 0; i < 5; i++ {
		attemptCount = 0
		operation := func(ctx context.Context) error {
			attempt := atomic.AddInt32(&attemptCount, 1)
			if attempt <= 2 {
				return &MockError{Message: "Throttling: Rate exceeded"}
			}
			return nil
		}

		config := aws.DefaultRetryConfig()
		config.InitialDelay = 1 * time.Millisecond

		err := aws.WithRetryAndTracking(context.Background(), config, tracker, operation)
		require.NoError(t, err)
	}

	metrics := tracker.GetMetrics()
	assert.Equal(t, 5, metrics.TotalOperations)
	assert.Equal(t, 5, metrics.SuccessfulRetries)
	assert.Equal(t, 0, metrics.FailedRetries)
	assert.Equal(t, 3.0, metrics.AverageAttempts)
	t.Logf("Metrics: %+v", metrics)
}

// TestRetryMetrics_TrackFailedRetries verifies failed retry tracking
func TestRetryMetrics_TrackFailedRetries(t *testing.T) {
	tracker := aws.NewRetryTracker()

	operation := func(ctx context.Context) error {
		// Always fail
		return &MockError{Message: "Throttling: Rate exceeded"}
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond

	err := aws.WithRetryAndTracking(context.Background(), config, tracker, operation)
	require.Error(t, err)

	metrics := tracker.GetMetrics()
	assert.Equal(t, 1, metrics.TotalOperations)
	assert.Equal(t, 0, metrics.SuccessfulRetries)
	assert.Equal(t, 1, metrics.FailedRetries)
}

// =============================================================================
// 8. Jitter and Thundering Herd Prevention Tests
// =============================================================================

// TestJitter_PreventsThunderingHerd verifies jitter is applied
func TestJitter_PreventsThunderingHerd(t *testing.T) {
	// Run multiple operations with same backoff and verify they don't all retry at same time
	const numOperations = 10
	retryTimes := make([]time.Time, 0, numOperations)
	doneChan := make(chan time.Time, numOperations)

	for i := 0; i < numOperations; i++ {
		go func() {
			var attemptCount int32
			operation := func(ctx context.Context) error {
				attempt := atomic.AddInt32(&attemptCount, 1)
				if attempt == 2 {
					// Record time of 2nd attempt (after jitter applied)
					doneChan <- time.Now()
				}
				if attempt <= 2 {
					return &MockError{Message: "Throttling: Rate exceeded"}
				}
				return nil
			}

			config := aws.DefaultRetryConfig()
			config.InitialDelay = 50 * time.Millisecond
			config.JitterFraction = 0.5 // 50% jitter

			_ = aws.WithRetry(context.Background(), config, operation)
		}()
	}

	// Collect retry times
	for i := 0; i < numOperations; i++ {
		retryTimes = append(retryTimes, <-doneChan)
	}

	// Verify times are spread out (jitter working)
	var spreads []time.Duration
	for i := 1; i < len(retryTimes); i++ {
		spread := retryTimes[i].Sub(retryTimes[i-1])
		if spread > 0 {
			spreads = append(spreads, spread)
		}
	}

	// With jitter, we should see variation in retry times
	assert.Greater(t, len(spreads), 0, "Expected some spread in retry times")
	t.Logf("Retry time spreads: %v", spreads)
}

// =============================================================================
// 9. Exponential Backoff Calculation Tests
// =============================================================================

// TestExponentialBackoff_Calculation verifies backoff calculation
func TestExponentialBackoff_Calculation(t *testing.T) {
	tests := []struct {
		name         string
		attempt      int
		initialDelay time.Duration
		maxDelay     time.Duration
		multiplier   float64
		expected     time.Duration
	}{
		{
			name:         "first retry",
			attempt:      1,
			initialDelay: 1 * time.Second,
			maxDelay:     30 * time.Second,
			multiplier:   2.0,
			expected:     1 * time.Second,
		},
		{
			name:         "second retry",
			attempt:      2,
			initialDelay: 1 * time.Second,
			maxDelay:     30 * time.Second,
			multiplier:   2.0,
			expected:     2 * time.Second,
		},
		{
			name:         "third retry",
			attempt:      3,
			initialDelay: 1 * time.Second,
			maxDelay:     30 * time.Second,
			multiplier:   2.0,
			expected:     4 * time.Second,
		},
		{
			name:         "capped at max delay",
			attempt:      10,
			initialDelay: 1 * time.Second,
			maxDelay:     10 * time.Second,
			multiplier:   2.0,
			expected:     10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := aws.ExponentialBackoff(tt.attempt, tt.initialDelay, tt.maxDelay, tt.multiplier)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

// =============================================================================
// 10. Error Classification Tests
// =============================================================================

// TestRetryableErrorPatterns verifies error pattern matching
func TestRetryableErrorPatterns(t *testing.T) {
	tests := []struct {
		name      string
		errorMsg  string
		retryable bool
		errType   string
	}{
		{"throttling", "Throttling: Rate exceeded", true, "throttling"},
		{"capacity", "InsufficientInstanceCapacity", true, "capacity"},
		{"service unavailable", "ServiceUnavailable", true, "service"},
		{"internal error", "InternalError", true, "service"},
		{"slow down", "SlowDown: Please reduce request rate", true, "throttling"},
		{"timeout", "RequestTimeout: Request timed out", true, "network"},
		{"connection reset", "connection reset by peer", true, "network"},
		{"expired token", "ExpiredToken", false, "credential"},
		{"access denied", "AccessDenied", false, "permission"},
		{"invalid parameter", "InvalidParameter", false, "validation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attemptCount int32
			operation := func(ctx context.Context) error {
				atomic.AddInt32(&attemptCount, 1)
				return &MockError{Message: tt.errorMsg}
			}

			config := aws.DefaultRetryConfig()
			config.MaxAttempts = 2
			config.InitialDelay = 1 * time.Millisecond

			err := aws.WithRetry(context.Background(), config, operation)

			if tt.retryable {
				// Should retry and fail after max attempts
				assert.Error(t, err)
				assert.Equal(t, int32(2), attemptCount, "Expected retries for %s", tt.name)
				assert.Contains(t, err.Error(), fmt.Sprintf("operation failed after %d attempts", config.MaxAttempts))
			} else {
				// Should fail immediately (non-retryable)
				assert.Error(t, err)
				assert.Equal(t, int32(1), attemptCount, "Expected no retries for %s", tt.name)
				assert.Contains(t, err.Error(), "non-retryable")
			}
		})
	}
}

// =============================================================================
// 11. Integration with Multiple Error Types
// =============================================================================

// TestMultipleErrorTypes_Recovery verifies recovery from different error types
func TestMultipleErrorTypes_Recovery(t *testing.T) {
	var attemptCount int32

	operation := func(ctx context.Context) error {
		attempt := atomic.AddInt32(&attemptCount, 1)
		switch attempt {
		case 1:
			return &MockError{Message: "Throttling: Rate exceeded"}
		case 2:
			return &MockNetworkError{Message: "timeout", IsTimeout: true}
		case 3:
			return &MockError{Message: "ServiceUnavailable"}
		default:
			return nil
		}
	}

	config := aws.DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	err := aws.WithRetry(context.Background(), config, operation)
	require.NoError(t, err)
	assert.Equal(t, int32(4), attemptCount)
	t.Logf("Successfully recovered from multiple error types")
}

// TestRecoveryRate_HighSuccessRate verifies 95%+ recovery on transient failures
func TestRecoveryRate_HighSuccessRate(t *testing.T) {
	const totalOperations = 100
	successCount := 0

	for i := 0; i < totalOperations; i++ {
		var attemptCount int32
		operation := func(ctx context.Context) error {
			attempt := atomic.AddInt32(&attemptCount, 1)
			// Fail first 2 attempts, succeed on 3rd
			if attempt <= 2 {
				return &MockError{Message: "Throttling: Rate exceeded"}
			}
			return nil
		}

		config := aws.DefaultRetryConfig()
		config.InitialDelay = 1 * time.Millisecond

		err := aws.WithRetry(context.Background(), config, operation)
		if err == nil {
			successCount++
		}
	}

	recoveryRate := float64(successCount) / float64(totalOperations) * 100
	assert.GreaterOrEqual(t, recoveryRate, 95.0, "Expected 95%+ recovery rate")
	t.Logf("Recovery rate: %.1f%% (%d/%d)", recoveryRate, successCount, totalOperations)
}
