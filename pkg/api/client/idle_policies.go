// Package client provides the CloudWorkstation API client implementation
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
)

// ListIdlePolicies returns all available idle policy templates
func (c *HTTPClient) ListIdlePolicies(ctx context.Context) ([]*idle.PolicyTemplate, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/idle/policies", nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list idle policies: %s", resp.Status)
	}

	var policies []*idle.PolicyTemplate
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, fmt.Errorf("failed to decode policies: %w", err)
	}

	return policies, nil
}

// GetIdlePolicy returns a specific idle policy template
func (c *HTTPClient) GetIdlePolicy(ctx context.Context, policyID string) (*idle.PolicyTemplate, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/idle/policies/%s", policyID), nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get idle policy: %s", resp.Status)
	}

	var policy idle.PolicyTemplate
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode policy: %w", err)
	}

	return &policy, nil
}

// ApplyIdlePolicy applies an idle policy to an instance
func (c *HTTPClient) ApplyIdlePolicy(ctx context.Context, instanceName string, policyID string) error {
	req := map[string]string{
		"instance_name": instanceName,
		"policy_id":     policyID,
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/idle/policies/apply", req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to apply idle policy: %s", resp.Status)
	}

	return nil
}

// RemoveIdlePolicy removes an idle policy from an instance
func (c *HTTPClient) RemoveIdlePolicy(ctx context.Context, instanceName string, policyID string) error {
	req := map[string]string{
		"instance_name": instanceName,
		"policy_id":     policyID,
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/idle/policies/remove", req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to remove idle policy: %s", resp.Status)
	}

	return nil
}

// GetInstanceIdlePolicies returns all idle policies applied to an instance
func (c *HTTPClient) GetInstanceIdlePolicies(ctx context.Context, instanceName string) ([]*idle.PolicyTemplate, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/instances/%s/idle-policies", instanceName), nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get instance idle policies: %s", resp.Status)
	}

	var policies []*idle.PolicyTemplate
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, fmt.Errorf("failed to decode policies: %w", err)
	}

	return policies, nil
}

// RecommendIdlePolicy recommends an idle policy for an instance
func (c *HTTPClient) RecommendIdlePolicy(ctx context.Context, instanceName string) (*idle.PolicyTemplate, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/instances/%s/recommend-idle-policy", instanceName), nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get idle policy recommendation: %s", resp.Status)
	}

	var policy idle.PolicyTemplate
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode policy: %w", err)
	}

	return &policy, nil
}

// GetIdleSavingsReport returns the idle cost savings report
func (c *HTTPClient) GetIdleSavingsReport(ctx context.Context, period string) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/idle/savings?period=%s", period), nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get idle savings report: %s", resp.Status)
	}

	var report map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode report: %w", err)
	}

	return report, nil
}
