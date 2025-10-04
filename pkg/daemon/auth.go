package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
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

	_ = json.NewEncoder(w).Encode(response)
}

// handleGetAuthStatus gets the current authentication status
func (s *Server) handleGetAuthStatus(w http.ResponseWriter, r *http.Request) {
	apiKey, createdAt, err := s.stateManager.GetAPIKey()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve authentication status")
		return
	}

	// Don't return the actual API key, just whether it exists
	response := map[string]interface{}{
		"auth_enabled": apiKey != "",
		"created_at":   createdAt,
	}

	// If the request is authenticated, include more information
	if isAuthenticated(r.Context()) {
		response["authenticated"] = true
	}

	_ = json.NewEncoder(w).Encode(response)
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
