package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// handleStorage handles storage collection operations
func (s *Server) handleStorage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListStorage(w, r)
	case http.MethodPost:
		s.handleCreateStorage(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListStorage lists all storage volumes
func (s *Server) handleListStorage(w http.ResponseWriter, r *http.Request) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	// Build unified storage list from both EBS and EFS volumes
	storage := make([]*types.StorageVolume, 0, len(state.EBSVolumes)+len(state.Volumes))

	// Add EBS volumes (local storage)
	for _, ebsVol := range state.EBSVolumes {
		vol := ebsVol // Create copy to get address
		storage = append(storage, types.EBSVolumeToStorageVolume(&vol))
	}

	// Add EFS volumes (shared storage)
	for _, efsVol := range state.Volumes {
		vol := efsVol // Create copy to get address
		storage = append(storage, types.EFSVolumeToStorageVolume(&vol))
	}

	_ = json.NewEncoder(w).Encode(storage)
}

// handleCreateStorage creates a new storage volume
func (s *Server) handleCreateStorage(w http.ResponseWriter, r *http.Request) {
	var req types.StorageCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	volume, err := awsManager.CreateStorage(req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create storage: %v", err))
		return
	}

	// Save state
	if err := s.stateManager.SaveEBSVolume(*volume); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save storage state")
		return
	}

	// Convert to unified StorageVolume for response
	storageVolume := types.EBSVolumeToStorageVolume(volume)
	_ = json.NewEncoder(w).Encode(storageVolume)
}

// handleStorageOperations handles operations on specific storage volumes
func (s *Server) handleStorageOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/v1/storage/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing storage name")
		return
	}

	storageName := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleGetStorage(w, r, storageName)
		case http.MethodDelete:
			s.handleDeleteStorage(w, r, storageName)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 {
		operation := parts[1]
		switch operation {
		case "attach":
			s.handleAttachStorage(w, r, storageName)
		case "detach":
			s.handleDetachStorage(w, r, storageName)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown operation")
		}
	}
}

// handleGetStorage gets details of a specific storage volume
func (s *Server) handleGetStorage(w http.ResponseWriter, r *http.Request, name string) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	// Check EBS volumes (local storage) first
	if ebsVol, exists := state.EBSVolumes[name]; exists {
		storageVolume := types.EBSVolumeToStorageVolume(&ebsVol)
		_ = json.NewEncoder(w).Encode(storageVolume)
		return
	}

	// Check EFS volumes (shared storage)
	if efsVol, exists := state.Volumes[name]; exists {
		storageVolume := types.EFSVolumeToStorageVolume(&efsVol)
		_ = json.NewEncoder(w).Encode(storageVolume)
		return
	}

	s.writeError(w, http.StatusNotFound, "Storage not found")
}

// handleDeleteStorage deletes a specific storage volume
func (s *Server) handleDeleteStorage(w http.ResponseWriter, r *http.Request, name string) {
	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.DeleteStorage(name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete storage: %v", err))
		return
	}

	// Remove from state
	if err := s.stateManager.RemoveEBSVolume(name); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to update state")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleAttachStorage attaches a storage volume to an instance
func (s *Server) handleAttachStorage(w http.ResponseWriter, r *http.Request, storageName string) {
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

	err = awsManager.AttachStorage(storageName, instanceName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to attach storage: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDetachStorage detaches a storage volume from its instance
func (s *Server) handleDetachStorage(w http.ResponseWriter, r *http.Request, storageName string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.DetachStorage(storageName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detach storage: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
