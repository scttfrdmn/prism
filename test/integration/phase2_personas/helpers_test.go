//go:build integration
// +build integration

package phase2_personas

import (
	"context"
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/test/integration"
)

// getProjectMember returns the project member with the given user ID, failing
// the test if it is not present. AddProjectMember returns only an error, so
// tests that assert on the added member's role look it up afterwards.
func getProjectMember(t *testing.T, ctx *integration.TestContext, projectID, userID string) types.ProjectMember {
	t.Helper()

	members, err := ctx.Client.GetProjectMembers(context.Background(), projectID)
	integration.AssertNoError(t, err, "Failed to list project members")

	for _, m := range members {
		if m.UserID == userID {
			return m
		}
	}
	t.Fatalf("project member %q not found in project %s", userID, projectID)
	return types.ProjectMember{}
}
