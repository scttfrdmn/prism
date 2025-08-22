// Package daemon provides the daemon server implementation
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
)

// RegisterHibernationPolicyRoutes registers all hibernation policy API routes
func (s *Server) RegisterHibernationPolicyRoutes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Hibernation policy endpoints
	mux.HandleFunc("/api/v1/hibernation/policies", applyMiddleware(s.handleHibernationPolicies))
	mux.HandleFunc("/api/v1/hibernation/policies/", applyMiddleware(s.handleHibernationPolicyOperations))
	mux.HandleFunc("/api/v1/hibernation/schedules", applyMiddleware(s.handleHibernationSchedules))
	mux.HandleFunc("/api/v1/hibernation/savings", applyMiddleware(s.handleHibernationSavings))
}

// handleHibernationPolicies handles /api/v1/hibernation/policies
func (s *Server) handleHibernationPolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listHibernationPolicies(w, r)
	case "POST":
		if strings.HasSuffix(r.URL.Path, "/recommend") {
			s.recommendHibernationPolicy(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHibernationPolicyOperations handles /api/v1/hibernation/policies/{policyId}
func (s *Server) handleHibernationPolicyOperations(w http.ResponseWriter, r *http.Request) {
	// Extract policy ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/hibernation/policies/")
	if path == "" {
		http.Error(w, "Policy ID required", http.StatusBadRequest)
		return
	}
	
	// Handle special routes
	if path == "recommend" {
		if r.Method == "POST" {
			s.recommendHibernationPolicy(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	switch r.Method {
	case "GET":
		s.getHibernationPolicy(w, r, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHibernationSchedules handles /api/v1/hibernation/schedules
func (s *Server) handleHibernationSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listHibernationSchedules(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHibernationSavings handles /api/v1/hibernation/savings
func (s *Server) handleHibernationSavings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getHibernationSavingsReport(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listHibernationPolicies returns all available hibernation policy templates
func (s *Server) listHibernationPolicies(w http.ResponseWriter, r *http.Request) {

	policies := s.awsManager.ListHibernationPolicies()
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policies); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode policies: %v", err), http.StatusInternalServerError)
		return
	}
}

// getHibernationPolicy returns a specific hibernation policy template
func (s *Server) getHibernationPolicy(w http.ResponseWriter, r *http.Request, policyID string) {

	policy, err := s.awsManager.GetHibernationPolicy(policyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Policy not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policy); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode policy: %v", err), http.StatusInternalServerError)
		return
	}
}

// recommendHibernationPolicy recommends a hibernation policy for an instance
func (s *Server) recommendHibernationPolicy(w http.ResponseWriter, r *http.Request) {
	var request struct {
		InstanceName string `json:"instance_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}


	policy, err := s.awsManager.RecommendHibernationPolicy(request.InstanceName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get recommendation: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policy); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode recommendation: %v", err), http.StatusInternalServerError)
		return
	}
}

// listHibernationSchedules returns active hibernation schedules
func (s *Server) listHibernationSchedules(w http.ResponseWriter, r *http.Request) {

	// This would get schedules from the scheduler
	// For now, return a placeholder
	schedules := []hibernation.Schedule{}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(schedules); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode schedules: %v", err), http.StatusInternalServerError)
		return
	}
}

// getHibernationSavingsReport generates a hibernation cost savings report
func (s *Server) getHibernationSavingsReport(w http.ResponseWriter, r *http.Request) {
	// Get period from query parameter
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}


	// Parse period
	days := 30
	switch period {
	case "7d":
		days = 7
	case "90d":
		days = 90
	}

	// Create a placeholder report
	// In a real implementation, this would aggregate data from the savings calculator
	report := map[string]interface{}{
		"period_days":          days,
		"total_saved":          245.67,
		"hibernation_hours":    1234.5,
		"active_hours":         2345.6,
		"savings_percentage":   34.5,
		"projected_monthly":    320.00,
		"recommendations": []map[string]string{
			{
				"type":        "enable_hibernation",
				"description": "Enable hibernation on 2 more instances",
				"impact":      "$80/month additional savings",
			},
			{
				"type":        "policy_optimization",
				"description": "Consider 'aggressive-cost' policy for dev instances",
				"impact":      "$45/month additional savings",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode report: %v", err), http.StatusInternalServerError)
		return
	}
}

// Instance-specific hibernation policy handlers
// These would be added to the existing instance operations handler

// handleInstanceHibernationPolicies handles /api/v1/instances/{instanceName}/hibernation/policies
func (s *Server) handleInstanceHibernationPolicies(w http.ResponseWriter, r *http.Request, instanceName string) {
	switch r.Method {
	case "GET":
		s.getInstanceHibernationPolicies(w, r, instanceName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleInstanceHibernationPolicy handles /api/v1/instances/{instanceName}/hibernation/policies/{policyId}
func (s *Server) handleInstanceHibernationPolicy(w http.ResponseWriter, r *http.Request, instanceName string, policyID string) {
	switch r.Method {
	case "PUT":
		s.applyHibernationPolicyToInstance(w, r, instanceName, policyID)
	case "DELETE":
		s.removeHibernationPolicyFromInstance(w, r, instanceName, policyID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getInstanceHibernationPolicies returns policies applied to an instance
func (s *Server) getInstanceHibernationPolicies(w http.ResponseWriter, r *http.Request, instanceName string) {

	policies, err := s.awsManager.GetInstancePolicies(instanceName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get instance policies: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policies); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode policies: %v", err), http.StatusInternalServerError)
		return
	}
}

// applyHibernationPolicyToInstance applies a hibernation policy to an instance
func (s *Server) applyHibernationPolicyToInstance(w http.ResponseWriter, r *http.Request, instanceName string, policyID string) {

	if err := s.awsManager.ApplyHibernationPolicy(instanceName, policyID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply policy: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"message": fmt.Sprintf("Successfully applied hibernation policy %s to instance %s", policyID, instanceName),
		"status":  "success",
	}
	json.NewEncoder(w).Encode(response)
}

// removeHibernationPolicyFromInstance removes a hibernation policy from an instance
func (s *Server) removeHibernationPolicyFromInstance(w http.ResponseWriter, r *http.Request, instanceName string, policyID string) {

	if err := s.awsManager.RemoveHibernationPolicy(instanceName, policyID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove policy: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"message": fmt.Sprintf("Successfully removed hibernation policy %s from instance %s", policyID, instanceName),
		"status":  "success",
	}
	json.NewEncoder(w).Encode(response)
}