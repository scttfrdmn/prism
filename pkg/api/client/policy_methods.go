package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// Policy management API methods (Phase 5A.5)

// GetPolicyStatus returns the current policy enforcement status
func (c *HTTPClient) GetPolicyStatus(ctx context.Context) (*PolicyStatusResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/policies/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy status: %w", err)
	}

	// Parse the raw response first to handle field mapping
	var rawResponse struct {
		Enabled          bool   `json:"enabled"`
		CurrentPolicySet string `json:"current_policy_set"`
		LastUpdated      string `json:"last_updated"`
		Status           string `json:"status"`
		StatusIcon       string `json:"status_icon"`
		Message          string `json:"message,omitempty"`
	}

	if err := c.handleResponse(resp, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to get policy status: %w", err)
	}

	// Map to the response type expected by tests
	response := &PolicyStatusResponse{
		Enabled:          rawResponse.Enabled,
		Status:           rawResponse.Status,
		StatusIcon:       rawResponse.StatusIcon,
		Message:          rawResponse.Message,
		AssignedPolicies: []string{},
	}

	// Add current_policy_set to AssignedPolicies if present
	if rawResponse.CurrentPolicySet != "" {
		response.AssignedPolicies = []string{rawResponse.CurrentPolicySet}
	}

	return response, nil
}

// ListPolicySets returns available policy sets
func (c *HTTPClient) ListPolicySets(ctx context.Context) (*PolicySetsResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/policies/sets", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list policy sets: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to list policy sets: status %d", resp.StatusCode)
	}

	// Read and decode the response body
	decoder := json.NewDecoder(resp.Body)
	var rawData map[string]interface{}
	if err := decoder.Decode(&rawData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	response := &PolicySetsResponse{
		PolicySets: make(map[string]PolicySetInfo),
	}

	// Check if policy_sets is a map or array
	if policySets, ok := rawData["policy_sets"].(map[string]interface{}); ok {
		// Map format (expected)
		for id, setData := range policySets {
			if setMap, ok := setData.(map[string]interface{}); ok {
				info := PolicySetInfo{ID: id}
				if name, ok := setMap["name"].(string); ok {
					info.Name = name
				}
				if desc, ok := setMap["description"].(string); ok {
					info.Description = desc
				}
				if policies, ok := setMap["policies"].(float64); ok {
					info.Policies = int(policies)
				}
				if status, ok := setMap["status"].(string); ok {
					info.Status = status
				}
				response.PolicySets[id] = info
			}
		}
	} else if policySets, ok := rawData["policy_sets"].([]interface{}); ok {
		// Array format (test/legacy)
		for _, setData := range policySets {
			if setMap, ok := setData.(map[string]interface{}); ok {
				var name, id string
				if n, ok := setMap["name"].(string); ok {
					name = n
					id = n // Use name as ID if ID not present
				}
				if i, ok := setMap["id"].(string); ok {
					id = i
				}

				info := PolicySetInfo{
					ID:   id,
					Name: name,
				}
				if desc, ok := setMap["description"].(string); ok {
					info.Description = desc
				}
				if policies, ok := setMap["policies"].(float64); ok {
					info.Policies = int(policies)
				} else {
					info.Policies = 1 // Default
				}
				if status, ok := setMap["status"].(string); ok {
					info.Status = status
				} else {
					info.Status = "active" // Default
				}
				response.PolicySets[id] = info
			}
		}
	}

	return response, nil
}

// AssignPolicySet assigns a policy set to the current user
func (c *HTTPClient) AssignPolicySet(ctx context.Context, policySet string) (*PolicyAssignResponse, error) {
	if policySet == "" {
		return nil, fmt.Errorf("policy set name cannot be empty")
	}

	// Create request body
	requestBody := map[string]string{
		"policy_set": policySet,
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/policies/assign", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to assign policy set '%s': %w", policySet, err)
	}

	// Parse the raw response to handle field mapping
	var rawResponse struct {
		Success       bool   `json:"success"`
		PolicySet     string `json:"policy_set"`
		Message       string `json:"message"`
		EffectiveDate string `json:"effective_date"`
	}

	if err := c.handleResponse(resp, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to assign policy set '%s': %w", policySet, err)
	}

	// Map to expected response type
	response := &PolicyAssignResponse{
		Success:           rawResponse.Success,
		Message:           rawResponse.Message,
		AssignedPolicySet: rawResponse.PolicySet,
		EnforcementStatus: "active", // Default status
	}

	return response, nil
}

// SetPolicyEnforcement enables or disables policy enforcement
func (c *HTTPClient) SetPolicyEnforcement(ctx context.Context, enabled bool) (*PolicyEnforcementResponse, error) {
	// Create request body
	requestBody := map[string]bool{
		"enabled": enabled,
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/policies/enforcement", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to set policy enforcement to %t: %w", enabled, err)
	}

	// Parse the raw response to handle field mapping
	var rawResponse struct {
		Enabled       bool   `json:"enabled"`
		PreviousState bool   `json:"previous_state"`
		Message       string `json:"message"`
		UpdatedAt     string `json:"updated_at"`
	}

	if err := c.handleResponse(resp, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to set policy enforcement to %t: %w", enabled, err)
	}

	// Map to expected response type with Success field
	response := &PolicyEnforcementResponse{
		Success: true, // If we got here without error, it was successful
		Message: rawResponse.Message,
		Enabled: rawResponse.Enabled,
		Status:  "active",
	}

	if !rawResponse.Enabled {
		response.Status = "disabled"
	}

	return response, nil
}

// CheckTemplateAccess checks if a template is accessible under current policies
func (c *HTTPClient) CheckTemplateAccess(ctx context.Context, templateName string) (*PolicyCheckResponse, error) {
	if templateName == "" {
		return nil, fmt.Errorf("template name cannot be empty")
	}

	// Create request body
	requestBody := map[string]string{
		"template_name": templateName,
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/policies/check", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to check template access for '%s': %w", templateName, err)
	}

	// Parse the raw response to handle field mapping
	var rawResponse struct {
		Allowed      bool     `json:"allowed"`
		Template     string   `json:"template"`
		Reason       string   `json:"reason"`
		PolicySet    string   `json:"policy_set"`
		Restrictions []string `json:"restrictions,omitempty"`
	}

	if err := c.handleResponse(resp, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to check template access for '%s': %w", templateName, err)
	}

	// Map to expected response type
	response := &PolicyCheckResponse{
		Allowed:         rawResponse.Allowed,
		TemplateName:    rawResponse.Template,
		Reason:          rawResponse.Reason,
		MatchedPolicies: []string{},
		Suggestions:     []string{},
	}

	// Add policy_set to MatchedPolicies if present
	if rawResponse.PolicySet != "" {
		response.MatchedPolicies = []string{rawResponse.PolicySet}
	}

	return response, nil
}
