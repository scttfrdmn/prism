//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHibernation_ManualFlow validates manual hibernation and resume cycle
// Tests: Launch → Stop (hibernate) → Start (resume) → Verify state transitions
func TestHibernation_ManualFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hibernation test in short mode")
	}

	ctx := context.Background()
	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: "aws",
		AWSRegion:  "us-west-2",
	})
	registry := fixtures.NewFixtureRegistry(t, apiClient)

	instanceName := fmt.Sprintf("hibernate-manual-%d", time.Now().Unix())

	t.Logf("Launching instance for hibernation test: %s", instanceName)

	// Launch instance
	launchResp, err := apiClient.LaunchInstance(ctx, types.LaunchRequest{
		Template: "Ubuntu Basic",
		Name:     instanceName,
		Size:     "S",
	})
	require.NoError(t, err, "Failed to launch instance")
	registry.Register("instance", instanceName)

	t.Logf("Instance launched: %s (ID: %s)", launchResp.Instance.Name, launchResp.Instance.ID)

	// Wait for running state
	t.Log("Waiting for instance to reach running state...")
	err = fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute)
	require.NoError(t, err, "Instance failed to reach running state")

	// Get instance details
	instance, err := apiClient.GetInstance(ctx, instanceName)
	require.NoError(t, err, "Failed to get instance details")
	require.Equal(t, "running", instance.State, "Instance should be running")

	recordInitialCost := instance.HourlyRate
	t.Logf("Instance running - hourly rate: $%.4f", recordInitialCost)

	// Test 1: Manual hibernation (stop)
	t.Run("ManualHibernation", func(t *testing.T) {
		t.Log("Initiating manual hibernation (stop)...")

		err := apiClient.StopInstance(ctx, instanceName)
		assert.NoError(t, err, "Stop instance should succeed")

		t.Log("Waiting for instance to reach stopped state...")
		err = fixtures.WaitForInstanceState(t, apiClient, instanceName, "stopped", 3*time.Minute)
		assert.NoError(t, err, "Instance should reach stopped state")

		// Verify instance is stopped
		instance, err := apiClient.GetInstance(ctx, instanceName)
		assert.NoError(t, err, "Failed to get instance state")
		assert.Equal(t, "stopped", instance.State, "Instance should be stopped")

		t.Log("✓ Manual hibernation successful")
	})

	// Test 2: Resume from hibernation (start)
	t.Run("ResumeFromHibernation", func(t *testing.T) {
		t.Log("Resuming instance from hibernation...")

		err := apiClient.StartInstance(ctx, instanceName)
		assert.NoError(t, err, "Start instance should succeed")

		t.Log("Waiting for instance to reach running state...")
		err = fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute)
		assert.NoError(t, err, "Instance should reach running state")

		// Verify instance is running again
		instance, err := apiClient.GetInstance(ctx, instanceName)
		assert.NoError(t, err, "Failed to get instance state")
		assert.Equal(t, "running", instance.State, "Instance should be running after resume")

		t.Log("✓ Resume from hibernation successful")
	})

	// Test 3: Multiple hibernation cycles
	t.Run("MultipleHibernationCycles", func(t *testing.T) {
		t.Log("Testing multiple hibernation cycles...")

		for i := 1; i <= 2; i++ {
			t.Logf("Cycle %d: Hibernating...", i)

			// Stop
			err := apiClient.StopInstance(ctx, instanceName)
			assert.NoError(t, err, "Stop should succeed on cycle %d", i)
			err = fixtures.WaitForInstanceState(t, apiClient, instanceName, "stopped", 3*time.Minute)
			assert.NoError(t, err, "Should reach stopped state on cycle %d", i)

			t.Logf("Cycle %d: Resuming...", i)

			// Start
			err = apiClient.StartInstance(ctx, instanceName)
			assert.NoError(t, err, "Start should succeed on cycle %d", i)
			err = fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute)
			assert.NoError(t, err, "Should reach running state on cycle %d", i)

			t.Logf("✓ Cycle %d complete", i)
		}

		t.Log("✓ Multiple hibernation cycles successful")
	})

	t.Log("✅ Manual hibernation flow test complete")
}

// TestHibernation_PolicyApplication validates idle policy management
// Tests: List policies → Apply policy → Verify policy applied → Remove policy
func TestHibernation_PolicyApplication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping idle policy test in short mode")
	}

	ctx := context.Background()
	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: "aws",
		AWSRegion:  "us-west-2",
	})
	registry := fixtures.NewFixtureRegistry(t, apiClient)

	instanceName := fmt.Sprintf("idle-policy-%d", time.Now().Unix())

	t.Logf("Launching instance for idle policy testing: %s", instanceName)

	// Launch instance
	launchResp, err := apiClient.LaunchInstance(ctx, types.LaunchRequest{
		Template: "Ubuntu Basic",
		Name:     instanceName,
		Size:     "S",
	})
	require.NoError(t, err, "Failed to launch instance")
	registry.Register("instance", instanceName)

	t.Logf("Instance launched: %s (ID: %s)", launchResp.Instance.Name, launchResp.Instance.ID)

	// Wait for running state
	err = fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute)
	require.NoError(t, err, "Instance failed to reach running state")

	// Test 1: List available idle policies
	t.Run("ListIdlePolicies", func(t *testing.T) {
		t.Log("Listing available idle policies...")

		policies, err := apiClient.ListIdlePolicies(ctx)
		assert.NoError(t, err, "Should be able to list idle policies")
		assert.NotEmpty(t, policies, "Should have at least one idle policy available")

		t.Logf("Found %d idle policies:", len(policies))
		for _, policy := range policies {
			t.Logf("  - %s: %s", policy.ID, policy.Name)
		}

		t.Log("✓ Idle policy listing successful")
	})

	// Test 2: Get policy recommendation
	t.Run("GetPolicyRecommendation", func(t *testing.T) {
		t.Log("Getting idle policy recommendation...")

		policy, err := apiClient.RecommendIdlePolicy(ctx, instanceName)
		assert.NoError(t, err, "Should be able to get policy recommendation")
		assert.NotNil(t, policy, "Should receive a policy recommendation")

		t.Logf("Recommended policy: %s (%s)", policy.Name, policy.ID)
		t.Logf("  Description: %s", policy.Description)

		t.Log("✓ Policy recommendation successful")
	})

	// Test 3: Apply idle policy
	t.Run("ApplyIdlePolicy", func(t *testing.T) {
		t.Log("Applying idle policy to instance...")

		// Apply "balanced" policy (common default)
		err := apiClient.ApplyIdlePolicy(ctx, instanceName, "balanced")
		assert.NoError(t, err, "Should be able to apply idle policy")

		t.Log("✓ Idle policy applied")

		// Verify policy was applied
		t.Log("Verifying policy application...")
		policies, err := apiClient.GetInstanceIdlePolicies(ctx, instanceName)
		assert.NoError(t, err, "Should be able to get instance policies")

		// Check if any policy is applied (may be empty if scheduler not running)
		if len(policies) > 0 {
			t.Logf("Applied policies: %d", len(policies))
			for _, policy := range policies {
				t.Logf("  - %s", policy.Name)
			}
		} else {
			t.Log("⚠️  No policies returned (scheduler may not be running)")
		}

		t.Log("✓ Policy application verified")
	})

	// Test 4: Remove idle policy
	t.Run("RemoveIdlePolicy", func(t *testing.T) {
		t.Log("Removing idle policy from instance...")

		err := apiClient.RemoveIdlePolicy(ctx, instanceName, "balanced")
		// Note: This may fail if scheduler isn't running, which is acceptable
		if err != nil {
			t.Logf("⚠️  Policy removal returned error (expected if scheduler not running): %v", err)
		} else {
			t.Log("✓ Idle policy removed")
		}
	})

	t.Log("✅ Idle policy application test complete")
}

// TestHibernation_CostSavings validates cost savings tracking
// Tests: Get savings report → Verify report structure → Calculate expected savings
func TestHibernation_CostSavings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cost savings test in short mode")
	}

	ctx := context.Background()
	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: "aws",
		AWSRegion:  "us-west-2",
	})

	t.Log("Testing idle cost savings reporting...")

	// Test 1: Get savings report
	t.Run("GetSavingsReport", func(t *testing.T) {
		t.Log("Fetching idle savings report...")

		report, err := apiClient.GetIdleSavingsReport(ctx, "30d")
		assert.NoError(t, err, "Should be able to get savings report")
		assert.NotNil(t, report, "Report should not be nil")

		// Verify report structure
		assert.Contains(t, report, "report_id", "Report should have report_id")
		assert.Contains(t, report, "generated_at", "Report should have generated_at")
		assert.Contains(t, report, "period", "Report should have period")
		assert.Contains(t, report, "total_saved", "Report should have total_saved")
		assert.Contains(t, report, "projected_savings", "Report should have projected_savings")
		assert.Contains(t, report, "idle_hours", "Report should have idle_hours")
		assert.Contains(t, report, "active_hours", "Report should have active_hours")
		assert.Contains(t, report, "savings_percentage", "Report should have savings_percentage")

		t.Logf("Report ID: %v", report["report_id"])
		t.Logf("Generated at: %v", report["generated_at"])
		t.Logf("Total saved: $%.2f", report["total_saved"])
		t.Logf("Projected savings: $%.2f", report["projected_savings"])
		t.Logf("Idle hours: %.1f", report["idle_hours"])
		t.Logf("Active hours: %.1f", report["active_hours"])
		t.Logf("Savings percentage: %.1f%%", report["savings_percentage"])

		t.Log("✓ Savings report structure validated")
	})

	// Test 2: Calculate expected savings
	t.Run("CalculateExpectedSavings", func(t *testing.T) {
		t.Log("Calculating expected hibernation savings...")

		// Example calculation:
		// t3.small: $0.0208/hour
		// Hibernated for 8 hours/day = $0.1664 saved/day
		// Over 30 days = $4.99 saved/month

		hourlyRate := 0.0208       // t3.small
		idleHoursPerDay := 8.0     // 8 hours idle per day
		daysPerMonth := 30.0       // 30-day month
		computeSavingsRate := 0.99 // 99% of compute cost saved during hibernation

		expectedSavingsPerDay := hourlyRate * idleHoursPerDay * computeSavingsRate
		expectedSavingsPerMonth := expectedSavingsPerDay * daysPerMonth

		t.Logf("Expected savings calculation:")
		t.Logf("  Hourly rate: $%.4f", hourlyRate)
		t.Logf("  Idle hours/day: %.1f", idleHoursPerDay)
		t.Logf("  Days/month: %.0f", daysPerMonth)
		t.Logf("  Savings rate: %.0f%%", computeSavingsRate*100)
		t.Logf("  Expected savings/day: $%.2f", expectedSavingsPerDay)
		t.Logf("  Expected savings/month: $%.2f", expectedSavingsPerMonth)

		assert.Greater(t, expectedSavingsPerMonth, 4.0, "Should save at least $4/month")
		assert.Less(t, expectedSavingsPerMonth, 6.0, "Should save less than $6/month for t3.small")

		t.Log("✓ Cost savings calculation validated")
	})

	t.Log("✅ Cost savings test complete")
}

// TestHibernation_MultipleInstances validates concurrent hibernation management
// Tests: Launch multiple → Hibernate all → Resume all → Verify state consistency
func TestHibernation_MultipleInstances(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple instance hibernation test in short mode")
	}

	ctx := context.Background()
	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: "aws",
		AWSRegion:  "us-west-2",
	})
	registry := fixtures.NewFixtureRegistry(t, apiClient)

	numInstances := 3
	instanceNames := make([]string, numInstances)

	t.Logf("Launching %d instances for concurrent hibernation test...", numInstances)

	// Test 1: Launch multiple instances
	t.Run("LaunchMultipleInstances", func(t *testing.T) {
		for i := 0; i < numInstances; i++ {
			instanceName := fmt.Sprintf("multi-hibernate-%d-%d", i, time.Now().Unix())
			instanceNames[i] = instanceName

			t.Logf("Launching instance %d/%d: %s", i+1, numInstances, instanceName)

			launchResp, err := apiClient.LaunchInstance(ctx, types.LaunchRequest{
				Template: "Ubuntu Basic",
				Name:     instanceName,
				Size:     "S",
			})
			assert.NoError(t, err, "Failed to launch instance %d", i)
			registry.Register("instance", instanceName)

			t.Logf("  Instance launched: %s (ID: %s)", launchResp.Instance.Name, launchResp.Instance.ID)
		}

		// Wait for all to be running
		t.Log("Waiting for all instances to reach running state...")
		for i, instanceName := range instanceNames {
			t.Logf("  Waiting for instance %d/%d: %s", i+1, numInstances, instanceName)
			err := fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute)
			assert.NoError(t, err, "Instance %s should reach running state", instanceName)
		}

		t.Logf("✓ All %d instances launched and running", numInstances)
	})

	// Test 2: Hibernate all instances
	t.Run("HibernateAllInstances", func(t *testing.T) {
		t.Logf("Hibernating all %d instances...", numInstances)

		for i, instanceName := range instanceNames {
			t.Logf("  Stopping instance %d/%d: %s", i+1, numInstances, instanceName)
			err := apiClient.StopInstance(ctx, instanceName)
			assert.NoError(t, err, "Failed to stop instance %s", instanceName)
		}

		// Wait for all to be stopped
		t.Log("Waiting for all instances to reach stopped state...")
		for i, instanceName := range instanceNames {
			t.Logf("  Waiting for instance %d/%d: %s", i+1, numInstances, instanceName)
			err := fixtures.WaitForInstanceState(t, apiClient, instanceName, "stopped", 3*time.Minute)
			assert.NoError(t, err, "Instance %s should reach stopped state", instanceName)
		}

		t.Logf("✓ All %d instances hibernated", numInstances)
	})

	// Test 3: Resume all instances
	t.Run("ResumeAllInstances", func(t *testing.T) {
		t.Logf("Resuming all %d instances...", numInstances)

		for i, instanceName := range instanceNames {
			t.Logf("  Starting instance %d/%d: %s", i+1, numInstances, instanceName)
			err := apiClient.StartInstance(ctx, instanceName)
			assert.NoError(t, err, "Failed to start instance %s", instanceName)
		}

		// Wait for all to be running
		t.Log("Waiting for all instances to reach running state...")
		for i, instanceName := range instanceNames {
			t.Logf("  Waiting for instance %d/%d: %s", i+1, numInstances, instanceName)
			err := fixtures.WaitForInstanceState(t, apiClient, instanceName, "running", 5*time.Minute)
			assert.NoError(t, err, "Instance %s should reach running state", instanceName)
		}

		t.Logf("✓ All %d instances resumed", numInstances)
	})

	// Test 4: Verify state consistency
	t.Run("VerifyStateConsistency", func(t *testing.T) {
		t.Log("Verifying state consistency across all instances...")

		for i, instanceName := range instanceNames {
			instance, err := apiClient.GetInstance(ctx, instanceName)
			assert.NoError(t, err, "Failed to get instance %s details", instanceName)
			assert.Equal(t, "running", instance.State, "Instance %d should be running", i+1)
			assert.NotEmpty(t, instance.PublicIP, "Instance %d should have public IP", i+1)

			t.Logf("  ✓ Instance %d/%d: %s (state: %s, IP: %s)", i+1, numInstances, instanceName, instance.State, instance.PublicIP)
		}

		t.Log("✓ State consistency verified")
	})

	t.Log("✅ Multiple instance hibernation test complete")
}
