package throttle

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLaunchThrottler(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true

	throttler := NewLaunchThrottler(config)

	assert.NotNil(t, throttler)
	assert.NotNil(t, throttler.globalBucket)
	assert.Equal(t, config, throttler.config)
}

func TestAllowLaunch_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false

	throttler := NewLaunchThrottler(config)
	ctx := context.Background()

	req := LaunchRequest{
		UserID:    "user1",
		ProjectID: "proj1",
	}

	// Should allow all launches when disabled
	for i := 0; i < 100; i++ {
		err := throttler.AllowLaunch(ctx, req)
		assert.NoError(t, err, "Launch %d should succeed (throttling disabled)", i+1)
	}
}

func TestAllowLaunch_GlobalThrottle(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.MaxLaunches = 3
	config.TimeWindow = "1m"
	config.PerUser = false
	config.PerProject = false

	throttler := NewLaunchThrottler(config)
	ctx := context.Background()

	req := LaunchRequest{
		UserID:    "user1",
		ProjectID: "proj1",
	}

	// First 3 launches should succeed
	for i := 0; i < 3; i++ {
		err := throttler.AllowLaunch(ctx, req)
		assert.NoError(t, err, "Launch %d should succeed", i+1)
	}

	// 4th launch should fail
	err := throttler.AllowLaunch(ctx, req)
	require.Error(t, err)

	throttleErr, ok := err.(*ThrottleError)
	require.True(t, ok, "Error should be ThrottleError")
	assert.Equal(t, "global", throttleErr.Scope)
	assert.Equal(t, 3, throttleErr.MaxLaunches)
	assert.Greater(t, throttleErr.RetryAfter, time.Duration(0))
}

func TestAllowLaunch_PerUserThrottle(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.MaxLaunches = 100 // High global limit so per-user limits are what matters
	config.TimeWindow = "1m"
	config.PerUser = true
	config.PerProject = false

	throttler := NewLaunchThrottler(config)

	// Override the user bucket configuration to use 2 launches per user
	throttler.mu.Lock()
	throttler.config.MaxLaunches = 2 // This will apply to new user buckets
	throttler.mu.Unlock()

	ctx := context.Background()

	// User1 launches 2 instances
	user1Req := LaunchRequest{UserID: "user1"}
	for i := 0; i < 2; i++ {
		err := throttler.AllowLaunch(ctx, user1Req)
		assert.NoError(t, err, "User1 launch %d should succeed", i+1)
	}

	// User1 3rd launch should fail
	err := throttler.AllowLaunch(ctx, user1Req)
	require.Error(t, err)
	throttleErr, _ := err.(*ThrottleError)
	assert.Contains(t, throttleErr.Scope, "user:user1")

	// User2 should still be able to launch 2 times (separate bucket)
	user2Req := LaunchRequest{UserID: "user2"}
	for i := 0; i < 2; i++ {
		err := throttler.AllowLaunch(ctx, user2Req)
		assert.NoError(t, err, "User2 launch %d should succeed", i+1)
	}
}

func TestAllowLaunch_PerProjectThrottle(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.MaxLaunches = 100 // High global limit so per-project limits are what matters
	config.TimeWindow = "1m"
	config.PerUser = false
	config.PerProject = true

	throttler := NewLaunchThrottler(config)

	// Override the project bucket configuration to use 2 launches per project
	throttler.mu.Lock()
	throttler.config.MaxLaunches = 2 // This will apply to new project buckets
	throttler.mu.Unlock()

	ctx := context.Background()

	// Project1 launches 2 instances
	proj1Req := LaunchRequest{ProjectID: "proj1"}
	for i := 0; i < 2; i++ {
		err := throttler.AllowLaunch(ctx, proj1Req)
		assert.NoError(t, err, "Project1 launch %d should succeed", i+1)
	}

	// Project1 3rd launch should fail
	err := throttler.AllowLaunch(ctx, proj1Req)
	require.Error(t, err)
	throttleErr, _ := err.(*ThrottleError)
	assert.Contains(t, throttleErr.Scope, "project:proj1")

	// Project2 should still be able to launch 2 times (separate bucket)
	proj2Req := LaunchRequest{ProjectID: "proj2"}
	for i := 0; i < 2; i++ {
		err := throttler.AllowLaunch(ctx, proj2Req)
		assert.NoError(t, err, "Project2 launch %d should succeed", i+1)
	}
}

func TestAllowLaunch_CombinedThrottles(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.MaxLaunches = 5
	config.TimeWindow = "1m"
	config.PerUser = true
	config.PerProject = true

	throttler := NewLaunchThrottler(config)
	ctx := context.Background()

	req := LaunchRequest{
		UserID:    "user1",
		ProjectID: "proj1",
	}

	// First 5 launches should succeed (all limits at 5)
	for i := 0; i < 5; i++ {
		err := throttler.AllowLaunch(ctx, req)
		assert.NoError(t, err, "Launch %d should succeed", i+1)
	}

	// 6th launch should fail (global, user, and project all exhausted)
	err := throttler.AllowLaunch(ctx, req)
	require.Error(t, err)
}

func TestGetStatus(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.MaxLaunches = 10
	config.TimeWindow = "1h"

	throttler := NewLaunchThrottler(config)
	ctx := context.Background()

	// Launch 3 instances
	req := LaunchRequest{UserID: "user1"}
	for i := 0; i < 3; i++ {
		_ = throttler.AllowLaunch(ctx, req)
	}

	// Get global status
	status := throttler.GetStatus("global")
	assert.Equal(t, "global", status.Scope)
	assert.True(t, status.Enabled)
	assert.Equal(t, 10, status.MaxLaunches)
	assert.Equal(t, "1h", status.TimeWindow)
	assert.Equal(t, 3, status.LaunchesInWindow)
	assert.InDelta(t, 7.0, status.CurrentTokens, 0.1) // ~7 tokens remaining

	// Get user status
	userStatus := throttler.GetStatus("user:user1")
	assert.Equal(t, "user:user1", userStatus.Scope)
	assert.Equal(t, 3, userStatus.LaunchesInWindow)
}

func TestUpdateConfig(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.MaxLaunches = 5

	throttler := NewLaunchThrottler(config)
	ctx := context.Background()

	// Launch 5 instances
	req := LaunchRequest{}
	for i := 0; i < 5; i++ {
		_ = throttler.AllowLaunch(ctx, req)
	}

	// Should fail (limit reached)
	err := throttler.AllowLaunch(ctx, req)
	require.Error(t, err)

	// Update config to 10 launches
	newConfig := config
	newConfig.MaxLaunches = 10
	throttler.UpdateConfig(newConfig)

	// Should now succeed (new bucket with fresh tokens)
	for i := 0; i < 5; i++ {
		err := throttler.AllowLaunch(ctx, req)
		assert.NoError(t, err, "Launch after config update should succeed")
	}
}

func TestProjectOverride(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.MaxLaunches = 100 // Very high global limit so project limits are what matters
	config.TimeWindow = "1m"
	config.PerProject = true
	config.PerUser = false // Disable user throttling for clarity

	throttler := NewLaunchThrottler(config)

	// Override the project bucket configuration after creation
	throttler.mu.Lock()
	throttler.config.MaxLaunches = 20 // Default for new project buckets
	throttler.mu.Unlock()

	ctx := context.Background()

	// Set override for proj1 (10 launches instead of 20)
	override := Override{
		ProjectID:   "proj1",
		MaxLaunches: 10,
		TimeWindow:  "1m",
		CreatedBy:   "admin",
		CreatedAt:   time.Now(),
		Reason:      "High-priority research project",
	}
	throttler.SetProjectOverride(override)

	// proj1 should allow 10 launches (limited by override, not default 20)
	proj1Req := LaunchRequest{ProjectID: "proj1"}
	for i := 0; i < 10; i++ {
		err := throttler.AllowLaunch(ctx, proj1Req)
		assert.NoError(t, err, "proj1 launch %d should succeed (override)", i+1)
	}

	// proj1 11th launch should fail (override limit reached)
	err := throttler.AllowLaunch(ctx, proj1Req)
	require.Error(t, err, "proj1 11th launch should fail (override limit)")

	// proj2 should allow 20 launches (default config limit)
	proj2Req := LaunchRequest{ProjectID: "proj2"}
	for i := 0; i < 20; i++ {
		err := throttler.AllowLaunch(ctx, proj2Req)
		assert.NoError(t, err, "proj2 launch %d should succeed", i+1)
	}
	err = throttler.AllowLaunch(ctx, proj2Req)
	require.Error(t, err, "proj2 21st launch should fail")

	// Verify override exists
	retrievedOverride, exists := throttler.GetProjectOverride("proj1")
	assert.True(t, exists)
	assert.Equal(t, override.ProjectID, retrievedOverride.ProjectID)
	assert.Equal(t, override.MaxLaunches, retrievedOverride.MaxLaunches)

	// Remove override
	throttler.RemoveProjectOverride("proj1")
	_, exists = throttler.GetProjectOverride("proj1")
	assert.False(t, exists)
}

func TestListProjectOverrides(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true

	throttler := NewLaunchThrottler(config)

	// Add 3 overrides
	override1 := Override{ProjectID: "proj1", MaxLaunches: 10, TimeWindow: "1h"}
	override2 := Override{ProjectID: "proj2", MaxLaunches: 20, TimeWindow: "1h"}
	override3 := Override{ProjectID: "proj3", MaxLaunches: 5, TimeWindow: "30m"}

	throttler.SetProjectOverride(override1)
	throttler.SetProjectOverride(override2)
	throttler.SetProjectOverride(override3)

	// List all overrides
	overrides := throttler.ListProjectOverrides()
	assert.Len(t, overrides, 3)

	// Verify all overrides present
	projectIDs := make(map[string]bool)
	for _, o := range overrides {
		projectIDs[o.ProjectID] = true
	}
	assert.True(t, projectIDs["proj1"])
	assert.True(t, projectIDs["proj2"])
	assert.True(t, projectIDs["proj3"])
}

func TestConcurrentLaunches(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.MaxLaunches = 10
	config.TimeWindow = "1m"
	config.PerUser = false
	config.PerProject = false

	throttler := NewLaunchThrottler(config)
	ctx := context.Background()

	const numGoroutines = 50
	var wg sync.WaitGroup
	results := make([]error, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			req := LaunchRequest{UserID: "user1"}
			results[index] = throttler.AllowLaunch(ctx, req)
		}(i)
	}

	wg.Wait()

	// Count successes and failures
	successes := 0
	failures := 0
	for _, err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	// Exactly 10 should succeed (burst capacity)
	assert.Equal(t, 10, successes, "Should have exactly 10 successful launches")
	assert.Equal(t, 40, failures, "Should have exactly 40 throttled launches")
}

func TestCalculateRate(t *testing.T) {
	throttler := NewLaunchThrottler(DefaultConfig())

	tests := []struct {
		name         string
		maxLaunches  int
		timeWindow   string
		expectedRate float64
	}{
		{"10 per hour", 10, "1h", 10.0 / 60.0},
		{"20 per hour", 20, "1h", 20.0 / 60.0},
		{"5 per 30min", 5, "30m", 5.0 / 30.0},
		{"100 per day", 100, "24h", 100.0 / (24 * 60)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := throttler.calculateRate(tt.maxLaunches, tt.timeWindow)
			assert.InDelta(t, tt.expectedRate, rate, 0.001)
		})
	}
}

func TestThrottleError_Error(t *testing.T) {
	err := &ThrottleError{
		Scope:       "global",
		MaxLaunches: 10,
		TimeWindow:  "1h",
		RetryAfter:  30 * time.Minute,
		Message:     "Custom error message",
	}

	assert.Equal(t, "Custom error message", err.Error())
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.False(t, config.Enabled, "Should be disabled by default")
	assert.Equal(t, 10, config.MaxLaunches)
	assert.Equal(t, "1h", config.TimeWindow)
	assert.True(t, config.PerUser)
	assert.True(t, config.PerProject)
	assert.True(t, config.BudgetAware)
	assert.Equal(t, 0.80, config.BudgetThreshold)
	assert.Equal(t, "reject", config.QueueMode)
}

// TestCheckLaunch_DoesNotConsume: CheckLaunch gives AllowLaunch's verdict without using any
// scope's capacity (#705 follow-up: dry runs used to throttle the real launch).
func TestCheckLaunch_DoesNotConsume(t *testing.T) {
	ctx := context.Background()
	lt := NewLaunchThrottler(Config{Enabled: true, MaxLaunches: 1, TimeWindow: "1h", BurstSize: 1, PerUser: true})
	req := LaunchRequest{UserID: "alice", TemplateName: "python-ml"}

	for i := 0; i < 5; i++ {
		require.NoError(t, lt.CheckLaunch(ctx, req), "dry run #%d", i+1)
	}
	require.NoError(t, lt.AllowLaunch(ctx, req), "the real launch must still be admitted")
	assert.Error(t, lt.CheckLaunch(ctx, req), "after the real launch, a preview reports the throttle")
}
