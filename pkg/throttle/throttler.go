package throttle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/scttfrdmn/prism/pkg/ratelimit"
)

// LaunchThrottler manages launch rate limiting across multiple scopes
type LaunchThrottler struct {
	config Config
	mu     sync.RWMutex

	// Token buckets for different scopes
	globalBucket   *ratelimit.TokenBucket
	userBuckets    map[string]*ratelimit.TokenBucket // key: userID
	projectBuckets map[string]*ratelimit.TokenBucket // key: projectID

	// Overrides
	overrides map[string]Override // key: projectID

	// Budget callback (optional)
	budgetUsageFunc func(projectID string) (used float64, limit float64, err error)
}

// NewLaunchThrottler creates a new launch throttler
func NewLaunchThrottler(config Config) *LaunchThrottler {
	lt := &LaunchThrottler{
		config:         config,
		userBuckets:    make(map[string]*ratelimit.TokenBucket),
		projectBuckets: make(map[string]*ratelimit.TokenBucket),
		overrides:      make(map[string]Override),
	}

	// Create global bucket if enabled
	if config.Enabled {
		rate, burst := lt.calculateRateAndBurst(config.MaxLaunches, config.TimeWindow, config.BurstSize)
		lt.globalBucket = ratelimit.NewTokenBucket(rate, burst)
	}

	return lt
}

// SetBudgetUsageFunc sets the callback for budget usage checking
func (lt *LaunchThrottler) SetBudgetUsageFunc(fn func(projectID string) (used float64, limit float64, err error)) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.budgetUsageFunc = fn
}

// AllowLaunch checks if a launch is allowed under current throttling rules
//
// This method checks all applicable throttling scopes (global, user, project)
// and returns an error if any scope rejects the launch.
func (lt *LaunchThrottler) AllowLaunch(ctx context.Context, req LaunchRequest) error {
	return lt.evaluate(ctx, req, (*ratelimit.TokenBucket).Allow)
}

// CheckLaunch reports whether AllowLaunch would admit req, without consuming any scope's
// capacity. A launch dry run uses it, so previewing a launch never throttles the real one.
func (lt *LaunchThrottler) CheckLaunch(ctx context.Context, req LaunchRequest) error {
	return lt.evaluate(ctx, req, (*ratelimit.TokenBucket).Check)
}

// evaluate runs req through every applicable scope, using admit (Allow or Check) per bucket.
func (lt *LaunchThrottler) evaluate(ctx context.Context, req LaunchRequest, admit func(*ratelimit.TokenBucket, context.Context) error) error {
	lt.mu.RLock()
	enabled := lt.config.Enabled
	lt.mu.RUnlock()

	if !enabled {
		return nil // Throttling disabled
	}

	// Check global throttle
	if lt.globalBucket != nil {
		if err := admit(lt.globalBucket, ctx); err != nil {
			return lt.formatThrottleError("global", err)
		}
	}

	// Check per-user throttle
	if lt.config.PerUser && req.UserID != "" {
		bucket := lt.getUserBucket(req.UserID)
		if err := admit(bucket, ctx); err != nil {
			return lt.formatThrottleError(fmt.Sprintf("user:%s", req.UserID), err)
		}
	}

	// Check per-project throttle (with budget awareness)
	if lt.config.PerProject && req.ProjectID != "" {
		bucket := lt.getProjectBucket(req.ProjectID)

		// Apply budget-aware throttling if enabled
		if lt.config.BudgetAware {
			lt.applyBudgetThrottling(req.ProjectID, bucket)
		}

		if err := admit(bucket, ctx); err != nil {
			return lt.formatThrottleError(fmt.Sprintf("project:%s", req.ProjectID), err)
		}
	}

	return nil
}

// GetStatus returns current throttling status for a scope
func (lt *LaunchThrottler) GetStatus(scope string) Status {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	status := Status{
		Scope:          scope,
		Enabled:        lt.config.Enabled,
		MaxLaunches:    lt.config.MaxLaunches,
		TimeWindow:     lt.config.TimeWindow,
		ConfiguredRate: lt.calculateRate(lt.config.MaxLaunches, lt.config.TimeWindow),
	}

	// Get bucket for scope
	var bucket *ratelimit.TokenBucket
	if scope == "global" {
		bucket = lt.globalBucket
	} else if len(scope) > 5 && scope[:5] == "user:" {
		userID := scope[5:]
		bucket = lt.userBuckets[userID]
	} else if len(scope) > 8 && scope[:8] == "project:" {
		projectID := scope[8:]
		bucket = lt.projectBuckets[projectID]
	}

	if bucket == nil {
		return status
	}

	// Get metrics from bucket
	metrics := bucket.GetMetrics()
	status.CurrentTokens = metrics.CurrentTokens
	status.LaunchesInWindow = int(metrics.AllowedRequests)
	status.TotalThrottled = metrics.RateLimited
	status.AllowedLaunches = metrics.AllowedRequests
	status.SuccessRate = metrics.SuccessRate
	status.EffectiveRate = metrics.Rate

	// Calculate next refill time
	if status.CurrentTokens < 1.0 {
		tokensNeeded := 1.0 - status.CurrentTokens
		secondsPerToken := 60.0 / metrics.Rate
		status.TimeUntilRefill = time.Duration(tokensNeeded*secondsPerToken) * time.Second
		status.NextTokenRefill = time.Now().Add(status.TimeUntilRefill)
	}

	return status
}

// UpdateConfig updates the throttling configuration
func (lt *LaunchThrottler) UpdateConfig(config Config) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.config = config

	// Recreate global bucket with new config
	if config.Enabled {
		rate, burst := lt.calculateRateAndBurst(config.MaxLaunches, config.TimeWindow, config.BurstSize)
		lt.globalBucket = ratelimit.NewTokenBucket(rate, burst)
	} else {
		lt.globalBucket = nil
	}

	// Clear existing buckets (they'll be recreated on next use with new config)
	lt.userBuckets = make(map[string]*ratelimit.TokenBucket)
	lt.projectBuckets = make(map[string]*ratelimit.TokenBucket)
}

// SetProjectOverride sets a project-specific throttling override
func (lt *LaunchThrottler) SetProjectOverride(override Override) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.overrides[override.ProjectID] = override

	// Recreate project bucket with override settings
	rate, burst := lt.calculateRateAndBurst(override.MaxLaunches, override.TimeWindow, lt.config.BurstSize)
	lt.projectBuckets[override.ProjectID] = ratelimit.NewTokenBucket(rate, burst)
}

// RemoveProjectOverride removes a project-specific override
func (lt *LaunchThrottler) RemoveProjectOverride(projectID string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	delete(lt.overrides, projectID)

	// Recreate project bucket with default config
	rate, burst := lt.calculateRateAndBurst(lt.config.MaxLaunches, lt.config.TimeWindow, lt.config.BurstSize)
	lt.projectBuckets[projectID] = ratelimit.NewTokenBucket(rate, burst)
}

// GetProjectOverride returns the override for a project, if any
func (lt *LaunchThrottler) GetProjectOverride(projectID string) (Override, bool) {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	override, exists := lt.overrides[projectID]
	return override, exists
}

// ListProjectOverrides returns all project overrides
func (lt *LaunchThrottler) ListProjectOverrides() []Override {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	overrides := make([]Override, 0, len(lt.overrides))
	for _, override := range lt.overrides {
		overrides = append(overrides, override)
	}
	return overrides
}

// getUserBucket gets or creates a user bucket
func (lt *LaunchThrottler) getUserBucket(userID string) *ratelimit.TokenBucket {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	bucket, exists := lt.userBuckets[userID]
	if !exists {
		rate, burst := lt.calculateRateAndBurst(lt.config.MaxLaunches, lt.config.TimeWindow, lt.config.BurstSize)
		bucket = ratelimit.NewTokenBucket(rate, burst)
		lt.userBuckets[userID] = bucket
	}
	return bucket
}

// getProjectBucket gets or creates a project bucket (respects overrides)
func (lt *LaunchThrottler) getProjectBucket(projectID string) *ratelimit.TokenBucket {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	bucket, exists := lt.projectBuckets[projectID]
	if !exists {
		// Check for override
		override, hasOverride := lt.overrides[projectID]
		var rate float64
		var burst int

		if hasOverride {
			rate, burst = lt.calculateRateAndBurst(override.MaxLaunches, override.TimeWindow, lt.config.BurstSize)
		} else {
			rate, burst = lt.calculateRateAndBurst(lt.config.MaxLaunches, lt.config.TimeWindow, lt.config.BurstSize)
		}

		bucket = ratelimit.NewTokenBucket(rate, burst)
		lt.projectBuckets[projectID] = bucket
	}
	return bucket
}

// applyBudgetThrottling adjusts rate based on budget usage
func (lt *LaunchThrottler) applyBudgetThrottling(projectID string, bucket *ratelimit.TokenBucket) {
	if lt.budgetUsageFunc == nil {
		return
	}

	used, limit, err := lt.budgetUsageFunc(projectID)
	if err != nil || limit == 0 {
		return // Can't determine budget, skip adjustment
	}

	// Calculate budget usage percentage
	usagePercent := used / limit
	if usagePercent < lt.config.BudgetThreshold {
		return // under threshold, no adjustment needed
	}

	// Over threshold: the bucket's rate should be reduced here. This is not yet
	// implemented — it requires extending TokenBucket to support dynamic rate
	// updates — so no adjustment is currently made.
}

// calculateRateAndBurst converts max launches and time window to rate and burst
func (lt *LaunchThrottler) calculateRateAndBurst(maxLaunches int, timeWindow string, burstSize int) (rate float64, burst int) {
	rate = lt.calculateRate(maxLaunches, timeWindow)

	if burstSize > 0 {
		burst = burstSize
	} else {
		burst = maxLaunches // Default: burst = max launches
	}

	return rate, burst
}

// calculateRate converts max launches and time window to rate per minute
func (lt *LaunchThrottler) calculateRate(maxLaunches int, timeWindow string) float64 {
	duration, err := time.ParseDuration(timeWindow)
	if err != nil {
		duration = 1 * time.Hour // Default to 1 hour
	}

	// Convert to rate per minute
	minutes := duration.Minutes()
	if minutes == 0 {
		minutes = 60 // Default to 1 hour
	}

	return float64(maxLaunches) / minutes
}

// formatThrottleError converts a rate limit error to a throttle error
func (lt *LaunchThrottler) formatThrottleError(scope string, err error) error {
	rateLimitErr, ok := err.(*ratelimit.RateLimitError)
	if !ok {
		return err // Not a rate limit error
	}

	message := fmt.Sprintf(
		"❌ Launch rate limit exceeded (%s: %d launches in %s)\n"+
			"   Current tokens: %.2f\n"+
			"   Available in: %v\n\n"+
			"💡 Check status: prism admin throttling status\n"+
			"💡 Contact admin for override if needed",
		scope,
		lt.config.MaxLaunches,
		lt.config.TimeWindow,
		rateLimitErr.CurrentTokens,
		rateLimitErr.RetryAfter.Round(time.Second),
	)

	return &ThrottleError{
		Scope:         scope,
		MaxLaunches:   lt.config.MaxLaunches,
		TimeWindow:    lt.config.TimeWindow,
		RetryAfter:    rateLimitErr.RetryAfter,
		CurrentTokens: rateLimitErr.CurrentTokens,
		Message:       message,
	}
}
