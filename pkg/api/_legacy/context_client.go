package api

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// ContextClient wraps the legacy client to implement the context-aware CloudWorkstationAPI interface
// and supports profile-aware operations through ProfileAwareAPI
type ContextClient struct {
	client *Client
	profileManager *profile.ManagerEnhanced
}

// NewContextClient creates a new context-aware client
func NewContextClient(client *Client) *ContextClient {
	return &ContextClient{
		client: client,
	}
}

// NewContextClientWithURL creates a new context-aware client with a URL
func NewContextClientWithURL(baseURL string) *ContextClient {
	return &ContextClient{
		client: NewClient(baseURL),
	}
}

// WithProfileOverride creates a new client using the specified profile
func (c *ContextClient) WithProfileOverride(profileManager *profile.ManagerEnhanced, profileID string) (CloudWorkstationAPI, error) {
	// Create a base client with profile
	baseClient, err := c.client.WithProfile(profileManager, profileID)
	if err != nil {
		return nil, err
	}
	
	// Create a new context client with the configured base client
	return &ContextClient{
		client: baseClient,
		profileManager: profileManager,
	}, nil
}

// SetOptions updates the client's configuration options
func (c *ContextClient) SetOptions(options ClientOptions) {
	// Pass options to the underlying client
	if c.client != nil {
		// Set AWS profile
		if options.AWSProfile != "" {
			c.client.SetAWSProfile(options.AWSProfile)
		}
		
		// Set AWS region
		if options.AWSRegion != "" {
			c.client.SetAWSRegion(options.AWSRegion)
		}
		
		// Set invitation details
		if options.InvitationToken != "" {
			c.client.SetInvitationToken(options.InvitationToken, options.OwnerAccount, options.S3ConfigPath)
		}
	}
}

// Ensure ContextClient implements CloudWorkstationAPI
var _ CloudWorkstationAPI = (*ContextClient)(nil)

// Instance operations

// LaunchInstance launches a new instance with context
func (c *ContextClient) LaunchInstance(ctx context.Context, req types.LaunchRequest) (*types.LaunchResponse, error) {
	return c.client.LaunchInstance(req)
}

// ListInstances lists all instances with context
func (c *ContextClient) ListInstances(ctx context.Context) (*types.ListResponse, error) {
	return c.client.ListInstances()
}

// GetInstance gets a specific instance with context
func (c *ContextClient) GetInstance(ctx context.Context, name string) (*types.Instance, error) {
	return c.client.GetInstance(name)
}

// StartInstance starts an instance with context
func (c *ContextClient) StartInstance(ctx context.Context, name string) error {
	return c.client.StartInstance(name)
}

// StopInstance stops an instance with context
func (c *ContextClient) StopInstance(ctx context.Context, name string) error {
	return c.client.StopInstance(name)
}

// DeleteInstance deletes an instance with context
func (c *ContextClient) DeleteInstance(ctx context.Context, name string) error {
	return c.client.DeleteInstance(name)
}

// ConnectInstance returns connection information for an instance
func (c *ContextClient) ConnectInstance(ctx context.Context, name string) (string, error) {
	return c.client.ConnectInstance(name)
}

// Template operations

// ListTemplates lists all templates with context
func (c *ContextClient) ListTemplates(ctx context.Context) (map[string]types.Template, error) {
	return c.client.ListTemplates()
}

// GetTemplate gets a specific template with context
func (c *ContextClient) GetTemplate(ctx context.Context, name string) (*types.Template, error) {
	return c.client.GetTemplate(name)
}

// Volume operations (EFS)

// CreateVolume creates a new EFS volume with context
func (c *ContextClient) CreateVolume(ctx context.Context, req types.VolumeCreateRequest) (*types.EFSVolume, error) {
	return c.client.CreateVolume(req)
}

// ListVolumes lists all EFS volumes with context
func (c *ContextClient) ListVolumes(ctx context.Context) ([]types.EFSVolume, error) {
	volumes, err := c.client.ListVolumes()
	if err != nil {
		return nil, err
	}
	
	// Convert map to slice
	var result []types.EFSVolume
	for _, volume := range volumes {
		result = append(result, volume)
	}
	
	return result, nil
}

// GetVolume gets a specific EFS volume with context
func (c *ContextClient) GetVolume(ctx context.Context, name string) (*types.EFSVolume, error) {
	return c.client.GetVolume(name)
}

// DeleteVolume deletes an EFS volume with context
func (c *ContextClient) DeleteVolume(ctx context.Context, name string) error {
	return c.client.DeleteVolume(name)
}

// AttachVolume attaches an EFS volume to an instance with context
func (c *ContextClient) AttachVolume(ctx context.Context, volumeName, instanceName string) error {
	// Implement based on legacy client
	return fmt.Errorf("AttachVolume not implemented in legacy client")
}

// DetachVolume detaches an EFS volume from an instance with context
func (c *ContextClient) DetachVolume(ctx context.Context, volumeName string) error {
	// Implement based on legacy client
	return fmt.Errorf("DetachVolume not implemented in legacy client")
}

// Storage operations (EBS)

// CreateStorage creates a new EBS volume with context
func (c *ContextClient) CreateStorage(ctx context.Context, req types.StorageCreateRequest) (*types.EBSVolume, error) {
	return c.client.CreateStorage(req)
}

// ListStorage lists all EBS volumes with context
func (c *ContextClient) ListStorage(ctx context.Context) ([]types.EBSVolume, error) {
	storage, err := c.client.ListStorage()
	if err != nil {
		return nil, err
	}
	
	// Convert map to slice
	var result []types.EBSVolume
	for _, volume := range storage {
		result = append(result, volume)
	}
	
	return result, nil
}

// GetStorage gets a specific EBS volume with context
func (c *ContextClient) GetStorage(ctx context.Context, name string) (*types.EBSVolume, error) {
	return c.client.GetStorage(name)
}

// DeleteStorage deletes an EBS volume with context
func (c *ContextClient) DeleteStorage(ctx context.Context, name string) error {
	return c.client.DeleteStorage(name)
}

// AttachStorage attaches an EBS volume to an instance with context
func (c *ContextClient) AttachStorage(ctx context.Context, volumeName, instanceName string) error {
	return c.client.AttachStorage(volumeName, instanceName)
}

// DetachStorage detaches an EBS volume from an instance with context
func (c *ContextClient) DetachStorage(ctx context.Context, volumeName string) error {
	return c.client.DetachStorage(volumeName)
}

// Status operations

// GetStatus gets daemon status with context
func (c *ContextClient) GetStatus(ctx context.Context) (*types.DaemonStatus, error) {
	return c.client.GetStatus()
}

// Ping checks if the daemon is responsive with context
func (c *ContextClient) Ping(ctx context.Context) error {
	return c.client.Ping()
}

// Registry operations

// GetRegistryStatus gets registry status with context
func (c *ContextClient) GetRegistryStatus(ctx context.Context) (*RegistryStatusResponse, error) {
	// Implement based on legacy client or return not implemented
	return nil, fmt.Errorf("GetRegistryStatus not implemented")
}

// SetRegistryStatus sets registry status with context
func (c *ContextClient) SetRegistryStatus(ctx context.Context, enabled bool) error {
	// Implement based on legacy client or return not implemented
	return fmt.Errorf("SetRegistryStatus not implemented")
}

// LookupAMI looks up an AMI with context
func (c *ContextClient) LookupAMI(ctx context.Context, template, region, arch string) (*AMIReferenceResponse, error) {
	// Implement based on legacy client or return not implemented
	return nil, fmt.Errorf("LookupAMI not implemented")
}

// ListTemplateAMIs lists all AMIs for a template with context
func (c *ContextClient) ListTemplateAMIs(ctx context.Context, template string) ([]AMIReferenceResponse, error) {
	// Implement based on legacy client or return not implemented
	return nil, fmt.Errorf("ListTemplateAMIs not implemented")
}

// Repository operations

// ListRepositories lists all template repositories
func (c *ContextClient) ListRepositories(ctx context.Context) ([]types.TemplateRepository, error) {
	// Stub implementation for now
	return []types.TemplateRepository{}, nil
}

// GetRepository gets a specific template repository
func (c *ContextClient) GetRepository(ctx context.Context, name string) (*types.TemplateRepository, error) {
	// Stub implementation for now
	return &types.TemplateRepository{
		Name:     name,
		URL:      "https://example.com/repo",
		Priority: 0,
		Enabled:  true,
	}, nil
}

// AddRepository adds a new template repository
func (c *ContextClient) AddRepository(ctx context.Context, repo types.TemplateRepositoryUpdate) error {
	// Stub implementation for now
	return nil
}

// UpdateRepository updates an existing template repository
func (c *ContextClient) UpdateRepository(ctx context.Context, repo types.TemplateRepositoryUpdate) error {
	// Stub implementation for now
	return nil
}

// RemoveRepository removes a template repository
func (c *ContextClient) RemoveRepository(ctx context.Context, name string) error {
	// Stub implementation for now
	return nil
}

// SyncRepositories synchronizes all template repositories
func (c *ContextClient) SyncRepositories(ctx context.Context) error {
	// Stub implementation for now
	return nil
}