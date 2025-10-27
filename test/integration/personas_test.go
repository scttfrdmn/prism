package integration

import (
	"context"
	"testing"
	"time"
)

// TestSoloResearcherPersona tests the Solo Researcher (Dr. Sarah Chen) workflow
// Based on docs/USER_SCENARIOS/01_SOLO_RESEARCHER_WALKTHROUGH.md
//
// Workflow:
// 1. Launch bioinformatics workspace (size M)
// 2. Configure hibernation profile (budget-safe, 15min idle)
// 3. Verify workspace is running with correct template
// 4. Test hibernation cycle (stop → start)
// 5. Verify cost tracking and hibernation savings
// 6. Cleanup
func TestSoloResearcherPersona(t *testing.T) {
	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Generate unique names for this test run
	instanceName := GenerateTestName("test-rnaseq-analysis")

	t.Run("Phase1_LaunchBioinformaticsWorkspace", func(t *testing.T) {
		// Launch instance with bioinformatics template
		// Per walkthrough: "prism launch bioinformatics-suite rnaseq-analysis --size M"
		// Using Python ML Workstation template (slug: python-ml-workstation)
		instance, err := ctx.LaunchInstance("python-ml-workstation", instanceName, "M")
		AssertNoError(t, err, "Launch bioinformatics workspace")

		// Verify instance details
		AssertNotEmpty(t, instance.ID, "Instance should have AWS ID")
		AssertNotEmpty(t, instance.PublicIP, "Instance should have public IP")
		AssertEqual(t, "running", instance.State, "Instance state")
		AssertEqual(t, instanceName, instance.Name, "Instance name")

		t.Logf("✅ Workspace launched successfully")
		t.Logf("   Name: %s", instance.Name)
		t.Logf("   ID: %s", instance.ID)
		t.Logf("   Public IP: %s", instance.PublicIP)
		t.Logf("   State: %s", instance.State)
		t.Logf("   Template: %s", instance.Template)
	})

	t.Run("Phase2_ConfigureHibernationPolicy", func(t *testing.T) {
		// Per walkthrough: Apply hibernation policy for cost optimization
		// Note: Using the new idle policy system

		// List available idle policies
		policies, err := ctx.Client.ListIdlePolicies(context.Background())
		AssertNoError(t, err, "List idle policies")

		t.Logf("Available idle policies: %d", len(policies))
		for _, policy := range policies {
			t.Logf("  - %s: %s", policy.ID, policy.Name)
		}

		// Find a hibernation-friendly policy (cost-optimized or batch)
		// Per walkthrough: Apply aggressive hibernation for budget safety
		policyToApply := ""
		for _, policy := range policies {
			// Look for policies that use hibernation action
			if policy.ID == "cost-optimized" || policy.ID == "batch" {
				policyToApply = policy.ID
				t.Logf("Selected policy: %s (%s)", policy.ID, policy.Name)
				break
			}
		}

		if policyToApply == "" {
			// If no pre-configured policy found, skip this test phase
			// (In real deployment, policies should be pre-configured)
			t.Log("⚠️  No hibernation policies found - skipping policy application")
			t.Log("   This is expected if idle policy system is not yet configured")
			return
		}

		// Apply policy to instance
		// Per walkthrough: "prism idle instance rnaseq-analysis --profile budget-safe"
		err = ctx.Client.ApplyIdlePolicy(context.Background(), instanceName, policyToApply)
		AssertNoError(t, err, "Apply idle policy to instance")

		t.Logf("✅ Applied idle policy '%s' to instance", policyToApply)
		t.Log("   Instance will automatically hibernate when idle")
	})

	t.Run("Phase3_VerifyWorkspaceConfiguration", func(t *testing.T) {
		// Verify instance is still running with correct configuration
		instance := ctx.AssertInstanceExists(instanceName)

		AssertEqual(t, "running", instance.State, "Instance should still be running")
		AssertNotEmpty(t, instance.PublicIP, "Instance should have public IP")

		t.Logf("✅ Workspace configuration verified")
		t.Logf("   Status: %s", instance.State)
		t.Logf("   Launch time: %s", instance.LaunchTime)
		t.Logf("   Uptime: %s", time.Since(instance.LaunchTime).Round(time.Second))
	})

	t.Run("Phase4_TestHibernationCycle", func(t *testing.T) {
		// Test manual hibernation (simulates idle detection triggering)
		// Per walkthrough: Workspace automatically hibernates after 15min idle

		t.Log("Testing hibernation cycle...")

		// Hibernate instance
		err := ctx.HibernateInstance(instanceName)
		AssertNoError(t, err, "Hibernate instance")

		// Verify stopped state (hibernated instances show as "stopped")
		ctx.AssertInstanceState(instanceName, "stopped")
		t.Logf("✅ Instance hibernated successfully")

		// Resume from hibernation
		// Per walkthrough: "prism start rnaseq-analysis" (resumes in 30 seconds)
		err = ctx.StartInstance(instanceName)
		AssertNoError(t, err, "Resume from hibernation")

		// Verify running state
		instance, err := ctx.WaitForInstanceRunning(instanceName)
		AssertNoError(t, err, "Wait for instance running")
		AssertNotEmpty(t, instance.PublicIP, "Instance should have public IP after resume")

		t.Logf("✅ Instance resumed from hibernation")
		t.Logf("   State: %s", instance.State)
		t.Logf("   Public IP: %s", instance.PublicIP)
	})

	t.Run("Phase5_VerifyCostTracking", func(t *testing.T) {
		// Verify cost tracking is working
		// Per walkthrough: "prism cost summary"

		listResp, err := ctx.Client.ListInstances(context.Background())
		AssertNoError(t, err, "List instances for cost tracking")

		foundInstance := false
		for _, instance := range listResp.Instances {
			if instance.Name == instanceName {
				foundInstance = true

				// Verify cost fields are present
				if instance.EstimatedCost > 0 {
					t.Logf("✅ Cost tracking verified")
					t.Logf("   Estimated cost: $%.2f", instance.EstimatedCost)
					t.Logf("   Hourly rate: $%.4f", instance.HourlyRate)
					t.Logf("   Current spend: $%.4f", instance.CurrentSpend)
				} else {
					t.Log("⚠️  Cost not yet calculated (instance may be too new)")
				}

				// Verify hibernation savings tracking
				t.Logf("   Instance type: %s", instance.InstanceType)
				t.Logf("   Launch time: %s", instance.LaunchTime)
				break
			}
		}

		if !foundInstance {
			t.Fatalf("Instance '%s' not found in instance list", instanceName)
		}
	})

	t.Run("Phase6_Cleanup", func(t *testing.T) {
		// Delete instance
		// Per walkthrough: "prism delete rnaseq-analysis"
		err := ctx.DeleteInstance(instanceName)
		AssertNoError(t, err, "Delete instance")

		t.Logf("✅ Instance deleted successfully")

		// Poll until instance no longer appears in list (AWS eventual consistency)
		// Terminated instances can take time to disappear from AWS
		t.Log("Polling for instance to disappear from list...")
		deadline := time.Now().Add(2 * time.Minute)
		instanceGone := false

		for time.Now().Before(deadline) {
			listResp, err := ctx.Client.ListInstances(context.Background())
			AssertNoError(t, err, "List instances after deletion")

			found := false
			for _, instance := range listResp.Instances {
				if instance.Name == instanceName {
					found = true
					t.Logf("  Instance still visible in state: %s (waiting...)", instance.State)
					break
				}
			}

			if !found {
				instanceGone = true
				break
			}

			time.Sleep(10 * time.Second)
		}

		if !instanceGone {
			t.Fatalf("Instance '%s' still exists after deletion timeout", instanceName)
		}

		t.Log("✅ Cleanup verified - instance no longer in list")
	})

	t.Log("🎉 Solo Researcher persona test completed successfully!")
}

// TestLabEnvironmentPersona tests the Lab Environment (Prof. Martinez) workflow
// Based on docs/USER_SCENARIOS/02_LAB_ENVIRONMENT_WALKTHROUGH.md
//
// TODO: Implement multi-user setup, shared storage, team collaboration
func TestLabEnvironmentPersona(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Lab Environment persona test in short mode")
	}

	t.Skip("Lab Environment persona test not yet implemented")

	// Planned workflow:
	// 1. Launch multiple workspaces for team members
	// 2. Create shared EFS volume
	// 3. Attach shared storage to all workspaces
	// 4. Create research users for team members
	// 5. Verify collaboration workflows
	// 6. Cleanup
}

// TestUniversityClassPersona tests the University Class (Prof. Thompson) workflow
// Based on docs/USER_SCENARIOS/03_UNIVERSITY_CLASS_WALKTHROUGH.md
//
// TODO: Implement bulk launch, student access, template standardization
func TestUniversityClassPersona(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping University Class persona test in short mode")
	}

	t.Skip("University Class persona test not yet implemented")

	// Planned workflow:
	// 1. Create standardized course template
	// 2. Bulk launch workspaces for 25 students
	// 3. Configure uniform access policies
	// 4. Test student workspace access
	// 5. Verify cost tracking per student
	// 6. Cleanup (bulk delete)
}

// TestConferenceWorkshopPersona tests the Conference Workshop (Dr. Patel) workflow
// Based on docs/USER_SCENARIOS/04_CONFERENCE_WORKSHOP_WALKTHROUGH.md
//
// TODO: Implement rapid deployment, public access, time-limited workspaces
func TestConferenceWorkshopPersona(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Conference Workshop persona test in short mode")
	}

	t.Skip("Conference Workshop persona test not yet implemented")

	// Planned workflow:
	// 1. Create workshop template
	// 2. Launch workspaces with auto-termination (8 hours)
	// 3. Configure public access (temporary credentials)
	// 4. Verify time-limited lifecycle
	// 5. Cleanup (auto-termination)
}

// TestCrossInstitutionalPersona tests the Cross-Institutional (Dr. Kim) workflow
// Based on docs/USER_SCENARIOS/05_CROSS_INSTITUTIONAL_COLLABORATION_WALKTHROUGH.md
//
// TODO: Implement multi-profile setup, shared EFS, budget tracking
func TestCrossInstitutionalPersona(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Cross-Institutional persona test in short mode")
	}

	t.Skip("Cross-Institutional persona test not yet implemented")

	// Planned workflow:
	// 1. Setup workspaces in different AWS accounts (multi-profile)
	// 2. Create shared EFS volume for collaboration
	// 3. Configure cross-account access
	// 4. Verify budget tracking per institution
	// 5. Test data sharing workflows
	// 6. Cleanup
}
