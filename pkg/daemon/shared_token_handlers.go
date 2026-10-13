// Package daemon provides shared token management API handlers for Prism v0.5.13+
//
// This file implements REST API endpoints for the shared token system,
// enabling project owners to create tokens that multiple users can redeem.
package daemon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/skip2/go-qrcode"
)

// isProjectSharedTokenPath reports whether path matches /api/v1/projects/{id}/invitations/shared.
func isProjectSharedTokenPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/invitations/shared")
}

// isBaseSharedTokenPath reports whether path is under /api/v1/invitations/shared/.
func isBaseSharedTokenPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/invitations/shared/")
}

// handleSharedTokenOperations routes shared token-related HTTP requests
func (s *Server) handleSharedTokenOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	case path == "/api/v1/invitations/shared/redeem" && method == http.MethodPost:
		s.handleRedeemSharedToken(w, r)

	case isProjectSharedTokenPath(path):
		switch method {
		case http.MethodPost:
			s.handleCreateSharedToken(w, r)
		case http.MethodGet:
			s.handleListSharedTokens(w, r)
		}

	case strings.Contains(path, "/qr") && method == http.MethodGet:
		s.handleGenerateQRCode(w, r)

	case strings.Contains(path, "/extend") && method == http.MethodPatch:
		s.handleExtendSharedToken(w, r)

	case isBaseSharedTokenPath(path):
		switch method {
		case http.MethodGet:
			s.handleGetSharedToken(w, r)
		case http.MethodDelete:
			s.handleRevokeSharedToken(w, r)
		}

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleCreateSharedToken creates a new shared invitation token
// POST /api/v1/projects/{id}/invitations/shared
func (s *Server) handleCreateSharedToken(w http.ResponseWriter, r *http.Request) {
	// Parse project ID from path
	path := r.URL.Path[len("/api/v1/projects/"):]
	parts := splitPath(path)
	if len(parts) < 2 {
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

	// Parse request
	var req types.CreateSharedTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Set project ID
	req.ProjectID = projectID

	// Set default role if not specified
	if req.Role == "" {
		req.Role = types.ProjectRoleMember
	}

	// Use caller-supplied CreatedBy; fall back to "system" for backward compatibility (Gap M fix).
	if req.CreatedBy == "" {
		req.CreatedBy = "system"
	}

	// Verify the requester is an owner or admin of the project (Gap N fix).
	if req.CreatedBy != "system" {
		if s.requireProjectRole(r.Context(), w, projectID, req.CreatedBy,
			types.ProjectRoleOwner, types.ProjectRoleAdmin) == nil {
			return
		}
	}

	// Create shared token
	token, err := s.sharedTokenManager.CreateSharedToken(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create shared token: %v", err), http.StatusBadRequest)
		return
	}

	// Build response
	redemptionURL := fmt.Sprintf("https://prism.dev/join/%s", token.Token)
	qrCodeURL := fmt.Sprintf("/api/v1/invitations/shared/%s/qr", token.Token)

	response := map[string]interface{}{
		"token":            token.Token,
		"id":               token.ID,
		"name":             token.Name,
		"role":             token.Role,
		"redemption_limit": token.RedemptionLimit,
		"redemptions":      token.RedemptionCount,
		"status":           token.Status,
		"expires_at":       token.ExpiresAt,
		"created_at":       token.CreatedAt,
		"redemption_url":   redemptionURL,
		"qr_code_url":      qrCodeURL,
		"project":          project,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encodeJSONResponse(w, response)
}

// handleRedeemSharedToken redeems a shared token
// POST /api/v1/invitations/shared/redeem
func (s *Server) handleRedeemSharedToken(w http.ResponseWriter, r *http.Request) {
	var req types.RedeemSharedTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Redeem token
	resp, err := s.sharedTokenManager.RedeemToken(r.Context(), &req)
	if err != nil {
		if err == types.ErrSharedTokenNotFound {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		if err == types.ErrSharedTokenExpired {
			http.Error(w, "Token has expired", http.StatusBadRequest)
			return
		}
		if err == types.ErrSharedTokenRevoked {
			http.Error(w, "Token has been revoked", http.StatusBadRequest)
			return
		}
		if err == types.ErrSharedTokenExhausted {
			http.Error(w, "Token has reached redemption limit", http.StatusBadRequest)
			return
		}
		if err == types.ErrAlreadyRedeemed {
			http.Error(w, "You have already redeemed this token", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to redeem token: %v", err), http.StatusBadRequest)
		return
	}

	// Add the redeemer as a project member with the role from the token
	if s.projectManager != nil && req.RedeemedBy != "" && resp.ProjectID != "" {
		member := &types.ProjectMember{
			UserID:  req.RedeemedBy,
			Role:    resp.Role,
			AddedAt: time.Now(),
			AddedBy: "shared-token:" + req.Token,
		}
		if err := s.projectManager.AddProjectMember(r.Context(), resp.ProjectID, member); err != nil {
			// Duplicate member is acceptable; log other errors but don't fail the redemption
			fmt.Printf("[shared-token] add member %s to project %s: %v\n", req.RedeemedBy, resp.ProjectID, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, resp)
}

// handleGetSharedToken retrieves shared token information
// GET /api/v1/invitations/shared/{token}
func (s *Server) handleGetSharedToken(w http.ResponseWriter, r *http.Request) {
	// Parse token from path
	path := r.URL.Path[len("/api/v1/invitations/shared/"):]
	tokenCode := strings.TrimSuffix(path, "/")

	// Get token
	token, err := s.sharedTokenManager.GetSharedToken(r.Context(), tokenCode)
	if err != nil {
		if err == types.ErrSharedTokenNotFound {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, token)
}

// handleListSharedTokens lists all shared tokens for a project
// GET /api/v1/projects/{id}/invitations/shared
func (s *Server) handleListSharedTokens(w http.ResponseWriter, r *http.Request) {
	// Parse project ID from path
	path := r.URL.Path[len("/api/v1/projects/"):]
	parts := splitPath(path)
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	projectID := parts[0]

	// Optional: verify requester is a project member (Gap N fix).
	if requesterID := r.URL.Query().Get("requester_id"); requesterID != "" {
		if s.requireProjectRole(r.Context(), w, projectID, requesterID,
			types.ProjectRoleOwner, types.ProjectRoleAdmin,
			types.ProjectRoleMember, types.ProjectRoleViewer) == nil {
			return
		}
	}

	// List tokens
	tokens, err := s.sharedTokenManager.ListSharedTokens(r.Context(), projectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list tokens: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"tokens": tokens,
		"count":  len(tokens),
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleExtendSharedToken extends the expiration of a shared token
// PATCH /api/v1/invitations/shared/{token}/extend
func (s *Server) handleExtendSharedToken(w http.ResponseWriter, r *http.Request) {
	// Parse token from path
	path := r.URL.Path[len("/api/v1/invitations/shared/"):]
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	tokenCode := parts[0]

	// Parse request
	var req struct {
		AddDays int `json:"add_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.AddDays <= 0 {
		http.Error(w, "add_days must be positive", http.StatusBadRequest)
		return
	}

	// Look up token to obtain projectID for auth check (Gap N fix).
	tokenInfo, lookupErr := s.sharedTokenManager.GetSharedToken(r.Context(), tokenCode)
	if lookupErr == nil {
		if requesterID := r.URL.Query().Get("requester_id"); requesterID != "" {
			if s.requireProjectRole(r.Context(), w, tokenInfo.ProjectID, requesterID,
				types.ProjectRoleOwner, types.ProjectRoleAdmin) == nil {
				return
			}
		}
	}

	// Extend expiration
	duration := time.Duration(req.AddDays) * 24 * time.Hour
	err := s.sharedTokenManager.ExtendExpiration(r.Context(), tokenCode, duration)
	if err != nil {
		if err == types.ErrSharedTokenNotFound {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		if err == types.ErrSharedTokenRevoked {
			http.Error(w, "Cannot extend revoked token", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to extend token: %v", err), http.StatusInternalServerError)
		return
	}

	// Get updated token
	token, _ := s.sharedTokenManager.GetSharedToken(r.Context(), tokenCode)

	response := map[string]interface{}{
		"success":    true,
		"token":      tokenCode,
		"expires_at": token.ExpiresAt,
		"message":    fmt.Sprintf("Token expiration extended by %d days", req.AddDays),
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleRevokeSharedToken revokes a shared token
// DELETE /api/v1/invitations/shared/{token}
func (s *Server) handleRevokeSharedToken(w http.ResponseWriter, r *http.Request) {
	// Parse token from path
	path := r.URL.Path[len("/api/v1/invitations/shared/"):]
	tokenCode := strings.TrimSuffix(path, "/")

	// Look up token to obtain projectID for auth check.
	tokenInfo, err := s.sharedTokenManager.GetSharedToken(r.Context(), tokenCode)
	if err != nil {
		if err == types.ErrSharedTokenNotFound {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	// Optional: verify requester is owner or admin (Gap N fix).
	if requesterID := r.URL.Query().Get("requester_id"); requesterID != "" {
		if s.requireProjectRole(r.Context(), w, tokenInfo.ProjectID, requesterID,
			types.ProjectRoleOwner, types.ProjectRoleAdmin) == nil {
			return
		}
	}

	// Revoke token
	if err := s.sharedTokenManager.RevokeSharedToken(r.Context(), tokenCode); err != nil {
		if err == types.ErrSharedTokenNotFound {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to revoke token: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"token":   tokenCode,
		"message": "Token revoked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleGenerateQRCode generates a QR code for a shared token
// GET /api/v1/invitations/shared/{token}/qr
func (s *Server) handleGenerateQRCode(w http.ResponseWriter, r *http.Request) {
	// Parse token from path
	path := r.URL.Path[len("/api/v1/invitations/shared/"):]
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	tokenCode := parts[0]

	// Verify token exists
	token, err := s.sharedTokenManager.GetSharedToken(r.Context(), tokenCode)
	if err != nil {
		if err == types.ErrSharedTokenNotFound {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate redemption URL
	redemptionURL := fmt.Sprintf("https://prism.dev/join/%s", tokenCode)

	// Check format parameter (default: json with base64)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Generate QR code
	qrCode, err := generateQRCode(redemptionURL, format)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate QR code: %v", err), http.StatusInternalServerError)
		return
	}

	// Return based on format
	switch format {
	case "png":
		// Return PNG image directly
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.png", tokenCode))
		if _, err := w.Write(qrCode); err != nil { //nolint:gosec // G705: server-generated PNG bytes served as image/png, not HTML
			log.Printf("Error writing QR code response: %v", err)
		}

	case "json":
		// Return JSON with base64-encoded PNG
		response := map[string]interface{}{
			"token":          tokenCode,
			"name":           token.Name,
			"redemption_url": redemptionURL,
			"qr_code_data":   base64EncodeQRCode(qrCode),
			"format":         "png",
			"message":        "QR code generated successfully",
		}
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(w, response)

	default:
		http.Error(w, "Invalid format (use 'json' or 'png')", http.StatusBadRequest)
	}
}

// generateQRCode generates a QR code image for the given URL
func generateQRCode(url string, format string) ([]byte, error) {
	// Generate QR code with medium recovery level
	// Size 256x256 pixels is good for both printing and display
	qr, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	}

	return qr, nil
}

// base64EncodeQRCode encodes a QR code image to base64 string
func base64EncodeQRCode(qrCode []byte) string {
	return base64.StdEncoding.EncodeToString(qrCode)
}
