package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/throttle"
	"github.com/scttfrdmn/prism/pkg/types"
)

// handleThrottlingStatus returns current throttling status
// GET /api/v1/throttling/status?scope=<scope>
func (s *Server) handleThrottlingStatus(w http.ResponseWriter, r *http.Request) {
	if s.launchThrottler == nil {
		s.writeError(w, http.StatusNotFound, "Launch throttling not configured")
		return
	}

	// Get scope from query parameter (default: global)
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "global"
	}

	status := s.launchThrottler.GetStatus(scope)
	s.writeJSON(w, http.StatusOK, status)
}

// handleThrottlingConfigure updates throttling configuration
// POST /api/v1/throttling/configure
func (s *Server) handleThrottlingConfigure(w http.ResponseWriter, r *http.Request) {
	var config throttle.Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Create new throttler with updated config
	s.launchThrottler = throttle.NewLaunchThrottler(config)

	if config.BudgetAware && s.budgetManager != nil {
		s.launchThrottler.SetBudgetUsageFunc(func(projectID string) (used float64, limit float64, err error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			summary, err := s.budgetManager.GetProjectFundingSummary(ctx, projectID)
			if err != nil {
				return 0, 0, err
			}

			return summary.TotalSpent, summary.TotalAllocated, nil
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Throttling configuration updated successfully",
		"config":  config,
	})
}

// handleThrottlingRemaining returns remaining launch tokens
// GET /api/v1/throttling/remaining?scope=<scope>
func (s *Server) handleThrottlingRemaining(w http.ResponseWriter, r *http.Request) {
	if s.launchThrottler == nil {
		s.writeError(w, http.StatusNotFound, "Launch throttling not configured")
		return
	}

	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "global"
	}

	status := s.launchThrottler.GetStatus(scope)

	response := map[string]interface{}{
		"scope":             status.Scope,
		"enabled":           status.Enabled,
		"current_tokens":    status.CurrentTokens,
		"max_launches":      status.MaxLaunches,
		"time_window":       status.TimeWindow,
		"next_refill":       status.NextTokenRefill,
		"time_until_refill": status.TimeUntilRefill.String(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleSetProjectOverride sets a project-specific throttling override
// POST /api/v1/throttling/projects/{projectID}/override
func (s *Server) handleSetProjectOverride(w http.ResponseWriter, r *http.Request) {
	if s.launchThrottler == nil {
		s.writeError(w, http.StatusNotFound, "Launch throttling not configured")
		return
	}

	projectID := r.URL.Path[len("/api/v1/throttling/projects/"):]
	// Remove "/override" suffix
	if len(projectID) > 9 && projectID[len(projectID)-9:] == "/override" {
		projectID = projectID[:len(projectID)-9]
	}

	var override throttle.Override
	if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Set project ID from URL
	override.ProjectID = projectID

	s.launchThrottler.SetProjectOverride(override)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  fmt.Sprintf("Override set for project %s", projectID),
		"override": override,
	})
}

// handleRemoveProjectOverride removes a project-specific override
// DELETE /api/v1/throttling/projects/{projectID}/override
func (s *Server) handleRemoveProjectOverride(w http.ResponseWriter, r *http.Request) {
	if s.launchThrottler == nil {
		s.writeError(w, http.StatusNotFound, "Launch throttling not configured")
		return
	}

	projectID := r.URL.Path[len("/api/v1/throttling/projects/"):]
	// Remove "/override" suffix
	if len(projectID) > 9 && projectID[len(projectID)-9:] == "/override" {
		projectID = projectID[:len(projectID)-9]
	}

	s.launchThrottler.RemoveProjectOverride(projectID)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Override removed for project %s", projectID),
	})
}

// handleListProjectOverrides lists all project overrides
// GET /api/v1/throttling/projects/overrides
func (s *Server) handleListProjectOverrides(w http.ResponseWriter, r *http.Request) {
	if s.launchThrottler == nil {
		s.writeError(w, http.StatusNotFound, "Launch throttling not configured")
		return
	}

	overrides := s.launchThrottler.ListProjectOverrides()
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"overrides": overrides,
		"count":     len(overrides),
	})
}

// handleProjectThrottlingOperations routes project-specific throttling operations
// POST /api/v1/throttling/projects/{projectID}/override - Set override
// DELETE /api/v1/throttling/projects/{projectID}/override - Remove override
func (s *Server) handleProjectThrottlingOperations(w http.ResponseWriter, r *http.Request) {
	// Extract path after /api/v1/throttling/projects/
	path := r.URL.Path[len("/api/v1/throttling/projects/"):]

	// Check if it's an override operation
	if strings.HasSuffix(path, "/override") {
		switch r.Method {
		case http.MethodPost:
			s.handleSetProjectOverride(w, r)
		case http.MethodDelete:
			s.handleRemoveProjectOverride(w, r)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	s.writeError(w, http.StatusNotFound, "Not found")
}

// checkLaunchThrottling checks if a launch is allowed under throttling rules
// This is called from handleLaunchInstance before actually launching
func (s *Server) checkLaunchThrottling(req *types.LaunchRequest, w http.ResponseWriter) bool {
	if s.launchThrottler == nil {
		return true // Throttling not configured, allow launch
	}

	// Build throttling request
	throttleReq := throttle.LaunchRequest{
		UserID:       req.ResearchUser, // Research user (Phase 5A+)
		ProjectID:    req.ProjectID,    // Project ID if applicable
		TemplateName: req.Template,     // Template being launched
		InstanceType: req.Size,         // Size acts as instance type hint
		// IsGPU will be determined by size
	}

	// Check if size contains GPU indicators
	if throttleReq.InstanceType != "" {
		sizeLower := strings.ToLower(req.Size)
		throttleReq.IsGPU = strings.Contains(sizeLower, "gpu")
	}

	// Check throttling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// A dry run sees the same verdict but consumes no throttle capacity.
	allow := s.launchThrottler.AllowLaunch
	if req.DryRun {
		allow = s.launchThrottler.CheckLaunch
	}
	err := allow(ctx, throttleReq)
	if err != nil {
		// Throttle error - reject launch
		if throttleErr, ok := err.(*throttle.ThrottleError); ok {
			s.writeError(w, http.StatusTooManyRequests, throttleErr.Message)
		} else {
			s.writeError(w, http.StatusTooManyRequests, err.Error())
		}
		return false
	}

	return true // Launch allowed
}
