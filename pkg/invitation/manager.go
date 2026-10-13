// Package invitation provides invitation management for Prism v0.5.11+
//
// This package implements the user invitation system enabling project
// owners to invite collaborators with role-based permissions.
package invitation

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/scttfrdmn/prism/pkg/types"
)

// DefaultExpirationDays is the default invitation expiration in days
const DefaultExpirationDays = 7

// Manager handles invitation lifecycle
type Manager struct {
	invitationsPath string
	mutex           sync.RWMutex
	invitations     map[string]*types.Invitation   // invitation_id → Invitation
	tokenIndex      map[string]*types.Invitation   // token → Invitation
	projectIndex    map[string][]*types.Invitation // project_id → [Invitations]
	emailIndex      map[string][]*types.Invitation // email → [Invitations]
	emailSender     EmailSender                    // Email sending interface
}

// EmailSender defines the interface for sending invitation emails
type EmailSender interface {
	SendInvitation(ctx context.Context, invitation *types.Invitation, project *types.Project, inviter string) error
	SendAcceptanceConfirmation(ctx context.Context, invitation *types.Invitation, project *types.Project) error
}

// NewManager creates a new invitation manager
func NewManager(emailSender EmailSender) (*Manager, error) {
	// Check for custom state directory via environment variable (for test isolation)
	stateDir := os.Getenv("PRISM_STATE_DIR")
	if stateDir == "" {
		// Default to ~/.prism for backward compatibility
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		stateDir = filepath.Join(homeDir, ".prism")
	}

	if err := os.MkdirAll(stateDir, 0700); err != nil { //nolint:gosec // G703: PRISM_STATE_DIR is set by the local operator/test harness, not remote input
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	invitationsPath := filepath.Join(stateDir, "invitations.json")

	manager := &Manager{
		invitationsPath: invitationsPath,
		invitations:     make(map[string]*types.Invitation),
		tokenIndex:      make(map[string]*types.Invitation),
		projectIndex:    make(map[string][]*types.Invitation),
		emailIndex:      make(map[string][]*types.Invitation),
		emailSender:     emailSender,
	}

	// Load existing invitations
	if err := manager.load(); err != nil {
		return nil, fmt.Errorf("failed to load invitations: %w", err)
	}

	// Build indexes
	manager.rebuildIndexes()

	return manager, nil
}

// CreateInvitation creates a new invitation
func (m *Manager) CreateInvitation(ctx context.Context, req *CreateInvitationRequest) (*types.Invitation, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid invitation request: %w", err)
	}

	// Check for duplicate pending invitation
	for _, inv := range m.invitations {
		if inv.ProjectID == req.ProjectID &&
			strings.EqualFold(inv.Email, req.Email) &&
			inv.Status == types.InvitationPending {
			return nil, types.ErrDuplicateInvitation
		}
	}

	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create invitation
	now := time.Now()
	expiresAt := now.AddDate(0, 0, DefaultExpirationDays)
	if req.ExpiresAt != nil {
		expiresAt = *req.ExpiresAt
	}

	invitation := &types.Invitation{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		Email:       strings.ToLower(req.Email),
		Role:        req.Role,
		Token:       token,
		InvitedBy:   req.InvitedBy,
		InvitedAt:   now,
		ExpiresAt:   expiresAt,
		Status:      types.InvitationPending,
		ResendCount: 0,
		Message:     req.Message,
	}

	// Store invitation
	m.invitations[invitation.ID] = invitation
	m.tokenIndex[invitation.Token] = invitation
	m.projectIndex[invitation.ProjectID] = append(m.projectIndex[invitation.ProjectID], invitation)
	m.emailIndex[invitation.Email] = append(m.emailIndex[invitation.Email], invitation)

	if err := m.save(); err != nil {
		// Rollback
		delete(m.invitations, invitation.ID)
		delete(m.tokenIndex, invitation.Token)
		m.rebuildIndexes()
		return nil, fmt.Errorf("failed to save invitation: %w", err)
	}

	return invitation, nil
}

// GetInvitation retrieves an invitation by ID
func (m *Manager) GetInvitation(ctx context.Context, invitationID string) (*types.Invitation, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	invitation, exists := m.invitations[invitationID]
	if !exists {
		return nil, types.ErrInvitationNotFound
	}

	// Return copy
	invitationCopy := *invitation
	return &invitationCopy, nil
}

// GetInvitationByToken retrieves an invitation by token
func (m *Manager) GetInvitationByToken(ctx context.Context, token string) (*types.Invitation, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	invitation, exists := m.tokenIndex[token]
	if !exists {
		return nil, types.ErrInvitationNotFound
	}

	// Auto-expire if needed
	if invitation.IsExpired() {
		// Unlock for write
		m.mutex.RUnlock()
		m.mutex.Lock()
		invitation.Status = types.InvitationExpired
		_ = m.save()
		m.mutex.Unlock()
		m.mutex.RLock()
		return nil, types.ErrInvitationExpired
	}

	// Return copy
	invitationCopy := *invitation
	return &invitationCopy, nil
}

// invitationMatchesFilter returns true if the invitation passes all active filter criteria.
func invitationMatchesFilter(inv *types.Invitation, filter *types.InvitationFilter) bool {
	if filter.ProjectID != "" && inv.ProjectID != filter.ProjectID {
		return false
	}
	if filter.Email != "" && !strings.EqualFold(inv.Email, filter.Email) {
		return false
	}
	if filter.Status != "" && inv.Status != filter.Status {
		return false
	}
	if filter.InvitedBy != "" && inv.InvitedBy != filter.InvitedBy {
		return false
	}
	if !filter.IncludeExpired && inv.IsExpired() {
		return false
	}
	return true
}

// ListInvitations lists invitations with optional filtering
func (m *Manager) ListInvitations(ctx context.Context, filter *types.InvitationFilter) ([]*types.Invitation, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var results []*types.Invitation

	for _, inv := range m.invitations {
		if invitationMatchesFilter(inv, filter) {
			results = append(results, inv)
		}
	}

	// Sort by InvitedAt descending (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].InvitedAt.After(results[j].InvitedAt)
	})

	// Apply pagination
	limit := filter.Limit
	if limit == 0 {
		limit = 50 // Default
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	if offset >= len(results) {
		return []*types.Invitation{}, nil
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end], nil
}

// AcceptInvitation accepts an invitation
func (m *Manager) AcceptInvitation(ctx context.Context, token string) (*types.Invitation, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	invitation, exists := m.tokenIndex[token]
	if !exists {
		return nil, types.ErrInvitationNotFound
	}

	// Validate can accept
	if !invitation.CanAccept() {
		if invitation.IsExpired() {
			invitation.Status = types.InvitationExpired
			_ = m.save()
			return nil, types.ErrInvitationExpired
		}
		return nil, types.ErrInvitationAlreadyUsed
	}

	// Accept invitation
	now := time.Now()
	invitation.Status = types.InvitationAccepted
	invitation.AcceptedAt = &now

	if err := m.save(); err != nil {
		return nil, fmt.Errorf("failed to save invitation: %w", err)
	}

	// Return copy
	invitationCopy := *invitation
	return &invitationCopy, nil
}

// DeclineInvitation declines an invitation
func (m *Manager) DeclineInvitation(ctx context.Context, token string, reason string) (*types.Invitation, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	invitation, exists := m.tokenIndex[token]
	if !exists {
		return nil, types.ErrInvitationNotFound
	}

	// Validate can decline
	if !invitation.CanDecline() {
		if invitation.IsExpired() {
			invitation.Status = types.InvitationExpired
			_ = m.save()
			return nil, types.ErrInvitationExpired
		}
		return nil, types.ErrInvitationAlreadyUsed
	}

	// Decline invitation
	now := time.Now()
	invitation.Status = types.InvitationDeclined
	invitation.DeclinedAt = &now
	invitation.DeclineReason = reason

	if err := m.save(); err != nil {
		return nil, fmt.Errorf("failed to save invitation: %w", err)
	}

	// Return copy
	invitationCopy := *invitation
	return &invitationCopy, nil
}

// RevokeInvitation revokes a pending invitation
func (m *Manager) RevokeInvitation(ctx context.Context, invitationID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	invitation, exists := m.invitations[invitationID]
	if !exists {
		return types.ErrInvitationNotFound
	}

	if !invitation.CanRevoke() {
		return types.ErrInvitationNotPending
	}

	invitation.Status = types.InvitationRevoked

	if err := m.save(); err != nil {
		return fmt.Errorf("failed to save invitation: %w", err)
	}

	return nil
}

// ResendInvitation resends an invitation email
func (m *Manager) ResendInvitation(ctx context.Context, invitationID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	invitation, exists := m.invitations[invitationID]
	if !exists {
		return types.ErrInvitationNotFound
	}

	if !invitation.CanResend() {
		return types.ErrInvitationNotPending
	}

	now := time.Now()
	invitation.ResendCount++
	invitation.LastResent = &now

	if err := m.save(); err != nil {
		return fmt.Errorf("failed to save invitation: %w", err)
	}

	// Email is sent by the daemon handler after this call returns,
	// where the project object is available for email context.

	return nil
}

// GetProjectSummary gets invitation summary for a project
func (m *Manager) GetProjectSummary(ctx context.Context, projectID string) (*types.InvitationSummary, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	invitations := m.projectIndex[projectID]

	summary := &types.InvitationSummary{
		ProjectID:        projectID,
		TotalInvitations: len(invitations),
	}

	// Count by status
	for _, inv := range invitations {
		switch inv.Status {
		case types.InvitationPending:
			summary.PendingCount++
		case types.InvitationAccepted:
			summary.AcceptedCount++
		case types.InvitationDeclined:
			summary.DeclinedCount++
		case types.InvitationExpired:
			summary.ExpiredCount++
		}
	}

	// Get recent invitations (last 10)
	if len(invitations) > 0 {
		// Sort by invited_at desc (most recent first)
		// For now, just take last 10
		recent := invitations
		if len(recent) > 10 {
			recent = recent[len(recent)-10:]
		}
		summary.RecentInvitations = recent
	}

	return summary, nil
}

// CleanupExpired marks expired invitations as expired
func (m *Manager) CleanupExpired(ctx context.Context) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	count := 0
	for _, inv := range m.invitations {
		if inv.IsExpired() {
			inv.Status = types.InvitationExpired
			count++
		}
	}

	if count > 0 {
		if err := m.save(); err != nil {
			return 0, fmt.Errorf("failed to save invitations: %w", err)
		}
	}

	return count, nil
}

// load loads invitations from JSON file
func (m *Manager) load() error {
	data, err := os.ReadFile(m.invitationsPath)
	if os.IsNotExist(err) {
		return nil // No invitations yet
	}
	if err != nil {
		return fmt.Errorf("failed to read invitations file: %w", err)
	}

	var invitations []*types.Invitation
	if err := json.Unmarshal(data, &invitations); err != nil {
		return fmt.Errorf("failed to unmarshal invitations: %w", err)
	}

	for _, inv := range invitations {
		m.invitations[inv.ID] = inv
	}

	return nil
}

// save saves invitations to JSON file
func (m *Manager) save() error {
	invitations := make([]*types.Invitation, 0, len(m.invitations))
	for _, inv := range m.invitations {
		invitations = append(invitations, inv)
	}

	data, err := json.MarshalIndent(invitations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal invitations: %w", err)
	}

	if err := os.WriteFile(m.invitationsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write invitations file: %w", err)
	}

	return nil
}

// rebuildIndexes rebuilds all indexes
func (m *Manager) rebuildIndexes() {
	m.tokenIndex = make(map[string]*types.Invitation)
	m.projectIndex = make(map[string][]*types.Invitation)
	m.emailIndex = make(map[string][]*types.Invitation)

	for _, inv := range m.invitations {
		m.tokenIndex[inv.Token] = inv
		m.projectIndex[inv.ProjectID] = append(m.projectIndex[inv.ProjectID], inv)
		m.emailIndex[inv.Email] = append(m.emailIndex[inv.Email], inv)
	}
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateBulkInvitations creates multiple invitations concurrently
func (m *Manager) CreateBulkInvitations(ctx context.Context, projectID string, invitedBy string, bulkReq *types.BulkInvitationRequest) (*types.BulkInvitationResponse, error) {
	// Initialize response
	response := &types.BulkInvitationResponse{
		Summary: types.BulkInvitationSummary{
			Total: len(bulkReq.Invitations),
		},
		Results: make([]types.BulkInvitationResult, 0, len(bulkReq.Invitations)),
	}

	// Parse expiration
	var expiresAt time.Time
	if bulkReq.ExpiresAt != nil {
		expiresAt = *bulkReq.ExpiresAt
	} else if bulkReq.ExpiresIn != "" {
		// Parse duration string (e.g., "7d", "30d")
		duration, err := parseDuration(bulkReq.ExpiresIn)
		if err != nil {
			return nil, fmt.Errorf("invalid expires_in duration: %w", err)
		}
		expiresAt = time.Now().Add(duration)
	} else {
		// Default to 7 days
		expiresAt = time.Now().AddDate(0, 0, DefaultExpirationDays)
	}

	// Check for existing members (would need project manager integration)
	// For now, we'll just check for duplicate pending invitations

	// Process each invitation
	for _, entry := range bulkReq.Invitations {
		// Determine role (entry-specific or default)
		role := entry.Role
		if role == "" {
			role = bulkReq.DefaultRole
		}

		// Determine message (entry-specific or default)
		message := entry.Message
		if message == "" {
			message = bulkReq.DefaultMessage
		}

		// Create invitation request
		req := &CreateInvitationRequest{
			ProjectID: projectID,
			Email:     entry.Email,
			Role:      role,
			InvitedBy: invitedBy,
			Message:   message,
			ExpiresAt: &expiresAt,
		}

		// Validate and create invitation
		if err := req.Validate(); err != nil {
			response.Results = append(response.Results, types.BulkInvitationResult{
				Email:  entry.Email,
				Status: "failed",
				Error:  err.Error(),
			})
			response.Summary.Failed++
			continue
		}

		// Check if already invited (pending invitation exists)
		m.mutex.RLock()
		duplicate := false
		for _, inv := range m.invitations {
			if inv.ProjectID == projectID &&
				strings.EqualFold(inv.Email, entry.Email) &&
				inv.Status == types.InvitationPending {
				duplicate = true
				break
			}
		}
		m.mutex.RUnlock()

		if duplicate {
			response.Results = append(response.Results, types.BulkInvitationResult{
				Email:  entry.Email,
				Status: "skipped",
				Reason: "pending invitation already exists",
			})
			response.Summary.Skipped++
			continue
		}

		// Create invitation
		inv, err := m.CreateInvitation(ctx, req)
		if err != nil {
			// Determine if this is a skip or failure
			if err == types.ErrDuplicateInvitation {
				response.Results = append(response.Results, types.BulkInvitationResult{
					Email:  entry.Email,
					Status: "skipped",
					Reason: "pending invitation already exists",
				})
				response.Summary.Skipped++
			} else {
				response.Results = append(response.Results, types.BulkInvitationResult{
					Email:  entry.Email,
					Status: "failed",
					Error:  err.Error(),
				})
				response.Summary.Failed++
			}
			continue
		}

		// Success
		response.Results = append(response.Results, types.BulkInvitationResult{
			Email:        entry.Email,
			Status:       "sent",
			InvitationID: inv.ID,
		})
		response.Summary.Sent++
	}

	// Generate summary message
	response.Message = fmt.Sprintf("Bulk invitation complete: %d sent, %d skipped, %d failed",
		response.Summary.Sent, response.Summary.Skipped, response.Summary.Failed)

	return response, nil
}

// parseDuration parses duration strings like "7d", "30d", "24h"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	// Get numeric value and unit
	valueStr := s[:len(s)-1]
	unit := s[len(s)-1:]

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return 0, fmt.Errorf("invalid duration value: %w", err)
	}

	switch unit {
	case "d":
		return time.Hour * 24 * time.Duration(value), nil
	case "h":
		return time.Hour * time.Duration(value), nil
	case "m":
		return time.Minute * time.Duration(value), nil
	default:
		return 0, fmt.Errorf("invalid duration unit: %s (use d, h, or m)", unit)
	}
}

// CreateInvitationRequest is a request to create an invitation
type CreateInvitationRequest struct {
	ProjectID string            `json:"project_id"`
	Email     string            `json:"email"`
	Role      types.ProjectRole `json:"role"`
	InvitedBy string            `json:"invited_by"`
	Message   string            `json:"message,omitempty"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
}

// Validate validates the create invitation request
func (r *CreateInvitationRequest) Validate() error {
	if r.ProjectID == "" {
		return types.ErrInvalidProjectID
	}
	if r.Email == "" {
		return types.ErrInvalidEmail
	}
	if r.Role == "" {
		return types.ErrInvalidRole
	}
	if r.InvitedBy == "" {
		return types.ErrInvalidInviter
	}
	return nil
}
