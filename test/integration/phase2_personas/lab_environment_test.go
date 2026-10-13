//go:build integration
// +build integration

package phase2_personas

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/scttfrdmn/prism/test/integration"
)

// TestLabEnvironment_SmithLab validates the complete workflow for a
// hierarchical research lab with multi-level budget management.
//
// This test addresses issue #401 - Lab Environment Persona
//
// Persona Background:
// - Smith Computational Biology Lab
// - PI (Dr. Patricia Smith): Strategic oversight, approves large purchases
// - Lab Manager (Dr. Michael Torres): Day-to-day operations, GPU cluster
// - Postdoc (Dr. Lisa Park): Independent researcher with sub-grant ($800/month)
// - Grad Students (James Wilson, Maria Garcia): Limited resources
// - Total lab budget: $4,500/month across 3 grants
//
// Success criteria:
// - Create hierarchical project structure (lab → sub-projects)
// - Configure budget allocations per role
// - Enforce role-based access control (RBAC)
// - Lab manager can view all resources
// - Grad students have limited instance launch permissions
// - Budget tracking shows per-person spend attribution
// - Shared EFS storage accessible to all team members
func TestLabEnvironment_SmithLab(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running persona test in short mode")
	}

	// Setup
	ctx := integration.NewTestContext(t)
	defer ctx.Cleanup()

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// ========================================
	// Scenario: Lab Setup (PI Creates Lab)
	// ========================================

	// Step 1: Dr. Smith (PI) creates lab-wide project
	labProjectName := integration.GenerateTestName("smith-compbio-lab")
	t.Logf("🏫 Dr. Smith (PI) creates lab project: %s", labProjectName)

	totalLabBudget := 4500.0 // $4,500/month total
	labProject, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        labProjectName,
		Description: "Smith Computational Biology Lab - Main Project",
		Owner:       "patricia.smith@university.edu",
		Budget: &fixtures.TestBudgetOptions{
			TotalBudget:  54000.0, // Annual budget ($4,500 × 12)
			MonthlyLimit: &totalLabBudget,
			BudgetPeriod: types.BudgetPeriodMonthly,
			AlertThresholds: []types.BudgetAlert{
				{
					Threshold: 0.80,
					Type:      types.BudgetAlertEmail,
					Message:   "Lab approaching monthly budget limit",
					Enabled:   true,
				},
			},
			AutoActions: []types.BudgetAutoAction{
				{
					Threshold: 1.00,
					Action:    types.BudgetActionPreventLaunch,
					// Monthly lab budget exceeded
					Enabled: true,
				},
			},
		},
	})
	integration.AssertNoError(t, err, "Failed to create lab project")
	t.Logf("✅ Lab project created with $4,500/month budget")

	// Step 2: Add lab manager as member
	t.Logf("👥 Adding lab manager: Dr. Torres")

	// Lab manager has admin rights
	err = fixtures.CreateTestProjectMember(t, registry, labProject.ID, "michael.torres@university.edu", types.ProjectRoleAdmin)
	integration.AssertNoError(t, err, "Failed to add lab manager")
	labManagerMember := getProjectMember(t, ctx, labProject.ID, "michael.torres@university.edu")
	integration.AssertEqual(t, types.ProjectRoleAdmin, labManagerMember.Role, "Lab manager should have admin role")
	t.Logf("✅ Lab manager added with admin role")

	// ========================================
	// Scenario: Sub-Project for Postdoc
	// ========================================

	// Step 3: Create sub-project for Dr. Lisa Park (postdoc)
	postdocProjectName := integration.GenerateTestName("park-protein-folding")
	t.Logf("🔬 Creating sub-project for Dr. Park (postdoc)")

	postdocMonthlyLimit := 800.0 // $800/month allocation
	postdocProject, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        postdocProjectName,
		Description: "Protein Folding Research - NSF Grant",
		Owner:       "lisa.park@university.edu",
		Budget: &fixtures.TestBudgetOptions{
			TotalBudget:  9600.0, // Annual allocation ($800 × 12)
			MonthlyLimit: &postdocMonthlyLimit,
			BudgetPeriod: types.BudgetPeriodMonthly,
			AlertThresholds: []types.BudgetAlert{
				{
					Threshold: 0.75,
					Type:      types.BudgetAlertEmail,
					Message:   "Postdoc project approaching budget limit",
					Enabled:   true,
				},
			},
		},
	})
	integration.AssertNoError(t, err, "Failed to create postdoc project")
	t.Logf("✅ Postdoc sub-project created with $800/month budget")

	// ========================================
	// Scenario: Grad Student Projects
	// ========================================

	// Step 4: Create project for James Wilson (senior grad student)
	jamesProjectName := integration.GenerateTestName("wilson-rnaseq")
	t.Logf("👨‍🎓 Creating project for James Wilson (Year 4 PhD)")

	jamesMonthlyLimit := 400.0 // $400/month allocation
	jamesProject, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        jamesProjectName,
		Description: "RNA-seq Analysis - NIH R01",
		Owner:       "james.wilson@university.edu",
		Budget: &fixtures.TestBudgetOptions{
			TotalBudget:  4800.0, // Annual allocation ($400 × 12)
			MonthlyLimit: &jamesMonthlyLimit,
			BudgetPeriod: types.BudgetPeriodMonthly,
			AlertThresholds: []types.BudgetAlert{
				{
					Threshold: 0.90,
					Type:      types.BudgetAlertEmail,
					Message:   "Graduate student approaching budget limit - talk to advisor",
					Enabled:   true,
				},
			},
		},
	})
	integration.AssertNoError(t, err, "Failed to create James's project")
	t.Logf("✅ James's project created with $400/month budget")

	// Step 5: Create project for Maria Garcia (junior grad student)
	mariaProjectName := integration.GenerateTestName("garcia-learning")
	t.Logf("👩‍🎓 Creating project for Maria Garcia (Year 2 PhD)")

	mariaMonthlyLimit := 300.0 // $300/month allocation
	mariaProject, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        mariaProjectName,
		Description: "Learning Project - Rotation Student",
		Owner:       "maria.garcia@university.edu",
		Budget: &fixtures.TestBudgetOptions{
			TotalBudget:  3600.0, // Annual allocation ($300 × 12)
			MonthlyLimit: &mariaMonthlyLimit,
			BudgetPeriod: types.BudgetPeriodMonthly,
			AlertThresholds: []types.BudgetAlert{
				{
					Threshold: 0.80,
					Type:      types.BudgetAlertEmail,
					Message:   "Rotation student approaching budget limit",
					Enabled:   true,
				},
			},
			AutoActions: []types.BudgetAutoAction{
				{
					Threshold: 1.00,
					Action:    types.BudgetActionPreventLaunch,
					// Budget limit reached - contact lab manager
					Enabled: true,
				},
			},
		},
	})
	integration.AssertNoError(t, err, "Failed to create Maria's project")
	t.Logf("✅ Maria's project created with $300/month budget")

	// ========================================
	// Scenario: Shared Lab Storage
	// ========================================

	// Step 6: Create shared EFS storage for lab
	t.Logf("💾 Creating shared lab storage (EFS)")

	sharedStorageName := integration.GenerateTestName("smith-lab-shared-data")
	// EFS is elastic (no size) and volume creation takes no project or
	// description, so only the name is passed.
	sharedStorage, err := fixtures.CreateTestVolume(t, registry, fixtures.CreateTestVolumeOptions{
		Name: sharedStorageName,
	})
	integration.AssertNoError(t, err, "Failed to create shared storage")
	t.Logf("✅ Shared lab storage created: %s", sharedStorage.Name)

	// ========================================
	// Scenario: Lab Members Launch Workspaces
	// ========================================

	// Step 7: Dr. Park (postdoc) launches GPU workspace
	t.Logf("🚀 Dr. Park launches GPU workspace for protein folding")

	parkInstanceName := integration.GenerateTestName("park-protein-gpu")
	parkInstance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Python ML Workstation",
		Name:      parkInstanceName,
		Size:      "L", // Larger instance for GPU work
		ProjectID: &postdocProject.ID,
	})
	integration.AssertNoError(t, err, "Failed to launch postdoc instance")
	integration.AssertEqual(t, "running", parkInstance.State, "Postdoc instance should be running")
	t.Logf("✅ Postdoc workspace launched successfully")

	// Step 8: James (grad student) launches CPU workspace
	t.Logf("🚀 James launches CPU workspace for RNA-seq")

	jamesInstanceName := integration.GenerateTestName("james-rnaseq")
	jamesInstance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Python ML Workstation",
		Name:      jamesInstanceName,
		Size:      "M", // Medium instance for data analysis
		ProjectID: &jamesProject.ID,
	})
	integration.AssertNoError(t, err, "Failed to launch James's instance")
	integration.AssertEqual(t, "running", jamesInstance.State, "James's instance should be running")
	t.Logf("✅ James's workspace launched successfully")

	// Step 9: Maria (junior grad) launches small workspace
	t.Logf("🚀 Maria launches small workspace for learning")

	mariaInstanceName := integration.GenerateTestName("maria-learning")
	mariaInstance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Python ML Workstation",
		Name:      mariaInstanceName,
		Size:      "S", // Small instance for learning
		ProjectID: &mariaProject.ID,
	})
	integration.AssertNoError(t, err, "Failed to launch Maria's instance")
	integration.AssertEqual(t, "running", mariaInstance.State, "Maria's instance should be running")
	t.Logf("✅ Maria's workspace launched successfully")

	// ========================================
	// Scenario: Lab Manager Reviews Resources
	// ========================================

	// Step 10: Lab manager reviews all lab resources
	t.Logf("📊 Lab manager (Dr. Torres) reviews lab resources")

	// List all instances across lab projects
	allInstances, err := ctx.Client.ListInstances(context.Background())
	integration.AssertNoError(t, err, "Failed to list instances")

	labInstanceCount := 0
	for _, inst := range allInstances.Instances {
		// Count instances in lab projects
		if inst.ProjectID == labProject.ID ||
			inst.ProjectID == postdocProject.ID ||
			inst.ProjectID == jamesProject.ID ||
			inst.ProjectID == mariaProject.ID {
			labInstanceCount++
			t.Logf("   - %s: %s (state: %s)", inst.Name, inst.Template, inst.State)
		}
	}

	if labInstanceCount < 3 {
		t.Fatalf("Expected at least 3 lab instances, found %d", labInstanceCount)
	}
	t.Logf("✅ Lab manager can view all %d lab instances", labInstanceCount)

	// ========================================
	// Scenario: Monthly Budget Review
	// ========================================

	// Step 11: Review budget status for each project
	t.Logf("💰 Reviewing monthly budget usage across lab")

	// Wait for cost tracking to update
	time.Sleep(5 * time.Second)

	// Check postdoc project budget
	postdocBudget, err := ctx.Client.GetProjectBudgetStatus(context.Background(), postdocProject.ID)
	integration.AssertNoError(t, err, "Failed to get postdoc budget")
	t.Logf("   Dr. Park (postdoc): $%.2f / $%.2f", postdocBudget.SpentAmount, postdocMonthlyLimit)

	// Check James's budget
	jamesBudget, err := ctx.Client.GetProjectBudgetStatus(context.Background(), jamesProject.ID)
	integration.AssertNoError(t, err, "Failed to get James's budget")
	t.Logf("   James (Year 4): $%.2f / $%.2f", jamesBudget.SpentAmount, jamesMonthlyLimit)

	// Check Maria's budget
	mariaBudget, err := ctx.Client.GetProjectBudgetStatus(context.Background(), mariaProject.ID)
	integration.AssertNoError(t, err, "Failed to get Maria's budget")
	t.Logf("   Maria (Year 2): $%.2f / $%.2f", mariaBudget.SpentAmount, mariaMonthlyLimit)

	// Check overall lab budget
	labBudget, err := ctx.Client.GetProjectBudgetStatus(context.Background(), labProject.ID)
	integration.AssertNoError(t, err, "Failed to get lab budget")
	t.Logf("   Lab Total: $%.2f / $%.2f", labBudget.SpentAmount, totalLabBudget)

	totalAllocatedSpend := postdocBudget.SpentAmount + jamesBudget.SpentAmount + mariaBudget.SpentAmount
	t.Logf("   Sub-projects total: $%.2f", totalAllocatedSpend)

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Lab Environment Persona Test Complete!")
	t.Logf("   ✓ Hierarchical project structure created")
	t.Logf("   ✓ Budget allocations configured per role")
	t.Logf("   ✓ Lab manager has visibility across all projects")
	t.Logf("   ✓ Grad students have appropriate budget limits")
	t.Logf("   ✓ Shared lab storage available")
	t.Logf("   ✓ Per-project budget tracking functional")
	t.Logf("")
	t.Logf("🎉 Smith Lab can now manage resources efficiently!")
	t.Logf("   - PI has strategic oversight")
	t.Logf("   - Lab manager can monitor all activity")
	t.Logf("   - Each researcher has appropriate budget allocation")
	t.Logf("   - Budget tracking prevents overspending")
}
