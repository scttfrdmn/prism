package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/research"
)

// ResearchUserRequest represents a request to create a research user
type ResearchUserRequest struct {
	Username string `json:"username"`
}

// ResearchUserSSHKeyRequest represents a request to manage SSH keys
type ResearchUserSSHKeyRequest struct {
	Username string `json:"username"`
	KeyType  string `json:"key_type,omitempty"` // "ed25519" or "rsa"
}

// handleResearchUsers handles research user collection operations
func (s *Server) handleResearchUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListResearchUsers(w, r)
	case http.MethodPost:
		s.handleCreateResearchUser(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListResearchUsers lists all research users
func (s *Server) handleListResearchUsers(w http.ResponseWriter, r *http.Request) {
	service, err := s.getResearchUserService()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize research user service: %v", err))
		return
	}

	users, err := service.ListResearchUsers()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list research users: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(users)
}

// handleCreateResearchUser creates a new research user
func (s *Server) handleCreateResearchUser(w http.ResponseWriter, r *http.Request) {
	var req ResearchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" {
		s.writeError(w, http.StatusBadRequest, "Username is required")
		return
	}

	service, err := s.getResearchUserService()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize research user service: %v", err))
		return
	}

	// Check if user already exists
	existingUser, err := service.GetResearchUser(req.Username)
	if err == nil {
		// User already exists, return it
		_ = json.NewEncoder(w).Encode(existingUser)
		return
	}

	// Create new user
	user, err := service.CreateResearchUser(req.Username, &research.CreateResearchUserOptions{
		GenerateSSHKey: true, // Generate SSH key by default
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create research user: %v", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(user)
}

// handleResearchUserOperations handles individual research user operations
func (s *Server) handleResearchUserOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/v1/research-users/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing username")
		return
	}

	username := parts[0]

	if len(parts) == 1 {
		// Operations on the user itself: GET /api/v1/research-users/{username}
		switch r.Method {
		case http.MethodGet:
			s.handleGetResearchUser(w, r, username)
		case http.MethodDelete:
			s.handleDeleteResearchUser(w, r, username)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 {
		// Sub-operations: /api/v1/research-users/{username}/{operation}
		operation := parts[1]
		switch operation {
		case "ssh-key":
			s.handleResearchUserSSHKey(w, r, username)
		case "status":
			s.handleResearchUserStatus(w, r, username)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown operation")
		}
	} else {
		s.writeError(w, http.StatusNotFound, "Invalid path")
	}
}

// handleGetResearchUser gets details for a specific research user
func (s *Server) handleGetResearchUser(w http.ResponseWriter, r *http.Request, username string) {
	service, err := s.getResearchUserService()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize research user service: %v", err))
		return
	}

	user, err := service.GetResearchUser(username)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Research user not found: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(user)
}

// handleDeleteResearchUser deletes a research user
func (s *Server) handleDeleteResearchUser(w http.ResponseWriter, r *http.Request, username string) {
	// For now, return method not implemented until we add DeleteResearchUser to service layer
	s.writeError(w, http.StatusNotImplemented, "Delete research user not yet implemented in service layer")
}

// handleResearchUserSSHKey handles SSH key operations for research users
func (s *Server) handleResearchUserSSHKey(w http.ResponseWriter, r *http.Request, username string) {
	switch r.Method {
	case http.MethodPost:
		s.handleGenerateResearchUserSSHKey(w, r, username)
	case http.MethodGet:
		s.handleListResearchUserSSHKeys(w, r, username)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGenerateResearchUserSSHKey generates SSH keys for a research user
func (s *Server) handleGenerateResearchUserSSHKey(w http.ResponseWriter, r *http.Request, username string) {
	var req ResearchUserSSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set default key type if not specified
	if req.KeyType == "" {
		req.KeyType = "ed25519"
	}

	service, err := s.getResearchUserService()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize research user service: %v", err))
		return
	}

	keyPair, _, err := service.ManageSSHKeys().GenerateKeyPair(username, req.KeyType)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate SSH key: %v", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"username":    username,
		"key_type":    req.KeyType,
		"public_key":  keyPair.PublicKey,
		"fingerprint": keyPair.Fingerprint,
	})
}

// handleListResearchUserSSHKeys lists SSH keys for a research user
func (s *Server) handleListResearchUserSSHKeys(w http.ResponseWriter, r *http.Request, username string) {
	service, err := s.getResearchUserService()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize research user service: %v", err))
		return
	}

	keys, err := service.ManageSSHKeys().ListKeys(username)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list SSH keys: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"username": username,
		"keys":     keys,
	})
}

// handleResearchUserStatus gets detailed status for a research user
func (s *Server) handleResearchUserStatus(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	service, err := s.getResearchUserService()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize research user service: %v", err))
		return
	}

	user, err := service.GetResearchUser(username)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Research user not found: %v", err))
		return
	}

	// Get current profile for context
	currentProfile, err := s.getCurrentProfile()
	if err != nil {
		currentProfile = "default" // Fallback
	}

	// Create detailed status response
	status := map[string]interface{}{
		"user":           user,
		"profile":        currentProfile,
		"ssh_keys_count": len(user.SSHPublicKeys),
		"status":         "active",
		"last_updated":   user.CreatedAt,
	}

	_ = json.NewEncoder(w).Encode(status)
}

// getResearchUserService creates a research user service instance
func (s *Server) getResearchUserService() (*research.ResearchUserService, error) {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".cloudworkstation")

	// Create profile adapter for daemon
	profileAdapter := &DaemonProfileAdapter{}

	// Create research user service with full functionality
	serviceConfig := &research.ResearchUserServiceConfig{
		ConfigDir:  configDir,
		ProfileMgr: profileAdapter,
	}

	return research.NewResearchUserService(serviceConfig), nil
}

// getCurrentProfile gets the current profile name
func (s *Server) getCurrentProfile() (string, error) {
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return "default", nil // Fallback to default
	}

	profile, err := profileManager.GetCurrentProfile()
	if err != nil {
		return "default", nil // Fallback to default
	}

	return profile.Name, nil
}

// DaemonProfileAdapter adapts the profile manager for daemon use
type DaemonProfileAdapter struct{}

func (d *DaemonProfileAdapter) GetCurrentProfile() (string, error) {
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return "default", nil
	}

	profile, err := profileManager.GetCurrentProfile()
	if err != nil {
		return "default", nil
	}

	return profile.Name, nil
}

func (d *DaemonProfileAdapter) GetProfileConfig(profileID string) (interface{}, error) {
	// Get profile using enhanced profile manager
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return nil, fmt.Errorf("failed to create profile manager: %w", err)
	}

	profileConfig, err := profileManager.GetProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile config: %w", err)
	}

	return profileConfig, nil
}

func (d *DaemonProfileAdapter) UpdateProfileConfig(profileID string, config interface{}) error {
	// Update profile using enhanced profile manager
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	// Convert config to profile.Profile if needed
	if profileConfig, ok := config.(*profile.Profile); ok {
		return profileManager.UpdateProfile(profileID, *profileConfig)
	}

	return fmt.Errorf("invalid profile config type")
}
