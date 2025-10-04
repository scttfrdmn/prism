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
func (s *Server) handleGetCircuitBreakers(w http.ResponseWriter, _ *http.Request) {
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
	// Validate HTTP method
	if !s.validateHTTPMethod(w, r, http.MethodPost) {
		return
	}

	// Parse and validate request
	request, err := s.parseRecoveryRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Execute recovery operation
	result, success := s.executeRecoveryOperation(w, request)
	if result == "" {
		return // Error already handled in executeRecoveryOperation
	}

	// Record recovery attempt and send response
	s.recordAndRespondRecovery(w, request, result, success)
}

// validateHTTPMethod checks if the request uses the correct HTTP method
func (s *Server) validateHTTPMethod(w http.ResponseWriter, r *http.Request, expectedMethod string) bool {
	if r.Method != expectedMethod {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}
	return true
}

// parseRecoveryRequest parses and validates the recovery request
func (s *Server) parseRecoveryRequest(r *http.Request) (*recoveryRequest, error) {
	var request recoveryRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, fmt.Errorf("invalid request body")
	}

	if request.Operation == "" {
		return nil, fmt.Errorf("recovery operation is required")
	}

	return &request, nil
}

// executeRecoveryOperation executes the specified recovery operation
func (s *Server) executeRecoveryOperation(w http.ResponseWriter, request *recoveryRequest) (string, bool) {
	switch request.Operation {
	case "memory_cleanup":
		return s.handleMemoryCleanup()
	case "memory_pressure":
		return s.handleMemoryPressure()
	case "goroutine_check":
		return s.handleGoroutineCheck()
	case "health_check":
		return s.handleHealthCheck()
	case "error_recovery":
		return s.handleErrorRecovery(w, request)
	default:
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Unknown recovery operation: %s", request.Operation))
		return "", false
	}
}

// handleMemoryCleanup performs memory cleanup operations
func (s *Server) handleMemoryCleanup() (string, bool) {
	s.stabilityManager.ForceGarbageCollection()
	return "Memory cleanup completed", true
}

// handleMemoryPressure handles memory pressure recovery
func (s *Server) handleMemoryPressure() (string, bool) {
	if err := s.recoveryManager.HandleMemoryPressure(); err != nil {
		return fmt.Sprintf("Memory pressure recovery failed: %v", err), false
	}
	return "Memory pressure recovery completed", true
}

// handleGoroutineCheck checks for goroutine leaks
func (s *Server) handleGoroutineCheck() (string, bool) {
	if err := s.recoveryManager.HandleGoroutineLeak(); err != nil {
		return fmt.Sprintf("Goroutine leak detected: %v", err), false
	}
	return "No goroutine leaks detected", true
}

// handleHealthCheck performs comprehensive health check
func (s *Server) handleHealthCheck() (string, bool) {
	if err := s.recoveryManager.HealthCheck(); err != nil {
		return fmt.Sprintf("Health check failed: %v", err), false
	}
	return "Health check passed", true
}

// handleErrorRecovery attempts error recovery for a specific component
func (s *Server) handleErrorRecovery(w http.ResponseWriter, request *recoveryRequest) (string, bool) {
	if request.Component == "" {
		s.writeError(w, http.StatusBadRequest, "Component is required for error recovery")
		return "", false
	}

	// Create a generic error for recovery attempt
	err := fmt.Errorf("manual recovery trigger")
	if recoveryErr := s.recoveryManager.RecoverFromError(request.Component, err); recoveryErr != nil {
		return fmt.Sprintf("Recovery failed for component '%s': %v", request.Component, recoveryErr), false
	}
	return fmt.Sprintf("Recovery completed for component '%s'", request.Component), true
}

// recordAndRespondRecovery records the recovery attempt and sends the response
func (s *Server) recordAndRespondRecovery(w http.ResponseWriter, request *recoveryRequest, result string, success bool) {
	// Record the manual recovery attempt
	if success {
		s.stabilityManager.RecordRecovery("manual", request.Operation)
	} else {
		s.stabilityManager.RecordError("manual", request.Operation, result, ErrorSeverityMedium)
	}

	// Determine status code
	statusCode := http.StatusOK
	if !success {
		statusCode = http.StatusInternalServerError
	}

	// Create and send response
	response := s.createRecoveryResponse(request, result, success)
	s.writeJSON(w, statusCode, response)
}

// createRecoveryResponse creates the response structure for recovery operations
func (s *Server) createRecoveryResponse(request *recoveryRequest, result string, success bool) interface{} {
	return struct {
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
}

// recoveryRequest represents the structure of a recovery operation request
type recoveryRequest struct {
	Operation string `json:"operation"`
	Component string `json:"component,omitempty"`
	Force     bool   `json:"force,omitempty"`
}
