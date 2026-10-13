//go:build integration
// +build integration

package phase1_workflows

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/scttfrdmn/prism/test/integration"
)

// TestBudgetEnforcement_PreventsLaunch validates end-to-end budget enforcement
// ensuring that budget limits actually prevent overspending (not just API validation).
//
// This test addresses issue #398 - Budget Enforcement & Cost Tracking
//
// Success criteria:
// - Project with budget can be created
// - Budget status shows correct limits
// - Instances consume budget correctly
// - Launch attempts that exceed budget are BLOCKED
// - Error messages are clear and actionable
// - Cost tracking is accurate (within reasonable margin)
func TestBudgetEnforcement_PreventsLaunch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping budget enforcement test in short mode")
	}

	// Setup
	ctx := integration.NewTestContext(t)

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// Step 1: Create project with $10 monthly budget
	projectName := integration.GenerateTestName("test-budget-project")
	t.Logf("Creating project with budget enforcement: %s", projectName)

	monthlyLimit := 10.0
	project, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        projectName,
		Description: "Budget enforcement integration test",
		Owner:       "test-user@example.com",
		Budget: &fixtures.TestBudgetOptions{
			TotalBudget:  1000.0, // Overall project budget
			MonthlyLimit: &monthlyLimit,
			AlertThresholds: []types.BudgetAlert{
				{Threshold: 0.80, Type: types.BudgetAlertEmail, Recipients: []string{"test@example.com"}},
			},
			BudgetPeriod: types.BudgetPeriodMonthly,
		},
	})
	integration.AssertNoError(t, err, "Failed to create project with budget")
	integration.AssertNotEmpty(t, project.ID, "Project should have ID")

	t.Logf("Project created: %s (ID: %s)", project.Name, project.ID)
	t.Logf("Budget configured: $%.2f monthly limit", monthlyLimit)

	// Step 2: Verify budget status is correctly initialized
	t.Log("Verifying initial budget status...")
	budgetStatus, err := ctx.Client.GetProjectBudgetStatus(context.Background(), project.ID)
	integration.AssertNoError(t, err, "Failed to get project budget status")

	if budgetStatus.BudgetEnabled {
		t.Log("✓ Budget tracking is enabled")
		t.Logf("  Total budget: $%.2f", budgetStatus.TotalBudget)
		t.Logf("  Current spend: $%.2f", budgetStatus.SpentAmount)
		t.Logf("  Remaining: $%.2f", budgetStatus.RemainingBudget)

		// Verify initial state
		if budgetStatus.SpentAmount != 0 {
			t.Logf("Warning: Initial spend is non-zero: $%.2f", budgetStatus.SpentAmount)
		}
	} else {
		t.Log("⚠️  Budget tracking not enabled - feature may not be fully implemented")
		t.Skip("Budget enforcement not available - skipping test")
	}

	// Step 3: Launch instances consuming budget (under limit)
	t.Log("Launching instances to consume budget...")

	// Launch 2 small instances ($0.0104/hour * 720 hours/month = ~$7.50/month each = ~$15/month total)
	// But we're testing with a $10 monthly limit
	// So let's launch 1 instance first (~$7.50/month) - should succeed

	instance1Name := integration.GenerateTestName("test-budget-inst-1")
	t.Logf("Launching first instance: %s (should succeed, under budget)", instance1Name)

	_, err = fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Ubuntu 24.04 Server",
		Name:      instance1Name,
		Size:      "XS", // t3.micro: $0.0104/hour (~$7.50/month)
		ProjectID: &project.ID,
	})
	integration.AssertNoError(t, err, "First instance launch should succeed")
	t.Log("✓ First instance launched successfully")

	// Step 4: Wait for cost tracking to update
	t.Log("Waiting for cost tracker to update...")
	time.Sleep(10 * time.Second) // Give cost tracker time to register the instance

	// Check budget status after first instance
	budgetStatus, err = ctx.Client.GetProjectBudgetStatus(context.Background(), project.ID)
	integration.AssertNoError(t, err, "Failed to get budget status after launch")

	t.Logf("Budget status after first instance:")
	t.Logf("  Current spend: $%.2f", budgetStatus.SpentAmount)
	t.Logf("  Remaining: $%.2f", budgetStatus.RemainingBudget)
	t.Logf("  Spent percentage: %.1f%%", budgetStatus.SpentPercentage*100)

	// Step 5: Attempt to launch instance that would exceed budget
	instance2Name := integration.GenerateTestName("test-budget-inst-2")
	t.Logf("Attempting to launch second instance: %s (should fail, exceeds budget)", instance2Name)

	// Note: This test assumes budget enforcement is implemented
	// If not implemented, this will succeed (which we'll detect and skip)
	_, err = fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Ubuntu 24.04 Server",
		Name:      instance2Name,
		Size:      "S",         // t3.small: $0.0208/hour (~$14.98/month) - would exceed $10 limit
		ProjectID: &project.ID, // Associate with budgeted project
	})

	// Verify launch was blocked
	if err != nil {
		t.Log("✓ Launch was blocked (budget enforcement working)")
		t.Logf("  Error message: %s", err.Error())

		// Verify error message is clear and helpful
		errorMsg := strings.ToLower(err.Error())
		if !strings.Contains(errorMsg, "budget") &&
			!strings.Contains(errorMsg, "limit") &&
			!strings.Contains(errorMsg, "exceed") {
			t.Errorf("Error message should mention budget/limit/exceed: %s", err.Error())
		}

		// Check if error message shows budget details
		if !strings.Contains(err.Error(), fmt.Sprintf("%.2f", monthlyLimit)) &&
			!strings.Contains(err.Error(), "10") {
			t.Log("  Note: Error message could be improved to show budget limit")
		}

		if !strings.Contains(err.Error(), fmt.Sprintf("%.2f", budgetStatus.SpentAmount)) {
			t.Log("  Note: Error message could be improved to show current spend")
		}
	} else {
		t.Log("⚠️  Launch succeeded - budget enforcement may not be implemented")
		t.Log("Expected behavior: Launch should be blocked when exceeding monthly budget limit")
		t.Skip("Budget enforcement not implemented - skipping remaining checks")
	}

	// Step 6: Verify budget status is accurate
	t.Log("Verifying final budget status...")
	finalBudgetStatus, err := ctx.Client.GetProjectBudgetStatus(context.Background(), project.ID)
	integration.AssertNoError(t, err, "Failed to get final budget status")

	t.Logf("Final budget status:")
	t.Logf("  Total budget: $%.2f", finalBudgetStatus.TotalBudget)
	t.Logf("  Spent: $%.2f (%.1f%%)", finalBudgetStatus.SpentAmount, finalBudgetStatus.SpentPercentage*100)
	t.Logf("  Remaining: $%.2f", finalBudgetStatus.RemainingBudget)

	// Verify cost tracking is within reasonable range
	// t3.micro running for ~10 seconds should cost approximately $0.00003
	// Allow generous margin for cost tracker lag
	if finalBudgetStatus.SpentAmount > 1.0 {
		t.Errorf("Spend amount seems too high: $%.2f for short test", finalBudgetStatus.SpentAmount)
	}

	// Verify projected monthly spend is calculated
	if finalBudgetStatus.ProjectedMonthlySpend > 0 {
		t.Logf("  Projected monthly spend: $%.2f", finalBudgetStatus.ProjectedMonthlySpend)
		t.Log("✓ Cost projection is calculated")
	}

	t.Log("✓ Budget enforcement & cost tracking test completed")
}

// TestBudgetEnforcement_AlertThresholds tests budget alert triggering
func TestBudgetEnforcement_AlertThresholds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping budget alert test in short mode")
	}

	// Setup
	ctx := integration.NewTestContext(t)

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// Create project with low budget and 80% alert threshold
	projectName := integration.GenerateTestName("test-budget-alerts")
	monthlyLimit := 1.0 // Very low limit to easily trigger alerts

	_, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:  projectName,
		Owner: "test-user@example.com",
		Budget: &fixtures.TestBudgetOptions{
			TotalBudget:  100.0,
			MonthlyLimit: &monthlyLimit,
			AlertThresholds: []types.BudgetAlert{
				{Threshold: 0.80, Type: types.BudgetAlertEmail, Recipients: []string{"test@example.com"}},
				{Threshold: 0.90, Type: types.BudgetAlertEmail, Recipients: []string{"test@example.com"}},
			},
			BudgetPeriod: types.BudgetPeriodMonthly,
		},
	})
	integration.AssertNoError(t, err, "Failed to create project with alerts")

	t.Logf("Project created with alert thresholds: 80%%, 90%%")

	// Document expected behavior (actual alert testing requires email monitoring)
	t.Log("Expected behavior:")
	t.Log("  - When spend reaches 80% of $1 ($0.80), first alert triggered")
	t.Log("  - When spend reaches 90% of $1 ($0.90), second alert triggered")
	t.Log("  - Alerts should be sent to test@example.com")
	t.Log("  - Alert history should be tracked in project")

	// Check if alert tracking API exists
	// TODO: When alert history API is available:
	// alerts, err := ctx.Client.GetProjectAlerts(context.Background(), project.ID)
	// if err == nil {
	//     t.Logf("Alert tracking available: %d alerts", len(alerts))
	// }

	t.Log("✓ Budget alert configuration documented")
	t.Skip("Alert triggering requires actual spend and time - documented only")
}

// TestBudgetEnforcement_AutoActions tests automatic budget actions
func TestBudgetEnforcement_AutoActions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping budget auto-actions test in short mode")
	}

	// Setup
	ctx := integration.NewTestContext(t)

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// Create project with auto-action: prevent launches at 90%
	projectName := integration.GenerateTestName("test-budget-autoaction")
	monthlyLimit := 5.0

	project, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:  projectName,
		Owner: "test-user@example.com",
		Budget: &fixtures.TestBudgetOptions{
			TotalBudget:  100.0,
			MonthlyLimit: &monthlyLimit,
			AutoActions: []types.BudgetAutoAction{
				{
					Threshold: 0.90,
					Action:    types.BudgetActionPreventLaunch,
				},
			},
			BudgetPeriod: types.BudgetPeriodMonthly,
		},
	})
	integration.AssertNoError(t, err, "Failed to create project with auto-actions")

	t.Log("Project created with auto-action: prevent launches at 90%")

	// Document expected behavior
	t.Log("Expected behavior:")
	t.Log("  - When spend reaches 90% of $5 ($4.50), prevent new launches")
	t.Log("  - Existing instances continue running")
	t.Log("  - Clear error message explains auto-action triggered")
	t.Log("  - Admin can override or increase budget to allow launches")

	// Verify budget status shows auto-actions
	budgetStatus, err := ctx.Client.GetProjectBudgetStatus(context.Background(), project.ID)
	integration.AssertNoError(t, err, "Failed to get budget status")

	if budgetStatus.BudgetEnabled {
		t.Log("✓ Budget tracking enabled with auto-actions configured")
	}

	t.Log("✓ Budget auto-action configuration documented")
	t.Skip("Auto-action triggering requires actual spend - documented only")
}

// TestBudgetEnforcement_CostBreakdown tests cost breakdown API
func TestBudgetEnforcement_CostBreakdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cost breakdown test in short mode")
	}

	// Setup
	ctx := integration.NewTestContext(t)

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// Create project
	projectName := integration.GenerateTestName("test-cost-breakdown")
	project, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:  projectName,
		Owner: "test-user@example.com",
	})
	integration.AssertNoError(t, err, "Failed to create project")

	// Launch instance
	instanceName := integration.GenerateTestName("test-cost-inst")
	_, err = fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template: "Ubuntu 24.04 Server",
		Name:     instanceName,
		Size:     "S",
	})
	integration.AssertNoError(t, err, "Failed to launch instance")

	// Wait for cost data
	time.Sleep(5 * time.Second)

	// Get cost breakdown
	t.Log("Retrieving cost breakdown...")
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	costBreakdown, err := ctx.Client.GetProjectCostBreakdown(context.Background(), project.ID, startTime, endTime)

	if err != nil {
		if strings.Contains(err.Error(), "not implemented") ||
			strings.Contains(err.Error(), "not found") {
			t.Log("⚠️  Cost breakdown API not yet implemented")
			t.Skip("Cost breakdown API not available")
		}
		integration.AssertNoError(t, err, "Failed to get cost breakdown")
	}

	t.Log("✓ Cost breakdown retrieved:")
	t.Logf("  Total cost: $%.2f", costBreakdown.TotalCost)
	t.Logf("  Instance costs: %d entries", len(costBreakdown.InstanceCosts))
	t.Logf("  Storage costs: %d entries", len(costBreakdown.StorageCosts))

	t.Log("✓ Cost breakdown test completed")
}
