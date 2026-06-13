// Package project provides approval workflow management for Prism.
//
// This file implements the approval system introduced in v0.12.0, enabling
// governance workflows for expensive resources, budget overages, and
// delegated budget access.
//
// Approval Types:
//   - gpu_instance: Required before launching GPU/p*/g* instance types (#149)
//   - expensive_instance: Required for instances costing >$2/hr (#149)
//   - budget_overage: Required when spending would exceed budget (#157)
//   - emergency: Emergency budget increase request (#157)
//   - sub_budget: Sub-budget delegation request (#148)
//
// Owner/Admin users bypass approval gates automatically.
// Member/Viewer users receive HTTP 202 with a request ID when approval is needed.
package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/scttfrdmn/prism/pkg/seam"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
)

// ApprovalType identifies the kind of approval being requested
type ApprovalType string

const (
	// ApprovalTypeGPUInstance requires approval before launching GPU/p*/g* instances (#149)
	ApprovalTypeGPUInstance ApprovalType = "gpu_instance"

	// ApprovalTypeExpensiveInstance requires approval for instances >$2/hr (#149)
	ApprovalTypeExpensiveInstance ApprovalType = "expensive_instance"

	// ApprovalTypeBudgetOverage requires approval when spending would exceed budget
	ApprovalTypeBudgetOverage ApprovalType = "budget_overage"

	// ApprovalTypeEmergency requests an emergency budget increase (#157)
	ApprovalTypeEmergency ApprovalType = "emergency"

	// ApprovalTypeSubBudget requests sub-budget delegation from PI to member (#148)
	ApprovalTypeSubBudget ApprovalType = "sub_budget"
)

// ApprovalStatus represents the current state of an approval request
type ApprovalStatus string

const (
	// ApprovalStatusPending indicates the request is awaiting review
	ApprovalStatusPending ApprovalStatus = "pending"

	// ApprovalStatusApproved indicates the request was approved
	ApprovalStatusApproved ApprovalStatus = "approved"

	// ApprovalStatusDenied indicates the request was denied
	ApprovalStatusDenied ApprovalStatus = "denied"

	// ApprovalStatusExpired indicates the request expired before being reviewed
	ApprovalStatusExpired ApprovalStatus = "expired"
)

// ApprovalRequest is a single governance approval request
type ApprovalRequest struct {
	// ID is the unique request identifier
	ID string `json:"id"`

	// ProjectID is the project this request is associated with
	ProjectID string `json:"project_id"`

	// RequestedBy is the user who submitted the request
	RequestedBy string `json:"requested_by"`

	// Type is the kind of approval needed
	Type ApprovalType `json:"type"`

	// Status is the current state of the request
	Status ApprovalStatus `json:"status"`

	// Details contains type-specific context (instance_type, amount, etc.)
	Details map[string]interface{} `json:"details"`

	// Reason is the requester's justification
	Reason string `json:"reason"`

	// ReviewedBy is the approver's user ID (empty until reviewed)
	ReviewedBy string `json:"reviewed_by,omitempty"`

	// ReviewNote is an optional note from the approver
	ReviewNote string `json:"review_note,omitempty"`

	// CreatedAt is when the request was submitted
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is when the pending request auto-expires (default: 7 days)
	ExpiresAt time.Time `json:"expires_at"`

	// ReviewedAt is when the request was approved or denied
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

// ApprovalManager handles approval request lifecycle, persisting through the seam (design §5).
//
// Persistence is no longer inlined file I/O: it goes through a seam.Store[ApprovalRequest], so the
// SAME manager logic backs the desktop (filestore) and the shared cloud (dynamostore). A record
// written by prp (web) is the same record Prism (desktop) reads — that is the shared-state
// guarantee (design §4), not a sync protocol.
//
// Approvals are not yet multi-tenant in Prism, so every record lives under the zero Scope
// (approvalScope). When Prism grows per-tenant approvals, the scope becomes the caller's Principal
// and nothing else here changes — that is the point of the seam.
type ApprovalManager struct {
	store seam.Store[ApprovalRequest]
	mu    sync.RWMutex
}

// approvalScope is the tenancy key approvals are stored under. Prism approvals are single-tenant
// today, so this is the zero Scope; the cloud deployment overrides it per Principal.
var approvalScope = seam.Scope{}

// NewApprovalManager creates an approval manager persisting to ~/.prism (via the file-backed seam
// store). Signature unchanged, so existing callers (the daemon) need no edit. Any legacy
// ~/.prism/approvals.json is migrated into the seam on first construction.
func NewApprovalManager() (*ApprovalManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".prism")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Records live under ~/.prism/approvals/ (one JSON file per request), via the seam.
	store := filestore.New[ApprovalRequest](filepath.Join(stateDir, "approvals"))
	am := NewApprovalManagerWithStore(store)

	// One-time migration: fold a legacy flat approvals.json into the seam, then retire it.
	if err := am.migrateLegacy(filepath.Join(stateDir, "approvals.json")); err != nil {
		return nil, fmt.Errorf("failed to migrate legacy approvals: %w", err)
	}
	return am, nil
}

// NewApprovalManagerWithStore builds a manager over an injected seam store — used by tests (with a
// filestore in a temp dir) and by the cloud (with a dynamostore). This is the seam in action: the
// manager logic is identical regardless of backend.
func NewApprovalManagerWithStore(store seam.Store[ApprovalRequest]) *ApprovalManager {
	return &ApprovalManager{store: store}
}

// migrateLegacy imports a pre-seam flat approvals.json (map[id]*ApprovalRequest) into the store,
// then renames it aside so the import runs once. Absent file → no-op.
func (am *ApprovalManager) migrateLegacy(legacyPath string) error {
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var requests map[string]*ApprovalRequest
	if err := json.Unmarshal(data, &requests); err != nil {
		return fmt.Errorf("parse legacy approvals: %w", err)
	}
	ctx := context.Background()
	for id, req := range requests {
		if req == nil {
			continue
		}
		if err := am.store.Put(ctx, approvalScope, id, *req); err != nil {
			return fmt.Errorf("migrate approval %q: %w", id, err)
		}
	}
	// Retire the legacy file so we don't re-import on next start.
	return os.Rename(legacyPath, legacyPath+".migrated")
}

// Submit creates a new pending approval request and persists it through the seam.
func (am *ApprovalManager) Submit(projectID, requestedBy string, typ ApprovalType, details map[string]interface{}, reason string) (*ApprovalRequest, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	req := ApprovalRequest{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		RequestedBy: requestedBy,
		Type:        typ,
		Status:      ApprovalStatusPending,
		Details:     details,
		Reason:      reason,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
	}

	if err := am.store.Put(context.Background(), approvalScope, req.ID, req); err != nil {
		return nil, err
	}
	out := req
	return &out, nil
}

// Get retrieves an approval request by ID.
func (am *ApprovalManager) Get(id string) (*ApprovalRequest, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	req, err := am.store.Get(context.Background(), approvalScope, id)
	if err != nil {
		if errorsIsNotFound(err) {
			return nil, fmt.Errorf("approval request %q not found", id)
		}
		return nil, err
	}
	return &req, nil
}

// List returns all approval requests, optionally filtered by project and status.
func (am *ApprovalManager) List(projectID string, status ApprovalStatus) ([]*ApprovalRequest, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	all, err := am.store.List(context.Background(), approvalScope)
	if err != nil {
		return nil, err
	}

	var results []*ApprovalRequest
	for i := range all {
		req := all[i]
		if projectID != "" && req.ProjectID != projectID {
			continue
		}
		if status != "" && req.Status != status {
			continue
		}
		out := req
		results = append(results, &out)
	}
	// Stable order (the old map-based List was unordered, but a deterministic order is strictly
	// better for callers and tests; sort by creation time, then ID).
	sort.Slice(results, func(i, j int) bool {
		if !results[i].CreatedAt.Equal(results[j].CreatedAt) {
			return results[i].CreatedAt.Before(results[j].CreatedAt)
		}
		return results[i].ID < results[j].ID
	})
	return results, nil
}

// Approve marks a request as approved.
func (am *ApprovalManager) Approve(id, reviewerID, note string) (*ApprovalRequest, error) {
	return am.review(id, ApprovalStatusApproved, reviewerID, note)
}

// Deny marks a request as denied.
func (am *ApprovalManager) Deny(id, reviewerID, note string) (*ApprovalRequest, error) {
	return am.review(id, ApprovalStatusDenied, reviewerID, note)
}

// review is the shared approve/deny transition: a pending request moves to the target status,
// stamped with the reviewer and time, then persisted.
func (am *ApprovalManager) review(id string, target ApprovalStatus, reviewerID, note string) (*ApprovalRequest, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	ctx := context.Background()
	req, err := am.store.Get(ctx, approvalScope, id)
	if err != nil {
		if errorsIsNotFound(err) {
			return nil, fmt.Errorf("approval request %q not found", id)
		}
		return nil, err
	}
	if req.Status != ApprovalStatusPending {
		verb := "approve"
		if target == ApprovalStatusDenied {
			verb = "deny"
		}
		return nil, fmt.Errorf("cannot %s request in status %q", verb, req.Status)
	}

	now := time.Now()
	req.Status = target
	req.ReviewedBy = reviewerID
	req.ReviewNote = note
	req.ReviewedAt = &now

	if err := am.store.Put(ctx, approvalScope, id, req); err != nil {
		return nil, err
	}
	out := req
	return &out, nil
}

// PruneExpired marks all pending requests that have passed their ExpiresAt as expired.
func (am *ApprovalManager) PruneExpired() (int, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	ctx := context.Background()
	all, err := am.store.List(ctx, approvalScope)
	if err != nil {
		return 0, err
	}

	count := 0
	now := time.Now()
	for i := range all {
		req := all[i]
		if req.Status == ApprovalStatusPending && now.After(req.ExpiresAt) {
			req.Status = ApprovalStatusExpired
			if err := am.store.Put(ctx, approvalScope, req.ID, req); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

// errorsIsNotFound reports whether err is the seam's not-found sentinel.
func errorsIsNotFound(err error) bool {
	return errors.Is(err, seam.ErrNotFound)
}
