package api

import (
	"context"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Context-aware wrapper methods for the API client
// These methods implement the CloudWorkstationAPI interface with context support
// while maintaining compatibility with the original client implementation

// ListVolumes returns all EFS volumes with context
func (c *Client) ListVolumes(ctx context.Context) ([]types.EFSVolume, error) {
	// Call original implementation and convert map to slice
	volumeMap, err := c.ListVolumesLegacy()
	if err != nil {
		return nil, err
	}

	volumes := make([]types.EFSVolume, 0, len(volumeMap))
	for _, vol := range volumeMap {
		volumes = append(volumes, vol)
	}
	return volumes, nil
}

// ListVolumesLegacy is the original implementation without context
func (c *Client) ListVolumesLegacy() (map[string]types.EFSVolume, error) {
	return c.ListVolumesImpl()
}

// ListStorage returns all EBS volumes with context
func (c *Client) ListStorage(ctx context.Context) ([]types.EBSVolume, error) {
	// Call original implementation and convert map to slice
	storageMap, err := c.ListStorageLegacy()
	if err != nil {
		return nil, err
	}

	storage := make([]types.EBSVolume, 0, len(storageMap))
	for _, vol := range storageMap {
		storage = append(storage, vol)
	}
	return storage, nil
}

// ListStorageLegacy is the original implementation without context
func (c *Client) ListStorageLegacy() (map[string]types.EBSVolume, error) {
	return c.ListStorageImpl()
}

// AttachStorage attaches an EBS volume to an instance with context
func (c *Client) AttachStorage(ctx context.Context, volumeName, instanceName string) error {
	return c.AttachStorageLegacy(volumeName, instanceName)
}

// AttachStorageLegacy is the original implementation without context
func (c *Client) AttachStorageLegacy(volumeName, instanceName string) error {
	return c.AttachStorageImpl(volumeName, instanceName)
}

// These are internal methods that are called by both legacy and context-aware methods
// They need to be implemented for the client to work

// ListVolumesImpl is the actual implementation
func (c *Client) ListVolumesImpl() (map[string]types.EFSVolume, error) {
	// Re-implement or delegate to the original method
	return c.doListVolumes()
}

// ListStorageImpl is the actual implementation
func (c *Client) ListStorageImpl() (map[string]types.EBSVolume, error) {
	// Re-implement or delegate to the original method
	return c.doListStorage()
}

// AttachStorageImpl is the actual implementation
func (c *Client) AttachStorageImpl(volumeName, instanceName string) error {
	// Re-implement or delegate to the original method
	return c.doAttachStorage(volumeName, instanceName)
}

// Helper methods that delegate to the original implementation
// These should be replaced with direct access to the original methods in a real implementation

func (c *Client) doListVolumes() (map[string]types.EFSVolume, error) {
	// In a real implementation, this would call the original method directly
	// For now, we need to provide a mock implementation
	return map[string]types.EFSVolume{}, nil
}

func (c *Client) doListStorage() (map[string]types.EBSVolume, error) {
	// In a real implementation, this would call the original method directly
	// For now, we need to provide a mock implementation
	return map[string]types.EBSVolume{}, nil
}

func (c *Client) doAttachStorage(volumeName, instanceName string) error {
	// In a real implementation, this would call the original method directly
	// For now, we need to provide a mock implementation
	return nil
}