// Package client provides the CloudWorkstation API client implementation
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/hibernation"
)

// ListHibernationPolicies returns all available hibernation policy templates
func (c *HTTPClient) ListHibernationPolicies(ctx context.Context) ([]*hibernation.PolicyTemplate, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/hibernation/policies", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list hibernation policies: %s", resp.Status)
	}

	var policies []*hibernation.PolicyTemplate
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, fmt.Errorf("failed to decode policies: %w", err)
	}

	return policies, nil
}

// GetHibernationPolicy returns a specific hibernation policy template
func (c *HTTPClient) GetHibernationPolicy(ctx context.Context, policyID string) (*hibernation.PolicyTemplate, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/hibernation/policies/%s", policyID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("policy not found: %s", policyID)
		}
		return nil, fmt.Errorf("failed to get hibernation policy: %s", resp.Status)
	}

	var policy hibernation.PolicyTemplate
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode policy: %w", err)
	}

	return &policy, nil
}

// ApplyHibernationPolicy applies a hibernation policy to an instance
func (c *HTTPClient) ApplyHibernationPolicy(ctx context.Context, instanceName string, policyID string) error {
	resp, err := c.makeRequest(ctx, "PUT", 
		fmt.Sprintf("/api/v1/instances/%s/hibernation/policies/%s", instanceName, policyID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to apply hibernation policy: %s", resp.Status)
	}

	return nil
}

// RemoveHibernationPolicy removes a hibernation policy from an instance
func (c *HTTPClient) RemoveHibernationPolicy(ctx context.Context, instanceName string, policyID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", 
		fmt.Sprintf("/api/v1/instances/%s/hibernation/policies/%s", instanceName, policyID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to remove hibernation policy: %s", resp.Status)
	}

	return nil
}

// GetInstanceHibernationPolicies returns policies applied to an instance
func (c *HTTPClient) GetInstanceHibernationPolicies(ctx context.Context, instanceName string) ([]*hibernation.PolicyTemplate, error) {
	resp, err := c.makeRequest(ctx, "GET", 
		fmt.Sprintf("/api/v1/instances/%s/hibernation/policies", instanceName), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get instance hibernation policies: %s", resp.Status)
	}

	var policies []*hibernation.PolicyTemplate
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, fmt.Errorf("failed to decode policies: %w", err)
	}

	return policies, nil
}

// RecommendHibernationPolicy recommends a hibernation policy for an instance
func (c *HTTPClient) RecommendHibernationPolicy(ctx context.Context, instanceName string) (*hibernation.PolicyTemplate, error) {
	request := map[string]string{
		"instance_name": instanceName,
	}
	
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/hibernation/policies/recommend", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get hibernation policy recommendation: %s", resp.Status)
	}

	var policy hibernation.PolicyTemplate
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode recommendation: %w", err)
	}

	return &policy, nil
}

// GetHibernationSavingsReport generates a hibernation cost savings report
func (c *HTTPClient) GetHibernationSavingsReport(ctx context.Context, period string) (map[string]interface{}, error) {
	url := "/api/v1/hibernation/savings"
	if period != "" {
		url = fmt.Sprintf("%s?period=%s", url, period)
	}

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get hibernation savings report: %s", resp.Status)
	}

	var report map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode report: %w", err)
	}

	return report, nil
}