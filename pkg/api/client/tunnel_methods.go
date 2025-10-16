package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// TunnelInfo represents tunnel information from the API
type TunnelInfo struct {
	InstanceName string `json:"instance_name"`
	ServiceName  string `json:"service_name"`
	ServiceDesc  string `json:"service_description"`
	RemotePort   int    `json:"remote_port"`
	LocalPort    int    `json:"local_port"`
	LocalURL     string `json:"local_url"`
	AuthToken    string `json:"auth_token,omitempty"` // Authentication token (e.g., Jupyter)
	Status       string `json:"status"`
	StartTime    string `json:"start_time,omitempty"`
}

// CreateTunnelsRequest is the request to create tunnels
type CreateTunnelsRequest struct {
	InstanceName string   `json:"instance_name"`
	Services     []string `json:"services,omitempty"` // If empty, create all
}

// CreateTunnelsResponse is the response from creating tunnels
type CreateTunnelsResponse struct {
	Tunnels []TunnelInfo `json:"tunnels"`
	Message string       `json:"message"`
}

// ListTunnelsResponse is the response from listing tunnels
type ListTunnelsResponse struct {
	Tunnels []TunnelInfo `json:"tunnels"`
	Count   int          `json:"count"`
}

// CreateTunnels creates SSH tunnels for an instance's web services
func (c *HTTPClient) CreateTunnels(ctx context.Context, instanceName string, services []string) (*CreateTunnelsResponse, error) {
	req := CreateTunnelsRequest{
		InstanceName: instanceName,
		Services:     services,
	}

	// makeRequest will marshal this for us - don't marshal twice!
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/tunnels", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CreateTunnelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// ListTunnels lists all active tunnels, optionally filtered by instance
func (c *HTTPClient) ListTunnels(ctx context.Context, instanceName string) (*ListTunnelsResponse, error) {
	url := "/api/v1/tunnels"
	if instanceName != "" {
		url += "?instance=" + instanceName
	}

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response ListTunnelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// CloseTunnel closes a specific tunnel or all tunnels for an instance
func (c *HTTPClient) CloseTunnel(ctx context.Context, instanceName string, serviceName string) error {
	url := "/api/v1/tunnels?instance=" + instanceName
	if serviceName != "" {
		url += "&service=" + serviceName
	}

	resp, err := c.makeRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// CloseInstanceTunnels closes all tunnels for an instance (convenience method)
func (c *HTTPClient) CloseInstanceTunnels(ctx context.Context, instanceName string) error {
	return c.CloseTunnel(ctx, instanceName, "")
}
