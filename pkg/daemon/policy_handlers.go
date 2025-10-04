package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RegisterPolicyRoutes registers all policy-related API endpoints
func (s *Server) RegisterPolicyRoutes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Policy management endpoints (Phase 5A.5)
	mux.HandleFunc("/api/v1/policies/status", applyMiddleware(s.handlePolicyStatus))
	mux.HandleFunc("/api/v1/policies/sets", applyMiddleware(s.handlePolicySets))
	mux.HandleFunc("/api/v1/policies/assign", applyMiddleware(s.handlePolicyAssign))
	mux.HandleFunc("/api/v1/policies/enforcement", applyMiddleware(s.handlePolicyEnforcement))
	mux.HandleFunc("/api/v1/policies/check", applyMiddleware(s.handlePolicyCheck))
}

// PolicyStatusResponse represents the policy enforcement status
type PolicyStatusResponse struct {
	Enabled          bool     `json:"enabled"`
	Status           string   `json:"status"`
	StatusIcon       string   `json:"status_icon"`
	AssignedPolicies []string `json:"assigned_policies"`
	Message          string   `json:"message,omitempty"`
}

// PolicySetsResponse represents available policy sets
type PolicySetsResponse struct {
	PolicySets map[string]PolicySetInfo `json:"policy_sets"`
}

// PolicySetInfo provides information about a policy set
type PolicySetInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Policies    int               `json:"policies"`
	Status      string            `json:"status"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// PolicyAssignRequest represents a policy assignment request
type PolicyAssignRequest struct {
	PolicySet string `json:"policy_set"`
	UserID    string `json:"user_id,omitempty"`
}

// PolicyAssignResponse represents the response to a policy assignment
type PolicyAssignResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	AssignedPolicySet string `json:"assigned_policy_set"`
	EnforcementStatus string `json:"enforcement_status"`
}

// PolicyEnforcementRequest represents an enforcement state change request
type PolicyEnforcementRequest struct {
	Enabled bool `json:"enabled"`
}

// PolicyEnforcementResponse represents the response to enforcement changes
type PolicyEnforcementResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

// PolicyCheckRequest represents a template access check request
type PolicyCheckRequest struct {
	TemplateName string `json:"template_name"`
	UserID       string `json:"user_id,omitempty"`
}

// PolicyCheckResponse represents the result of a policy check
type PolicyCheckResponse struct {
	Allowed         bool     `json:"allowed"`
	TemplateName    string   `json:"template_name"`
	Reason          string   `json:"reason"`
	MatchedPolicies []string `json:"matched_policies,omitempty"`
	Suggestions     []string `json:"suggestions,omitempty"`
}

// handlePolicyStatus returns the current policy enforcement status
func (s *Server) handlePolicyStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if policy service is available
	if s.policyService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Policy service not available")
		return
	}

	// Get policy status
	enabled := s.policyService.IsEnabled()
	currentPolicies := s.policyService.GetCurrentUserPolicies()

	status := "inactive"
	statusIcon := "🔓 Inactive"
	message := ""

	if enabled {
		status = "active"
		statusIcon = "🔒 Active"
	}

	if len(currentPolicies) == 0 {
		message = "No policies configured - using default allow"
	}

	response := PolicyStatusResponse{
		Enabled:          enabled,
		Status:           status,
		StatusIcon:       statusIcon,
		AssignedPolicies: currentPolicies,
		Message:          message,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handlePolicySets returns available policy sets
func (s *Server) handlePolicySets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if policy service is available
	if s.policyService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Policy service not available")
		return
	}

	// Get available policy sets
	policySets := s.policyService.ListAvailablePolicySets()

	// Convert to response format
	policySetInfos := make(map[string]PolicySetInfo)
	for id, policySet := range policySets {
		status := "Enabled"
		if !policySet.Enabled {
			status = "Disabled"
		}

		policySetInfos[id] = PolicySetInfo{
			ID:          id,
			Name:        policySet.Name,
			Description: policySet.Description,
			Policies:    len(policySet.Policies),
			Status:      status,
			Tags:        policySet.Tags,
		}
	}

	response := PolicySetsResponse{
		PolicySets: policySetInfos,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handlePolicyAssign assigns a policy set to a user
func (s *Server) handlePolicyAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if policy service is available
	if s.policyService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Policy service not available")
		return
	}

	// Parse request
	var req PolicyAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.PolicySet == "" {
		s.writeError(w, http.StatusBadRequest, "Policy set name is required")
		return
	}

	// Validate policy set exists
	policySets := s.policyService.ListAvailablePolicySets()
	if _, exists := policySets[req.PolicySet]; !exists {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Policy set '%s' not found", req.PolicySet))
		return
	}

	// Assign policy set
	var err error
	switch req.PolicySet {
	case "student":
		err = s.policyService.AssignStudentPolicies()
	case "researcher":
		err = s.policyService.AssignResearcherPolicies()
	default:
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Assignment not implemented for policy set: %s", req.PolicySet))
		return
	}

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to assign policy set: %v", err))
		return
	}

	// Get enforcement status for response
	enforcementStatus := "Disabled"
	if s.policyService.IsEnabled() {
		enforcementStatus = "Enabled"
	}

	response := PolicyAssignResponse{
		Success:           true,
		Message:           fmt.Sprintf("Successfully assigned '%s' policy set", req.PolicySet),
		AssignedPolicySet: req.PolicySet,
		EnforcementStatus: enforcementStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handlePolicyEnforcement controls policy enforcement state
func (s *Server) handlePolicyEnforcement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if policy service is available
	if s.policyService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Policy service not available")
		return
	}

	// Parse request
	var req PolicyEnforcementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Set enforcement state
	s.policyService.SetEnabled(req.Enabled)

	status := "Disabled"
	message := "⚠️  Policy enforcement disabled"
	if req.Enabled {
		status = "Enabled"
		message = "✅ Policy enforcement enabled"
	}

	response := PolicyEnforcementResponse{
		Success: true,
		Message: message,
		Enabled: req.Enabled,
		Status:  status,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handlePolicyCheck checks template access permissions
func (s *Server) handlePolicyCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if policy service is available
	if s.policyService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Policy service not available")
		return
	}

	// Parse request
	var req PolicyCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.TemplateName == "" {
		s.writeError(w, http.StatusBadRequest, "Template name is required")
		return
	}

	// Handle template names with multiple words (join with spaces)
	templateName := strings.TrimSpace(req.TemplateName)

	// Check template access
	policyResponse := s.policyService.CheckTemplateAccess(templateName)

	response := PolicyCheckResponse{
		Allowed:         policyResponse.Allowed,
		TemplateName:    templateName,
		Reason:          policyResponse.Reason,
		MatchedPolicies: policyResponse.MatchedPolicies,
		Suggestions:     policyResponse.Suggestions,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
