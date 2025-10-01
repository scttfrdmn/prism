package client

import (
	"context"
	"fmt"
)

// Policy management API methods (Phase 5A.5)

// GetPolicyStatus returns the current policy enforcement status
func (c *HTTPClient) GetPolicyStatus(ctx context.Context) (*PolicyStatusResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/policies/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy status: %w", err)
	}

	var response PolicyStatusResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// ListPolicySets returns available policy sets
func (c *HTTPClient) ListPolicySets(ctx context.Context) (*PolicySetsResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/policies/sets", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list policy sets: %w", err)
	}

	var response PolicySetsResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
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

	var response PolicyAssignResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
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

	var response PolicyEnforcementResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
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

	var response PolicyCheckResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
