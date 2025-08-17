package api

import (
	"fmt"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Authentication-related API client methods

// SetAPIKey sets the API key for authentication
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// GetAPIKey returns the current API key
func (c *Client) GetAPIKey() string {
	return c.apiKey
}

// GenerateAPIKey requests a new API key from the server
func (c *Client) GenerateAPIKey() (*types.AuthResponse, error) {
	var resp types.AuthResponse
	err := c.post("/api/v1/auth", nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Store the API key for future requests
	c.SetAPIKey(resp.APIKey)
	return &resp, nil
}

// GetAuthStatus returns the current authentication status
func (c *Client) GetAuthStatus() (*AuthStatusResponse, error) {
	var resp map[string]interface{}
	err := c.get("/api/v1/auth", &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication status: %w", err)
	}

	status := &AuthStatusResponse{}

	// Extract auth_enabled
	if enabled, ok := resp["auth_enabled"].(bool); ok {
		status.AuthEnabled = enabled
	}

	// Extract authenticated
	if authenticated, ok := resp["authenticated"].(bool); ok {
		status.Authenticated = authenticated
	}

	// Extract created_at
	if timeStr, ok := resp["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			status.CreatedAt = t
		}
	}

	return status, nil
}

// RevokeAPIKey revokes the current API key
func (c *Client) RevokeAPIKey() error {
	err := c.delete("/api/v1/auth")
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	// Clear the stored API key
	c.apiKey = ""
	return nil
}

// AuthStatusResponse represents the authentication status response
type AuthStatusResponse struct {
	AuthEnabled   bool      `json:"auth_enabled"`
	Authenticated bool      `json:"authenticated"`
	CreatedAt     time.Time `json:"created_at"`
}
