// Package daemon provides invitation management API handlers for Prism v0.5.11+
//
// This file implements REST API endpoints for the user invitation system,
// enabling project owners to invite collaborators via email.
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/invitation"
	"github.com/scttfrdmn/prism/pkg/research"
	"github.com/scttfrdmn/prism/pkg/types"
)

// requireProjectRole looks up the project and verifies that userID holds one of
// the given roles.  It returns the project on success.  On failure it writes the
// appropriate HTTP error and returns nil, so callers can just do:
//
//	project, err := s.requireProjectRole(ctx, w, projectID, userID, RoleOwner, RoleAdmin)
//	if project == nil { return }
func (s *Server) requireProjectRole(ctx context.Context, w http.ResponseWriter, projectID, userID string, roles ...types.ProjectRole) *types.Project {
	if s.projectManager == nil || userID == "" {
		// No project manager or no caller identity — skip check (permissive fallback).
		return nil
	}

	project, err := s.projectManager.GetProject(ctx, projectID)
	if err != nil {
		if err == types.ErrProjectNotFound {
			http.Error(w, "Project not found", http.StatusNotFound)
			return nil
		}
		http.Error(w, fmt.Sprintf("Failed to get project: %v", err), http.StatusInternalServerError)
		return nil
	}

	for _, m := range project.Members {
		if m.UserID != userID {
			continue
		}
		for _, allowed := range roles {
			if m.Role == allowed {
				return project
			}
		}
		// Member found but role not allowed.
		http.Error(w, "Forbidden: insufficient project role", http.StatusForbidden)
		return nil
	}

	// userID not in member list at all.
	http.Error(w, "Forbidden: not a project member", http.StatusForbidden)
	return nil
}

// invitationRoute maps a path/method pattern to a handler.
// Matching is performed by the matches() method using the exported fields.
// Routes are evaluated in order; the first matching route wins.
type invitationRoute struct {
	method   string // required HTTP method
	exact    string // path must equal this value (takes priority over prefix/suffix/contains)
	prefix   string // path must have this prefix (optional)
	suffix   string // path must have this suffix (optional)
	contains string // path must contain this substring (optional)
	handler  func(w http.ResponseWriter, r *http.Request)
}

// matches reports whether this route handles the given path and method.
func (rt invitationRoute) matches(path, method string) bool {
	if method != rt.method {
		return false
	}
	if rt.exact != "" {
		return path == rt.exact
	}
	if rt.prefix != "" && !strings.HasPrefix(path, rt.prefix) {
		return false
	}
	if rt.suffix != "" && !strings.HasSuffix(path, rt.suffix) {
		return false
	}
	if rt.contains != "" && !strings.Contains(path, rt.contains) {
		return false
	}
	return true
}

// invitationRoutes returns the ordered routing table for invitation endpoints.
func (s *Server) invitationRoutes() []invitationRoute {
	return []invitationRoute{
		{method: http.MethodPost, prefix: "/api/v1/projects/", suffix: "/invitations/bulk", handler: s.handleBulkInvitation},
		{method: http.MethodPost, prefix: "/api/v1/projects/", suffix: "/invitations", handler: s.handleSendInvitation},
		{method: http.MethodGet, prefix: "/api/v1/projects/", suffix: "/invitations", handler: s.handleListProjectInvitations},
		{method: http.MethodGet, exact: "/api/v1/invitations/my", handler: s.handleListMyInvitations},
		{method: http.MethodPost, contains: "/accept", handler: s.handleAcceptInvitation},
		{method: http.MethodPost, contains: "/decline", handler: s.handleDeclineInvitation},
		{method: http.MethodPost, contains: "/resend", handler: s.handleResendInvitation},
		{method: http.MethodDelete, prefix: "/api/v1/invitations/", handler: s.handleRevokeInvitation},
		{method: http.MethodPost, exact: "/api/v1/invitations/quota-check", handler: s.handleQuotaCheck},
		{method: http.MethodGet, prefix: "/api/v1/invitations/by-id/", handler: s.handleGetInvitationByID},
		{method: http.MethodPost, contains: "/validate", handler: s.handleValidateInvitation},
		{method: http.MethodGet, contains: "/credential-status", handler: s.handleGetCredentialStatus},
		{method: http.MethodPost, contains: "/test-credentials", handler: s.handleTestCredentials},
		{method: http.MethodGet, prefix: "/api/v1/invitations/", handler: s.handleGetInvitation},
	}
}

// handleInvitationOperations routes invitation-related HTTP requests.
// It delegates to specific handlers via the routing table returned by invitationRoutes().
func (s *Server) handleInvitationOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Shared-token operations take priority (v0.5.13)
	if strings.Contains(path, "/shared") || strings.HasPrefix(path, "/api/v1/invitations/shared/") {
		s.handleSharedTokenOperations(w, r)
		return
	}

	for _, route := range s.invitationRoutes() {
		if route.matches(path, r.Method) {
			route.handler(w, r)
			return
		}
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// handleSendInvitation creates and sends a new project invitation
// POST /api/v1/projects/{id}/invitations
func (s *Server) handleSendInvitation(w http.ResponseWriter, r *http.Request) {
	// Parse project ID from path: /api/v1/projects/{id}/invitations
	path := r.URL.Path[len("/api/v1/projects/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "invitations" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	projectID := parts[0]

	var req invitation.CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Set project ID from URL
	req.ProjectID = projectID

	// Set InvitedBy from request body (caller identity).
	if req.InvitedBy == "" {
		req.InvitedBy = "system" // Default for tests
	}

	// Verify the requester is an owner or admin of the project.
	project := s.requireProjectRole(r.Context(), w, projectID, req.InvitedBy,
		types.ProjectRoleOwner, types.ProjectRoleAdmin)
	if project == nil {
		return
	}

	// Create invitation
	inv, err := s.invitationManager.CreateInvitation(r.Context(), &req)
	if err != nil {
		if err == types.ErrDuplicateInvitation {
			http.Error(w, "An invitation for this email already exists for this project", http.StatusConflict)
			return
		}
		if err == types.ErrInvalidEmail || err == types.ErrInvalidRole || err == types.ErrInvalidInviter {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to create invitation: %v", err), http.StatusInternalServerError)
		return
	}

	s.invitationManager.SendInvitationEmail(r.Context(), inv, project, req.InvitedBy)

	response := map[string]interface{}{
		"invitation": inv,
		"project":    project,
		"message":    "Invitation created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encodeJSONResponse(w, response)
}

// handleListProjectInvitations lists all invitations for a project
// GET /api/v1/projects/{id}/invitations?status=pending&limit=50&offset=0
func (s *Server) handleListProjectInvitations(w http.ResponseWriter, r *http.Request) {
	// Parse project ID from path: /api/v1/projects/{id}/invitations
	path := r.URL.Path[len("/api/v1/projects/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "invitations" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	projectID := parts[0]

	// Require the requester to be at least a member of the project.
	requesterID := r.URL.Query().Get("requester_id")
	if requesterID != "" {
		if p := s.requireProjectRole(r.Context(), w, projectID, requesterID,
			types.ProjectRoleOwner, types.ProjectRoleAdmin, types.ProjectRoleMember, types.ProjectRoleViewer); p == nil {
			return
		}
	} else {
		// No requester_id — still verify the project exists.
		if _, err := s.projectManager.GetProject(r.Context(), projectID); err != nil {
			if err == types.ErrProjectNotFound {
				http.Error(w, "Project not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to get project: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Parse query parameters
	filter := &types.InvitationFilter{
		ProjectID: projectID,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = types.InvitationStatus(status)
	}

	if includeExpired := r.URL.Query().Get("include_expired"); includeExpired == "true" {
		filter.IncludeExpired = true
	}

	// Parse pagination
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if _, err := fmt.Sscanf(limit, "%d", &filter.Limit); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid limit parameter")
			return
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if _, err := fmt.Sscanf(offset, "%d", &filter.Offset); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid offset parameter")
			return
		}
	}

	// List invitations
	invitations, err := s.invitationManager.ListInvitations(r.Context(), filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list invitations: %v", err), http.StatusInternalServerError)
		return
	}

	// Get summary
	summary, err := s.invitationManager.GetProjectSummary(r.Context(), projectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get invitation summary: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"invitations": invitations,
		"summary":     summary,
		"filter":      filter,
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleListMyInvitations lists invitations received by the current user
// GET /api/v1/invitations/my?email=user@example.com&status=pending
func (s *Server) handleListMyInvitations(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Email parameter required", http.StatusBadRequest)
		return
	}

	filter := &types.InvitationFilter{
		Email:          email,
		IncludeExpired: false,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = types.InvitationStatus(status)
	}

	// Parse pagination
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if _, err := fmt.Sscanf(limit, "%d", &filter.Limit); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid limit parameter")
			return
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if _, err := fmt.Sscanf(offset, "%d", &filter.Offset); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid offset parameter")
			return
		}
	}

	// List invitations
	invitations, err := s.invitationManager.ListInvitations(r.Context(), filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list invitations: %v", err), http.StatusInternalServerError)
		return
	}

	// Enrich invitations with project names
	for _, inv := range invitations {
		if project, err := s.projectManager.GetProject(r.Context(), inv.ProjectID); err == nil {
			inv.ProjectName = project.Name
		}
	}

	response := map[string]interface{}{
		"invitations": invitations,
		"email":       email,
		"filter":      filter,
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleGetInvitation retrieves an invitation by token (public endpoint)
// GET /api/v1/invitations/{token}
func (s *Server) handleGetInvitation(w http.ResponseWriter, r *http.Request) {
	// Parse token from path: /api/v1/invitations/{token}
	token := r.URL.Path[len("/api/v1/invitations/"):]
	if token == "" {
		http.Error(w, "Missing invitation token", http.StatusBadRequest)
		return
	}

	inv, err := s.invitationManager.GetInvitationByToken(r.Context(), token)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		if err == types.ErrInvitationExpired {
			http.Error(w, "Invitation has expired", http.StatusGone)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get invitation: %v", err), http.StatusInternalServerError)
		return
	}

	// Get project details for context
	project, err := s.projectManager.GetProject(r.Context(), inv.ProjectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get project: %v", err), http.StatusInternalServerError)
		return
	}

	// Enrich invitation with project name for display
	inv.ProjectName = project.Name

	response := map[string]interface{}{
		"invitation": inv,
		"project":    project,
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleGetInvitationByID retrieves an invitation by its ID (v0.6.2)
// GET /api/v1/invitations/by-id/{id}
func (s *Server) handleGetInvitationByID(w http.ResponseWriter, r *http.Request) {
	// Parse invitation ID from path: /api/v1/invitations/by-id/{id}
	invitationID := r.URL.Path[len("/api/v1/invitations/by-id/"):]
	if invitationID == "" {
		http.Error(w, "Missing invitation ID", http.StatusBadRequest)
		return
	}

	inv, err := s.invitationManager.GetInvitation(r.Context(), invitationID)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		if err == types.ErrInvitationExpired {
			http.Error(w, "Invitation has expired", http.StatusGone)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get invitation: %v", err), http.StatusInternalServerError)
		return
	}

	// Get project details for context
	project, err := s.projectManager.GetProject(r.Context(), inv.ProjectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get project: %v", err), http.StatusInternalServerError)
		return
	}

	// Enrich invitation with project name for display
	inv.ProjectName = project.Name

	response := map[string]interface{}{
		"invitation": inv,
		"project":    project,
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleAcceptInvitation accepts an invitation and adds user to project
// POST /api/v1/invitations/{token}/accept
func (s *Server) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	// Parse token from path: /api/v1/invitations/{token}/accept
	path := r.URL.Path[len("/api/v1/invitations/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "accept" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	token := parts[0]

	// Accept invitation
	inv, err := s.invitationManager.AcceptInvitation(r.Context(), token)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		if err == types.ErrInvitationExpired {
			http.Error(w, "Invitation has expired", http.StatusGone)
			return
		}
		if err == types.ErrInvitationAlreadyUsed {
			http.Error(w, "Invitation has already been used", http.StatusConflict)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to accept invitation: %v", err), http.StatusInternalServerError)
		return
	}

	// Add user to project members (Issue #102)
	member := &types.ProjectMember{
		UserID:  inv.Email, // Use invitation email as user ID
		Role:    inv.Role,  // Use role from invitation
		AddedBy: inv.InvitedBy,
	}

	if err := s.projectManager.AddProjectMember(r.Context(), inv.ProjectID, member); err != nil {
		// Handle duplicate member case gracefully (user may already be added)
		if !strings.Contains(err.Error(), "already a member") {
			http.Error(w, fmt.Sprintf("Failed to add member to project: %v", err), http.StatusInternalServerError)
			return
		}
		// If already a member, that's fine - continue with acceptance response
	}

	// Auto-provision research user (Issue #106)
	// Extract username from email (e.g., "user@example.com" → "user")
	username := strings.Split(inv.Email, "@")[0]

	var researchUser interface{}
	var provisioningError string

	if researchUserService, err := s.getResearchUserService(); err == nil {
		// Try to create or get existing research user
		user, err := researchUserService.CreateResearchUser(username, &research.CreateResearchUserOptions{
			GenerateSSHKey: true, // Automatically generate SSH keys
		})
		if err != nil {
			// Log error but don't fail invitation acceptance
			provisioningError = fmt.Sprintf("Research user provisioning failed: %v", err)
		} else {
			researchUser = user
		}
	} else {
		provisioningError = fmt.Sprintf("Failed to initialize research user service: %v", err)
	}

	// Get project details
	project, err := s.projectManager.GetProject(r.Context(), inv.ProjectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get project: %v", err), http.StatusInternalServerError)
		return
	}

	s.invitationManager.SendAcceptanceEmail(r.Context(), inv, project)

	// Build response with provisioning status
	response := map[string]interface{}{
		"invitation": inv,
		"project":    project,
		"message":    "Invitation accepted successfully",
	}

	// Add provisioning information to response
	if researchUser != nil {
		response["research_user"] = researchUser
		response["provisioning_status"] = "success"
	} else if provisioningError != "" {
		response["provisioning_status"] = "failed"
		response["provisioning_error"] = provisioningError
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleDeclineInvitation declines an invitation
// POST /api/v1/invitations/{token}/decline
func (s *Server) handleDeclineInvitation(w http.ResponseWriter, r *http.Request) {
	// Parse token from path: /api/v1/invitations/{token}/decline
	path := r.URL.Path[len("/api/v1/invitations/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "decline" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	token := parts[0]

	// Parse optional decline reason
	var req struct {
		Reason string `json:"reason,omitempty"`
	}
	// The body is optional (empty or absent means no reason), so a decode
	// failure just leaves Reason empty.
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Decline invitation
	inv, err := s.invitationManager.DeclineInvitation(r.Context(), token, req.Reason)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		if err == types.ErrInvitationExpired {
			http.Error(w, "Invitation has expired", http.StatusGone)
			return
		}
		if err == types.ErrInvitationAlreadyUsed {
			http.Error(w, "Invitation has already been used", http.StatusConflict)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to decline invitation: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"invitation": inv,
		"message":    "Invitation declined",
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleResendInvitation resends an invitation email
// POST /api/v1/invitations/{id}/resend
func (s *Server) handleResendInvitation(w http.ResponseWriter, r *http.Request) {
	// Parse invitation ID from path: /api/v1/invitations/{id}/resend
	path := r.URL.Path[len("/api/v1/invitations/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "resend" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	invitationID := parts[0]

	// Get invitation
	inv, err := s.invitationManager.GetInvitation(r.Context(), invitationID)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get invitation: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify the requester is an owner or admin of the project.
	if requesterID := r.URL.Query().Get("requester_id"); requesterID != "" {
		if s.requireProjectRole(r.Context(), w, inv.ProjectID, requesterID,
			types.ProjectRoleOwner, types.ProjectRoleAdmin) == nil {
			return
		}
	}

	// Resend invitation
	if err := s.invitationManager.ResendInvitation(r.Context(), invitationID); err != nil {
		if err == types.ErrInvitationNotPending {
			http.Error(w, "Cannot resend: invitation is not pending", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to resend invitation: %v", err), http.StatusInternalServerError)
		return
	}

	// Send resend email (look up project for context; non-fatal on failure)
	if s.projectManager != nil {
		if resendProject, pErr := s.projectManager.GetProject(r.Context(), inv.ProjectID); pErr == nil {
			s.invitationManager.SendInvitationEmail(r.Context(), inv, resendProject, inv.InvitedBy)
		} else {
			s.invitationManager.SendInvitationEmail(r.Context(), inv, nil, inv.InvitedBy)
		}
	}

	response := map[string]interface{}{
		"invitation": inv,
		"message":    "Invitation resent successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleRevokeInvitation revokes a pending invitation
// DELETE /api/v1/invitations/{id}
func (s *Server) handleRevokeInvitation(w http.ResponseWriter, r *http.Request) {
	// Parse invitation ID from path: /api/v1/invitations/{id}
	invitationID := r.URL.Path[len("/api/v1/invitations/"):]
	if invitationID == "" {
		http.Error(w, "Missing invitation ID", http.StatusBadRequest)
		return
	}

	// Get invitation for project context
	inv, err := s.invitationManager.GetInvitation(r.Context(), invitationID)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get invitation: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify the requester is an owner or admin of the project.
	if requesterID := r.URL.Query().Get("requester_id"); requesterID != "" {
		if s.requireProjectRole(r.Context(), w, inv.ProjectID, requesterID,
			types.ProjectRoleOwner, types.ProjectRoleAdmin) == nil {
			return
		}
	}

	// Revoke invitation
	if err := s.invitationManager.RevokeInvitation(r.Context(), invitationID); err != nil {
		if err == types.ErrInvitationNotPending {
			http.Error(w, "Cannot revoke: invitation is not pending", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to revoke invitation: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"invitation": inv,
		"message":    "Invitation revoked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleBulkInvitation creates multiple invitations from a bulk request
// POST /api/v1/projects/{id}/invitations/bulk
func (s *Server) handleBulkInvitation(w http.ResponseWriter, r *http.Request) {
	// Parse project ID from path: /api/v1/projects/{id}/invitations/bulk
	path := r.URL.Path[len("/api/v1/projects/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "invitations" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	projectID := parts[0]

	// Validate project exists
	project, err := s.projectManager.GetProject(r.Context(), projectID)
	if err != nil {
		if err == types.ErrProjectNotFound {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get project: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse bulk invitation request
	var bulkReq types.BulkInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&bulkReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if len(bulkReq.Invitations) == 0 {
		http.Error(w, "No invitations provided", http.StatusBadRequest)
		return
	}

	// Set default role if not specified
	if bulkReq.DefaultRole == "" {
		bulkReq.DefaultRole = types.ProjectRoleMember
	}

	// Resolve inviter from request body; fall back to "system" for backward compatibility.
	invitedBy := bulkReq.InvitedBy
	if invitedBy == "" {
		invitedBy = "system"
	}

	// Verify the requester is an owner or admin of the project (Gap I fix).
	if invitedBy != "system" {
		if s.requireProjectRole(r.Context(), w, projectID, invitedBy,
			types.ProjectRoleOwner, types.ProjectRoleAdmin) == nil {
			return
		}
	}

	// Create bulk invitations
	bulkResp, err := s.invitationManager.CreateBulkInvitations(r.Context(), projectID, invitedBy, &bulkReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create bulk invitations: %v", err), http.StatusInternalServerError)
		return
	}

	// Send invitation emails for each successfully created invitation (Gap K fix, fire-and-forget).
	for _, result := range bulkResp.Results {
		if result.Status == "sent" && result.InvitationID != "" {
			if inv, err := s.invitationManager.GetInvitation(r.Context(), result.InvitationID); err == nil {
				s.invitationManager.SendInvitationEmail(r.Context(), inv, project, invitedBy)
			}
		}
	}

	response := map[string]interface{}{
		"summary": bulkResp.Summary,
		"results": bulkResp.Results,
		"message": bulkResp.Message,
		"project": project,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encodeJSONResponse(w, response)
}

// QuotaCheckRequest represents a request to check AWS quotas for bulk invitations
type QuotaCheckRequest struct {
	InstanceType string `json:"instance_type"`
	Count        int    `json:"count"` // Number of students/invitations
	TemplateName string `json:"template_name,omitempty"`
}

// handleQuotaCheck checks AWS quotas for bulk invitation launch requirements
// POST /api/v1/invitations/quota-check
func (s *Server) handleQuotaCheck(w http.ResponseWriter, r *http.Request) {
	var req QuotaCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.InstanceType == "" {
		http.Error(w, "instance_type is required", http.StatusBadRequest)
		return
	}
	if req.Count <= 0 {
		http.Error(w, "count must be greater than 0", http.StatusBadRequest)
		return
	}

	// Get AWS manager
	if s.awsManager == nil {
		http.Error(w, "AWS manager not initialized", http.StatusInternalServerError)
		return
	}

	// Check quota
	quotaCheck, err := s.awsManager.CheckQuotaForInvitations(r.Context(), req.InstanceType, req.Count)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check quota: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	response := map[string]interface{}{
		"has_sufficient_quota": quotaCheck.HasSufficientQuota,
		"required_vcpus":       quotaCheck.RequiredVCPUs,
		"current_usage":        quotaCheck.CurrentUsage,
		"quota_limit":          quotaCheck.QuotaLimit,
		"available_vcpus":      quotaCheck.AvailableVCPUs,
		"instance_type":        quotaCheck.InstanceType,
	}

	if !quotaCheck.HasSufficientQuota {
		response["warning"] = quotaCheck.WarningMessage
		w.WriteHeader(http.StatusPreconditionFailed) // 412 status for insufficient quota
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleValidateInvitation validates invitation profile association and credentials (#357)
// POST /api/v1/invitations/{id}/validate
func (s *Server) handleValidateInvitation(w http.ResponseWriter, r *http.Request) {
	// Parse invitation ID from path: /api/v1/invitations/{id}/validate
	path := r.URL.Path[len("/api/v1/invitations/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "validate" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	invitationID := parts[0]

	// Get invitation
	inv, err := s.invitationManager.GetInvitation(r.Context(), invitationID)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get invitation: %v", err), http.StatusInternalServerError)
		return
	}

	// Validate invitation with profile manager
	profileMgr := NewProfileManagerAdapter(s)
	result, err := s.invitationManager.ValidateInvitation(r.Context(), invitationID, profileMgr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to validate invitation: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"invitation": inv,
		"validation": result,
	}

	// Set appropriate HTTP status
	if !result.Valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		encodeJSONResponse(w, response)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleGetCredentialStatus returns the current credential status for an invitation (#357)
// GET /api/v1/invitations/{id}/credential-status
func (s *Server) handleGetCredentialStatus(w http.ResponseWriter, r *http.Request) {
	// Parse invitation ID from path: /api/v1/invitations/{id}/credential-status
	path := r.URL.Path[len("/api/v1/invitations/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "credential-status" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	invitationID := parts[0]

	// Get credential status
	profileMgr := NewProfileManagerAdapter(s)
	status, err := s.invitationManager.GetCredentialStatus(r.Context(), invitationID, profileMgr)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get credential status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, status)
}

// handleTestCredentials tests if credentials work for an invitation (#357)
// POST /api/v1/invitations/{id}/test-credentials
func (s *Server) handleTestCredentials(w http.ResponseWriter, r *http.Request) {
	// Parse invitation ID from path: /api/v1/invitations/{id}/test-credentials
	path := r.URL.Path[len("/api/v1/invitations/"):]
	parts := splitPath(path)
	if len(parts) < 2 || parts[1] != "test-credentials" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	invitationID := parts[0]

	// Get invitation
	inv, err := s.invitationManager.GetInvitation(r.Context(), invitationID)
	if err != nil {
		if err == types.ErrInvitationNotFound {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get invitation: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if profile ID exists
	if inv.ProfileID == "" {
		http.Error(w, "Invitation has no profile association", http.StatusBadRequest)
		return
	}

	// Test credentials
	profileMgr := NewProfileManagerAdapter(s)
	valid, identity, err := s.invitationManager.TestCredentials(r.Context(), profileMgr, inv.ProfileID)
	if err != nil {
		response := map[string]interface{}{
			"valid":         false,
			"error_message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		encodeJSONResponse(w, response)
		return
	}

	response := map[string]interface{}{
		"valid":               valid,
		"credential_identity": identity,
		"tested_at":           time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}
