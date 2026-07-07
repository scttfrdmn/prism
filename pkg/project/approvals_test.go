package project

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/prism/pkg/seam"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
)

// newTestApprovalManager creates an ApprovalManager rooted in a temp directory
// so that tests do not read or write to the real ~/.prism/approvals.json.
func newTestApprovalManager(t *testing.T) *ApprovalManager {
	t.Helper()
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", originalHome) })
	_ = os.Setenv("HOME", tempDir)
	am, err := NewApprovalManager()
	require.NoError(t, err)
	return am
}

// TestApprovalManager_ScopeIsolation proves the multi-tenant guarantee: two managers over the
// SAME store but different scopes see only their own records — the property that lets one shared
// cloud table serve every tenant without leakage (design §4, §6.2).
func TestApprovalManager_ScopeIsolation(t *testing.T) {
	store := filestore.New[ApprovalRequest](t.TempDir())
	harvard := NewApprovalManagerForScope(store, seam.Scope{Tenant: "harvard", PI: "curie"})
	mit := NewApprovalManagerForScope(store, seam.Scope{Tenant: "mit", PI: "bohr"})

	hReq, err := harvard.Submit("proj-h", "curie", ApprovalTypeGPUInstance, nil, "harvard work")
	require.NoError(t, err)
	_, err = mit.Submit("proj-m", "bohr", ApprovalTypeEmergency, nil, "mit work")
	require.NoError(t, err)

	// Each tenant lists only its own request.
	hList, err := harvard.List("", "")
	require.NoError(t, err)
	assert.Len(t, hList, 1)
	assert.Equal(t, "proj-h", hList[0].ProjectID)

	mList, err := mit.List("", "")
	require.NoError(t, err)
	assert.Len(t, mList, 1)
	assert.Equal(t, "proj-m", mList[0].ProjectID)

	// MIT cannot Get Harvard's request id (different scope partition).
	_, err = mit.Get(hReq.ID)
	assert.Error(t, err)
}

func TestNewApprovalManager(t *testing.T) {
	am := newTestApprovalManager(t)
	assert.NotNil(t, am)
	assert.NotNil(t, am.store)
	// A fresh manager lists no requests and errors on no particular id.
	results, err := am.List("", "")
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestApprovalManager_Submit(t *testing.T) {
	am := newTestApprovalManager(t)

	details := map[string]interface{}{"instance_type": "p3.2xlarge"}
	req, err := am.Submit("proj-1", "alice", ApprovalTypeGPUInstance, details, "need GPU for training")
	require.NoError(t, err)
	require.NotNil(t, req)

	// ID must be assigned
	assert.NotEmpty(t, req.ID)

	// Status must be pending
	assert.Equal(t, ApprovalStatusPending, req.Status)

	// ExpiresAt must be ~7 days in the future (within 1 minute tolerance)
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, req.ExpiresAt, time.Minute)

	// Fields must be stored correctly
	assert.Equal(t, "proj-1", req.ProjectID)
	assert.Equal(t, "alice", req.RequestedBy)
	assert.Equal(t, ApprovalTypeGPUInstance, req.Type)
	assert.Equal(t, "need GPU for training", req.Reason)
}

func TestApprovalManager_Submit_AllTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  ApprovalType
	}{
		{"gpu_instance", ApprovalTypeGPUInstance},
		{"expensive_instance", ApprovalTypeExpensiveInstance},
		{"budget_overage", ApprovalTypeBudgetOverage},
		{"emergency", ApprovalTypeEmergency},
		{"sub_budget", ApprovalTypeSubBudget},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			am := newTestApprovalManager(t)
			req, err := am.Submit("proj-1", "user1", tc.typ, nil, "reason")
			require.NoError(t, err)
			assert.Equal(t, tc.typ, req.Type)
			assert.Equal(t, ApprovalStatusPending, req.Status)
		})
	}
}

func TestApprovalManager_Get(t *testing.T) {
	am := newTestApprovalManager(t)

	submitted, err := am.Submit("proj-1", "alice", ApprovalTypeSubBudget, nil, "delegation request")
	require.NoError(t, err)

	got, err := am.Get(submitted.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, submitted.ID, got.ID)
	assert.Equal(t, submitted.ProjectID, got.ProjectID)
	assert.Equal(t, submitted.Type, got.Type)
}

func TestApprovalManager_Get_NotFound(t *testing.T) {
	am := newTestApprovalManager(t)

	got, err := am.Get("nonexistent-id")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestApprovalManager_List_ByProject(t *testing.T) {
	am := newTestApprovalManager(t)

	_, err := am.Submit("proj-A", "alice", ApprovalTypeGPUInstance, nil, "reason")
	require.NoError(t, err)
	_, err = am.Submit("proj-A", "bob", ApprovalTypeEmergency, nil, "reason")
	require.NoError(t, err)
	_, err = am.Submit("proj-B", "carol", ApprovalTypeBudgetOverage, nil, "reason")
	require.NoError(t, err)

	results, err := am.List("proj-A", "")
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "proj-A", r.ProjectID)
	}

	results, err = am.List("proj-B", "")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "proj-B", results[0].ProjectID)
}

func TestApprovalManager_List_ByStatus(t *testing.T) {
	am := newTestApprovalManager(t)

	req1, err := am.Submit("proj-1", "alice", ApprovalTypeGPUInstance, nil, "reason")
	require.NoError(t, err)
	_, err = am.Submit("proj-1", "bob", ApprovalTypeEmergency, nil, "reason")
	require.NoError(t, err)

	// Approve the first request
	_, err = am.Approve(req1.ID, "admin", "looks good")
	require.NoError(t, err)

	pending, err := am.List("", ApprovalStatusPending)
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, ApprovalStatusPending, pending[0].Status)

	approved, err := am.List("", ApprovalStatusApproved)
	require.NoError(t, err)
	assert.Len(t, approved, 1)
	assert.Equal(t, ApprovalStatusApproved, approved[0].Status)
}

func TestApprovalManager_List_Empty(t *testing.T) {
	am := newTestApprovalManager(t)

	results, err := am.List("no-such-project", "")
	require.NoError(t, err)
	// Must return zero results and no error (slice may be nil or empty)
	assert.Len(t, results, 0)
}

func TestApprovalManager_Approve(t *testing.T) {
	am := newTestApprovalManager(t)

	req, err := am.Submit("proj-1", "alice", ApprovalTypeExpensiveInstance, nil, "large instance needed")
	require.NoError(t, err)

	approved, err := am.Approve(req.ID, "admin-user", "approved for grant work")
	require.NoError(t, err)
	require.NotNil(t, approved)

	assert.Equal(t, ApprovalStatusApproved, approved.Status)
	assert.Equal(t, "admin-user", approved.ReviewedBy)
	assert.Equal(t, "approved for grant work", approved.ReviewNote)
	assert.NotNil(t, approved.ReviewedAt)

	// Verify the stored state matches
	stored, err := am.Get(req.ID)
	require.NoError(t, err)
	assert.Equal(t, ApprovalStatusApproved, stored.Status)
}

func TestApprovalManager_Deny(t *testing.T) {
	am := newTestApprovalManager(t)

	req, err := am.Submit("proj-1", "alice", ApprovalTypeBudgetOverage, nil, "went over budget")
	require.NoError(t, err)

	denied, err := am.Deny(req.ID, "admin-user", "budget already exhausted")
	require.NoError(t, err)
	require.NotNil(t, denied)

	assert.Equal(t, ApprovalStatusDenied, denied.Status)
	assert.Equal(t, "admin-user", denied.ReviewedBy)
	assert.Equal(t, "budget already exhausted", denied.ReviewNote)
	assert.NotNil(t, denied.ReviewedAt)

	// Verify the stored state matches
	stored, err := am.Get(req.ID)
	require.NoError(t, err)
	assert.Equal(t, ApprovalStatusDenied, stored.Status)
}

func TestApprovalManager_Approve_AlreadyReviewed(t *testing.T) {
	am := newTestApprovalManager(t)

	req, err := am.Submit("proj-1", "alice", ApprovalTypeGPUInstance, nil, "need GPU")
	require.NoError(t, err)

	// First approval succeeds
	_, err = am.Approve(req.ID, "admin", "ok")
	require.NoError(t, err)

	// Second approval on same request must fail (not pending anymore)
	_, err = am.Approve(req.ID, "admin", "trying again")
	assert.Error(t, err)

	// Deny on already-approved request must also fail
	_, err = am.Deny(req.ID, "admin", "changed my mind")
	assert.Error(t, err)
}

func TestApprovalManager_PruneExpired(t *testing.T) {
	am := newTestApprovalManager(t)

	_, err := am.Submit("proj-1", "alice", ApprovalTypeGPUInstance, nil, "reason")
	require.NoError(t, err)
	_, err = am.Submit("proj-1", "bob", ApprovalTypeEmergency, nil, "reason")
	require.NoError(t, err)

	// Bypass Submit's 7-day expiry by backdating the stored requests through the seam store.
	listed, err := am.List("", "")
	require.NoError(t, err)
	for _, req := range listed {
		req.ExpiresAt = time.Now().Add(-1 * time.Hour)
		require.NoError(t, am.store.Put(context.Background(), am.scope, req.ID, *req))
	}

	count, err := am.PruneExpired()
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// All requests should now be expired
	results, err := am.List("", ApprovalStatusExpired)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestApprovalManager_PruneExpired_None(t *testing.T) {
	am := newTestApprovalManager(t)

	// Submit a request with default 7-day expiry (not yet expired)
	_, err := am.Submit("proj-1", "alice", ApprovalTypeSubBudget, nil, "reason")
	require.NoError(t, err)

	count, err := am.PruneExpired()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestApprovalManager_MigratesLegacyFile(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", originalHome) })
	_ = os.Setenv("HOME", tempDir)

	// Seed a pre-seam flat approvals.json (the old on-disk format).
	stateDir := tempDir + "/.prism"
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	legacy := `{"legacy-1":{"id":"legacy-1","project_id":"proj-old","requested_by":"alice","type":"gpu_instance","status":"pending","reason":"old request","created_at":"2026-01-01T00:00:00Z","expires_at":"2030-01-01T00:00:00Z"}}`
	require.NoError(t, os.WriteFile(stateDir+"/approvals.json", []byte(legacy), 0o644))

	// Constructing the manager migrates it into the seam.
	am, err := NewApprovalManager()
	require.NoError(t, err)

	got, err := am.Get("legacy-1")
	require.NoError(t, err)
	assert.Equal(t, "proj-old", got.ProjectID)
	assert.Equal(t, ApprovalStatusPending, got.Status)

	// The legacy file is retired so it won't re-import.
	_, statErr := os.Stat(stateDir + "/approvals.json")
	assert.True(t, os.IsNotExist(statErr), "legacy approvals.json should be renamed aside")

	// A second manager (same HOME) sees the migrated record and does not double-import.
	am2, err := NewApprovalManager()
	require.NoError(t, err)
	all, err := am2.List("", "")
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestApprovalManager_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", originalHome) })
	_ = os.Setenv("HOME", tempDir)

	// Create first manager and submit a request
	am1, err := NewApprovalManager()
	require.NoError(t, err)

	submitted, err := am1.Submit("proj-persist", "alice", ApprovalTypeGPUInstance, nil, "persist test")
	require.NoError(t, err)

	// Create a second manager pointing to the same tempDir (same HOME)
	am2, err := NewApprovalManager()
	require.NoError(t, err)

	// The second manager should have loaded the persisted request
	got, err := am2.Get(submitted.ID)
	require.NoError(t, err)
	assert.Equal(t, submitted.ID, got.ID)
	assert.Equal(t, "proj-persist", got.ProjectID)
	assert.Equal(t, ApprovalStatusPending, got.Status)
}
