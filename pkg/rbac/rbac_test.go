package rbac

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testManager creates a Manager whose rbac.json lives in a temp dir.
func testManager(t *testing.T) *Manager {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	m, err := NewManager()
	require.NoError(t, err)
	return m
}

// ── Construction ──────────────────────────────────────────────────────────

func TestNewManager_BuiltinRoles(t *testing.T) {
	m := testManager(t)

	roles := m.ListRoles()
	assert.Len(t, roles, 4, "should have 4 built-in roles")

	ids := make(map[string]bool)
	for _, r := range roles {
		ids[r.ID] = true
	}
	assert.True(t, ids[RoleAdmin])
	assert.True(t, ids[RoleResearcher])
	assert.True(t, ids[RoleStudent])
	assert.True(t, ids[RoleViewer])
}

func TestNewManager_EmptyAssignments(t *testing.T) {
	m := testManager(t)
	assert.Empty(t, m.ListUserAssignments())
}

// ── AssignRole ────────────────────────────────────────────────────────────

func TestAssignRole_Valid(t *testing.T) {
	m := testManager(t)

	err := m.AssignRole("alice", RoleAdmin, "system")
	require.NoError(t, err)

	role := m.GetUserRole("alice")
	assert.Equal(t, RoleAdmin, role.ID)
}

func TestAssignRole_UnknownRole(t *testing.T) {
	m := testManager(t)

	err := m.AssignRole("bob", "superuser", "system")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ── RemoveRole ────────────────────────────────────────────────────────────

func TestRemoveRole(t *testing.T) {
	m := testManager(t)

	require.NoError(t, m.AssignRole("carol", RoleAdmin, "system"))
	require.NoError(t, m.RemoveRole("carol"))

	// Falls back to default researcher
	role := m.GetUserRole("carol")
	assert.Equal(t, RoleResearcher, role.ID)
}

func TestRemoveRole_NonExistent(t *testing.T) {
	m := testManager(t)
	// Removing an unassigned user should not error
	assert.NoError(t, m.RemoveRole("nobody"))
}

// ── GetUserRole ───────────────────────────────────────────────────────────

func TestGetUserRole_Default(t *testing.T) {
	m := testManager(t)

	// Unassigned users default to researcher
	role := m.GetUserRole("unknown-user")
	require.NotNil(t, role)
	assert.Equal(t, RoleResearcher, role.ID)
}

func TestGetUserRole_Assigned(t *testing.T) {
	m := testManager(t)

	require.NoError(t, m.AssignRole("dave", RoleStudent, "system"))
	role := m.GetUserRole("dave")
	assert.Equal(t, RoleStudent, role.ID)
}

// ── CanPerformAction ──────────────────────────────────────────────────────

func TestCanPerformAction_Admin(t *testing.T) {
	m := testManager(t)
	require.NoError(t, m.AssignRole("admin-user", RoleAdmin, "system"))

	allowed, reason := m.CanPerformAction("admin-user", ActionStorageDelete)
	assert.True(t, allowed)
	assert.Contains(t, reason, "full access")
}

func TestCanPerformAction_ExactMatch(t *testing.T) {
	m := testManager(t)
	// Student has instances:launch
	require.NoError(t, m.AssignRole("student1", RoleStudent, "system"))

	allowed, reason := m.CanPerformAction("student1", ActionInstancesLaunch)
	assert.True(t, allowed)
	assert.NotEmpty(t, reason)
}

func TestCanPerformAction_WildcardSegment(t *testing.T) {
	// Create a custom role with instances:* permission
	m := testManager(t)
	m.roles["instance-manager"] = &Role{
		ID:          "instance-manager",
		Name:        "Instance Manager",
		Permissions: []string{"instances:*"},
	}
	require.NoError(t, m.AssignRole("mgr1", "instance-manager", "system"))

	allowed, _ := m.CanPerformAction("mgr1", ActionInstancesTerminate)
	assert.True(t, allowed)

	allowed, _ = m.CanPerformAction("mgr1", ActionStorageDelete)
	assert.False(t, allowed)
}

func TestCanPerformAction_Denied(t *testing.T) {
	m := testManager(t)
	// Student cannot delete storage
	require.NoError(t, m.AssignRole("student2", RoleStudent, "system"))

	allowed, reason := m.CanPerformAction("student2", ActionStorageDelete)
	assert.False(t, allowed)
	assert.Contains(t, reason, "not permitted")
}

func TestCanPerformAction_FullWildcard(t *testing.T) {
	m := testManager(t)
	require.NoError(t, m.AssignRole("super", RoleAdmin, "system"))

	// Admin's "*" should allow any arbitrary action
	allowed, _ := m.CanPerformAction("super", "custom:exotic:action")
	assert.True(t, allowed)
}

func TestCanPerformAction_Viewer_ReadOnly(t *testing.T) {
	m := testManager(t)
	require.NoError(t, m.AssignRole("viewer1", RoleViewer, "system"))

	// Can view instances
	allowed, _ := m.CanPerformAction("viewer1", ActionInstancesView)
	assert.True(t, allowed)

	// Cannot launch
	allowed, _ = m.CanPerformAction("viewer1", ActionInstancesLaunch)
	assert.False(t, allowed)
}

// ── ListRoles ─────────────────────────────────────────────────────────────

func TestListRoles(t *testing.T) {
	m := testManager(t)
	roles := m.ListRoles()
	assert.Len(t, roles, 4)
}

// ── ListUserAssignments ───────────────────────────────────────────────────

func TestListUserAssignments(t *testing.T) {
	m := testManager(t)

	require.NoError(t, m.AssignRole("u1", RoleAdmin, "system"))
	require.NoError(t, m.AssignRole("u2", RoleStudent, "system"))

	assignments := m.ListUserAssignments()
	assert.Len(t, assignments, 2)
	assert.Equal(t, RoleAdmin, assignments["u1"])
	assert.Equal(t, RoleStudent, assignments["u2"])
}

func TestListUserAssignments_ReturnsCopy(t *testing.T) {
	m := testManager(t)
	require.NoError(t, m.AssignRole("u1", RoleAdmin, "system"))

	a := m.ListUserAssignments()
	a["u1"] = "tampered"

	// Original must be unchanged
	assert.Equal(t, RoleAdmin, m.ListUserAssignments()["u1"])
}

// ── Persistence ───────────────────────────────────────────────────────────

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	m1, err := NewManager()
	require.NoError(t, err)
	require.NoError(t, m1.AssignRole("persist-user", RoleAdmin, "system"))

	// Create a second manager from the same HOME — should load the persisted assignment
	m2, err := NewManager()
	require.NoError(t, err)

	role := m2.GetUserRole("persist-user")
	assert.Equal(t, RoleAdmin, role.ID)
}

func TestMigratesLegacyFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Seed a pre-seam rbac.json (the old single-state format).
	stateDir := filepath.Join(tmpDir, ".prism")
	require.NoError(t, os.MkdirAll(stateDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "rbac.json"),
		[]byte(`{"user_roles":{"legacy-user":"`+RoleAdmin+`"}}`), 0o600))

	m, err := NewManager()
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, m.GetUserRole("legacy-user").ID)

	// Legacy file retired; a second manager still sees the binding and doesn't double-import.
	_, statErr := os.Stat(filepath.Join(stateDir, "rbac.json"))
	assert.True(t, os.IsNotExist(statErr), "legacy rbac.json should be renamed aside")

	m2, err := NewManager()
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, m2.GetUserRole("legacy-user").ID)
}
