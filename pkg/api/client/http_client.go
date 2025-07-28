package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// HTTPClient provides an HTTP-based implementation of CloudWorkstationAPI
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	
	// Configuration
	awsProfile      string
	awsRegion       string
	invitationToken string
	ownerAccount    string
	s3ConfigPath    string
	apiKey          string  // API key for authentication
	lastOperation   string  // Last operation performed for error context
}

// NewClient creates a new HTTP API client
func NewClient(baseURL string) CloudWorkstationAPI {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// Create a client with default performance options
	httpClient := createHTTPClient(DefaultPerformanceOptions())

	return &HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// NewClientWithOptions creates a new HTTP API client with specific options
func NewClientWithOptions(baseURL string, opts Options) CloudWorkstationAPI {
	client := NewClient(baseURL).(*HTTPClient)
	client.SetOptions(opts)
	return client
}

// SetOptions configures the client with the provided options
func (c *HTTPClient) SetOptions(opts Options) {
	c.awsProfile = opts.AWSProfile
	c.awsRegion = opts.AWSRegion
	c.invitationToken = opts.InvitationToken
	c.ownerAccount = opts.OwnerAccount
	c.s3ConfigPath = opts.S3ConfigPath
}

// makeRequest makes an HTTP request to the daemon with proper headers
func (c *HTTPClient) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
	if c.awsProfile != "" {
		req.Header.Set("X-AWS-Profile", c.awsProfile)
	}
	if c.awsRegion != "" {
		req.Header.Set("X-AWS-Region", c.awsRegion)
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	c.lastOperation = fmt.Sprintf("%s %s", method, path)
	return c.httpClient.Do(req)
}

// handleResponse processes the HTTP response and unmarshals JSON if successful
func (c *HTTPClient) handleResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d for %s: %s", resp.StatusCode, c.lastOperation, string(body))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response for %s: %w", c.lastOperation, err)
		}
	}

	return nil
}

// Ping checks if the daemon is running
func (c *HTTPClient) Ping(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/ping", nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// Shutdown gracefully shuts down the daemon
func (c *HTTPClient) Shutdown(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/shutdown", nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// GetStatus gets the daemon status
func (c *HTTPClient) GetStatus(ctx context.Context) (*types.DaemonStatus, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/status", nil)
	if err != nil {
		return nil, err
	}

	var status types.DaemonStatus
	if err := c.handleResponse(resp, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// LaunchInstance launches a new instance
func (c *HTTPClient) LaunchInstance(ctx context.Context, req types.LaunchRequest) (*types.LaunchResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/instances", req)
	if err != nil {
		return nil, err
	}

	var result types.LaunchResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListInstances lists all instances
func (c *HTTPClient) ListInstances(ctx context.Context) (*types.ListResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/instances", nil)
	if err != nil {
		return nil, err
	}

	var result types.ListResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetInstance gets details of a specific instance
func (c *HTTPClient) GetInstance(ctx context.Context, name string) (*types.Instance, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/instances/%s", name), nil)
	if err != nil {
		return nil, err
	}

	var result types.Instance
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// StartInstance starts a stopped instance
func (c *HTTPClient) StartInstance(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/start", name), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// StopInstance stops a running instance
func (c *HTTPClient) StopInstance(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/stop", name), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// HibernateInstance hibernates a running instance
func (c *HTTPClient) HibernateInstance(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/hibernate", name), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// ResumeInstance resumes a hibernated instance
func (c *HTTPClient) ResumeInstance(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/resume", name), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// GetInstanceHibernationStatus gets hibernation status for an instance
func (c *HTTPClient) GetInstanceHibernationStatus(ctx context.Context, name string) (*types.HibernationStatus, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/instances/%s/hibernation-status", name), nil)
	if err != nil {
		return nil, err
	}

	var status types.HibernationStatus
	if err := c.handleResponse(resp, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// DeleteInstance deletes an instance
func (c *HTTPClient) DeleteInstance(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/instances/%s", name), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// ConnectInstance gets connection information for an instance
func (c *HTTPClient) ConnectInstance(ctx context.Context, name string) (string, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/instances/%s/connect", name), nil)
	if err != nil {
		return "", err
	}

	var result map[string]string
	if err := c.handleResponse(resp, &result); err != nil {
		return "", err
	}

	return result["connection_info"], nil
}

// ListTemplates lists all available templates
func (c *HTTPClient) ListTemplates(ctx context.Context) (map[string]types.Template, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/templates", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]types.Template
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetTemplate gets details of a specific template
func (c *HTTPClient) GetTemplate(ctx context.Context, name string) (*types.Template, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/templates/%s", name), nil)
	if err != nil {
		return nil, err
	}

	var result types.Template
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Volume operations

func (c *HTTPClient) CreateVolume(ctx context.Context, req types.VolumeCreateRequest) (*types.EFSVolume, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/volumes", req)
	if err != nil {
		return nil, err
	}

	var result types.EFSVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) ListVolumes(ctx context.Context) ([]types.EFSVolume, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/volumes", nil)
	if err != nil {
		return nil, err
	}

	var result []types.EFSVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *HTTPClient) GetVolume(ctx context.Context, name string) (*types.EFSVolume, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/volumes/%s", name), nil)
	if err != nil {
		return nil, err
	}

	var result types.EFSVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) DeleteVolume(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/volumes/%s", name), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) AttachVolume(ctx context.Context, volumeName, instanceName string) error {
	req := map[string]string{"instance": instanceName}
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/volumes/%s/attach", volumeName), req)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) DetachVolume(ctx context.Context, volumeName string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/volumes/%s/detach", volumeName), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// Storage operations

func (c *HTTPClient) CreateStorage(ctx context.Context, req types.StorageCreateRequest) (*types.EBSVolume, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/storage", req)
	if err != nil {
		return nil, err
	}

	var result types.EBSVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) ListStorage(ctx context.Context) ([]types.EBSVolume, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/storage", nil)
	if err != nil {
		return nil, err
	}

	var result []types.EBSVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *HTTPClient) GetStorage(ctx context.Context, name string) (*types.EBSVolume, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/storage/%s", name), nil)
	if err != nil {
		return nil, err
	}

	var result types.EBSVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) DeleteStorage(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/storage/%s", name), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) AttachStorage(ctx context.Context, storageName, instanceName string) error {
	req := map[string]string{"instance": instanceName}
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/storage/%s/attach", storageName), req)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) DetachStorage(ctx context.Context, storageName string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/storage/%s/detach", storageName), nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// Registry operations - these will need proper implementation based on actual API endpoints

func (c *HTTPClient) GetRegistryStatus(ctx context.Context) (*RegistryStatusResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/registry/status", nil)
	if err != nil {
		return nil, err
	}

	var result RegistryStatusResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) SetRegistryStatus(ctx context.Context, active bool) error {
	req := map[string]bool{"active": active}
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/registry/status", req)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) LookupAMI(ctx context.Context, templateName, region, architecture string) (*AMIReferenceResponse, error) {
	path := fmt.Sprintf("/api/v1/registry/ami?template=%s&region=%s&architecture=%s", 
		templateName, region, architecture)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result AMIReferenceResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) ListTemplateAMIs(ctx context.Context, templateName string) ([]AMIReferenceResponse, error) {
	path := fmt.Sprintf("/api/v1/registry/template/%s/amis", templateName)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []AMIReferenceResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Template application operations

func (c *HTTPClient) ApplyTemplate(ctx context.Context, req templates.ApplyRequest) (*templates.ApplyResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/templates/apply", req)
	if err != nil {
		return nil, err
	}

	var result templates.ApplyResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) DiffTemplate(ctx context.Context, req templates.DiffRequest) (*templates.TemplateDiff, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/templates/diff", req)
	if err != nil {
		return nil, err
	}

	var result templates.TemplateDiff
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) GetInstanceLayers(ctx context.Context, instanceName string) ([]templates.AppliedTemplate, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/layers", instanceName)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []templates.AppliedTemplate
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *HTTPClient) RollbackInstance(ctx context.Context, req types.RollbackRequest) error {
	path := fmt.Sprintf("/api/v1/instances/%s/rollback", req.InstanceName)
	resp, err := c.makeRequest(ctx, "POST", path, req)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, nil)
}