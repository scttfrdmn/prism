package api

import (
	"context"
	"fmt"
	"encoding/json"
	"net/http"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// IdlePolicyUpdateRequest represents a request to update an idle policy
type IdlePolicyUpdateRequest struct {
	Name      string `json:"name"`
	Threshold int    `json:"threshold"`
	Action    string `json:"action"`
}

// IdlePolicyResponse represents the API response for idle policy operations
type IdlePolicyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ListIdlePolicies retrieves all idle policies
func (c *Client) ListIdlePolicies(ctx context.Context) ([]types.IdlePolicy, error) {
	return c.ListIdlePoliciesLegacy()
}

// ListIdlePoliciesLegacy retrieves all idle policies without context
func (c *Client) ListIdlePoliciesLegacy() ([]types.IdlePolicy, error) {
	var policies []types.IdlePolicy
	err := c.get("/api/v1/idle/policies", &policies)
	if err != nil {
		return nil, fmt.Errorf("failed to list idle policies: %w", err)
	}
	return policies, nil
}

// GetIdlePolicy retrieves a specific idle policy
func (c *Client) GetIdlePolicy(ctx context.Context, name string) (*types.IdlePolicy, error) {
	return c.GetIdlePolicyLegacy(name)
}

// GetIdlePolicyLegacy retrieves a specific idle policy without context
func (c *Client) GetIdlePolicyLegacy(name string) (*types.IdlePolicy, error) {
	var policy types.IdlePolicy
	err := c.get(fmt.Sprintf("/api/v1/idle/policies/%s", name), &policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle policy: %w", err)
	}
	return &policy, nil
}

// UpdateIdlePolicy updates an idle policy
func (c *Client) UpdateIdlePolicy(ctx context.Context, req IdlePolicyUpdateRequest) error {
	return c.UpdateIdlePolicyLegacy(req)
}

// UpdateIdlePolicyLegacy updates an idle policy without context
func (c *Client) UpdateIdlePolicyLegacy(req IdlePolicyUpdateRequest) error {
	var resp IdlePolicyResponse
	err := c.put(fmt.Sprintf("/api/v1/idle/policies/%s", req.Name), req, &resp)
	if err != nil {
		return fmt.Errorf("failed to update idle policy: %w", err)
	}
	
	if !resp.Success {
		return fmt.Errorf("API error: %s", resp.Message)
	}
	
	return nil
}

// EnableIdleDetection enables idle detection for an instance
func (c *Client) EnableIdleDetection(ctx context.Context, instance, policy string) error {
	return c.EnableIdleDetectionLegacy(instance, policy)
}

// EnableIdleDetectionLegacy enables idle detection for an instance without context
func (c *Client) EnableIdleDetectionLegacy(instance, policy string) error {
	req := struct {
		Policy string `json:"policy"`
	}{
		Policy: policy,
	}
	
	var resp IdlePolicyResponse
	err := c.post(fmt.Sprintf("/api/v1/idle/instances/%s/enable", instance), req, &resp)
	if err != nil {
		return fmt.Errorf("failed to enable idle detection: %w", err)
	}
	
	if !resp.Success {
		return fmt.Errorf("API error: %s", resp.Message)
	}
	
	return nil
}

// DisableIdleDetection disables idle detection for an instance
func (c *Client) DisableIdleDetection(ctx context.Context, instance string) error {
	return c.DisableIdleDetectionLegacy(instance)
}

// DisableIdleDetectionLegacy disables idle detection for an instance without context
func (c *Client) DisableIdleDetectionLegacy(instance string) error {
	var resp IdlePolicyResponse
	err := c.post(fmt.Sprintf("/api/v1/idle/instances/%s/disable", instance), nil, &resp)
	if err != nil {
		return fmt.Errorf("failed to disable idle detection: %w", err)
	}
	
	if !resp.Success {
		return fmt.Errorf("API error: %s", resp.Message)
	}
	
	return nil
}

// GetIdleStatus retrieves the idle status of an instance
func (c *Client) GetIdleStatus(ctx context.Context, instance string) (*types.IdleStatus, error) {
	return c.GetIdleStatusLegacy(instance)
}

// GetIdleStatusLegacy retrieves the idle status of an instance without context
func (c *Client) GetIdleStatusLegacy(instance string) (*types.IdleStatus, error) {
	var status types.IdleStatus
	err := c.get(fmt.Sprintf("/api/v1/idle/instances/%s/status", instance), &status)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle status: %w", err)
	}
	return &status, nil
}

// Implementation of the PUT HTTP method
func (c *Client) put(path string, body, resp interface{}) error {
	url := c.buildURL(path)
	
	// Marshal body to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	
	// Create request
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	
	// Execute request
	return c.doRequest(req, jsonBody, resp)
}