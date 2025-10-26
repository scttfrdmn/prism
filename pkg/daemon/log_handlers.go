package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/scttfrdmn/prism/pkg/types"
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

		// Parse timestamps from log lines and filter
		filteredLogs := filterLogsByTimestamp(logs, cutoff)
		logs = filteredLogs
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

// filterLogsByTimestamp filters log lines by parsing timestamps and comparing to cutoff
func filterLogsByTimestamp(logs []string, cutoff time.Time) []string {
	if len(logs) == 0 {
		return logs
	}

	// Common log timestamp patterns
	// RFC3339: 2006-01-02T15:04:05Z07:00
	// Syslog: Jan 2 15:04:05
	// ISO8601: 2006-01-02 15:04:05
	// CloudWatch: 2006/01/02 15:04:05
	// Systemd: [  123.456789] (seconds since boot)

	timestampPatterns := []struct {
		regex  *regexp.Regexp
		layout string
		parse  func(string) (time.Time, error)
	}{
		{
			// RFC3339: 2024-10-07T12:34:56Z or 2024-10-07T12:34:56-07:00
			regex:  regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})`),
			layout: time.RFC3339,
			parse: func(s string) (time.Time, error) {
				return time.Parse(time.RFC3339, s)
			},
		},
		{
			// ISO8601: 2024-10-07 12:34:56
			regex:  regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`),
			layout: "2006-01-02 15:04:05",
			parse: func(s string) (time.Time, error) {
				return time.Parse("2006-01-02 15:04:05", s)
			},
		},
		{
			// CloudWatch: 2024/10/07 12:34:56
			regex:  regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`),
			layout: "2006/01/02 15:04:05",
			parse: func(s string) (time.Time, error) {
				return time.Parse("2006/01/02 15:04:05", s)
			},
		},
		{
			// Syslog: Oct  7 12:34:56 or Oct 07 12:34:56
			regex:  regexp.MustCompile(`^[A-Z][a-z]{2}\s+\d{1,2} \d{2}:\d{2}:\d{2}`),
			layout: "Jan _2 15:04:05",
			parse: func(s string) (time.Time, error) {
				// Syslog doesn't include year, assume current year
				t, err := time.Parse("Jan _2 15:04:05", s)
				if err != nil {
					return time.Time{}, err
				}
				now := time.Now()
				return time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
			},
		},
		{
			// Systemd journal: [  123.456789] kernel message
			regex:  regexp.MustCompile(`^\[\s*\d+\.\d+\]`),
			layout: "",
			parse: func(s string) (time.Time, error) {
				// Boot time offset - can't reliably convert to absolute time
				// Return zero time to skip filtering on these lines
				return time.Time{}, fmt.Errorf("relative timestamp")
			},
		},
	}

	filtered := make([]string, 0, len(logs))

	for _, line := range logs {
		if len(line) == 0 {
			filtered = append(filtered, line)
			continue
		}

		// Try each timestamp pattern
		var logTime time.Time
		var parsed bool

		for _, pattern := range timestampPatterns {
			if match := pattern.regex.FindString(line); match != "" {
				t, err := pattern.parse(match)
				if err == nil {
					logTime = t
					parsed = true
					break
				}
			}
		}

		// If we couldn't parse a timestamp, include the line (might be continuation)
		if !parsed {
			filtered = append(filtered, line)
			continue
		}

		// Filter by cutoff time
		if logTime.After(cutoff) || logTime.Equal(cutoff) {
			filtered = append(filtered, line)
		}
	}

	return filtered
}
