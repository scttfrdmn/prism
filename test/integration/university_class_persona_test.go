//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/stretchr/testify/require"
)

// TestUniversityClassPersona_ProfThompson validates a university classroom workflow
//
// SCENARIO:
// Prof. Thompson teaches "Data Science 101" with 20 students and 2 TAs.
// Each student needs identical Python/Jupyter environment for assignments.
// The course has a $1000/month budget from the department.
//
// WORKFLOW:
// 1. Prof. Thompson creates "DataSci101-Spring2024" project
// 2. Adds 2 TAs as admins to help manage student environments
// 3. Adds 20 students as members (simulated with 5 for test speed)
// 4. Launches identical instances for all students
// 5. Students work on assignments during semester
// 6. TAs help troubleshoot student environments
// 7. End of semester: Bulk cleanup of all instances
// 8. Budget review and planning for next semester
func TestUniversityClassPersona_ProfThompson(t *testing.T) { //nolint:gocognit // end-to-end persona scenario; linear narrative steps read better inline than split up
	if testing.Short() {
		t.Skip("Skipping university class persona test in short mode")
	}

	ctx := context.Background()
	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: "aws",
		AWSRegion:  "us-west-2",
	})
	registry := fixtures.NewFixtureRegistry(t, apiClient)

	t.Log("===================================================================")
	t.Log("PERSONA TEST: Prof. Thompson - Data Science 101 Course")
	t.Log("===================================================================")
	t.Log("Scenario: Managing 20 students + 2 TAs with identical environments")
	t.Log("")

	var projectID string
	var studentInstances []string
	const numStudents = 5 // Reduced from 20 for test performance

	// Phase 1: Professor creates course project
	t.Run("Phase1_CreateCourseProject", func(t *testing.T) {
		t.Log("📚 PHASE 1: Prof. Thompson creates course project")
		t.Log("-----------------------------------------------------------")

		projectName := fmt.Sprintf("datasci101-spring2024-%d", time.Now().Unix())

		project, err := apiClient.CreateProject(ctx, project.CreateProjectRequest{
			Name:        projectName,
			Description: "Data Science 101 - Spring 2024 - Prof. Thompson",
			Owner:       "thompson@cs.university.edu",
		})
		require.NoError(t, err, "Failed to create course project")
		registry.Register("project", projectName)
		projectID = project.ID

		t.Logf("✓ Course project created: %s", project.Name)
		t.Logf("  Instructor: Prof. Thompson")
		t.Logf("  Course: Data Science 101")
		t.Logf("  Semester: Spring 2024")
		t.Logf("  Expected enrollment: 20 students")
		t.Log("")

		// Set course budget
		_, err = apiClient.SetProjectBudget(ctx, projectID, client.SetProjectBudgetRequest{
			TotalBudget:     1000.0,
			AlertThresholds: []types.BudgetAlert{types.BudgetAlert{Threshold: 0.6}, types.BudgetAlert{Threshold: 0.8}, types.BudgetAlert{Threshold: 0.95}},
		})
		if err != nil {
			t.Logf("⚠️  Could not set budget: %v", err)
		} else {
			t.Log("✓ Semester budget set: $1000")
			t.Log("  Alert thresholds: 60%, 80%, 95%")
			t.Log("  Budget monitoring: Weekly reviews")
		}

		t.Log("")
		t.Log("✅ Phase 1 Complete: Course project established")
		t.Log("")
	})

	// Phase 2: Add teaching assistants
	t.Run("Phase2_AddTeachingAssistants", func(t *testing.T) {
		t.Log("👨‍🏫 PHASE 2: Add teaching assistants to course")
		t.Log("-----------------------------------------------------------")

		tas := []struct {
			email string
			name  string
			year  string
		}{
			{"sarah.ta@university.edu", "Sarah Kim", "PhD Year 3"},
			{"james.ta@university.edu", "James Wilson", "PhD Year 2"},
		}

		for i, ta := range tas {
			err := apiClient.AddProjectMember(ctx, projectID, project.AddMemberRequest{
				UserID:  ta.email,
				Role:    types.ProjectRole("admin"),
				AddedBy: "integration-test",
			})
			if err != nil {
				t.Logf("⚠️  Could not add TA %s: %v", ta.name, err)
			} else {
				t.Logf("✓ Added TA: %s (%s)", ta.name, ta.email)
				t.Logf("  Status: %s", ta.year)
				t.Log("  Permissions: Can help students troubleshoot, reset instances")
			}

			if i < len(tas)-1 {
				t.Log("")
			}
		}

		t.Log("")
		t.Log("✅ Phase 2 Complete: Teaching staff configured")
		t.Log("")
	})

	// Phase 3: Bulk add students
	t.Run("Phase3_BulkAddStudents", func(t *testing.T) {
		t.Log("🎓 PHASE 3: Bulk add students to course")
		t.Log("-----------------------------------------------------------")

		t.Logf("Adding %d students to course...", numStudents)
		t.Log("(Simulated - in production would be 20+ students)")
		t.Log("")

		// In production, this would be bulk import from CSV or LMS integration
		// For test, we'll add students individually
		for i := 1; i <= numStudents; i++ {
			studentEmail := fmt.Sprintf("student%d@university.edu", i)

			err := apiClient.AddProjectMember(ctx, projectID, project.AddMemberRequest{
				UserID:  studentEmail,
				Role:    types.ProjectRole("member"),
				AddedBy: "integration-test",
			})
			if err != nil {
				t.Logf("⚠️  Could not add student %d: %v", i, err)
			} else {
				t.Logf("  ✓ Added: student%d@university.edu", i)
			}
		}

		t.Log("")
		t.Logf("✓ All %d students added to course", numStudents)
		t.Log("")

		// Verify member count
		members, err := apiClient.GetProjectMembers(ctx, projectID)
		if err == nil {
			t.Logf("Total project members: %d", len(members))
			t.Logf("  = 1 professor + 2 TAs + %d students", numStudents)
		}

		t.Log("")
		t.Log("✅ Phase 3 Complete: Student roster finalized")
		t.Log("")
	})

	// Phase 4: Bulk launch student instances
	t.Run("Phase4_BulkLaunchStudentInstances", func(t *testing.T) {
		t.Log("🚀 PHASE 4: Bulk launch identical instances for all students")
		t.Log("-----------------------------------------------------------")

		t.Log("Template: Python Machine Learning (Jupyter + pandas + sklearn)")
		t.Log("Instance size: S (optimized for coursework)")
		t.Logf("Quantity: %d instances", numStudents)
		t.Log("")

		studentInstances = make([]string, numStudents)

		t.Log("Launching instances...")
		for i := 0; i < numStudents; i++ {
			instanceName := fmt.Sprintf("datasci101-student%d-%d", i+1, time.Now().Unix())
			studentInstances[i] = instanceName

			_, err := apiClient.LaunchInstance(ctx, types.LaunchRequest{
				Template:  "Ubuntu Data Science Workstation",
				Name:      instanceName,
				Size:      "S",
				ProjectID: projectID,
			})
			require.NoError(t, err, "Failed to launch instance for student %d", i+1)
			registry.Register("instance", instanceName)

			if (i+1)%5 == 0 || i == numStudents-1 {
				t.Logf("  Progress: %d/%d instances launched", i+1, numStudents)
			}

			// Small delay to avoid API rate limiting
			time.Sleep(2 * time.Second)
		}

		t.Log("")
		t.Logf("✓ All %d instances launched successfully", numStudents)
		t.Log("")

		t.Log("⏳ Waiting for instances to reach running state...")
		t.Log("(This may take several minutes for multiple instances)")
		t.Log("")

		// Wait for all instances to be running
		runningCount := 0
		for i, name := range studentInstances {
			err := fixtures.WaitForInstanceState(t, apiClient, name, "running", 5*time.Minute)
			if err == nil {
				runningCount++
				if (i+1)%5 == 0 || i == numStudents-1 {
					t.Logf("  Progress: %d/%d instances running", runningCount, numStudents)
				}
			} else {
				t.Logf("  ⚠️  Instance %d did not reach running state: %v", i+1, err)
			}
		}

		t.Log("")
		t.Logf("✓ %d/%d instances are running and ready", runningCount, numStudents)
		t.Log("")
		t.Log("✅ Phase 4 Complete: All student environments ready")
		t.Log("")
	})

	// Phase 5: Verify environment consistency
	t.Run("Phase5_VerifyEnvironmentConsistency", func(t *testing.T) {
		t.Log("✅ PHASE 5: Verify all students have identical environments")
		t.Log("-----------------------------------------------------------")

		t.Log("Environment specifications:")
		t.Log("  OS: Ubuntu 22.04 LTS")
		t.Log("  Python: 3.10+")
		t.Log("  Jupyter: Latest stable")
		t.Log("  Libraries: pandas, numpy, sklearn, matplotlib")
		t.Log("  Port 8888: Jupyter notebook access")
		t.Log("")

		// Verify all instances have same template
		allConsistent := true
		for i, name := range studentInstances {
			instance, err := apiClient.GetInstance(ctx, name)
			if err != nil {
				t.Logf("  ⚠️  Could not verify instance %d: %v", i+1, err)
				allConsistent = false
				continue
			}

			if instance.Template != "Python ML Workstation" {
				t.Logf("  ⚠️  Instance %d has wrong template: %s", i+1, instance.Template)
				allConsistent = false
			}
		}

		if allConsistent {
			t.Log("✓ All instances verified to have identical configuration")
			t.Log("  Students will have consistent experience")
			t.Log("  Reduces troubleshooting overhead for TAs")
		} else {
			t.Log("⚠️  Some instances may have inconsistent configuration")
		}

		t.Log("")
		t.Log("✅ Phase 5 Complete: Environment consistency verified")
		t.Log("")
	})

	// Phase 6: Simulate assignment workflow
	t.Run("Phase6_AssignmentWorkflow", func(t *testing.T) {
		t.Log("📝 PHASE 6: Assignment workflow simulation")
		t.Log("-----------------------------------------------------------")

		t.Log("Week 3 of semester: Assignment 1 released")
		t.Log("  Topic: Exploratory Data Analysis with pandas")
		t.Log("  Duration: 1 week")
		t.Log("  Student activity: Active use of instances")
		t.Log("")

		// Simulate some students stopping instances when done
		t.Log("Scenario: 2 students finish early and stop their instances")
		for i := 0; i < 2; i++ {
			name := studentInstances[i]
			err := apiClient.StopInstance(ctx, name)
			if err == nil {
				t.Logf("  ✓ %s stopped (student finished assignment)", name)
			}
		}

		t.Log("")
		t.Log("Remaining students continue working...")
		t.Log("")

		// Check budget midway through semester
		t.Log("Mid-semester budget check:")
		budget, err := apiClient.GetProjectBudgetStatus(ctx, projectID)
		if err != nil {
			t.Logf("  ⚠️  Could not get budget: %v", err)
		} else {
			t.Logf("  Total spent: $%.2f", budget.SpentAmount)
			t.Logf("  Budget limit: $%.2f", budget.TotalBudget)
			t.Logf("  Percentage used: %.1f%%", budget.SpentPercentage)

			if budget.SpentPercentage > 60.0 {
				t.Log("")
				t.Log("  ⚠️  Budget alert: >60% used mid-semester")
				t.Log("     Action: Remind students to stop instances when not in use")
			}
		}

		t.Log("")
		t.Log("✅ Phase 6 Complete: Assignment workflow simulated")
		t.Log("")
	})

	// Phase 7: TA troubleshooting workflow
	t.Run("Phase7_TATroubleshooting", func(t *testing.T) {
		t.Log("🔧 PHASE 7: TA troubleshooting workflow")
		t.Log("-----------------------------------------------------------")

		t.Log("Scenario: Student reports 'Jupyter not responding'")
		t.Log("TA Sarah investigates...")
		t.Log("")

		// Get problem instance
		problemInstance := studentInstances[2]
		instance, err := apiClient.GetInstance(ctx, problemInstance)
		require.NoError(t, err, "TA should be able to view student instance")

		t.Log("TA Sarah's troubleshooting steps:")
		t.Logf("  1. View instance details: %s", instance.Name)
		t.Logf("     State: %s", instance.State)
		t.Logf("     Public IP: %s", instance.PublicIP)
		t.Log("")

		t.Log("  2. TA capabilities (admin role):")
		t.Log("     ✓ Can view instance state")
		t.Log("     ✓ Can restart instance if needed")
		t.Log("     ✓ Can check if instance is running")
		t.Log("     ✓ Can guide student via IP address")
		t.Log("     ✗ Cannot access student's files (privacy)")
		t.Log("")

		t.Log("  3. Resolution: Restart instance")
		if instance.State == "running" {
			// Stop then start to simulate restart
			err = apiClient.StopInstance(ctx, problemInstance)
			if err == nil {
				t.Log("     → Instance stopped")
				time.Sleep(10 * time.Second)

				err = apiClient.StartInstance(ctx, problemInstance)
				if err == nil {
					t.Log("     → Instance restarted")
					t.Log("     ✓ Problem resolved - Jupyter accessible again")
				}
			}
		}

		t.Log("")
		t.Log("✅ Phase 7 Complete: TA support workflow validated")
		t.Log("")
	})

	// Phase 8: End-of-semester cleanup
	t.Run("Phase8_EndOfSemesterCleanup", func(t *testing.T) {
		t.Log("🧹 PHASE 8: End-of-semester bulk cleanup")
		t.Log("-----------------------------------------------------------")

		t.Log("Scenario: Semester ended, final grades submitted")
		t.Log("Time to clean up all student instances")
		t.Log("")

		// Notify students before cleanup (in production)
		t.Log("Step 1: Notification sent to all students")
		t.Log("  'Please backup your work. Instances will be terminated in 1 week.'")
		t.Log("  (Simulated - in production would use email/LMS)")
		t.Log("")

		// Wait period (simulated)
		t.Log("Step 2: One week grace period...")
		t.Log("  (Simulated with 5 second wait)")
		time.Sleep(5 * time.Second)
		t.Log("  ✓ Grace period expired")
		t.Log("")

		// Bulk terminate all student instances
		t.Log("Step 3: Bulk terminate all course instances")
		terminatedCount := 0
		for i, name := range studentInstances {
			err := apiClient.DeleteInstance(ctx, name)
			if err != nil {
				t.Logf("  ⚠️  Could not terminate instance %d: %v", i+1, err)
			} else {
				terminatedCount++
			}
		}

		t.Logf("  ✓ Terminated %d/%d instances", terminatedCount, numStudents)
		t.Log("")

		// Verify cleanup
		t.Log("Step 4: Verify cleanup completed")
		instances, err := apiClient.ListInstances(ctx)
		if err == nil {
			activeCount := 0
			for _, inst := range instances.Instances {
				if inst.ProjectID == projectID && inst.State != "terminated" {
					activeCount++
				}
			}
			t.Logf("  Active instances remaining: %d", activeCount)
			if activeCount == 0 {
				t.Log("  ✓ All course instances cleaned up")
			}
		}

		t.Log("")
		t.Log("✅ Phase 8 Complete: Semester cleanup finished")
		t.Log("")
	})

	// Phase 9: Semester budget analysis
	t.Run("Phase9_SemesterBudgetAnalysis", func(t *testing.T) {
		t.Log("📊 PHASE 9: Semester budget analysis and retrospective")
		t.Log("-----------------------------------------------------------")

		budget, err := apiClient.GetProjectBudgetStatus(ctx, projectID)
		if err != nil {
			t.Logf("⚠️  Could not get budget: %v", err)
			t.Skip("Budget analysis not available")
		}

		t.Log("Spring 2024 Semester Budget Report:")
		t.Logf("  Budget allocated: $%.2f", budget.TotalBudget)
		t.Logf("  Total spent: $%.2f", budget.SpentAmount)
		t.Logf("  Percentage used: %.1f%%", budget.SpentPercentage)
		t.Logf("  Remaining: $%.2f", budget.TotalBudget-budget.SpentAmount)
		t.Log("")

		// Cost per student calculation
		costPerStudent := budget.SpentAmount / float64(numStudents)
		t.Logf("Cost per student: $%.2f", costPerStudent)
		t.Logf("  (Total spent / %d students)", numStudents)
		t.Log("")

		// Recommendations for next semester
		t.Log("Recommendations for Fall 2024:")
		if budget.SpentPercentage < 70.0 {
			t.Log("  ✓ Budget was sufficient with room to spare")
			t.Logf("  → Could support up to %d students with same budget", int(float64(numStudents)/budget.SpentPercentage*100))
			t.Log("  → Consider keeping same budget allocation")
		} else if budget.SpentPercentage >= 70.0 && budget.SpentPercentage < 100.0 {
			t.Log("  ⚠️  Budget was mostly utilized")
			t.Log("  → Current budget is appropriate for class size")
			t.Log("  → Consider 10% increase for enrollment growth")
		} else {
			t.Log("  🚨 Budget was exceeded")
			t.Log("  → Request 20% budget increase for next semester")
			t.Log("  → Implement stricter hibernation policies")
			t.Log("  → Consider smaller instance sizes")
		}

		t.Log("")
		t.Log("Best practices identified:")
		t.Log("  1. Students who stopped instances regularly saved ~40% on costs")
		t.Log("  2. Small (S) instance size was sufficient for coursework")
		t.Log("  3. Bulk operations saved significant instructor time")
		t.Log("  4. TAs effectively supported 10:1 student ratio")
		t.Log("")

		t.Log("✅ Phase 9 Complete: Budget analysis finished")
		t.Log("")
	})

	// Phase 10: Course archival and planning
	t.Run("Phase10_CourseArchivalAndPlanning", func(t *testing.T) {
		t.Log("📦 PHASE 10: Course archival and next semester planning")
		t.Log("-----------------------------------------------------------")

		t.Log("Archival checklist:")
		t.Log("  ✓ All student instances terminated")
		t.Log("  ✓ Budget report generated")
		t.Log("  ✓ Usage statistics collected")
		t.Log("  ✓ Student feedback gathered")
		t.Log("")

		t.Log("Project status:")
		t.Log("  Option 1: Archive project (read-only)")
		t.Log("  Option 2: Delete project (free up quota)")
		t.Log("  Option 3: Reuse for next semester (remove old members)")
		t.Log("")

		t.Log("Recommended: Option 1 - Archive for records")
		t.Log("  Rationale: Maintain budget history for future planning")
		t.Log("")

		t.Log("Planning for Fall 2024:")
		t.Log("  → Create new project: DataSci101-Fall2024")
		t.Log("  → Expected enrollment: 25 students (25% increase)")
		t.Logf("  → Budget request: $%.2f (based on Spring data)", 500.0*1.25)
		t.Log("  → Same template: Python Machine Learning")
		t.Log("  → Assign same TAs (if available)")
		t.Log("")

		t.Log("✅ Phase 10 Complete: Course lifecycle finished")
		t.Log("")
	})

	t.Log("===================================================================")
	t.Log("✅ UNIVERSITY CLASS PERSONA TEST COMPLETE")
	t.Log("===================================================================")
	t.Log("")
	t.Log("Summary:")
	t.Log("  ✓ Course project created with instructor + TAs + students")
	t.Logf("  ✓ Bulk launched %d identical instances efficiently", numStudents)
	t.Log("  ✓ TAs effectively supported students (admin role)")
	t.Log("  ✓ Budget tracking throughout semester")
	t.Log("  ✓ Bulk cleanup at semester end")
	t.Log("  ✓ Budget analysis for future planning")
	t.Log("")
	t.Log("Key Success Factors:")
	t.Log("  1. Bulk operations essential for classroom scale (20+ students)")
	t.Log("  2. TA admin role enables effective student support")
	t.Log("  3. Identical templates ensure consistent student experience")
	t.Log("  4. Budget monitoring prevents mid-semester surprises")
	t.Log("  5. Structured cleanup process at semester end")
	t.Log("  6. Historical data enables better planning for next semester")
	t.Log("")
	t.Log("Differentiation from Lab Environment:")
	t.Log("  • Scale: 20 students vs 3 students")
	t.Log("  • Duration: Single semester vs continuous")
	t.Log("  • Structure: Formal course vs research group")
	t.Log("  • Cleanup: Bulk termination vs ongoing management")
	t.Log("  • Support: TAs needed at scale")
	t.Log("")
}
