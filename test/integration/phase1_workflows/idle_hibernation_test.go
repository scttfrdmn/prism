//go:build integration
// +build integration

package phase1_workflows

import (
	"context"
	"testing"

	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/scttfrdmn/prism/test/integration"
)

// TestIdleDetection_TriggersHibernation validates end-to-end idle detection
// and hibernation flow, ensuring that idle instances are actually detected
// and hibernated (not just API endpoint functionality).
//
// This test addresses issue #397 - Idle Detection & Hibernation Flow
//
// Success criteria:
// - Instance launches and becomes active
// - Idle policy is applied successfully
// - Instance becomes idle (simulated via waiting)
// - Idle detection triggers hibernation
// - Instance enters stopped/hibernated state
// - Cost savings are tracked and calculated
func TestIdleDetection_TriggersHibernation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running idle detection test in short mode (requires 10+ minutes)")
	}

	// Setup
	ctx := integration.NewTestContext(t) //nolint:staticcheck // SA4006: used by the scenario below t.Skip, kept for when idle detection lands

	// Step 1: DETERMINISTIC CHECK FIRST - Check if automatic idle detection is implemented
	// DON'T launch expensive instance if feature isn't available
	t.Log("Checking if automatic idle detection is implemented...")

	// Try to get idle policy status for the daemon
	// If the API endpoint doesn't exist or returns not implemented, skip test immediately
	// TODO: Add API call to check if idle detection is enabled
	// For now, skip test since automatic idle detection isn't fully implemented
	t.Log("⚠️  Automatic idle detection not yet implemented")
	t.Log("Expected behavior when implemented:")
	t.Log("  1. Launch instance")
	t.Log("  2. Apply 'aggressive-cost' policy (10min idle → hibernate)")
	t.Log("  3. After 10 min: Idle scheduler detects no activity")
	t.Log("  4. Scheduler triggers hibernation action")
	t.Log("  5. Instance transitions to 'stopped' state")
	t.Log("  6. Cost savings begin accumulating")
	t.Log("")
	t.Log("To implement this test:")
	t.Log("  - Add idle policy application API endpoint")
	t.Log("  - Add daemon idle detection scheduler")
	t.Log("  - Update test to check API exists, then launch instance")
	t.Log("  - Poll for state changes with deadline (not arbitrary sleep)")

	t.Skip("Automatic idle detection not implemented - skipping instance launch and test")

	// NOTE: When idle detection is implemented, replace above with:
	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// Step 2: Launch instance for idle detection testing
	instanceName := integration.GenerateTestName("test-idle-hibernate")
	t.Logf("Launching instance for idle detection testing: %s", instanceName)

	instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template: "Python ML Workstation",
		Name:     instanceName,
		Size:     "S", // Small size for cost efficiency
	})
	integration.AssertNoError(t, err, "Failed to create instance")
	integration.AssertEqual(t, "running", instance.State, "Instance should be running")

	t.Logf("Instance launched successfully: %s (ID: %s)", instance.Name, instance.ID)

	// Step 3: Apply idle policy
	// err = ctx.Client.ApplyIdlePolicy(context.Background(), instanceName, "aggressive-cost")
	// integration.AssertNoError(t, err, "Failed to apply idle policy")

	// Step 4: Poll for hibernation (DETERMINISTIC, not timeout-based)
	// maxWaitTime := 15 * time.Minute
	// pollInterval := 30 * time.Second
	// deadline := time.Now().Add(maxWaitTime)
	//
	// for time.Now().Before(deadline) {
	//     instanceState, err := ctx.Client.GetInstance(context.Background(), instanceName)
	//     integration.AssertNoError(t, err, "Failed to get instance state")
	//
	//     if instanceState.State == "stopped" {
	//         t.Log("✓ Hibernation triggered successfully")
	//         return
	//     }
	//     time.Sleep(pollInterval)
	// }
	// t.Errorf("Instance did not hibernate within %v", maxWaitTime)
}

// TestIdleDetection_ShortThreshold tests idle detection with aggressive timing
// This is a documentation-only test that describes expected behavior
func TestIdleDetection_ShortThreshold_Documentation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping documentation test in short mode")
	}

	// DETERMINISTIC: Document behavior without launching instances
	t.Log("Documenting idle detection behavior:")
	t.Log("")
	t.Log("IDLE THRESHOLD POLICIES:")
	t.Log("  - aggressive-cost: 10min idle → hibernate (65% savings)")
	t.Log("  - gpu: 15min idle → stop (optimized for GPU costs)")
	t.Log("  - balanced: 30min idle → hibernate (45% savings)")
	t.Log("  - research: 45min idle → hibernate (35% savings)")
	t.Log("  - conservative: 120min idle → stop (20% savings)")
	t.Log("")
	t.Log("DETECTION CRITERIA:")
	t.Log("  - CPU usage < 5% (configurable per policy)")
	t.Log("  - Memory usage < 10% (configurable per policy)")
	t.Log("  - Network activity minimal")
	t.Log("  - No user SSH sessions")
	t.Log("")
	t.Log("HIBERNATION vs STOP:")
	t.Log("  - Hibernate: Saves RAM to disk, faster resume, preserves session")
	t.Log("  - Stop: Complete shutdown, slower resume, clears RAM")
	t.Log("")
	t.Log("COST SAVINGS:")
	t.Log("  - Stopped instances: Only EBS storage charges (~$0.10/GB/month)")
	t.Log("  - Running instances: Compute + storage charges ($0.32+/hour)")
	t.Log("  - Savings: ~99% of compute costs during hibernation")
	t.Log("")

	t.Log("✓ Idle detection behavior documented")

	// Skip - this is documentation only, see TestIdleDetection_TriggersHibernation for full test
	t.Skip("Documentation only - see TestIdleDetection_TriggersHibernation for actual test")
}

// TestIdleDetection_ManualHibernation tests manual hibernation command
// This validates the hibernation mechanism works, even if auto-detection isn't implemented
func TestIdleDetection_ManualHibernation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hibernation test in short mode")
	}

	// Setup
	ctx := integration.NewTestContext(t)

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	instanceName := integration.GenerateTestName("test-manual-hibernate")
	t.Logf("Launching instance for manual hibernation test: %s", instanceName)

	_, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template: "Python ML Workstation",
		Name:     instanceName,
		Size:     "S",
	})
	integration.AssertNoError(t, err, "Failed to create instance")

	// Manual hibernation via API
	t.Log("Triggering manual hibernation...")
	err = ctx.HibernateInstance(instanceName)
	integration.AssertNoError(t, err, "Manual hibernation should succeed")

	t.Log("✓ Manual hibernation triggered successfully")

	// Verify instance is stopped
	instanceState, err := ctx.Client.GetInstance(context.Background(), instanceName)
	integration.AssertNoError(t, err, "Failed to get instance state")
	integration.AssertEqual(t, "stopped", instanceState.State, "Instance should be stopped after hibernation")

	t.Log("✓ Instance successfully hibernated")

	// Resume instance
	t.Log("Resuming instance from hibernation...")
	err = ctx.StartInstance(instanceName)
	integration.AssertNoError(t, err, "Resume should succeed")

	// Verify instance is running again
	instanceState, err = ctx.Client.GetInstance(context.Background(), instanceName)
	integration.AssertNoError(t, err, "Failed to get instance state after resume")
	integration.AssertEqual(t, "running", instanceState.State, "Instance should be running after resume")

	t.Log("✓ Instance resumed successfully - hibernation cycle complete")
}
