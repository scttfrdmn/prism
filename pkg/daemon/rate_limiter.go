// Package daemon provides rate limiting for instance launch operations
package daemon

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter provides token bucket-based rate limiting for instance launches
type RateLimiter struct {
	maxLaunches int           // Maximum launches per window
	window      time.Duration // Time window for rate limiting
	launches    []time.Time   // Timestamps of recent launches
	mutex       sync.Mutex    // Protects launches slice
	enabled     bool          // Whether rate limiting is enabled
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxLaunches int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxLaunches: maxLaunches,
		window:      window,
		launches:    make([]time.Time, 0),
		enabled:     true,
	}
}

// CheckAndRecordLaunch checks if a launch is allowed and records it if so
// Returns nil if allowed, error with retry time if rate limited
func (rl *RateLimiter) CheckAndRecordLaunch() error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// If disabled, always allow
	if !rl.enabled {
		return nil
	}

	now := time.Now()
	rl.pruneExpired(now)

	// Check if we've hit the limit
	if len(rl.launches) >= rl.maxLaunches {
		// Calculate when the oldest launch will expire
		oldestLaunch := rl.launches[0]
		retryAfter := rl.window - now.Sub(oldestLaunch)

		return &RateLimitError{
			Current:    len(rl.launches),
			Limit:      rl.maxLaunches,
			Window:     rl.window,
			RetryAfter: retryAfter,
		}
	}

	// Record this launch
	rl.launches = append(rl.launches, now)
	return nil
}

// CheckAndRecordLaunches admits a batch of n launches (e.g. a job array) as a
// unit and records n timestamps. Unlike CheckAndRecordLaunch, it is gated on the
// window NOT already being at the limit, but does not reject a batch merely for
// being larger than maxLaunches — an intentional array of N is not the accidental
// runaway the per-launch limiter guards against. If the window is already full,
// the whole batch is rejected (retry later); otherwise it is admitted and all n
// timestamps recorded so subsequent single launches see the array's footprint.
func (rl *RateLimiter) CheckAndRecordLaunches(n int) error {
	if n <= 1 {
		return rl.CheckAndRecordLaunch()
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if !rl.enabled {
		return nil
	}

	now := time.Now()
	rl.pruneExpired(now)

	// Reject only if the window is already saturated by prior launches.
	if len(rl.launches) >= rl.maxLaunches {
		oldestLaunch := rl.launches[0]
		return &RateLimitError{
			Current:    len(rl.launches),
			Limit:      rl.maxLaunches,
			Window:     rl.window,
			RetryAfter: rl.window - now.Sub(oldestLaunch),
		}
	}

	for i := 0; i < n; i++ {
		rl.launches = append(rl.launches, now)
	}
	return nil
}

// pruneExpired drops launch timestamps outside the sliding window. Caller holds the mutex.
func (rl *RateLimiter) pruneExpired(now time.Time) {
	cutoff := now.Add(-rl.window)
	validLaunches := make([]time.Time, 0, len(rl.launches))
	for _, launchTime := range rl.launches {
		if launchTime.After(cutoff) {
			validLaunches = append(validLaunches, launchTime)
		}
	}
	rl.launches = validLaunches
}

// GetStatus returns current rate limiter status
func (rl *RateLimiter) GetStatus() RateLimiterStatus {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Count valid launches in current window
	validCount := 0
	var oldestLaunch *time.Time
	for _, launchTime := range rl.launches {
		if launchTime.After(cutoff) {
			validCount++
			if oldestLaunch == nil || launchTime.Before(*oldestLaunch) {
				oldestLaunch = &launchTime
			}
		}
	}

	status := RateLimiterStatus{
		Enabled:     rl.enabled,
		MaxLaunches: rl.maxLaunches,
		Window:      rl.window,
		Current:     validCount,
		Remaining:   rl.maxLaunches - validCount,
	}

	// Calculate time until next slot available
	if validCount >= rl.maxLaunches && oldestLaunch != nil {
		status.ResetTime = oldestLaunch.Add(rl.window)
	}

	return status
}

// SetEnabled enables or disables rate limiting
func (rl *RateLimiter) SetEnabled(enabled bool) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.enabled = enabled
}

// UpdateConfig updates rate limiter configuration
func (rl *RateLimiter) UpdateConfig(maxLaunches int, window time.Duration) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.maxLaunches = maxLaunches
	rl.window = window
}

// RateLimiterStatus represents the current status of the rate limiter
type RateLimiterStatus struct {
	Enabled     bool          `json:"enabled"`
	MaxLaunches int           `json:"max_launches"`
	Window      time.Duration `json:"window"`
	Current     int           `json:"current"`
	Remaining   int           `json:"remaining"`
	ResetTime   time.Time     `json:"reset_time,omitempty"`
}

// RateLimitError is returned when a launch is rate limited
type RateLimitError struct {
	Current    int
	Limit      int
	Window     time.Duration
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	minutes := int(e.Window.Minutes())
	retrySeconds := int(e.RetryAfter.Seconds())

	return fmt.Sprintf(
		"Rate limit exceeded: %d launches in last %d minute(s) (limit: %d per %d min). Retry in %d second(s).",
		e.Current, minutes, e.Limit, minutes, retrySeconds,
	)
}
