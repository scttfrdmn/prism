// Package api provides an API client for the TUI.
package api

import (
	"context"

	pkgapi "github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TUIClient wraps the CloudWorkstationAPI interface to provide
// a consistent interface for the TUI models.
type TUIClient struct {
	client pkgapi.CloudWorkstationAPI
}

// NewTUIClient creates a new TUIClient.
func NewTUIClient(client pkgapi.CloudWorkstationAPI) *TUIClient {
	return &TUIClient{
		client: client,
	}
}

// Instance operations

// ListInstances returns all instances
func (c *TUIClient) ListInstances(ctx context.Context) (*ListInstancesResponse, error) {
	resp, err := c.client.ListInstances(ctx)
	if err != nil {
		return nil, err
	}
	return ToListInstancesResponse(resp), nil
}

// GetInstance returns a specific instance
func (c *TUIClient) GetInstance(ctx context.Context, name string) (*InstanceResponse, error) {
	instance, err := c.client.GetInstance(ctx, name)
	if err != nil {
		return nil, err
	}
	resp := ToInstanceResponse(*instance)
	return &resp, nil
}

// LaunchInstance launches a new instance
func (c *TUIClient) LaunchInstance(ctx context.Context, req LaunchInstanceRequest) (*LaunchInstanceResponse, error) {
	// Convert request
	launchReq := types.LaunchRequest{
		Template:   req.Template,
		Name:       req.Name,
		Size:       req.Size,
		Volumes:    req.Volumes,
		EBSVolumes: req.EBSVolumes,
		Region:     req.Region,
		Spot:       req.Spot,
		DryRun:     req.DryRun,
	}

	// Make API call
	resp, err := c.client.LaunchInstance(ctx, launchReq)
	if err != nil {
		return nil, err
	}

	// Convert response
	return &LaunchInstanceResponse{
		Instance:       ToInstanceResponse(resp.Instance),
		Message:        resp.Message,
		EstimatedCost:  resp.EstimatedCost,
		ConnectionInfo: resp.ConnectionInfo,
	}, nil
}

// StartInstance starts a stopped instance
func (c *TUIClient) StartInstance(ctx context.Context, name string) error {
	return c.client.StartInstance(ctx, name)
}

// StopInstance stops a running instance
func (c *TUIClient) StopInstance(ctx context.Context, name string) error {
	return c.client.StopInstance(ctx, name)
}

// DeleteInstance terminates an instance
func (c *TUIClient) DeleteInstance(ctx context.Context, name string) error {
	return c.client.DeleteInstance(ctx, name)
}

// Template operations

// ListTemplates returns all available templates
func (c *TUIClient) ListTemplates(ctx context.Context) (*ListTemplatesResponse, error) {
	templates, err := c.client.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	return ToListTemplatesResponse(templates), nil
}

// GetTemplate returns a specific template
func (c *TUIClient) GetTemplate(ctx context.Context, name string) (*TemplateResponse, error) {
	template, err := c.client.GetTemplate(ctx, name)
	if err != nil {
		return nil, err
	}
	resp := ToTemplateResponse(name, *template)
	return &resp, nil
}

// Volume operations

// ListVolumes returns all EFS volumes
func (c *TUIClient) ListVolumes(ctx context.Context) (*ListVolumesResponse, error) {
	volumes, err := c.client.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}
	return ToListVolumesResponse(volumes), nil
}

// Storage operations

// ListStorage returns all EBS volumes
func (c *TUIClient) ListStorage(ctx context.Context) (*ListStorageResponse, error) {
	storage, err := c.client.ListStorage(ctx)
	if err != nil {
		return nil, err
	}
	return ToListStorageResponse(storage), nil
}

// Status operations

// Ping checks if the daemon is responsive
func (c *TUIClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx)
}

// GetStatus returns the daemon status information
func (c *TUIClient) GetStatus(ctx context.Context) (*SystemStatusResponse, error) {
	status, err := c.client.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	return ToSystemStatusResponse(status), nil
}

// Idle detection operations

// ListIdlePolicies returns all idle detection policies
func (c *TUIClient) ListIdlePolicies(ctx context.Context) (*ListIdlePoliciesResponse, error) {
	// For now, return hardcoded policies
	return &ListIdlePoliciesResponse{
		Policies: map[string]IdlePolicyResponse{
			"default": {
				Name:      "default",
				Threshold: 30, // 30 minutes
				Action:    "stop",
			},
			"aggressive": {
				Name:      "aggressive",
				Threshold: 15, // 15 minutes
				Action:    "stop",
			},
			"conservative": {
				Name:      "conservative",
				Threshold: 60, // 60 minutes
				Action:    "stop",
			},
		},
	}, nil
}

// UpdateIdlePolicy updates an idle detection policy
func (c *TUIClient) UpdateIdlePolicy(ctx context.Context, req IdlePolicyUpdateRequest) error {
	// This is a stub implementation
	return nil
}

// GetInstanceIdleStatus returns idle detection status for an instance
func (c *TUIClient) GetInstanceIdleStatus(ctx context.Context, name string) (*IdleDetectionResponse, error) {
	// This is a stub implementation
	return &IdleDetectionResponse{
		Enabled:       true,
		Policy:        "default",
		IdleTime:      5,  // 5 minutes
		Threshold:     30, // 30 minutes
		ActionPending: false,
	}, nil
}

// EnableIdleDetection enables idle detection for an instance
func (c *TUIClient) EnableIdleDetection(ctx context.Context, name, policy string) error {
	// This is a stub implementation
	return nil
}

// DisableIdleDetection disables idle detection for an instance
func (c *TUIClient) DisableIdleDetection(ctx context.Context, name string) error {
	// This is a stub implementation
	return nil
}

// Volume mount/unmount operations

// MountVolume mounts an EFS volume to an instance
func (c *TUIClient) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	return c.client.MountVolume(ctx, volumeName, instanceName, mountPoint)
}

// UnmountVolume unmounts an EFS volume from an instance
func (c *TUIClient) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	return c.client.UnmountVolume(ctx, volumeName, instanceName)
}
