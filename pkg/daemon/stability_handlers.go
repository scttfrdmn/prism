package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// handleStabilityMetrics returns current stability metrics
func (s *Server) handleStabilityMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get stability metrics
	stabilityMetrics := s.stabilityManager.GetStabilityMetrics()

	// Get health summary for additional context
	healthSummary := s.healthMonitor.GetHealthSummary()

	response := struct {
		StabilityMetrics interface{} `json:"stability_metrics"`
		HealthSummary    interface{} `json:"health_summary"`
		Timestamp        time.Time   `json:"timestamp"`
	}{
		StabilityMetrics: stabilityMetrics,
		HealthSummary:    healthSummary,
		Timestamp:        time.Now(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleStabilityErrors returns error history and analysis
func (s *Server) handleStabilityErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	limitStr := query.Get("limit")
	severityFilter := query.Get("severity")
	componentFilter := query.Get("component")

	// Default limit
	limit := 100
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get error history
	errorHistory := s.stabilityManager.GetErrorHistory()

	// Apply filters
	var filteredErrors []ErrorRecord
	for _, err := range errorHistory {
		// Apply severity filter
		if severityFilter != "" && string(err.Severity) != severityFilter {
			continue
		}

		// Apply component filter
		if componentFilter != "" && err.Component != componentFilter {
			continue
		}

		filteredErrors = append(filteredErrors, err)

		// Apply limit
		if len(filteredErrors) >= limit {
			break
		}
	}

	// Calculate error statistics
	errorStats := struct {
		TotalErrors     int                   `json:"total_errors"`
		FilteredErrors  int                   `json:"filtered_errors"`
		BySeverity      map[ErrorSeverity]int `json:"by_severity"`
		ByComponent     map[string]int        `json:"by_component"`
		RecentErrors    int                   `json:"recent_errors"` // Last hour
		RecoveredErrors int                   `json:"recovered_errors"`
	}{
		TotalErrors:    len(errorHistory),
		FilteredErrors: len(filteredErrors),
		BySeverity:     make(map[ErrorSeverity]int),
		ByComponent:    make(map[string]int),
	}

	// Calculate statistics
	recentCutoff := time.Now().Add(-1 * time.Hour)
	for _, err := range errorHistory {
		errorStats.BySeverity[err.Severity]++
		errorStats.ByComponent[err.Component]++

		if err.Timestamp.After(recentCutoff) {
			errorStats.RecentErrors++
		}

		if err.Recovered {
			errorStats.RecoveredErrors++
		}
	}

	response := struct {
		Errors     []ErrorRecord `json:"errors"`
		Statistics interface{}   `json:"statistics"`
		Filters    interface{}   `json:"filters"`
		Timestamp  time.Time     `json:"timestamp"`
	}{
		Errors:     filteredErrors,
		Statistics: errorStats,
		Filters: map[string]interface{}{
			"limit":     limit,
			"severity":  severityFilter,
			"component": componentFilter,
		},
		Timestamp: time.Now(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleCircuitBreakers returns circuit breaker status
func (s *Server) handleCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetCircuitBreakers(w, r)
	case http.MethodPost:
		s.handleResetCircuitBreaker(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetCircuitBreakers returns all circuit breaker statuses
func (s *Server) handleGetCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	circuitBreakers := s.stabilityManager.GetCircuitBreakerStatus()

	// Calculate summary statistics
	summary := struct {
		TotalBreakers    int `json:"total_breakers"`
		OpenBreakers     int `json:"open_breakers"`
		HalfOpenBreakers int `json:"half_open_breakers"`
		ClosedBreakers   int `json:"closed_breakers"`
	}{}

	for _, cb := range circuitBreakers {
		summary.TotalBreakers++
		switch cb.State {
		case CircuitBreakerOpen:
			summary.OpenBreakers++
		case CircuitBreakerHalfOpen:
			summary.HalfOpenBreakers++
		case CircuitBreakerClosed:
			summary.ClosedBreakers++
		}
	}

	response := struct {
		CircuitBreakers map[string]*CircuitBreaker `json:"circuit_breakers"`
		Summary         interface{}                `json:"summary"`
		Timestamp       time.Time                  `json:"timestamp"`
	}{
		CircuitBreakers: circuitBreakers,
		Summary:         summary,
		Timestamp:       time.Now(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleResetCircuitBreaker resets a specific circuit breaker
func (s *Server) handleResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if request.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Circuit breaker name is required")
		return
	}

	// Get circuit breaker
	cb := s.stabilityManager.GetCircuitBreaker(request.Name)
	if cb == nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Circuit breaker '%s' not found", request.Name))
		return
	}

	// Reset circuit breaker
	cb.State = CircuitBreakerClosed
	cb.FailureCount = 0

	// Record the reset
	s.stabilityManager.RecordRecovery("circuit_breaker", request.Name)

	response := struct {
		Message   string    `json:"message"`
		Name      string    `json:"name"`
		NewState  string    `json:"new_state"`
		Timestamp time.Time `json:"timestamp"`
	}{
		Message:   fmt.Sprintf("Circuit breaker '%s' has been reset", request.Name),
		Name:      request.Name,
		NewState:  string(CircuitBreakerClosed),
		Timestamp: time.Now(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleRecoveryTrigger triggers manual recovery operations
func (s *Server) handleRecoveryTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var request struct {
		Operation string `json:"operation"`
		Component string `json:"component,omitempty"`
		Force     bool   `json:"force,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if request.Operation == "" {
		s.writeError(w, http.StatusBadRequest, "Recovery operation is required")
		return
	}

	var result string
	var success bool

	switch request.Operation {
	case "memory_cleanup":
		// Force garbage collection
		s.stabilityManager.ForceGarbageCollection()
		result = "Memory cleanup completed"
		success = true

	case "memory_pressure":
		// Handle memory pressure
		if err := s.recoveryManager.HandleMemoryPressure(); err != nil {
			result = fmt.Sprintf("Memory pressure recovery failed: %v", err)
			success = false
		} else {
			result = "Memory pressure recovery completed"
			success = true
		}

	case "goroutine_check":
		// Check for goroutine leaks
		if err := s.recoveryManager.HandleGoroutineLeak(); err != nil {
			result = fmt.Sprintf("Goroutine leak detected: %v", err)
			success = false
		} else {
			result = "No goroutine leaks detected"
			success = true
		}

	case "health_check":
		// Trigger comprehensive health check
		if err := s.recoveryManager.HealthCheck(); err != nil {
			result = fmt.Sprintf("Health check failed: %v", err)
			success = false
		} else {
			result = "Health check passed"
			success = true
		}

	case "error_recovery":
		// Attempt error recovery for component
		if request.Component == "" {
			s.writeError(w, http.StatusBadRequest, "Component is required for error recovery")
			return
		}

		// Create a generic error for recovery attempt
		err := fmt.Errorf("manual recovery trigger")
		if recoveryErr := s.recoveryManager.RecoverFromError(request.Component, err); recoveryErr != nil {
			result = fmt.Sprintf("Recovery failed for component '%s': %v", request.Component, recoveryErr)
			success = false
		} else {
			result = fmt.Sprintf("Recovery completed for component '%s'", request.Component)
			success = true
		}

	default:
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Unknown recovery operation: %s", request.Operation))
		return
	}

	// Record the manual recovery attempt
	if success {
		s.stabilityManager.RecordRecovery("manual", request.Operation)
	} else {
		s.stabilityManager.RecordError("manual", request.Operation, result, ErrorSeverityMedium)
	}

	statusCode := http.StatusOK
	if !success {
		statusCode = http.StatusInternalServerError
	}

	response := struct {
		Success   bool      `json:"success"`
		Operation string    `json:"operation"`
		Component string    `json:"component,omitempty"`
		Result    string    `json:"result"`
		Timestamp time.Time `json:"timestamp"`
	}{
		Success:   success,
		Operation: request.Operation,
		Component: request.Component,
		Result:    result,
		Timestamp: time.Now(),
	}

	s.writeJSON(w, statusCode, response)
}
