// Package ratelimit provides token bucket rate limiting for AWS API calls
//
// This implements a thread-safe token bucket algorithm to prevent
// AWS API throttling by limiting request rates. Key features:
//
// - Configurable rate (tokens per minute) and burst capacity
// - Thread-safe concurrent access with mutex protection
// - Automatic token refill based on elapsed time
// - Metrics tracking for monitoring and debugging
// - Context-aware operations with timeout support
//
// Example usage:
//
//	limiter := ratelimit.NewTokenBucket(10, 20) // 10/min, burst 20
//	if err := limiter.Allow(ctx); err != nil {
//	    // Rate limit exceeded, handle appropriately
//	}
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenBucket implements a thread-safe token bucket rate limiter
//
// The token bucket algorithm allows for burst capacity while maintaining
// a sustained rate limit over time. Tokens are refilled at a constant
// rate, and requests consume tokens. When tokens are exhausted, requests
// are rejected until tokens refill.
type TokenBucket struct {
	// Configuration
	rate  float64 // Tokens per minute
	burst int     // Maximum tokens (burst capacity)

	// State (protected by mutex)
	mu             sync.Mutex
	tokens         float64   // Current available tokens
	lastRefillTime time.Time // Last time tokens were refilled

	// Metrics
	totalRequests    int64
	allowedRequests  int64
	rateLimitedCount int64
	burstUtilization int64 // Cumulative burst usage
	successRate      float64
	metricsStartTime time.Time
}

// RateLimitError indicates a request was rejected due to rate limiting
type RateLimitError struct {
	Rate          float64
	Burst         int
	CurrentTokens float64
	RetryAfter    time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf(
		"rate limit exceeded: %d requests/minute (burst: %d), %.2f tokens available, retry after %v",
		int(e.Rate), e.Burst, e.CurrentTokens, e.RetryAfter.Round(time.Second),
	)
}

// NewTokenBucket creates a new token bucket rate limiter
//
// Parameters:
//   - rate: Maximum sustained requests per minute
//   - burst: Maximum burst capacity (initial tokens)
//
// The burst capacity allows for short-term spikes in traffic while
// maintaining the overall rate limit. For example:
//
//	rate=10, burst=20 allows 20 immediate requests, then 10/minute sustained
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	if rate <= 0 {
		rate = 10 // Default: 10 requests per minute
	}
	if burst <= 0 {
		burst = int(rate * 2) // Default: 2x rate
	}

	return &TokenBucket{
		rate:             rate,
		burst:            burst,
		tokens:           float64(burst), // Start with full burst capacity
		lastRefillTime:   time.Now(),
		metricsStartTime: time.Now(),
	}
}

// Allow checks if a request is allowed under the current rate limit
//
// This method is thread-safe and can be called concurrently from multiple
// goroutines. It automatically refills tokens based on elapsed time since
// the last refill.
//
// Returns:
//   - nil if the request is allowed
//   - RateLimitError if rate limit is exceeded
//   - context.DeadlineExceeded if context times out
//   - context.Canceled if context is canceled
func (tb *TokenBucket) Allow(ctx context.Context) error {
	// Check context first
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Update metrics
	tb.totalRequests++

	// Refill tokens based on elapsed time
	tb.refillTokens()

	// Check if we have tokens available
	if tb.tokens >= 1.0 {
		// Consume one token
		tb.tokens--
		tb.allowedRequests++
		tb.updateSuccessRate()

		// Track burst utilization (inverse of remaining tokens)
		burstUsage := float64(tb.burst) - tb.tokens
		if burstUsage > 0 {
			tb.burstUtilization += int64(burstUsage)
		}

		return nil
	}

	// Rate limit exceeded
	tb.rateLimitedCount++
	tb.updateSuccessRate()

	return tb.limitError()
}

// Check reports whether a request would be allowed right now, without consuming a token or
// counting toward the metrics. It is for previews such as a launch dry run, which must see the
// same verdict as Allow but must not use up capacity a real request needs.
func (tb *TokenBucket) Check(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refillTokens()
	if tb.tokens >= 1.0 {
		return nil
	}
	return tb.limitError()
}

// limitError builds the RateLimitError for the current token count.
// Must be called with mutex held
func (tb *TokenBucket) limitError() *RateLimitError {
	// Calculate retry after duration
	// Multiply by float64(time.Second) before converting to time.Duration
	// to preserve sub-second precision (time.Duration is in nanoseconds)
	tokensNeeded := 1.0 - tb.tokens
	secondsPerToken := 60.0 / tb.rate
	retryAfter := time.Duration(tokensNeeded * secondsPerToken * float64(time.Second))

	return &RateLimitError{
		Rate:          tb.rate,
		Burst:         tb.burst,
		CurrentTokens: tb.tokens,
		RetryAfter:    retryAfter,
	}
}

// refillTokens adds tokens based on elapsed time since last refill
// Must be called with mutex held
func (tb *TokenBucket) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime)

	// Calculate tokens to add based on elapsed time
	// rate is per minute, so convert elapsed to minutes
	tokensToAdd := tb.rate * elapsed.Minutes()

	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		tb.lastRefillTime = now

		// Cap at burst capacity (don't accumulate beyond burst)
		if tb.tokens > float64(tb.burst) {
			tb.tokens = float64(tb.burst)
		}
	}
}

// updateSuccessRate recalculates the success rate metric
// Must be called with mutex held
func (tb *TokenBucket) updateSuccessRate() {
	if tb.totalRequests > 0 {
		tb.successRate = float64(tb.allowedRequests) / float64(tb.totalRequests)
	}
}

// GetMetrics returns current rate limiter metrics
//
// Metrics include:
//   - Total requests attempted
//   - Requests allowed (successful)
//   - Requests rate limited (rejected)
//   - Success rate percentage
//   - Average burst utilization
//   - Current token count
//   - Uptime duration
func (tb *TokenBucket) GetMetrics() Metrics {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens for accurate current count
	tb.refillTokens()

	uptime := time.Since(tb.metricsStartTime)

	return Metrics{
		TotalRequests:    tb.totalRequests,
		AllowedRequests:  tb.allowedRequests,
		RateLimited:      tb.rateLimitedCount,
		SuccessRate:      tb.successRate,
		CurrentTokens:    tb.tokens,
		BurstUtilization: tb.burstUtilization,
		Rate:             tb.rate,
		Burst:            tb.burst,
		Uptime:           uptime,
	}
}

// ResetMetrics resets all metrics counters
//
// This preserves the rate limiter configuration and current token count,
// but clears historical metrics. Useful for testing or periodic resets.
func (tb *TokenBucket) ResetMetrics() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.totalRequests = 0
	tb.allowedRequests = 0
	tb.rateLimitedCount = 0
	tb.burstUtilization = 0
	tb.successRate = 0
	tb.metricsStartTime = time.Now()
}

// Metrics contains rate limiter performance metrics
type Metrics struct {
	TotalRequests    int64         // Total requests attempted
	AllowedRequests  int64         // Requests that were allowed
	RateLimited      int64         // Requests that were rate limited
	SuccessRate      float64       // Percentage of allowed requests (0.0-1.0)
	CurrentTokens    float64       // Current available tokens
	BurstUtilization int64         // Cumulative burst usage
	Rate             float64       // Configured rate (per minute)
	Burst            int           // Configured burst capacity
	Uptime           time.Duration // Time since metrics started
}

// String returns a human-readable representation of metrics
func (m Metrics) String() string {
	return fmt.Sprintf(
		"Rate Limiter Metrics:\n"+
			"  Configuration: %.0f req/min, burst %d\n"+
			"  Total Requests: %d\n"+
			"  Allowed: %d (%.1f%%)\n"+
			"  Rate Limited: %d\n"+
			"  Current Tokens: %.2f\n"+
			"  Burst Utilization: %d\n"+
			"  Uptime: %v",
		m.Rate, m.Burst,
		m.TotalRequests,
		m.AllowedRequests, m.SuccessRate*100,
		m.RateLimited,
		m.CurrentTokens,
		m.BurstUtilization,
		m.Uptime.Round(time.Second),
	)
}
