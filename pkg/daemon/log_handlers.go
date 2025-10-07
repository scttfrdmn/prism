package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// handleLogs handles log collection operations
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetLogs(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleLogOperations handles individual log operations
func (s *Server) handleLogOperations(w http.ResponseWriter, r *http.Request) {
	// Parse path to get instance identifier and operation
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/logs/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		s.writeError(w, http.StatusBadRequest, "Instance identifier required")
		return
	}

	instanceIdentifier := parts[0]

	// Resolve instance identifier to actual instance name
	instanceName, found := s.resolveInstanceIdentifier(instanceIdentifier)
	if !found {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %s", instanceIdentifier))
		return
	}

	// Handle different log operations
	if len(parts) == 1 {
		// /api/v1/logs/{instance} - Get logs for instance
		switch r.Method {
		case http.MethodGet:
			s.handleGetInstanceLogs(w, r, instanceName)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 && parts[1] == "types" {
		// /api/v1/logs/{instance}/types - Get available log types
		switch r.Method {
		case http.MethodGet:
			s.handleGetLogTypes(w, r, instanceName)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else {
		s.writeError(w, http.StatusNotFound, "Log endpoint not found")
	}
}

// handleGetLogs handles listing all instances with log availability
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	// Load current state
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load state: %v", err))
		return
	}

	// Create log availability summary
	var logSummary []types.InstanceLogSummary
	for name, instance := range state.Instances {
		summary := types.InstanceLogSummary{
			Name:          name,
			ID:            instance.ID,
			State:         instance.State,
			LogsAvailable: instance.State == "running" || instance.State == "stopped", // Console logs always available, system logs need SSM
		}
		logSummary = append(logSummary, summary)
	}

	response := types.LogSummaryResponse{
		Instances:         logSummary,
		AvailableLogTypes: []string{"console", "cloud-init", "cloud-init-out", "messages", "secure", "boot", "dmesg", "kern", "syslog"},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleGetInstanceLogs retrieves logs for a specific instance
func (s *Server) handleGetInstanceLogs(w http.ResponseWriter, r *http.Request, instanceName string) {
	// Get query parameters
	logType := r.URL.Query().Get("type")
	if logType == "" {
		logType = "console" // Default to console logs
	}

	tailStr := r.URL.Query().Get("tail")
	tail := 0
	if tailStr != "" {
		if parsed, err := strconv.Atoi(tailStr); err == nil && parsed > 0 {
			tail = parsed
		}
	}

	since := r.URL.Query().Get("since")
	follow := r.URL.Query().Get("follow") == "true"

	// Get instance info from state
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load state: %v", err))
		return
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %s", instanceName))
		return
	}

	// Retrieve logs using AWS manager
	var logs []string
	var timestamp time.Time
	var logError error

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		if logType == "console" {
			// Get EC2 console output
			consoleOutput, err := awsManager.GetInstanceConsoleOutput(instance.ID)
			if err != nil {
				logError = err
				return fmt.Errorf("failed to get console logs: %w", err)
			}
			logs = strings.Split(consoleOutput, "\n")
			timestamp = time.Now()
		} else {
			// Get system logs via SSM
			systemLogs, err := awsManager.GetInstanceSystemLogs(instance.ID, logType)
			if err != nil {
				logError = err
				return fmt.Errorf("failed to get system logs: %w", err)
			}
			logs = systemLogs
			timestamp = time.Now()
		}
		return nil
	})

	if logError != nil {
		// Error already handled by withAWSManager
		return
	}

	// Apply tail limit if specified
	if tail > 0 && len(logs) > tail {
		logs = logs[len(logs)-tail:]
	}

	// Filter by time if since parameter is provided
	if since != "" {
		// Parse since duration (e.g., "1h", "30m", "2h30m")
		duration, err := time.ParseDuration(since)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid since duration: %s", since))
			return
		}
		cutoff := time.Now().Add(-duration)

		// Note: This is a simplified implementation
		// In a real implementation, you'd parse timestamps from log lines
		// For now, we return all logs if within the duration
		if timestamp.Before(cutoff) {
			logs = []string{"No logs available within the specified time range"}
		}
	}

	response := types.LogResponse{
		InstanceName: instanceName,
		InstanceID:   instance.ID,
		LogType:      logType,
		Lines:        logs,
		Timestamp:    timestamp,
		Follow:       follow,
		Tail:         tail,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleGetLogTypes returns available log types for an instance
func (s *Server) handleGetLogTypes(w http.ResponseWriter, r *http.Request, instanceName string) {
	// Get instance info from state
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load state: %v", err))
		return
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %s", instanceName))
		return
	}

	// Get available log types using AWS manager
	var logTypes []string
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		logTypes = awsManager.GetAvailableLogTypes()
		return nil
	})

	response := types.LogTypesResponse{
		InstanceName:      instanceName,
		InstanceID:        instance.ID,
		AvailableLogTypes: logTypes,
		SSMEnabled:        instance.State == "running", // SSM only works on running instances
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
