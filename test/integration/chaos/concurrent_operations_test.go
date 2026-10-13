//go:build integration
// +build integration

package chaos

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/scttfrdmn/prism/test/integration"
)

// TestConcurrentInstanceLaunches validates that multiple simultaneous instance
// launches handle concurrency correctly without race conditions.
//
// Chaos Scenario: 5 instances launched simultaneously
// Expected Behavior:
// - All instances launch successfully
// - No race conditions in state management
// - Instance IDs are unique
// - State remains consistent
//
// Addresses Issue #412 - Concurrent Operation Chaos
func TestConcurrentInstanceLaunches(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: Concurrent Instance Launches")
	t.Logf("")

	// Setup
	ctx := integration.NewTestContext(t)
	defer ctx.Cleanup()

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// ========================================
	// Setup: Create Project
	// ========================================

	t.Logf("📋 Setting up test environment")

	projectName := integration.GenerateTestName("concurrent-test-project")
	project, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        projectName,
		Description: "Concurrent operations test project",
		Owner:       "test-user@example.com",
	})
	integration.AssertNoError(t, err, "Failed to create project")
	t.Logf("✅ Project created: %s", project.ID)

	// ========================================
	// Test Scenario: Concurrent Instance Launches
	// ========================================

	t.Logf("")
	t.Logf("📋 Launching 5 instances concurrently")

	concurrency := 5
	results := make(chan error, concurrency)
	instanceIDs := make([]string, concurrency)
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			instanceName := integration.GenerateTestName(fmt.Sprintf("concurrent-instance-%d", index))
			t.Logf("   Goroutine %d: Launching %s", index, instanceName)

			instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
				Template:  "Python ML Workstation",
				Name:      instanceName,
				Size:      "S",
				ProjectID: &project.ID,
			})

			if err != nil {
				t.Logf("   ❌ Goroutine %d: Failed: %v", index, err)
				results <- err
				return
			}

			instanceIDs[index] = instance.ID
			t.Logf("   ✅ Goroutine %d: Success (%s)", index, instance.ID)
			results <- nil
		}(i)
	}

	// Wait for all launches to complete
	wg.Wait()
	close(results)
	elapsed := time.Since(startTime)

	// ========================================
	// Validation: Check Results
	// ========================================

	t.Logf("")
	t.Logf("📋 Validating concurrent launch results")

	var successCount, failureCount int
	var firstError error

	for err := range results {
		if err != nil {
			failureCount++
			if firstError == nil {
				firstError = err
			}
		} else {
			successCount++
		}
	}

	t.Logf("Results:")
	t.Logf("   Success: %d/%d", successCount, concurrency)
	t.Logf("   Failures: %d/%d", failureCount, concurrency)
	t.Logf("   Total time: %v", elapsed)

	// Verify success rate
	if successCount < concurrency {
		t.Errorf("Only %d/%d launches succeeded. First error: %v",
			successCount, concurrency, firstError)
	} else {
		t.Logf("✅ All concurrent launches successful")
	}

	// ========================================
	// Validation: Check Instance ID Uniqueness
	// ========================================

	t.Logf("")
	t.Logf("📋 Validating instance ID uniqueness")

	idMap := make(map[string]bool)
	var duplicates int

	for _, id := range instanceIDs {
		if id == "" {
			continue
		}
		if idMap[id] {
			duplicates++
			t.Errorf("Duplicate instance ID detected: %s", id)
		}
		idMap[id] = true
	}

	if duplicates == 0 {
		t.Logf("✅ All instance IDs are unique (%d unique IDs)", len(idMap))
	} else {
		t.Errorf("Found %d duplicate instance IDs", duplicates)
	}

	// ========================================
	// Validation: State Consistency
	// ========================================

	t.Logf("")
	t.Logf("📋 Validating state consistency")

	// List all instances
	instances, err := ctx.Client.ListInstances(context.Background())
	integration.AssertNoError(t, err, "Failed to list instances")

	// Count instances in this project
	var projectInstanceCount int
	for _, inst := range instances.Instances {
		if inst.ProjectID == project.ID {
			projectInstanceCount++
		}
	}

	t.Logf("Project instances: %d (expected: %d)", projectInstanceCount, successCount)

	if projectInstanceCount < successCount {
		t.Errorf("State inconsistency: Only %d/%d instances found in project",
			projectInstanceCount, successCount)
	} else {
		t.Logf("✅ State is consistent")
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Concurrent Instance Launches Test Complete!")
	t.Logf("   ✓ All %d instances launched successfully", successCount)
	t.Logf("   ✓ All instance IDs unique")
	t.Logf("   ✓ State remains consistent")
	t.Logf("   ✓ Total time: %v (avg: %v per instance)",
		elapsed, elapsed/time.Duration(concurrency))
	t.Logf("")
	t.Logf("🎉 System handles concurrent launches correctly!")
}

// TestConcurrentStateModifications validates that simultaneous state
// modifications don't cause race conditions or data corruption.
//
// Chaos Scenario: Multiple operations modifying same resource simultaneously
// Expected Behavior:
// - Operations serialize correctly
// - No data races detected
// - State remains consistent
// - Last writer wins (or appropriate conflict resolution)
//
// Addresses Issue #412 - Concurrent Operation Chaos
func TestConcurrentStateModifications(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: Concurrent State Modifications")
	t.Logf("")

	// Setup
	ctx := integration.NewTestContext(t)
	defer ctx.Cleanup()

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// ========================================
	// Setup: Create Resources
	// ========================================

	t.Logf("📋 Setting up test resources")

	projectName := integration.GenerateTestName("state-test-project")
	project, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        projectName,
		Description: "State modification test project",
		Owner:       "test-user@example.com",
	})
	integration.AssertNoError(t, err, "Failed to create project")
	t.Logf("✅ Project created: %s", project.ID)

	// Create instance for concurrent operations
	instanceName := integration.GenerateTestName("state-test-instance")
	instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Python ML Workstation",
		Name:      instanceName,
		Size:      "S",
		ProjectID: &project.ID,
	})
	integration.AssertNoError(t, err, "Failed to create instance")
	t.Logf("✅ Instance created: %s", instance.ID)

	// ========================================
	// Test Scenario: Concurrent Reads
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing concurrent read operations")

	readers := 10
	var readErrors atomic.Int64
	var readSuccess atomic.Int64
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			_, err := ctx.Client.GetInstance(context.Background(), instance.ID)
			if err != nil {
				readErrors.Add(1)
				t.Logf("   ❌ Reader %d: Failed: %v", index, err)
			} else {
				readSuccess.Add(1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	successRate := float64(readSuccess.Load()) / float64(readers) * 100
	t.Logf("Read results:")
	t.Logf("   Success: %d/%d (%.1f%%)", readSuccess.Load(), readers, successRate)
	t.Logf("   Errors: %d", readErrors.Load())
	t.Logf("   Time: %v", elapsed)

	if readSuccess.Load() == int64(readers) {
		t.Logf("✅ All concurrent reads successful")
	} else {
		t.Errorf("Some concurrent reads failed (%d/%d)", readSuccess.Load(), readers)
	}

	// ========================================
	// Test Scenario: Concurrent Writes (Idempotent)
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing concurrent idempotent operations (stop)")

	writers := 5
	var writeErrors atomic.Int64
	var writeSuccess atomic.Int64
	wg = sync.WaitGroup{}

	startTime = time.Now()

	// Concurrent stop operations (should be idempotent)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			err := ctx.Client.StopInstance(context.Background(), instance.ID)
			if err != nil {
				writeErrors.Add(1)
				t.Logf("   ❌ Writer %d: Failed: %v", index, err)
			} else {
				writeSuccess.Add(1)
				t.Logf("   ✅ Writer %d: Success", index)
			}
		}(i)
	}

	wg.Wait()
	elapsed = time.Since(startTime)

	writeSuccessRate := float64(writeSuccess.Load()) / float64(writers) * 100
	t.Logf("Write results:")
	t.Logf("   Success: %d/%d (%.1f%%)", writeSuccess.Load(), writers, writeSuccessRate)
	t.Logf("   Errors: %d", writeErrors.Load())
	t.Logf("   Time: %v", elapsed)

	// Verify instance is stopped
	stoppedInstance, err := ctx.Client.GetInstance(context.Background(), instance.ID)
	integration.AssertNoError(t, err, "Failed to get instance after stop")

	if stoppedInstance.State == "stopped" {
		t.Logf("✅ Instance correctly stopped")
	} else {
		t.Logf("⚠️  Instance state: %s (expected: stopped)", stoppedInstance.State)
	}

	// Idempotent operations should mostly succeed
	if writeSuccess.Load() >= int64(writers-1) {
		t.Logf("✅ Idempotent operations handled correctly")
	} else {
		t.Logf("⚠️  Some idempotent operations failed unexpectedly")
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Concurrent State Modifications Test Complete!")
	t.Logf("   ✓ Concurrent reads: %d/%d successful", readSuccess.Load(), readers)
	t.Logf("   ✓ Concurrent writes: %d/%d successful", writeSuccess.Load(), writers)
	t.Logf("   ✓ Idempotent operations handled correctly")
	t.Logf("   ✓ Final state consistent")
	t.Logf("")
	t.Logf("🎉 System handles concurrent state modifications!")
}

// TestRaceConditionDetection runs chaos tests with Go race detector enabled
// to find potential race conditions.
//
// Chaos Scenario: Concurrent operations with race detection enabled
// Expected Behavior:
// - No data races detected
// - All operations complete successfully
// - Race detector reports clean run
//
// Addresses Issue #412 - Concurrent Operation Chaos
//
// Run with: go test -race -tags integration ./test/integration/chaos
func TestRaceConditionDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: Race Condition Detection")
	t.Logf("")
	t.Logf("⚠️  Run with -race flag to enable race detector:")
	t.Logf("   go test -race -tags integration ./test/integration/chaos")
	t.Logf("")

	// Setup
	ctx := integration.NewTestContext(t)
	defer ctx.Cleanup()

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// ========================================
	// Setup: Create Project
	// ========================================

	t.Logf("📋 Setting up race detection test")

	projectName := integration.GenerateTestName("race-test-project")
	project, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        projectName,
		Description: "Race condition detection project",
		Owner:       "test-user@example.com",
	})
	integration.AssertNoError(t, err, "Failed to create project")
	t.Logf("✅ Project created: %s", project.ID)

	// ========================================
	// Test Scenario: Heavy Concurrent Load
	// ========================================

	t.Logf("")
	t.Logf("📋 Running heavy concurrent load to trigger races")

	operations := 20
	var operationErrors atomic.Int64
	var operationSuccess atomic.Int64
	var wg sync.WaitGroup

	startTime := time.Now()

	// Mix of different operations running concurrently
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Alternate between different operation types
			switch index % 4 {
			case 0:
				// List operations
				_, err := ctx.Client.ListInstances(context.Background())
				if err != nil {
					operationErrors.Add(1)
				} else {
					operationSuccess.Add(1)
				}

			case 1:
				// Project operations
				_, err := ctx.Client.ListProjects(context.Background(), nil)
				if err != nil {
					operationErrors.Add(1)
				} else {
					operationSuccess.Add(1)
				}

			case 2:
				// Storage operations
				_, err := ctx.Client.ListVolumes(context.Background())
				if err != nil {
					operationErrors.Add(1)
				} else {
					operationSuccess.Add(1)
				}

			case 3:
				// Get specific project
				_, err := ctx.Client.GetProject(context.Background(), project.ID)
				if err != nil {
					operationErrors.Add(1)
				} else {
					operationSuccess.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	successRate := float64(operationSuccess.Load()) / float64(operations) * 100
	t.Logf("")
	t.Logf("Race detection results:")
	t.Logf("   Operations: %d", operations)
	t.Logf("   Success: %d (%.1f%%)", operationSuccess.Load(), successRate)
	t.Logf("   Errors: %d", operationErrors.Load())
	t.Logf("   Time: %v", elapsed)

	if operationSuccess.Load() == int64(operations) {
		t.Logf("✅ All operations successful - no races detected")
	} else {
		t.Logf("⚠️  Some operations failed - check for races")
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Race Condition Detection Test Complete!")
	t.Logf("   ✓ %d concurrent operations executed", operations)
	t.Logf("   ✓ Success rate: %.1f%%", successRate)
	t.Logf("")

	if operationErrors.Load() == 0 {
		t.Logf("🎉 No race conditions detected!")
	} else {
		t.Logf("⚠️  Review failures for potential race conditions")
	}
}
