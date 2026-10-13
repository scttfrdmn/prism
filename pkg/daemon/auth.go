package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// handleAuth handles API authentication endpoints
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Generate new API key
		s.handleGenerateAPIKey(w, r)
	case http.MethodGet:
		// Get authentication status
		s.handleGetAuthStatus(w, r)
	case http.MethodDelete:
		// Revoke API key
		s.handleRevokeAPIKey(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGenerateAPIKey generates a new API key
func (s *Server) handleGenerateAPIKey(w http.ResponseWriter, _ *http.Request) {
	// Generate a secure random API key
	apiKeyBytes := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(apiKeyBytes); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to generate secure API key")
		return
	}

	// Convert to hex string for easier handling
	apiKey := hex.EncodeToString(apiKeyBytes)

	// Save the API key
	if err := s.stateManager.SaveAPIKey(apiKey); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save API key")
		return
	}

	log.Printf("Generated new API key")

	// Return the API key in the response
	response := types.AuthResponse{
		APIKey:    apiKey,
		CreatedAt: time.Now(),
		Message:   "API key generated successfully. This key will not be shown again.",
	}

	// The generated key is intentionally returned once to the caller.
	_ = json.NewEncoder(w).Encode(response) //nolint:gosec // G117: returning the newly generated API key to its requester is the purpose of this endpoint
}

// handleGetAuthStatus gets the current authentication status
func (s *Server) handleGetAuthStatus(w http.ResponseWriter, r *http.Request) {
	apiKey, createdAt, err := s.stateManager.GetAPIKey()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve authentication status")
		return
	}

	// Only return detailed auth status to authenticated requests (#598)
	if isAuthenticated(r.Context()) {
		response := map[string]interface{}{
			"auth_enabled":  apiKey != "",
			"created_at":    createdAt,
			"authenticated": true,
		}
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	// Unauthenticated: return minimal info (don't leak whether auth is configured)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	})
}

// handleRevokeAPIKey revokes the current API key
func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	// Check if the request is authenticated
	if !isAuthenticated(r.Context()) {
		s.writeError(w, http.StatusUnauthorized, "Authentication required to revoke API key")
		return
	}

	// Clear the API key
	if err := s.stateManager.ClearAPIKey(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to revoke API key")
		return
	}

	log.Printf("API key revoked")

	w.WriteHeader(http.StatusNoContent)
}
