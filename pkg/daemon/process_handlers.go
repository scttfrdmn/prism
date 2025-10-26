// Package daemon provides HTTP handlers for daemon process management operations.
//
// This file implements REST API endpoints for daemon process detection, management,
// and cleanup operations supporting Prism uninstallation scenarios.
//
// Endpoints:
//   GET  /api/v1/daemon/processes  - List all daemon processes
//   POST /api/v1/daemon/cleanup    - Perform comprehensive cleanup

package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// ProcessListResponse represents the response for listing daemon processes
type ProcessListResponse struct {
	Processes []ProcessInfo `json:"processes"`
	Total     int           `json:"total"`
	Status    string        `json:"status"`
}

// CleanupRequest represents a request for daemon cleanup
type CleanupRequest struct {
	ForceKill bool `json:"force_kill,omitempty"`
	RemoveAll bool `json:"remove_all,omitempty"`
}

// CleanupResponse represents the response for cleanup operations
type CleanupResponse struct {
	ProcessesFound   int      `json:"processes_found"`
	ProcessesCleaned int      `json:"processes_cleaned"`
	ProcessesFailed  int      `json:"processes_failed"`
	FailedProcesses  []int    `json:"failed_processes,omitempty"`
	FilesRemoved     []string `json:"files_removed,omitempty"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
}

// handleDaemonProcesses handles GET requests to list daemon processes
func (s *Server) handleDaemonProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed,
			fmt.Sprintf("Method %s not allowed", r.Method))
		return
	}

	log.Printf("Listing daemon processes")

	// Find all daemon processes
	processes, err := s.processManager.FindDaemonProcesses()
	if err != nil {
		log.Printf("Failed to find daemon processes: %v", err)
		s.writeError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to discover daemon processes: %v", err))
		return
	}

	response := ProcessListResponse{
		Processes: processes,
		Total:     len(processes),
		Status:    "success",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode processes response: %v", err)
	}

	log.Printf("Found %d daemon processes", len(processes))
}

// handleDaemonCleanup handles POST requests for daemon cleanup
func (s *Server) handleDaemonCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed,
			fmt.Sprintf("Method %s not allowed", r.Method))
		return
	}

	log.Printf("Starting daemon cleanup operation")

	// Parse request body
	var req CleanupRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Failed to parse cleanup request: %v", err)
			s.writeError(w, http.StatusBadRequest,
				fmt.Sprintf("Invalid request body: %v", err))
			return
		}
	}

	// Find processes before cleanup
	processesBeforeCleanup, err := s.processManager.FindDaemonProcesses()
	if err != nil {
		log.Printf("Failed to find processes before cleanup: %v", err)
		s.writeError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to discover processes before cleanup: %v", err))
		return
	}

	processesFound := len(processesBeforeCleanup)
	log.Printf("Found %d daemon processes to cleanup", processesFound)

	// Prepare response
	response := CleanupResponse{
		ProcessesFound:   processesFound,
		ProcessesCleaned: 0,
		ProcessesFailed:  0,
		FailedProcesses:  []int{},
		FilesRemoved:     []string{},
		Status:           "success",
	}

	// If no processes found, still perform file cleanup
	if processesFound == 0 {
		log.Printf("No daemon processes found, performing file cleanup only")
		response.Message = "No daemon processes found. File cleanup performed."

		// Clean up files directly
		s.cleanupDaemonFiles(&response)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode cleanup response: %v", err)
		}
		return
	}

	// Perform cleanup based on request options
	if req.ForceKill {
		log.Printf("Force kill requested for all daemon processes")
		response = s.performForceCleanup(processesBeforeCleanup, response)
	} else {
		log.Printf("Performing graceful cleanup of daemon processes")
		response = s.performGracefulCleanup(processesBeforeCleanup, response)
	}

	// Clean up daemon files
	s.cleanupDaemonFiles(&response)

	// Verify cleanup success
	processesAfterCleanup, err := s.processManager.FindDaemonProcesses()
	if err == nil && len(processesAfterCleanup) > 0 {
		response.Status = "partial_success"
		response.Message = fmt.Sprintf("Cleanup completed but %d processes may still be running",
			len(processesAfterCleanup))
		log.Printf("Warning: %d processes still running after cleanup", len(processesAfterCleanup))
	} else {
		response.Message = "All daemon processes cleaned up successfully"
		log.Printf("Daemon cleanup completed successfully")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode cleanup response: %v", err)
	}
}

// performGracefulCleanup attempts graceful shutdown of all processes
func (s *Server) performGracefulCleanup(processes []ProcessInfo, response CleanupResponse) CleanupResponse {
	for _, proc := range processes {
		log.Printf("Attempting graceful shutdown of PID %d", proc.PID)

		if err := s.processManager.GracefulShutdown(proc.PID); err != nil {
			log.Printf("Failed to gracefully shutdown PID %d: %v", proc.PID, err)
			response.ProcessesFailed++
			response.FailedProcesses = append(response.FailedProcesses, proc.PID)
		} else {
			log.Printf("Successfully shut down PID %d", proc.PID)
			response.ProcessesCleaned++
		}
	}

	return response
}

// performForceCleanup forcefully kills all processes
func (s *Server) performForceCleanup(processes []ProcessInfo, response CleanupResponse) CleanupResponse {
	for _, proc := range processes {
		log.Printf("Force killing PID %d", proc.PID)

		if err := s.processManager.ForceKill(proc.PID); err != nil {
			log.Printf("Failed to force kill PID %d: %v", proc.PID, err)
			response.ProcessesFailed++
			response.FailedProcesses = append(response.FailedProcesses, proc.PID)
		} else {
			log.Printf("Successfully force killed PID %d", proc.PID)
			response.ProcessesCleaned++
		}
	}

	return response
}

// cleanupDaemonFiles removes daemon-related files and updates response
func (s *Server) cleanupDaemonFiles(response *CleanupResponse) {
	// Get file paths from process manager
	pidFile := s.processManager.GetPIDFilePath()
	registryFile := s.processManager.GetRegistryPath()

	// List of files to clean up
	filesToClean := []string{pidFile, registryFile}

	for _, file := range filesToClean {
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				log.Printf("Warning: Failed to remove file %s: %v", file, err)
			} else {
				log.Printf("Removed file: %s", file)
				response.FilesRemoved = append(response.FilesRemoved, file)
			}
		}
	}
}
