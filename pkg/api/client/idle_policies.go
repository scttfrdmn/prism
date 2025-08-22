// Package client provides the CloudWorkstation API client implementation
package client

import (
	"bytes"
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/idle/policies/apply", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/idle/policies/remove", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get idle savings report: %s", resp.Status)
	}

	var report map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode report: %w", err)
	}

	return report, nil
}

// Deprecated: Hibernation policy operations - these now call the idle policy functions

// ListHibernationPolicies returns all available hibernation policy templates (deprecated: use ListIdlePolicies)
func (c *HTTPClient) ListHibernationPolicies(ctx context.Context) ([]*idle.PolicyTemplate, error) {
	return c.ListIdlePolicies(ctx)
}

// GetHibernationPolicy returns a specific hibernation policy template (deprecated: use GetIdlePolicy)
func (c *HTTPClient) GetHibernationPolicy(ctx context.Context, policyID string) (*idle.PolicyTemplate, error) {
	return c.GetIdlePolicy(ctx, policyID)
}

// ApplyHibernationPolicy applies a hibernation policy to an instance (deprecated: use ApplyIdlePolicy)
func (c *HTTPClient) ApplyHibernationPolicy(ctx context.Context, instanceName string, policyID string) error {
	return c.ApplyIdlePolicy(ctx, instanceName, policyID)
}

// RemoveHibernationPolicy removes a hibernation policy from an instance (deprecated: use RemoveIdlePolicy)
func (c *HTTPClient) RemoveHibernationPolicy(ctx context.Context, instanceName string, policyID string) error {
	return c.RemoveIdlePolicy(ctx, instanceName, policyID)
}

// GetInstanceHibernationPolicies returns all hibernation policies applied to an instance (deprecated: use GetInstanceIdlePolicies)
func (c *HTTPClient) GetInstanceHibernationPolicies(ctx context.Context, instanceName string) ([]*idle.PolicyTemplate, error) {
	return c.GetInstanceIdlePolicies(ctx, instanceName)
}

// RecommendHibernationPolicy recommends a hibernation policy for an instance (deprecated: use RecommendIdlePolicy)
func (c *HTTPClient) RecommendHibernationPolicy(ctx context.Context, instanceName string) (*idle.PolicyTemplate, error) {
	return c.RecommendIdlePolicy(ctx, instanceName)
}

// GetHibernationSavingsReport returns the hibernation cost savings report (deprecated: use GetIdleSavingsReport)
func (c *HTTPClient) GetHibernationSavingsReport(ctx context.Context, period string) (map[string]interface{}, error) {
	return c.GetIdleSavingsReport(ctx, period)
}