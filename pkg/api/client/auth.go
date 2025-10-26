package client

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// AuthStatusResponse represents the authentication status response
type AuthStatusResponse struct {
	AuthEnabled   bool      `json:"auth_enabled"`
	Authenticated bool      `json:"authenticated"`
	CreatedAt     time.Time `json:"created_at"`
}

// Authentication methods for HTTPClient

// SetAPIKey sets the API key for authentication
func (c *HTTPClient) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// GetAPIKey returns the current API key
func (c *HTTPClient) GetAPIKey() string {
	return c.apiKey
}

// GenerateAPIKey requests a new API key from the server
func (c *HTTPClient) GenerateAPIKey(ctx context.Context) (*types.AuthResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/auth", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}
	defer resp.Body.Close() // Explicit close for static analysis

	var result types.AuthResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	// Store the API key for future requests
	c.SetAPIKey(result.APIKey)
	return &result, nil
}

// GetAuthStatus returns the current authentication status
func (c *HTTPClient) GetAuthStatus(ctx context.Context) (*AuthStatusResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/auth", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication status: %w", err)
	}
	defer resp.Body.Close() // Explicit close for static analysis

	var rawResp map[string]interface{}
	if err := c.handleResponse(resp, &rawResp); err != nil {
		return nil, err
	}

	status := &AuthStatusResponse{}

	// Extract auth_enabled
	if enabled, ok := rawResp["auth_enabled"].(bool); ok {
		status.AuthEnabled = enabled
	}

	// Extract authenticated
	if authenticated, ok := rawResp["authenticated"].(bool); ok {
		status.Authenticated = authenticated
	}

	// Extract created_at
	if timeStr, ok := rawResp["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			status.CreatedAt = t
		}
	}

	return status, nil
}

// RevokeAPIKey revokes the current API key
func (c *HTTPClient) RevokeAPIKey(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/api/v1/auth", nil)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}
	defer resp.Body.Close() // Explicit close for static analysis

	if err := c.handleResponse(resp, nil); err != nil {
		return err
	}

	// Clear the stored API key
	c.apiKey = ""
	return nil
}
