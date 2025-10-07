package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// handleSnapshotOperations routes snapshot operations based on URL path
func (s *Server) handleSnapshotOperations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/snapshots/")

	// Split path to handle nested operations
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		s.writeError(w, http.StatusBadRequest, "Missing snapshot identifier")
		return
	}

	snapshotName := parts[0]

	// Handle restore operation
	if len(parts) > 1 && parts[1] == "restore" {
		s.handleRestoreInstanceFromSnapshot(w, r, snapshotName)
		return
	}

	// Handle individual snapshot operations
	s.handleSnapshot(w, r, snapshotName)
}

// handleSnapshots handles instance snapshot collection operations
func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListInstanceSnapshots(w, r)
	case http.MethodPost:
		s.handleCreateInstanceSnapshot(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleSnapshot handles individual snapshot operations
func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request, snapshotName string) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetInstanceSnapshot(w, r, snapshotName)
	case http.MethodDelete:
		s.handleDeleteInstanceSnapshot(w, r, snapshotName)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleCreateInstanceSnapshot creates a snapshot from an instance
func (s *Server) handleCreateInstanceSnapshot(w http.ResponseWriter, r *http.Request) {
	var req types.InstanceSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate required fields
	if req.InstanceName == "" {
		s.writeError(w, http.StatusBadRequest, "instance_name is required")
		return
	}
	if req.SnapshotName == "" {
		s.writeError(w, http.StatusBadRequest, "snapshot_name is required")
		return
	}

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		// Create the snapshot
		result, err := awsManager.CreateInstanceAMISnapshot(
			req.InstanceName,
			req.SnapshotName,
			req.Description,
			req.NoReboot,
		)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				s.writeError(w, http.StatusNotFound, err.Error())
			} else if strings.Contains(err.Error(), "must be running") {
				s.writeError(w, http.StatusBadRequest, err.Error())
			} else {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create snapshot: %v", err))
			}
			return err
		}

		// If wait is requested, monitor snapshot creation
		if req.Wait {
			s.writeJSON(w, http.StatusAccepted, map[string]interface{}{
				"snapshot": result,
				"message":  "Snapshot creation initiated. Monitoring progress...",
			})
		} else {
			s.writeJSON(w, http.StatusCreated, result)
		}
		return nil
	})
}

// handleListInstanceSnapshots lists all instance snapshots
func (s *Server) handleListInstanceSnapshots(w http.ResponseWriter, r *http.Request) {
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		// List snapshots
		snapshots, err := awsManager.ListInstanceSnapshots()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list snapshots: %v", err))
			return err
		}

		response := types.InstanceSnapshotListResponse{
			Snapshots: snapshots,
			Count:     len(snapshots),
		}

		s.writeJSON(w, http.StatusOK, response)
		return nil
	})
}

// handleGetInstanceSnapshot gets information about a specific snapshot
func (s *Server) handleGetInstanceSnapshot(w http.ResponseWriter, r *http.Request, snapshotName string) {
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		// Get snapshot info
		snapshot, err := awsManager.GetInstanceSnapshotInfo(snapshotName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				s.writeError(w, http.StatusNotFound, err.Error())
			} else {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get snapshot info: %v", err))
			}
			return err
		}

		s.writeJSON(w, http.StatusOK, snapshot)
		return nil
	})
}

// handleDeleteInstanceSnapshot deletes a snapshot
func (s *Server) handleDeleteInstanceSnapshot(w http.ResponseWriter, r *http.Request, snapshotName string) {
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		// Delete the snapshot
		result, err := awsManager.DeleteInstanceSnapshot(snapshotName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				s.writeError(w, http.StatusNotFound, err.Error())
			} else {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete snapshot: %v", err))
			}
			return err
		}

		s.writeJSON(w, http.StatusOK, result)
		return nil
	})
}

// handleRestoreInstanceFromSnapshot restores a new instance from a snapshot
func (s *Server) handleRestoreInstanceFromSnapshot(w http.ResponseWriter, r *http.Request, snapshotName string) {
	var req types.InstanceRestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate required fields
	if req.NewInstanceName == "" {
		s.writeError(w, http.StatusBadRequest, "new_instance_name is required")
		return
	}

	// Override snapshot name from URL path
	req.SnapshotName = snapshotName

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		// Restore the instance
		result, err := awsManager.RestoreInstanceFromSnapshot(req.SnapshotName, req.NewInstanceName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				s.writeError(w, http.StatusNotFound, err.Error())
			} else {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to restore instance: %v", err))
			}
			return err
		}

		if req.Wait {
			s.writeJSON(w, http.StatusAccepted, map[string]interface{}{
				"restore": result,
				"message": "Instance restore initiated. Monitoring progress...",
			})
		} else {
			s.writeJSON(w, http.StatusCreated, result)
		}
		return nil
	})
}
