package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Client provides an interface to communicate with the CloudWorkstation daemon
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CloudWorkstationAPI defines the interface for all CloudWorkstation operations
type CloudWorkstationAPI interface {
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

	// Daemon operations
	GetStatus() (*types.DaemonStatus, error)
	Ping() error
}

// Ensure Client implements CloudWorkstationAPI
var _ CloudWorkstationAPI = (*Client)(nil)

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

// HTTP helper methods

func (c *Client) get(path string, result interface{}) error {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", reqBody)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *Client) delete(path string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, nil)
}

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