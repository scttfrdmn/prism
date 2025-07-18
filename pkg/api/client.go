package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/usermgmt"
)

// Client provides an interface to communicate with the CloudWorkstation daemon
type Client struct {
	baseURL    string
	httpClient *http.Client
	
	// Configuration
	awsProfile      string
	awsRegion       string
	invitationToken string
	ownerAccount    string
	s3ConfigPath    string
	profileID       string
	apiKey          string  // API key for authentication
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// Create a client with default performance options
	httpClient := createHTTPClient(DefaultClientPerformanceOptions())

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// LegacyCloudWorkstationAPI defines the interface for CloudWorkstation operations without context
type LegacyCloudWorkstationAPI interface {
	// Instance operations
	LaunchInstance(req types.LaunchRequest) (*types.LaunchResponse, error)
	ListInstances() (*types.ListResponse, error)
	GetInstance(name string) (*types.Instance, error)
	DeleteInstance(name string) error
	StopInstance(name string) error
	StartInstance(name string) error
	ConnectInstance(name string) (string, error)

	// Template operations
	ListTemplates() (map[string]types.Template, error)
	GetTemplate(name string) (*types.Template, error)

	// Volume operations
	CreateVolume(req types.VolumeCreateRequest) (*types.EFSVolume, error)
	ListVolumes() (map[string]types.EFSVolume, error)
	GetVolume(name string) (*types.EFSVolume, error)
	DeleteVolume(name string) error

	// Storage (EBS) operations
	CreateStorage(req types.StorageCreateRequest) (*types.EBSVolume, error)
	ListStorage() (map[string]types.EBSVolume, error)
	GetStorage(name string) (*types.EBSVolume, error)
	DeleteStorage(name string) error
	AttachStorage(volumeName, instanceName string) error
	DetachStorage(volumeName string) error

	// User management operations
	ListUsers(filter *usermgmt.UserFilter, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedUsers, error)
	GetUser(id string) (*usermgmt.User, error)
	GetUserByUsername(username string) (*usermgmt.User, error)
	CreateUser(user *usermgmt.User) (*usermgmt.User, error)
	UpdateUser(user *usermgmt.User) (*usermgmt.User, error)
	DeleteUser(id string) error
	EnableUser(id string) error
	DisableUser(id string) error
	GetUserGroups(id string) ([]*usermgmt.Group, error)
	UpdateUserGroups(id string, groupNames []string) error
	ListGroups(filter *usermgmt.GroupFilter, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedGroups, error)
	GetGroup(id string) (*usermgmt.Group, error)
	GetGroupByName(name string) (*usermgmt.Group, error)
	CreateGroup(group *usermgmt.Group) (*usermgmt.Group, error)
	UpdateGroup(group *usermgmt.Group) (*usermgmt.Group, error)
	DeleteGroup(id string) error
	GetGroupUsers(id string, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedUsers, error)
	UpdateGroupUsers(id string, userIDs []string) error
	Authenticate(username, password string) (*AuthenticationResult, error)

	// Daemon operations
	GetStatus() (*types.DaemonStatus, error)
	Ping() error
}

// Ensure Client implements LegacyCloudWorkstationAPI
var _ LegacyCloudWorkstationAPI = (*Client)(nil)

// LaunchInstance launches a new instance
func (c *Client) LaunchInstance(req types.LaunchRequest) (*types.LaunchResponse, error) {
	var resp types.LaunchResponse
	err := c.post("/api/v1/instances", req, &resp)
	return &resp, err
}

// ListInstances returns all instances
func (c *Client) ListInstances() (*types.ListResponse, error) {
	var resp types.ListResponse
	err := c.get("/api/v1/instances", &resp)
	return &resp, err
}

// GetInstance returns a specific instance
func (c *Client) GetInstance(name string) (*types.Instance, error) {
	var instance types.Instance
	err := c.get(fmt.Sprintf("/api/v1/instances/%s", name), &instance)
	return &instance, err
}

// DeleteInstance terminates an instance
func (c *Client) DeleteInstance(name string) error {
	return c.delete(fmt.Sprintf("/api/v1/instances/%s", name))
}

// StopInstance stops an instance
func (c *Client) StopInstance(name string) error {
	return c.post(fmt.Sprintf("/api/v1/instances/%s/stop", name), nil, nil)
}

// StartInstance starts a stopped instance
func (c *Client) StartInstance(name string) error {
	return c.post(fmt.Sprintf("/api/v1/instances/%s/start", name), nil, nil)
}

// ConnectInstance returns connection information for an instance
func (c *Client) ConnectInstance(name string) (string, error) {
	var response struct {
		ConnectionInfo string `json:"connection_info"`
	}
	err := c.get(fmt.Sprintf("/api/v1/instances/%s/connect", name), &response)
	return response.ConnectionInfo, err
}

// ListTemplates returns all available templates
func (c *Client) ListTemplates() (map[string]types.Template, error) {
	var templates map[string]types.Template
	err := c.get("/api/v1/templates", &templates)
	return templates, err
}

// GetTemplate returns a specific template
func (c *Client) GetTemplate(name string) (*types.Template, error) {
	var template types.Template
	err := c.get(fmt.Sprintf("/api/v1/templates/%s", name), &template)
	return &template, err
}

// CreateVolume creates a new EFS volume
func (c *Client) CreateVolume(req types.VolumeCreateRequest) (*types.EFSVolume, error) {
	var volume types.EFSVolume
	err := c.post("/api/v1/volumes", req, &volume)
	return &volume, err
}

// ListVolumes returns all EFS volumes
func (c *Client) ListVolumes() (map[string]types.EFSVolume, error) {
	var volumes map[string]types.EFSVolume
	err := c.get("/api/v1/volumes", &volumes)
	return volumes, err
}

// GetVolume returns a specific EFS volume
func (c *Client) GetVolume(name string) (*types.EFSVolume, error) {
	var volume types.EFSVolume
	err := c.get(fmt.Sprintf("/api/v1/volumes/%s", name), &volume)
	return &volume, err
}

// DeleteVolume deletes an EFS volume
func (c *Client) DeleteVolume(name string) error {
	return c.delete(fmt.Sprintf("/api/v1/volumes/%s", name))
}

// CreateStorage creates a new EBS volume
func (c *Client) CreateStorage(req types.StorageCreateRequest) (*types.EBSVolume, error) {
	var volume types.EBSVolume
	err := c.post("/api/v1/storage", req, &volume)
	return &volume, err
}

// ListStorage returns all EBS volumes
func (c *Client) ListStorage() (map[string]types.EBSVolume, error) {
	var volumes map[string]types.EBSVolume
	err := c.get("/api/v1/storage", &volumes)
	return volumes, err
}

// GetStorage returns a specific EBS volume
func (c *Client) GetStorage(name string) (*types.EBSVolume, error) {
	var volume types.EBSVolume
	err := c.get(fmt.Sprintf("/api/v1/storage/%s", name), &volume)
	return &volume, err
}

// DeleteStorage deletes an EBS volume
func (c *Client) DeleteStorage(name string) error {
	return c.delete(fmt.Sprintf("/api/v1/storage/%s", name))
}

// AttachStorage attaches an EBS volume to an instance
func (c *Client) AttachStorage(volumeName, instanceName string) error {
	req := map[string]string{"instance": instanceName}
	return c.post(fmt.Sprintf("/api/v1/storage/%s/attach", volumeName), req, nil)
}

// DetachStorage detaches an EBS volume from its instance
func (c *Client) DetachStorage(volumeName string) error {
	return c.post(fmt.Sprintf("/api/v1/storage/%s/detach", volumeName), nil, nil)
}

// GetStatus returns daemon status
func (c *Client) GetStatus() (*types.DaemonStatus, error) {
	var status types.DaemonStatus
	err := c.get("/api/v1/status", &status)
	return &status, err
}

// Ping checks if the daemon is responsive
func (c *Client) Ping() error {
	return c.get("/api/v1/ping", nil)
}

// Configuration methods

// SetAWSProfile sets the AWS profile to use for requests
func (c *Client) SetAWSProfile(profile string) {
	c.awsProfile = profile
}

// SetAWSRegion sets the AWS region to use for requests
func (c *Client) SetAWSRegion(region string) {
	c.awsRegion = region
}

// SetInvitationToken sets the invitation token and related information
func (c *Client) SetInvitationToken(token, ownerAccount, s3ConfigPath string) {
	c.invitationToken = token
	c.ownerAccount = ownerAccount
	c.s3ConfigPath = s3ConfigPath
}

// SetProfileID sets the profile ID for the client
func (c *Client) SetProfileID(profileID string) {
	c.profileID = profileID
}

// SetAPIKey sets the API key for authentication
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// GenerateAPIKey requests a new API key from the server
func (c *Client) GenerateAPIKey() (*types.AuthResponse, error) {
	var resp types.AuthResponse
	err := c.post("/api/v1/auth", nil, &resp)
	if err != nil {
		return nil, err
	}
	
	// Store the API key for future requests
	c.SetAPIKey(resp.APIKey)
	return &resp, nil
}

// GetAuthStatus returns the current authentication status
func (c *Client) GetAuthStatus() (map[string]interface{}, error) {
	var resp map[string]interface{}
	err := c.get("/api/v1/auth", &resp)
	return resp, err
}

// RevokeAPIKey revokes the current API key
func (c *Client) RevokeAPIKey() error {
	err := c.delete("/api/v1/auth")
	if err == nil {
		// Clear the stored API key
		c.apiKey = ""
	}
	return err
}

// HTTP helper methods

// addRequestHeaders adds common headers and auth headers to requests
func (c *Client) addRequestHeaders(req *http.Request) {
	// Add AWS authentication headers
	if c.awsProfile != "" {
		req.Header.Set("X-AWS-Profile", c.awsProfile)
	}
	
	if c.awsRegion != "" {
		req.Header.Set("X-AWS-Region", c.awsRegion)
	}
	
	// Add invitation headers if configured
	if c.invitationToken != "" {
		req.Header.Set("X-Invitation-Token", c.invitationToken)
		req.Header.Set("X-Owner-Account", c.ownerAccount)
		
		if c.s3ConfigPath != "" {
			req.Header.Set("X-S3-Config-Path", c.s3ConfigPath)
		}
	}
	
	// Add profile ID header if configured
	if c.profileID != "" {
		req.Header.Set("X-Profile-ID", c.profileID)
	}
	
	// Add API key if configured
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}

// get sends a GET request to the specified path and decodes response into result
func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers
	c.addRequestHeaders(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

// post sends a POST request to the specified path with optional body and decodes response into result
func (c *Client) post(path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest("POST", c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set content type
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// Add headers
	c.addRequestHeaders(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

// put sends a PUT request to the specified path with optional body and decodes response into result
func (c *Client) put(path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add headers
	c.addRequestHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

// delete sends a DELETE request to the specified path
func (c *Client) delete(path string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers
	c.addRequestHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, nil)
}

// handleResponse processes the HTTP response and decodes JSON into the result if provided
func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	if resp.StatusCode >= 400 {
		var apiErr types.APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		return apiErr
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}