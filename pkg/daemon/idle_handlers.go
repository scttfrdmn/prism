// Package daemon provides the daemon server implementation
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
)

// RegisterIdleRoutes registers all idle policy API routes
func (s *Server) RegisterIdleRoutes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Idle policy endpoints
	mux.HandleFunc("/api/v1/idle/policies", applyMiddleware(s.handleIdlePolicies))
	mux.HandleFunc("/api/v1/idle/policies/", applyMiddleware(s.handleIdlePolicyOperations))
	mux.HandleFunc("/api/v1/idle/schedules", applyMiddleware(s.handleIdleSchedules))
	mux.HandleFunc("/api/v1/idle/savings", applyMiddleware(s.handleIdleSavings))
}

// handleIdlePolicies handles /api/v1/idle/policies
func (s *Server) handleIdlePolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listIdlePolicies(w, r)
	case "POST":
		if strings.HasSuffix(r.URL.Path, "/recommend") {
			s.recommendIdlePolicy(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleIdlePolicyOperations handles /api/v1/idle/policies/{policyId}
func (s *Server) handleIdlePolicyOperations(w http.ResponseWriter, r *http.Request) {
	// Extract policy ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/idle/policies/")
	if path == "" {
		http.Error(w, "Policy ID required", http.StatusBadRequest)
		return
	}

	// Handle special routes
	if path == "recommend" {
		if r.Method == "POST" {
			s.recommendIdlePolicy(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case "GET":
		s.getIdlePolicy(w, r, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleIdleSchedules handles /api/v1/idle/schedules
func (s *Server) handleIdleSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listIdleSchedules(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleIdleSavings handles /api/v1/idle/savings
func (s *Server) handleIdleSavings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getIdleSavingsReport(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listIdlePolicies returns all available idle policy templates
func (s *Server) listIdlePolicies(w http.ResponseWriter, r *http.Request) {
	policyManager := idle.NewPolicyManager()
	policies := policyManager.ListTemplates()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policies); err != nil {
		http.Error(w, "Failed to encode policies", http.StatusInternalServerError)
		return
	}
}

// getIdlePolicy returns a specific idle policy template
func (s *Server) getIdlePolicy(w http.ResponseWriter, r *http.Request, policyID string) {
	policyManager := idle.NewPolicyManager()
	policy, err := policyManager.GetTemplate(policyID)
	if err != nil {
		http.Error(w, "Policy not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policy); err != nil {
		http.Error(w, "Failed to encode policy", http.StatusInternalServerError)
		return
	}
}

// recommendIdlePolicy recommends an idle policy for an instance
func (s *Server) recommendIdlePolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstanceType string            `json:"instance_type"`
		Tags         map[string]string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	policyManager := idle.NewPolicyManager()
	policy, err := policyManager.RecommendTemplate(req.InstanceType, req.Tags)
	if err != nil {
		http.Error(w, "Failed to recommend policy", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policy); err != nil {
		http.Error(w, "Failed to encode recommendation", http.StatusInternalServerError)
		return
	}
}

// listIdleSchedules returns active idle schedules
func (s *Server) listIdleSchedules(w http.ResponseWriter, r *http.Request) {
	// Get scheduler from AWS manager
	scheduler := s.awsManager.GetIdleScheduler()
	if scheduler == nil {
		http.Error(w, "Scheduler not available", http.StatusServiceUnavailable)
		return
	}

	// Get all schedules from scheduler
	schedules := scheduler.ListSchedules()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(schedules); err != nil {
		http.Error(w, "Failed to encode schedules", http.StatusInternalServerError)
		return
	}
}

// getIdleSavingsReport generates an idle cost savings report
func (s *Server) getIdleSavingsReport(w http.ResponseWriter, r *http.Request) {
	// TODO: Integrate with actual cost tracking
	// For now, return sample data
	report := map[string]interface{}{
		"report_id":    "report-sample",
		"generated_at": "2024-01-01T00:00:00Z",
		"period": map[string]string{
			"start": "2024-01-01",
			"end":   "2024-01-31",
		},
		"total_saved":        1234.56,
		"projected_savings":  1500.00,
		"idle_hours":         1234.5,
		"active_hours":       500.5,
		"savings_percentage": 71.1,
		"recommendations": []map[string]interface{}{
			{
				"type":        "enable_idle_detection",
				"description": "Enable idle detection on 2 more instances",
				"priority":    "high",
				"impact":      200.00,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, "Failed to encode report", http.StatusInternalServerError)
		return
	}
}

// Instance-specific idle policy handlers

// handleInstanceIdlePolicies handles /api/v1/instances/{instanceName}/idle/policies
func (s *Server) handleInstanceIdlePolicies(w http.ResponseWriter, r *http.Request, instanceName string) {
	switch r.Method {
	case "GET":
		s.getInstanceIdlePolicies(w, r, instanceName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleInstanceIdlePolicy handles /api/v1/instances/{instanceName}/idle/policies/{policyId}
func (s *Server) handleInstanceIdlePolicy(w http.ResponseWriter, r *http.Request, instanceName, policyID string) {
	switch r.Method {
	case "PUT":
		s.applyIdlePolicyToInstance(w, r, instanceName, policyID)
	case "DELETE":
		s.removeIdlePolicyFromInstance(w, r, instanceName, policyID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getInstanceIdlePolicies returns idle policies applied to an instance
func (s *Server) getInstanceIdlePolicies(w http.ResponseWriter, r *http.Request, instanceName string) {
	// Get applied policies from AWS manager
	policies, err := s.awsManager.GetInstancePolicies(instanceName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get instance policies: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policies); err != nil {
		http.Error(w, "Failed to encode policies", http.StatusInternalServerError)
		return
	}
}

// applyIdlePolicyToInstance applies an idle policy to an instance
func (s *Server) applyIdlePolicyToInstance(w http.ResponseWriter, r *http.Request, instanceName, policyID string) {
	// Apply the idle policy via AWS manager
	if err := s.awsManager.ApplyHibernationPolicy(instanceName, policyID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply idle policy: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Successfully applied idle policy %s to instance %s", policyID, instanceName),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// removeIdlePolicyFromInstance removes an idle policy from an instance
func (s *Server) removeIdlePolicyFromInstance(w http.ResponseWriter, r *http.Request, instanceName, policyID string) {
	// Remove the idle policy via AWS manager
	if err := s.awsManager.RemoveHibernationPolicy(instanceName, policyID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove idle policy: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Successfully removed idle policy %s from instance %s", policyID, instanceName),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
