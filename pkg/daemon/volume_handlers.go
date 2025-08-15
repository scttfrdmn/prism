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

	// Convert map to slice for API consistency
	volumes := make([]types.EFSVolume, 0, len(state.Volumes))
	for _, volume := range state.Volumes {
		volumes = append(volumes, volume)
	}

	json.NewEncoder(w).Encode(volumes)
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
	path := r.URL.Path[len("/api/v1/volumes/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing volume name")
		return
	}

	volumeName := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleGetVolume(w, r, volumeName)
		case http.MethodDelete:
			s.handleDeleteVolume(w, r, volumeName)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 {
		operation := parts[1]
		switch operation {
		case "mount":
			s.handleMountVolume(w, r, volumeName)
		case "unmount":
			s.handleUnmountVolume(w, r, volumeName)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown operation")
		}
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

// handleMountVolume mounts an EFS volume to an instance
func (s *Server) handleMountVolume(w http.ResponseWriter, r *http.Request, volumeName string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	instanceName, ok := req["instance"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Missing instance name")
		return
	}

	mountPoint, ok := req["mount_point"]
	if !ok {
		mountPoint = "/mnt/" + volumeName // Default mount point
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.MountVolume(volumeName, instanceName, mountPoint)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to mount volume: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUnmountVolume unmounts an EFS volume from an instance
func (s *Server) handleUnmountVolume(w http.ResponseWriter, r *http.Request, volumeName string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	instanceName, ok := req["instance"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Missing instance name")
		return
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.UnmountVolume(volumeName, instanceName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to unmount volume: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
