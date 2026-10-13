package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/scttfrdmn/prism/pkg/profile"
)

// ProfileResponse represents a profile in API responses
type ProfileResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	AWSProfile    string  `json:"aws_profile"`
	Region        string  `json:"region"`
	Default       bool    `json:"default"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
	LastUsed      *string `json:"last_used,omitempty"`
	SSHKeyName    string  `json:"ssh_key_name,omitempty"`
	SSHKeyPath    string  `json:"ssh_key_path,omitempty"`
	UseDefaultKey bool    `json:"use_default_key,omitempty"`
}

// ProfileCreateRequest represents a request to create a profile
type ProfileCreateRequest struct {
	Name          string `json:"name"`
	AWSProfile    string `json:"aws_profile"`
	Region        string `json:"region"`
	SSHKeyName    string `json:"ssh_key_name,omitempty"`
	SSHKeyPath    string `json:"ssh_key_path,omitempty"`
	UseDefaultKey bool   `json:"use_default_key,omitempty"`
}

// ProfileUpdateRequest represents a request to update a profile
type ProfileUpdateRequest struct {
	Name          *string `json:"name,omitempty"`
	AWSProfile    *string `json:"aws_profile,omitempty"`
	Region        *string `json:"region,omitempty"`
	SSHKeyName    *string `json:"ssh_key_name,omitempty"`
	SSHKeyPath    *string `json:"ssh_key_path,omitempty"`
	UseDefaultKey *bool   `json:"use_default_key,omitempty"`
}

// parseProfilePath extracts the profile ID from a path like /api/v1/profiles/{id}
func parseProfilePath(path string) (string, []string, error) {
	// Expected format: /api/v1/profiles/{id} or /api/v1/profiles/{id}/action
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Minimum: ["api", "v1", "profiles", "{id}"]
	if len(parts) < 4 {
		return "", nil, fmt.Errorf("invalid profile path: %s", path)
	}

	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "profiles" {
		return "", nil, fmt.Errorf("invalid profile path: %s", path)
	}

	// Profile ID is at index 3
	profileID := parts[3]
	if profileID == "" {
		return "", nil, fmt.Errorf("profile ID is required")
	}

	// Return ID and any remaining path parts
	remainingParts := parts[4:]
	return profileID, remainingParts, nil
}

// handleListProfiles returns all profiles
func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	// Operation tracking removed for consistency

	// Use daemon's singleton profile manager (prevents race conditions)
	if s.profileManager == nil {
		s.writeError(w, http.StatusInternalServerError, "profile manager not initialized")
		return
	}

	// Get profiles with IDs
	profilesWithIDs, err := s.profileManager.ListProfilesWithIDs()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list profiles: %v", err))
		return
	}

	// Get current profile ID
	currentProfileID, _ := s.profileManager.GetCurrentProfileID()

	// Convert to response format
	var responses []ProfileResponse
	for _, pw := range profilesWithIDs {
		resp := profileToResponse(pw.ID, pw.Profile)
		if pw.ID == currentProfileID {
			resp.Default = true
		}
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, responses)
}

// handleGetCurrentProfile returns the currently active profile
func (s *Server) handleGetCurrentProfile(w http.ResponseWriter, r *http.Request) {
	// Operation tracking removed for consistency

	// Use daemon's singleton profile manager (prevents race conditions)
	if s.profileManager == nil {
		s.writeError(w, http.StatusInternalServerError, "profile manager not initialized")
		return
	}

	// Get current profile
	currentProfile, err := s.profileManager.GetCurrentProfile()
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("failed to get current profile: %v", err))
		return
	}

	// Get current profile ID
	currentProfileID, _ := s.profileManager.GetCurrentProfileID()

	resp := profileToResponse(currentProfileID, *currentProfile)
	resp.Default = true

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, resp)
}

// handleProfileOperations handles operations on specific profiles
func (s *Server) handleProfileOperations(w http.ResponseWriter, r *http.Request) {
	profileID, pathParts, err := parseProfilePath(r.URL.Path)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Handle /api/v1/profiles/{id}/activate
	if len(pathParts) > 0 && pathParts[0] == "activate" {
		if r.Method == "POST" {
			s.handleActivateProfile(w, r, profileID)
		} else {
			s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// Handle /api/v1/profiles/{id}
	switch r.Method {
	case "GET":
		s.handleGetProfile(w, r, profileID)
	case "PUT":
		s.handleUpdateProfile(w, r, profileID)
	case "DELETE":
		s.handleDeleteProfile(w, r, profileID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleGetProfile returns a specific profile by ID
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request, profileID string) {
	// Operation tracking removed for consistency)

	// Use daemon's singleton profile manager (prevents race conditions)
	if s.profileManager == nil {
		s.writeError(w, http.StatusInternalServerError, "profile manager not initialized")
		return
	}

	// Get profile
	prof, err := s.profileManager.GetProfile(profileID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("profile not found: %v", err))
		return
	}

	// Get current profile ID to check if this is the active one
	currentProfileID, _ := s.profileManager.GetCurrentProfileID()

	resp := profileToResponse(profileID, *prof)
	if profileID == currentProfileID {
		resp.Default = true
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, resp)
}

// handleProfiles handles /api/v1/profiles (list or create)
func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListProfiles(w, r)
	case "POST":
		s.handleCreateProfile(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleCreateProfile creates a new profile
func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	// Operation tracking removed for consistency

	// Parse request
	var req ProfileCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Validate required fields
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "profile name is required")
		return
	}
	if req.AWSProfile == "" {
		s.writeError(w, http.StatusBadRequest, "AWS profile is required")
		return
	}

	// Use daemon's singleton profile manager (prevents race conditions)
	if s.profileManager == nil {
		s.writeError(w, http.StatusInternalServerError, "profile manager not initialized")
		return
	}

	// Create profile object
	newProfile := profile.Profile{
		Type:          profile.ProfileTypePersonal,
		Name:          req.Name,
		AWSProfile:    req.AWSProfile,
		Region:        req.Region,
		SSHKeyName:    req.SSHKeyName,
		SSHKeyPath:    req.SSHKeyPath,
		UseDefaultKey: req.UseDefaultKey,
	}

	// Add profile (manager will generate ID and timestamps)
	if err := s.profileManager.AddProfile(newProfile); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			s.writeError(w, http.StatusConflict, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create profile: %v", err))
		}
		return
	}

	// Get the newly created profile to return its ID
	profilesWithIDs, err := s.profileManager.ListProfilesWithIDs()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("profile created but failed to retrieve: %v", err))
		return
	}

	// Find the profile we just created (by name)
	var createdProfile *ProfileResponse
	for _, pw := range profilesWithIDs {
		if pw.Profile.Name == req.Name {
			resp := profileToResponse(pw.ID, pw.Profile)
			createdProfile = &resp
			break
		}
	}

	if createdProfile == nil {
		s.writeError(w, http.StatusInternalServerError, "profile created but not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encodeJSONResponse(w, createdProfile)
}

// handleUpdateProfile updates an existing profile
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request, profileID string) {
	// Operation tracking removed for consistency)

	// Parse request
	var req ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Use daemon's singleton profile manager (prevents race conditions)
	if s.profileManager == nil {
		s.writeError(w, http.StatusInternalServerError, "profile manager not initialized")
		return
	}

	// Get existing profile
	existingProfile, err := s.profileManager.GetProfile(profileID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("profile not found: %v", err))
		return
	}

	// Apply updates
	updates := *existingProfile
	if req.Name != nil {
		updates.Name = *req.Name
	}
	if req.AWSProfile != nil {
		updates.AWSProfile = *req.AWSProfile
	}
	if req.Region != nil {
		updates.Region = *req.Region
	}
	if req.SSHKeyName != nil {
		updates.SSHKeyName = *req.SSHKeyName
	}
	if req.SSHKeyPath != nil {
		updates.SSHKeyPath = *req.SSHKeyPath
	}
	if req.UseDefaultKey != nil {
		updates.UseDefaultKey = *req.UseDefaultKey
	}

	// Update profile
	if err := s.profileManager.UpdateProfile(profileID, updates); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update profile: %v", err))
		return
	}

	// Get updated profile
	updatedProfile, err := s.profileManager.GetProfile(profileID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("profile updated but failed to retrieve: %v", err))
		return
	}

	// Check if this is the current profile
	currentProfileID, _ := s.profileManager.GetCurrentProfileID()
	resp := profileToResponse(profileID, *updatedProfile)
	if profileID == currentProfileID {
		resp.Default = true
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, resp)
}

// handleDeleteProfile deletes a profile
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request, profileID string) {
	// Operation tracking removed for consistency)

	// Use daemon's singleton profile manager (prevents race conditions)
	if s.profileManager == nil {
		s.writeError(w, http.StatusInternalServerError, "profile manager not initialized")
		return
	}

	// Check if this is the current profile
	currentProfileID, _ := s.profileManager.GetCurrentProfileID()
	if profileID == currentProfileID {
		s.writeError(w, http.StatusBadRequest, "cannot delete the currently active profile")
		return
	}

	// Delete profile
	if err := s.profileManager.RemoveProfile(profileID); err != nil {
		if err == profile.ErrProfileNotFound {
			s.writeError(w, http.StatusNotFound, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete profile: %v", err))
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleActivateProfile switches to a different profile
func (s *Server) handleActivateProfile(w http.ResponseWriter, r *http.Request, profileID string) {
	// Operation tracking removed for consistency)

	// Use daemon's singleton profile manager (prevents race conditions)
	if s.profileManager == nil {
		s.writeError(w, http.StatusInternalServerError, "profile manager not initialized")
		return
	}

	// Switch profile
	if err := s.profileManager.SwitchProfile(profileID); err != nil {
		if err == profile.ErrProfileNotFound {
			s.writeError(w, http.StatusNotFound, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to switch profile: %v", err))
		}
		return
	}

	// Get the activated profile
	activatedProfile, err := s.profileManager.GetProfile(profileID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("profile activated but failed to retrieve: %v", err))
		return
	}

	resp := profileToResponse(profileID, *activatedProfile)
	resp.Default = true

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, resp)
}

// profileToResponse converts a profile.Profile to ProfileResponse
func profileToResponse(id string, p profile.Profile) ProfileResponse {
	resp := ProfileResponse{
		ID:            id,
		Name:          p.Name,
		Type:          string(p.Type),
		AWSProfile:    p.AWSProfile,
		Region:        p.Region,
		Default:       false, // Always false - caller must set based on CurrentProfile
		CreatedAt:     p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		SSHKeyName:    p.SSHKeyName,
		SSHKeyPath:    p.SSHKeyPath,
		UseDefaultKey: p.UseDefaultKey,
	}

	if !p.UpdatedAt.IsZero() {
		updatedAt := p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.UpdatedAt = updatedAt
	}

	if p.LastUsed != nil && !p.LastUsed.IsZero() {
		lastUsed := p.LastUsed.Format("2006-01-02T15:04:05Z07:00")
		resp.LastUsed = &lastUsed
	}

	return resp
}
