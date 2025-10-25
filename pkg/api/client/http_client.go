package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// HTTPClient provides an HTTP-based implementation of CloudWorkstationAPI
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client

	// Configuration (protected by mutex for thread safety)
	mu              sync.RWMutex
	awsProfile      string
	awsRegion       string
	invitationToken string
	ownerAccount    string
	s3ConfigPath    string
	apiKey          string // API key for authentication
	lastOperation   string // Last operation performed for error context
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
	c.mu.Lock()
	defer c.mu.Unlock()
	c.awsProfile = opts.AWSProfile
	c.awsRegion = opts.AWSRegion
	c.invitationToken = opts.InvitationToken
	c.ownerAccount = opts.OwnerAccount
	c.s3ConfigPath = opts.S3ConfigPath
	c.apiKey = opts.APIKey
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

	// Set headers (thread-safe read)
	req.Header.Set("Content-Type", "application/json")

	c.mu.RLock()
	if c.awsProfile != "" {
		req.Header.Set("X-AWS-Profile", c.awsProfile)
	}
	if c.awsRegion != "" {
		req.Header.Set("X-AWS-Region", c.awsRegion)
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// Update lastOperation while holding lock
	c.mu.RUnlock()
	c.mu.Lock()
	c.lastOperation = fmt.Sprintf("%s %s", method, path)
	c.mu.Unlock()
	return c.httpClient.Do(req)
}

// handleResponse processes the HTTP response and unmarshals JSON if successful
func (c *HTTPClient) handleResponse(resp *http.Response, result interface{}) error {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log but don't fail on cleanup error - response body cleanup is not critical
			_ = err // Explicitly ignore error
		}
	}()

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
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

// Shutdown gracefully shuts down the daemon
func (c *HTTPClient) Shutdown(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/shutdown", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

// GetStatus gets the daemon status
func (c *HTTPClient) GetStatus(ctx context.Context) (*types.DaemonStatus, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status types.DaemonStatus
	if err := c.handleResponse(resp, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// MakeRequest makes a generic HTTP request to any API endpoint
func (c *HTTPClient) MakeRequest(method, path string, body interface{}) ([]byte, error) {
	resp, err := c.makeRequest(context.Background(), method, path, body)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log but don't fail on cleanup error - response body cleanup is not critical
			_ = err // Explicitly ignore error
		}
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d for %s %s: %s", resp.StatusCode, method, path, string(body))
	}

	return io.ReadAll(resp.Body)
}

// LaunchInstance launches a new instance
func (c *HTTPClient) LaunchInstance(ctx context.Context, req types.LaunchRequest) (*types.LaunchResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/instances", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.LaunchResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListInstances lists all instances
func (c *HTTPClient) ListInstances(ctx context.Context) (*types.ListResponse, error) {
	return c.ListInstancesWithRefresh(ctx, false)
}

// ListInstancesWithRefresh lists all instances with optional AWS refresh
func (c *HTTPClient) ListInstancesWithRefresh(ctx context.Context, refresh bool) (*types.ListResponse, error) {
	url := "/api/v1/instances"
	if refresh {
		url += "?refresh=true"
	}

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

// StopInstance stops a running instance
func (c *HTTPClient) StopInstance(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/stop", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

// HibernateInstance hibernates a running instance
func (c *HTTPClient) HibernateInstance(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/hibernate", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

// ResumeInstance resumes a hibernated instance
func (c *HTTPClient) ResumeInstance(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/resume", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

// GetInstanceHibernationStatus gets hibernation status for an instance
func (c *HTTPClient) GetInstanceHibernationStatus(ctx context.Context, name string) (*types.HibernationStatus, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/instances/%s/hibernation-status", name), nil)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
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
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
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
	defer resp.Body.Close()

	var result map[string]string
	if err := c.handleResponse(resp, &result); err != nil {
		return "", err
	}

	return result["connection_info"], nil
}

// ExecInstance executes a command on an instance
func (c *HTTPClient) ExecInstance(ctx context.Context, instanceName string, execRequest types.ExecRequest) (*types.ExecResult, error) {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/exec", instanceName), execRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.ExecResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ResizeInstance resizes an instance to a new instance type
func (c *HTTPClient) ResizeInstance(ctx context.Context, resizeRequest types.ResizeRequest) (*types.ResizeResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/instances/%s/resize", resizeRequest.InstanceName), resizeRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.ResizeResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetInstanceLogs retrieves logs for a specific instance
func (c *HTTPClient) GetInstanceLogs(ctx context.Context, instanceName string, logRequest types.LogRequest) (*types.LogResponse, error) {
	// Build query parameters
	params := url.Values{}
	if logRequest.LogType != "" {
		params.Set("type", logRequest.LogType)
	}
	if logRequest.Tail > 0 {
		params.Set("tail", strconv.Itoa(logRequest.Tail))
	}
	if logRequest.Since != "" {
		params.Set("since", logRequest.Since)
	}
	if logRequest.Follow {
		params.Set("follow", "true")
	}

	path := fmt.Sprintf("/api/v1/logs/%s", instanceName)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.LogResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetInstanceLogTypes retrieves available log types for a specific instance
func (c *HTTPClient) GetInstanceLogTypes(ctx context.Context, instanceName string) (*types.LogTypesResponse, error) {
	path := fmt.Sprintf("/api/v1/logs/%s/types", instanceName)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.LogTypesResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetLogsSummary retrieves log availability summary for all instances
func (c *HTTPClient) GetLogsSummary(ctx context.Context) (*types.LogSummaryResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/logs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.LogSummaryResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ==========================================
// Instance Snapshot Operations
// ==========================================

// CreateInstanceSnapshot creates a snapshot from an instance
func (c *HTTPClient) CreateInstanceSnapshot(ctx context.Context, req types.InstanceSnapshotRequest) (*types.InstanceSnapshotResult, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/snapshots", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.InstanceSnapshotResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListInstanceSnapshots lists all instance snapshots
func (c *HTTPClient) ListInstanceSnapshots(ctx context.Context) (*types.InstanceSnapshotListResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/snapshots", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.InstanceSnapshotListResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetInstanceSnapshot gets information about a specific snapshot
func (c *HTTPClient) GetInstanceSnapshot(ctx context.Context, snapshotName string) (*types.InstanceSnapshotInfo, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/snapshots/%s", snapshotName), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.InstanceSnapshotInfo
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteInstanceSnapshot deletes a snapshot
func (c *HTTPClient) DeleteInstanceSnapshot(ctx context.Context, snapshotName string) (*types.InstanceSnapshotDeleteResult, error) {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/snapshots/%s", snapshotName), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.InstanceSnapshotDeleteResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RestoreInstanceFromSnapshot restores a new instance from a snapshot
func (c *HTTPClient) RestoreInstanceFromSnapshot(ctx context.Context, snapshotName string, req types.InstanceRestoreRequest) (*types.InstanceRestoreResult, error) {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/snapshots/%s/restore", snapshotName), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.InstanceRestoreResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListTemplates lists all available templates
func (c *HTTPClient) ListTemplates(ctx context.Context) (map[string]types.Template, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/templates", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	var result types.Template
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Volume operations

func (c *HTTPClient) CreateVolume(ctx context.Context, req types.VolumeCreateRequest) (*types.StorageVolume, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/volumes", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.StorageVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) ListVolumes(ctx context.Context) ([]*types.StorageVolume, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/volumes", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []*types.StorageVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *HTTPClient) GetVolume(ctx context.Context, name string) (*types.StorageVolume, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/volumes/%s", name), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.StorageVolume
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
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) AttachVolume(ctx context.Context, volumeName, instanceName string) error {
	req := map[string]string{"instance": instanceName}
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/volumes/%s/attach", volumeName), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) DetachVolume(ctx context.Context, volumeName string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/volumes/%s/detach", volumeName), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	req := map[string]string{
		"instance":    instanceName,
		"mount_point": mountPoint,
	}
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/volumes/%s/mount", volumeName), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	req := map[string]string{"instance": instanceName}
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/volumes/%s/unmount", volumeName), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

// Storage operations

func (c *HTTPClient) CreateStorage(ctx context.Context, req types.StorageCreateRequest) (*types.StorageVolume, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/storage", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.StorageVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) ListStorage(ctx context.Context) ([]*types.StorageVolume, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/storage", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []*types.StorageVolume
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *HTTPClient) GetStorage(ctx context.Context, name string) (*types.StorageVolume, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/storage/%s", name), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.StorageVolume
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
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) AttachStorage(ctx context.Context, storageName, instanceName string) error {
	req := map[string]string{"instance": instanceName}
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/storage/%s/attach", storageName), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) DetachStorage(ctx context.Context, storageName string) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/storage/%s/detach", storageName), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

// Registry operations - these will need proper implementation based on actual API endpoints

func (c *HTTPClient) GetRegistryStatus(ctx context.Context) (*RegistryStatusResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/registry/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()
	return c.handleResponse(resp, nil)
}

func (c *HTTPClient) LookupAMI(ctx context.Context, templateName, region, architecture string) (*AMIReferenceResponse, error) {
	path := fmt.Sprintf("/api/v1/registry/ami?template=%s&region=%s&architecture=%s",
		templateName, region, architecture)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	return c.handleResponse(resp, nil)
}

// Idle detection operations (new system)

func (c *HTTPClient) GetIdlePendingActions(ctx context.Context) ([]types.IdleState, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/idle/pending-actions", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []types.IdleState
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *HTTPClient) ExecuteIdleActions(ctx context.Context) (*types.IdleExecutionResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/idle/execute-actions", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.IdleExecutionResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPClient) GetIdleHistory(ctx context.Context) ([]types.IdleHistoryEntry, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/idle/history", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []types.IdleHistoryEntry
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Project management operations

// CreateProject creates a new project
func (c *HTTPClient) CreateProject(ctx context.Context, req project.CreateProjectRequest) (*types.Project, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/projects", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var project types.Project
	if err := c.handleResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// ListProjects lists projects with optional filtering
func (c *HTTPClient) ListProjects(ctx context.Context, filter *project.ProjectFilter) (*project.ProjectListResponse, error) {
	// Build query parameters
	params := url.Values{}
	if filter != nil {
		if filter.Owner != "" {
			params.Set("owner", filter.Owner)
		}
		if filter.Status != nil {
			params.Set("status", string(*filter.Status))
		}
		if filter.HasBudget != nil {
			params.Set("has_budget", strconv.FormatBool(*filter.HasBudget))
		}
		if filter.CreatedAfter != nil {
			params.Set("created_after", filter.CreatedAfter.Format(time.RFC3339))
		}
		if filter.CreatedBefore != nil {
			params.Set("created_before", filter.CreatedBefore.Format(time.RFC3339))
		}
	}

	path := "/api/v1/projects"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result project.ProjectListResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetProject retrieves a specific project
func (c *HTTPClient) GetProject(ctx context.Context, projectID string) (*types.Project, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/projects/%s", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var project types.Project
	if err := c.handleResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// UpdateProject updates a project
func (c *HTTPClient) UpdateProject(ctx context.Context, projectID string, req project.UpdateProjectRequest) (*types.Project, error) {
	resp, err := c.makeRequest(ctx, "PUT", fmt.Sprintf("/api/v1/projects/%s", projectID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var project types.Project
	if err := c.handleResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// DeleteProject deletes a project
func (c *HTTPClient) DeleteProject(ctx context.Context, projectID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/projects/%s", projectID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, nil)
}

// AddProjectMember adds a member to a project
func (c *HTTPClient) AddProjectMember(ctx context.Context, projectID string, req project.AddMemberRequest) error {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/projects/%s/members", projectID), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, nil)
}

// UpdateProjectMember updates a project member's role
func (c *HTTPClient) UpdateProjectMember(ctx context.Context, projectID, userID string, req project.UpdateMemberRequest) error {
	resp, err := c.makeRequest(ctx, "PUT", fmt.Sprintf("/api/v1/projects/%s/members/%s", projectID, userID), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, nil)
}

// RemoveProjectMember removes a member from a project
func (c *HTTPClient) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/projects/%s/members/%s", projectID, userID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, nil)
}

// GetProjectMembers retrieves project members
func (c *HTTPClient) GetProjectMembers(ctx context.Context, projectID string) ([]types.ProjectMember, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/projects/%s/members", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var members []types.ProjectMember
	if err := c.handleResponse(resp, &members); err != nil {
		return nil, err
	}

	return members, nil
}

// GetProjectBudgetStatus retrieves budget status for a project
func (c *HTTPClient) GetProjectBudgetStatus(ctx context.Context, projectID string) (*project.BudgetStatus, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/projects/%s/budget", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var budgetStatus project.BudgetStatus
	if err := c.handleResponse(resp, &budgetStatus); err != nil {
		return nil, err
	}

	return &budgetStatus, nil
}

// GetProjectCostBreakdown retrieves detailed cost analysis for a project
func (c *HTTPClient) GetProjectCostBreakdown(ctx context.Context, projectID string, startDate, endDate time.Time) (*types.ProjectCostBreakdown, error) {
	params := url.Values{}
	params.Set("start_date", startDate.Format(time.RFC3339))
	params.Set("end_date", endDate.Format(time.RFC3339))

	path := fmt.Sprintf("/api/v1/projects/%s/costs?%s", projectID, params.Encode())
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var costBreakdown types.ProjectCostBreakdown
	if err := c.handleResponse(resp, &costBreakdown); err != nil {
		return nil, err
	}

	return &costBreakdown, nil
}

// GetProjectResourceUsage retrieves resource utilization metrics for a project
func (c *HTTPClient) GetProjectResourceUsage(ctx context.Context, projectID string, period time.Duration) (*types.ProjectResourceUsage, error) {
	params := url.Values{}
	params.Set("period", period.String())

	path := fmt.Sprintf("/api/v1/projects/%s/usage?%s", projectID, params.Encode())
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var usage types.ProjectResourceUsage
	if err := c.handleResponse(resp, &usage); err != nil {
		return nil, err
	}

	return &usage, nil
}

// Universal AMI System methods (Phase 5.1 Week 2)

// ResolveAMI resolves AMI for a template
func (c *HTTPClient) ResolveAMI(ctx context.Context, templateName string, params map[string]interface{}) (map[string]interface{}, error) {
	var queryParams url.Values
	if params != nil {
		queryParams = url.Values{}
		if details, ok := params["details"].(bool); ok && details {
			queryParams.Set("details", "true")
		}
		if region, ok := params["region"].(string); ok && region != "" {
			queryParams.Set("region", region)
		}
	}

	path := fmt.Sprintf("/api/v1/ami/resolve/%s", templateName)
	if len(queryParams) > 0 {
		path += "?" + queryParams.Encode()
	}

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// TestAMIAvailability tests AMI availability across regions
func (c *HTTPClient) TestAMIAvailability(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	path := "/api/v1/ami/test"

	resp, err := c.makeRequest(ctx, "POST", path, request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetAMICosts provides cost analysis for AMI vs script deployment
func (c *HTTPClient) GetAMICosts(ctx context.Context, templateName string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/ami/costs/%s", templateName)

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// PreviewAMIResolution shows what would happen during AMI resolution
func (c *HTTPClient) PreviewAMIResolution(ctx context.Context, templateName string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/ami/preview/%s", templateName)

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// AMI Creation methods (Phase 5.1 AMI Creation)

// CreateAMI creates an AMI from a running instance
func (c *HTTPClient) CreateAMI(ctx context.Context, request types.AMICreationRequest) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/ami/create", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAMIStatus checks the status of AMI creation
func (c *HTTPClient) GetAMIStatus(ctx context.Context, creationID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/ami/status/%s", creationID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListUserAMIs lists AMIs created by the user
func (c *HTTPClient) ListUserAMIs(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/ami/list", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// AMI Lifecycle Management operations

// CleanupAMIs removes old and unused AMIs
func (c *HTTPClient) CleanupAMIs(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/ami/cleanup", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteAMI deletes a specific AMI by ID
func (c *HTTPClient) DeleteAMI(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/ami/delete", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// AMI Snapshot operations

// ListAMISnapshots lists available snapshots
func (c *HTTPClient) ListAMISnapshots(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/ami/snapshots", filters)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateAMISnapshot creates a snapshot from an instance
func (c *HTTPClient) CreateAMISnapshot(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/ami/snapshot/create", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RestoreAMIFromSnapshot creates an AMI from a snapshot
func (c *HTTPClient) RestoreAMIFromSnapshot(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/ami/snapshot/restore", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteAMISnapshot deletes a specific snapshot
func (c *HTTPClient) DeleteAMISnapshot(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/ami/snapshot/delete", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CheckAMIFreshness checks static AMI IDs against latest SSM versions (v0.5.4 - Universal Version System)
func (c *HTTPClient) CheckAMIFreshness(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/ami/check-freshness", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Template Marketplace operations (Phase 5.2)

// SearchMarketplace searches the marketplace for templates
func (c *HTTPClient) SearchMarketplace(ctx context.Context, query map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/marketplace/templates", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMarketplaceTemplate gets a specific template from the marketplace
func (c *HTTPClient) GetMarketplaceTemplate(ctx context.Context, templateID string) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/marketplace/templates/%s", templateID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PublishMarketplaceTemplate publishes a template to the marketplace
func (c *HTTPClient) PublishMarketplaceTemplate(ctx context.Context, template map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/marketplace/publish", template)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// AddMarketplaceReview adds a review for a marketplace template
func (c *HTTPClient) AddMarketplaceReview(ctx context.Context, templateID string, review map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/marketplace/templates/%s/reviews", templateID), review)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ForkMarketplaceTemplate forks a marketplace template for customization
func (c *HTTPClient) ForkMarketplaceTemplate(ctx context.Context, templateID string, fork map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/marketplace/templates/%s/fork", templateID), fork)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMarketplaceFeatured gets featured templates from the marketplace
func (c *HTTPClient) GetMarketplaceFeatured(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/marketplace/featured", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMarketplaceTrending gets trending templates from the marketplace
func (c *HTTPClient) GetMarketplaceTrending(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/marketplace/trending", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetProjectBudgetRequest represents a request to set or enable a project budget
type SetProjectBudgetRequest struct {
	TotalBudget     float64                  `json:"total_budget"`
	MonthlyLimit    *float64                 `json:"monthly_limit,omitempty"`
	DailyLimit      *float64                 `json:"daily_limit,omitempty"`
	AlertThresholds []types.BudgetAlert      `json:"alert_thresholds,omitempty"`
	AutoActions     []types.BudgetAutoAction `json:"auto_actions,omitempty"`
	BudgetPeriod    types.BudgetPeriod       `json:"budget_period"`
	EndDate         *time.Time               `json:"end_date,omitempty"`
}

// UpdateProjectBudgetRequest represents a request to update a project budget
type UpdateProjectBudgetRequest struct {
	TotalBudget     *float64                 `json:"total_budget,omitempty"`
	MonthlyLimit    *float64                 `json:"monthly_limit,omitempty"`
	DailyLimit      *float64                 `json:"daily_limit,omitempty"`
	AlertThresholds []types.BudgetAlert      `json:"alert_thresholds,omitempty"`
	AutoActions     []types.BudgetAutoAction `json:"auto_actions,omitempty"`
	EndDate         *time.Time               `json:"end_date,omitempty"`
}

// SetProjectBudget sets or enables budget tracking for a project
func (c *HTTPClient) SetProjectBudget(ctx context.Context, projectID string, req SetProjectBudgetRequest) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "PUT", fmt.Sprintf("/api/v1/projects/%s/budget", projectID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateProjectBudget updates an existing project budget
func (c *HTTPClient) UpdateProjectBudget(ctx context.Context, projectID string, req UpdateProjectBudgetRequest) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/api/v1/projects/%s/budget", projectID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DisableProjectBudget disables budget tracking for a project
func (c *HTTPClient) DisableProjectBudget(ctx context.Context, projectID string) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/projects/%s/budget", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetCostTrends retrieves cost trends for analysis
func (c *HTTPClient) GetCostTrends(ctx context.Context, projectID, period string) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/cost/trends?project_id=%s&period=%s", projectID, period), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Data Backup operations

// CreateBackup creates a data backup from an instance
func (c *HTTPClient) CreateBackup(ctx context.Context, req types.BackupCreateRequest) (*types.BackupCreateResult, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/backups", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.BackupCreateResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListBackups lists all data backups
func (c *HTTPClient) ListBackups(ctx context.Context) (*types.BackupListResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/backups", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.BackupListResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetBackup gets detailed information about a backup
func (c *HTTPClient) GetBackup(ctx context.Context, backupName string) (*types.BackupInfo, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/backups/%s", backupName), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.BackupInfo
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteBackup deletes a backup
func (c *HTTPClient) DeleteBackup(ctx context.Context, backupName string) (*types.BackupDeleteResult, error) {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/backups/%s", backupName), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.BackupDeleteResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetBackupContents lists the contents of a backup
func (c *HTTPClient) GetBackupContents(ctx context.Context, req types.BackupContentsRequest) (*types.BackupContentsResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/backups/contents", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.BackupContentsResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// VerifyBackup verifies backup integrity
func (c *HTTPClient) VerifyBackup(ctx context.Context, req types.BackupVerifyRequest) (*types.BackupVerifyResult, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/backups/verify", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.BackupVerifyResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Data Restore operations

// RestoreBackup restores data from a backup
func (c *HTTPClient) RestoreBackup(ctx context.Context, req types.RestoreRequest) (*types.RestoreResult, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/backups/restore", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.RestoreResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetRestoreStatus gets the status of a restore operation
func (c *HTTPClient) GetRestoreStatus(ctx context.Context, restoreID string) (*types.RestoreResult, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/backups/restore/%s", restoreID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.RestoreResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListRestoreOperations lists all restore operations
func (c *HTTPClient) ListRestoreOperations(ctx context.Context) ([]types.RestoreResult, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/backups/restore", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []types.RestoreResult
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ==========================================
// Version Compatibility Check
// ==========================================

// CheckVersionCompatibility verifies that the client and daemon versions are compatible
func (c *HTTPClient) CheckVersionCompatibility(ctx context.Context, clientVersion string) error {
	// Parse client version
	clientMajor, clientMinor, _ := parseVersion(clientVersion)

	// Get daemon status which includes version information
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/status", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	var status map[string]interface{}
	if err := c.handleResponse(resp, &status); err != nil {
		return fmt.Errorf("failed to get daemon status: %w", err)
	}

	// Extract daemon version from status
	daemonVersionStr, ok := status["version"].(string)
	if !ok {
		return fmt.Errorf("daemon did not return version information")
	}

	daemonMajor, daemonMinor, _ := parseVersion(daemonVersionStr)

	// Version compatibility rules:
	// 1. Major versions must match exactly
	// 2. Client minor version must be <= daemon minor version (daemon can be newer)
	// 3. Patch versions are ignored for compatibility

	if clientMajor != daemonMajor {
		return fmt.Errorf("❌ VERSION MISMATCH ERROR\n\n"+
			"Client version:  v%s\n"+
			"Daemon version:  v%s\n\n"+
			"The client and daemon have incompatible major versions.\n"+
			"Both must be updated to the same major version.\n\n"+
			"💡 To fix this:\n"+
			"   1. Stop the daemon: cws daemon stop\n"+
			"   2. Update CloudWorkstation: brew upgrade cloudworkstation\n"+
			"   3. Restart the daemon: cws daemon start\n"+
			"   4. Verify versions match: cws version && cws daemon status",
			clientVersion, daemonVersionStr)
	}

	if clientMinor > daemonMinor {
		return fmt.Errorf("❌ VERSION MISMATCH ERROR\n\n"+
			"Client version:  v%s\n"+
			"Daemon version:  v%s\n\n"+
			"Your CLI client is newer than the daemon.\n"+
			"The daemon needs to be updated.\n\n"+
			"💡 To fix this:\n"+
			"   1. Stop the daemon: cws daemon stop\n"+
			"   2. The daemon will auto-start with the new version\n"+
			"   3. Or manually restart: cws daemon start\n"+
			"   4. Verify versions match: cws version && cws daemon status",
			clientVersion, daemonVersionStr)
	}

	// Versions are compatible
	return nil
}

// parseVersion parses a version string like "v0.5.1" or "0.5.1" into major, minor, patch
func parseVersion(version string) (major, minor, patch int) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Split into parts
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, 0, 0
	}

	major, _ = strconv.Atoi(parts[0])
	minor, _ = strconv.Atoi(parts[1])
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}

	return major, minor, patch
}
