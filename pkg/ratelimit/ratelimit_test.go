// Package ratelimit provides unit tests for the token bucket rate limiter.
// These tests verify core rate limiting logic without relying on real-time delays.
package ratelimit_test

import (
	"context"
	"sync"
	"testing"

	"github.com/scttfrdmn/prism/pkg/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTokenBucket_BurstCapacity verifies that burst capacity is honored:
// exactly burst-many requests succeed immediately, and the next fails.
func TestTokenBucket_BurstCapacity(t *testing.T) {
	limiter := ratelimit.NewTokenBucket(60, 5)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err := limiter.Allow(ctx)
		assert.NoError(t, err, "request %d should succeed within burst", i+1)
	}

	err := limiter.Allow(ctx)
	assert.Error(t, err, "6th request should fail (burst exhausted)")

	var rateLimitErr *ratelimit.RateLimitError
	require.ErrorAs(t, err, &rateLimitErr)
	assert.Equal(t, 60.0, rateLimitErr.Rate)
	assert.Equal(t, 5, rateLimitErr.Burst)
	assert.Greater(t, rateLimitErr.RetryAfter.Nanoseconds(), int64(0))
}

// TestTokenBucket_Concurrent verifies thread safety: with burst=10 and 50 goroutines,
// exactly 10 requests succeed and 40 are rate-limited.
func TestTokenBucket_Concurrent(t *testing.T) {
	limiter := ratelimit.NewTokenBucket(10, 10)
	ctx := context.Background()

	const n = 50
	var wg sync.WaitGroup
	results := make([]error, n)

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = limiter.Allow(ctx)
		}(i)
	}
	wg.Wait()

	successes, failures := 0, 0
	for _, err := range results {
		if err == nil {
			successes++
		} else {
			failures++
			var rlErr *ratelimit.RateLimitError
			assert.ErrorAs(t, err, &rlErr)
		}
	}

	assert.Equal(t, 10, successes, "exactly burst=10 requests should succeed")
	assert.Equal(t, 40, failures, "remaining 40 should be rate-limited")
}

// TestTokenBucket_Metrics verifies that metrics are tracked correctly.
func TestTokenBucket_Metrics(t *testing.T) {
	limiter := ratelimit.NewTokenBucket(60, 3)
	ctx := context.Background()

	// 3 succeed, 2 fail
	for i := 0; i < 3; i++ {
		_ = limiter.Allow(ctx)
	}
	for i := 0; i < 2; i++ {
		_ = limiter.Allow(ctx)
	}

	m := limiter.GetMetrics()
	assert.Equal(t, int64(5), m.TotalRequests)
	assert.Equal(t, int64(3), m.AllowedRequests)
	assert.Equal(t, int64(2), m.RateLimited)
	assert.InDelta(t, 0.6, m.SuccessRate, 0.01)
	assert.Equal(t, 60.0, m.Rate)
	assert.Equal(t, 3, m.Burst)

	// Verify reset works
	limiter.ResetMetrics()
	m2 := limiter.GetMetrics()
	assert.Equal(t, int64(0), m2.TotalRequests)
	assert.Equal(t, int64(0), m2.AllowedRequests)
	assert.Equal(t, int64(0), m2.RateLimited)
}

// TestTokenBucket_MetricsString verifies the String() method is readable.
func TestTokenBucket_MetricsString(t *testing.T) {
	limiter := ratelimit.NewTokenBucket(60, 10)
	m := limiter.GetMetrics()
	s := m.String()
	assert.Contains(t, s, "Rate Limiter Metrics")
	assert.Contains(t, s, "60 req/min")
	assert.Contains(t, s, "burst 10")
}

// TestTokenBucket_ContextCancellation verifies Allow respects context cancellation.
func TestTokenBucket_ContextCancellation(t *testing.T) {
	limiter := ratelimit.NewTokenBucket(1, 1)
	ctx := context.Background()

	// Exhaust burst
	_ = limiter.Allow(ctx)

	// Cancelled context should return an error
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	err := limiter.Allow(cancelled)
	assert.Error(t, err)
}

// TestTokenBucket_CheckDoesNotConsume: Check gives Allow's verdict without using a token, so a
// preview (a launch dry run) never takes capacity from the real request.
func TestTokenBucket_CheckDoesNotConsume(t *testing.T) {
	ctx := context.Background()
	tb := ratelimit.NewTokenBucket(1, 1) // one token, ~1/min refill

	for i := 0; i < 5; i++ {
		require.NoError(t, tb.Check(ctx), "Check #%d should pass while a token is available", i+1)
	}
	assert.Equal(t, int64(0), tb.GetMetrics().TotalRequests, "Check must not count as a request")

	require.NoError(t, tb.Allow(ctx), "the token must still be there for the real request")
	assert.Error(t, tb.Check(ctx), "once the token is spent, Check reports the limit like Allow")
}
