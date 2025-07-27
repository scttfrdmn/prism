package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// handleVolumes handles volume collection operations
func (s *Server) handleVolumes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListVolumes(w, r)
	case http.MethodPost:
		s.handleCreateVolume(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListVolumes lists all volumes
func (s *Server) handleListVolumes(w http.ResponseWriter, r *http.Request) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	json.NewEncoder(w).Encode(state.Volumes)
}

// handleCreateVolume creates a new volume
func (s *Server) handleCreateVolume(w http.ResponseWriter, r *http.Request) {
	var req types.VolumeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	volume, err := awsManager.CreateVolume(req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create volume: %v", err))
		return
	}

	// Save state
	if err := s.stateManager.SaveVolume(*volume); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save volume state")
		return
	}

	json.NewEncoder(w).Encode(volume)
}

// handleVolumeOperations handles operations on specific volumes
func (s *Server) handleVolumeOperations(w http.ResponseWriter, r *http.Request) {
	volumeName := r.URL.Path[len("/api/v1/volumes/"):]
	
	switch r.Method {
	case http.MethodGet:
		s.handleGetVolume(w, r, volumeName)
	case http.MethodDelete:
		s.handleDeleteVolume(w, r, volumeName)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetVolume gets details of a specific volume
func (s *Server) handleGetVolume(w http.ResponseWriter, r *http.Request, name string) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	volume, exists := state.Volumes[name]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Volume not found")
		return
	}

	json.NewEncoder(w).Encode(volume)
}

// handleDeleteVolume deletes a specific volume
func (s *Server) handleDeleteVolume(w http.ResponseWriter, r *http.Request, name string) {
	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.DeleteVolume(name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete volume: %v", err))
		return
	}

	// Remove from state
	if err := s.stateManager.RemoveVolume(name); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to update state")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}