package fixtures

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
)

// FixtureRegistry tracks test resources for automatic cleanup
// Uses t.Cleanup() to ensure resources are always cleaned up, even on test failure
type FixtureRegistry struct {
	t             *testing.T
	client        client.PrismAPI
	instances     []string
	backups       []string
	volumes       []string
	ebsStorages   []string
	profiles      []string
	projects      []string
	budgets       []string // v0.6.2: Budget IDs for cleanup
	allocations   []string // v0.6.2: Allocation IDs for cleanup
	invitations   []string // v0.6.2: Invitation IDs for cleanup (Issue #383)
	courses       []string // v0.14.0: Course IDs for cleanup
	workshops     []string // v0.18.0: Workshop IDs for cleanup
	cleanupCalled bool
}

// NewFixtureRegistry creates a new fixture registry with automatic cleanup
// The cleanup function is registered with t.Cleanup() to run after the test
func NewFixtureRegistry(t *testing.T, apiClient client.PrismAPI) *FixtureRegistry {
	t.Helper()

	registry := &FixtureRegistry{
		t:      t,
		client: apiClient,
	}

	// Register cleanup function to run after test completes
	t.Cleanup(func() {
		if !registry.cleanupCalled {
			registry.cleanup()
		}
	})

	return registry
}

// Register adds a resource ID to the registry for cleanup
func (r *FixtureRegistry) Register(resourceType, id string) {
	r.t.Helper()

	switch resourceType {
	case "instance":
		r.instances = append(r.instances, id)
	case "backup":
		r.backups = append(r.backups, id)
	case "volume":
		r.volumes = append(r.volumes, id)
	case "ebs_storage":
		r.ebsStorages = append(r.ebsStorages, id)
	case "profile":
		r.profiles = append(r.profiles, id)
	case "project":
		r.projects = append(r.projects, id)
	case "budget": // v0.6.2
		r.budgets = append(r.budgets, id)
	case "allocation": // v0.6.2
		r.allocations = append(r.allocations, id)
	case "invitation": // v0.6.2 (Issue #383)
		r.invitations = append(r.invitations, id)
	case "course": // v0.14.0
		r.courses = append(r.courses, id)
	case "workshop": // v0.18.0
		r.workshops = append(r.workshops, id)
	default:
		r.t.Logf("Warning: Unknown resource type %q", resourceType)
	}
}

// cleanupResourceItems deletes each resource by ID, logging success or failure.
func (r *FixtureRegistry) cleanupResourceItems(label string, ids []string, del func(string) error) {
	for _, id := range ids {
		if err := del(id); err != nil {
			r.t.Logf("Warning: Failed to cleanup %s %s: %v", label, id, err)
		} else {
			r.t.Logf("Cleaned up %s: %s", label, id)
		}
	}
}

// RegisterCourse adds a course ID to the registry for automatic cleanup.
func (r *FixtureRegistry) RegisterCourse(courseID string) {
	r.t.Helper()
	r.courses = append(r.courses, courseID)
}

// RegisterWorkshop adds a workshop ID to the registry for automatic cleanup.
func (r *FixtureRegistry) RegisterWorkshop(workshopID string) {
	r.t.Helper()
	r.workshops = append(r.workshops, workshopID)
}

// cleanup removes all registered test resources in the correct order
// Order: allocations → budgets → invitations → backups → instances → EBS volumes → EFS volumes → projects → courses → profiles
func (r *FixtureRegistry) cleanup() {
	r.t.Helper()
	r.cleanupCalled = true

	ctx := context.Background()
	r.t.Log("Cleaning up test fixtures...")

	// Clean up allocations first (must be deleted before budgets) - v0.6.2
	r.cleanupResourceItems("allocation", r.allocations, func(id string) error { return r.client.DeleteAllocation(ctx, id) })

	// Clean up budgets - v0.6.2
	r.cleanupResourceItems("budget", r.budgets, func(id string) error { return r.client.DeleteBudget(ctx, id) })

	// Clean up invitations (may already be accepted/declined/expired) - v0.6.2 (Issue #383)
	for _, invitationID := range r.invitations {
		if err := r.client.RevokeInvitation(ctx, invitationID); err != nil {
			r.t.Logf("Warning: Failed to cleanup invitation %s: %v (may already be accepted/declined)", invitationID, err)
		} else {
			r.t.Logf("Cleaned up invitation: %s", invitationID)
		}
	}

	// Clean up backups (fastest, no dependencies)
	for _, backupID := range r.backups {
		result, err := r.client.DeleteBackup(ctx, backupID)
		if err != nil {
			r.t.Logf("Warning: Failed to cleanup backup %s: %v", backupID, err)
		} else {
			r.t.Logf("Cleaned up backup: %s (savings: $%.2f/month)", backupID, result.StorageSavingsMonthly)
		}
	}

	// Clean up instances (must be deleted before volumes can be deleted)
	for _, instanceName := range r.instances {
		if err := r.client.DeleteInstance(ctx, instanceName); err != nil {
			r.t.Logf("Warning: Failed to cleanup instance %s: %v", instanceName, err)
		} else {
			r.t.Logf("Cleaned up instance: %s", instanceName)
			// Give AWS time to begin termination
			time.Sleep(2 * time.Second)
		}
	}

	// Clean up EBS and EFS volumes, then projects
	r.cleanupResourceItems("EBS volume", r.ebsStorages, func(id string) error { return r.client.DeleteStorage(ctx, id) })
	r.cleanupResourceItems("EFS volume", r.volumes, func(id string) error { return r.client.DeleteVolume(ctx, id) })
	r.cleanupResourceItems("project", r.projects, func(id string) error { return r.client.DeleteProject(ctx, id) })
	r.cleanupResourceItems("course", r.courses, func(id string) error { return r.client.DeleteCourse(ctx, id) })
	r.cleanupResourceItems("workshop", r.workshops, func(id string) error { return r.client.DeleteWorkshop(ctx, id) })

	// Note: Profiles are managed locally via pkg/profile package, not through daemon API
	if len(r.profiles) > 0 {
		r.t.Logf("Note: %d profiles registered but profile cleanup not implemented (profiles managed locally)", len(r.profiles))
	}

	r.t.Log("Fixture cleanup complete")
}

// Cleanup manually triggers cleanup (useful for defer statements)
// Note: Cleanup is automatically called via t.Cleanup(), so this is optional
func (r *FixtureRegistry) Cleanup() {
	r.cleanup()
}
