package client

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// MockClient provides a mock implementation of CloudWorkstationAPI for testing
type MockClient struct {
	// Mock data
	instances map[string]*types.Instance
	templates map[string]types.Template
	volumes   []types.EFSVolume
	storage   []types.EBSVolume

	// Mock responses
	pingError    error
	statusResult *types.DaemonStatus

	// Configuration
	options Options
}

// NewMockClient creates a new mock client for testing
func NewMockClient() *MockClient {
	return &MockClient{
		instances: make(map[string]*types.Instance),
		templates: make(map[string]types.Template),
		volumes:   make([]types.EFSVolume, 0),
		storage:   make([]types.EBSVolume, 0),
		statusResult: &types.DaemonStatus{
			Version:   "mock-1.0.0",
			Status:    "running",
			StartTime: time.Now(),
		},
	}
}

// Configuration
func (m *MockClient) SetOptions(opts Options) {
	m.options = opts
}

// Instance operations
func (m *MockClient) LaunchInstance(ctx context.Context, req types.LaunchRequest) (*types.LaunchResponse, error) {
	instance := &types.Instance{
		ID:       "mock-" + req.Name,
		Name:     req.Name,
		Template: req.Template,
		State:    "running",
	}
	m.instances[req.Name] = instance

	return &types.LaunchResponse{
		Instance: *instance,
		Message:  "Mock instance launched",
	}, nil
}

func (m *MockClient) ListInstances(ctx context.Context) (*types.ListResponse, error) {
	instances := make([]types.Instance, 0, len(m.instances))
	for _, instance := range m.instances {
		instances = append(instances, *instance)
	}

	return &types.ListResponse{
		Instances: instances,
		TotalCost: 0.0,
	}, nil
}

func (m *MockClient) GetInstance(ctx context.Context, name string) (*types.Instance, error) {
	if instance, exists := m.instances[name]; exists {
		return instance, nil
	}
	return nil, fmt.Errorf("instance not found: %s", name)
}

func (m *MockClient) StartInstance(ctx context.Context, name string) error {
	if instance, exists := m.instances[name]; exists {
		instance.State = "running"
		return nil
	}
	return fmt.Errorf("instance not found: %s", name)
}

func (m *MockClient) StopInstance(ctx context.Context, name string) error {
	if instance, exists := m.instances[name]; exists {
		instance.State = "stopped"
		return nil
	}
	return fmt.Errorf("instance not found: %s", name)
}

func (m *MockClient) DeleteInstance(ctx context.Context, name string) error {
	delete(m.instances, name)
	return nil
}

func (m *MockClient) ConnectInstance(ctx context.Context, name string) (string, error) {
	return "ssh mock@mock-ip", nil
}

// Template operations
func (m *MockClient) ListTemplates(ctx context.Context) (map[string]types.Template, error) {
	return m.templates, nil
}

func (m *MockClient) GetTemplate(ctx context.Context, name string) (*types.Template, error) {
	if template, exists := m.templates[name]; exists {
		return &template, nil
	}
	return nil, fmt.Errorf("template not found: %s", name)
}

// Volume operations
func (m *MockClient) CreateVolume(ctx context.Context, req types.VolumeCreateRequest) (*types.EFSVolume, error) {
	volume := types.EFSVolume{
		Name:         req.Name,
		FileSystemId: "mock-fs-" + req.Name,
		State:        "available",
	}
	m.volumes = append(m.volumes, volume)
	return &volume, nil
}

func (m *MockClient) ListVolumes(ctx context.Context) ([]types.EFSVolume, error) {
	return m.volumes, nil
}

func (m *MockClient) GetVolume(ctx context.Context, name string) (*types.EFSVolume, error) {
	for _, volume := range m.volumes {
		if volume.Name == name {
			return &volume, nil
		}
	}
	return nil, fmt.Errorf("volume not found: %s", name)
}

func (m *MockClient) DeleteVolume(ctx context.Context, name string) error {
	for i, volume := range m.volumes {
		if volume.Name == name {
			m.volumes = append(m.volumes[:i], m.volumes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("volume not found: %s", name)
}

func (m *MockClient) AttachVolume(ctx context.Context, volumeName, instanceName string) error {
	return nil // Mock implementation
}

func (m *MockClient) DetachVolume(ctx context.Context, volumeName string) error {
	return nil // Mock implementation
}

func (m *MockClient) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	return nil // Mock implementation
}

func (m *MockClient) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	return nil // Mock implementation
}

// Storage operations
func (m *MockClient) CreateStorage(ctx context.Context, req types.StorageCreateRequest) (*types.EBSVolume, error) {
	volume := types.EBSVolume{
		Name:     req.Name,
		VolumeID: "mock-vol-" + req.Name,
		State:    "available",
	}
	m.storage = append(m.storage, volume)
	return &volume, nil
}

func (m *MockClient) ListStorage(ctx context.Context) ([]types.EBSVolume, error) {
	return m.storage, nil
}

func (m *MockClient) GetStorage(ctx context.Context, name string) (*types.EBSVolume, error) {
	for _, volume := range m.storage {
		if volume.Name == name {
			return &volume, nil
		}
	}
	return nil, fmt.Errorf("storage not found: %s", name)
}

func (m *MockClient) DeleteStorage(ctx context.Context, name string) error {
	for i, volume := range m.storage {
		if volume.Name == name {
			m.storage = append(m.storage[:i], m.storage[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("storage not found: %s", name)
}

func (m *MockClient) AttachStorage(ctx context.Context, storageName, instanceName string) error {
	return nil // Mock implementation
}

func (m *MockClient) DetachStorage(ctx context.Context, storageName string) error {
	return nil // Mock implementation
}

// Status operations
func (m *MockClient) GetStatus(ctx context.Context) (*types.DaemonStatus, error) {
	return m.statusResult, nil
}

func (m *MockClient) Ping(ctx context.Context) error {
	return m.pingError
}

func (m *MockClient) Shutdown(ctx context.Context) error {
	return nil // Mock always succeeds
}

// Registry operations
func (m *MockClient) GetRegistryStatus(ctx context.Context) (*RegistryStatusResponse, error) {
	return &RegistryStatusResponse{
		Active:        true,
		TemplateCount: len(m.templates),
		Status:        "active",
	}, nil
}

func (m *MockClient) SetRegistryStatus(ctx context.Context, active bool) error {
	return nil // Mock implementation
}

func (m *MockClient) LookupAMI(ctx context.Context, templateName, region, architecture string) (*AMIReferenceResponse, error) {
	return &AMIReferenceResponse{
		AMIID:        "mock-ami-123",
		Region:       region,
		Architecture: architecture,
		TemplateName: templateName,
		Status:       "available",
	}, nil
}

func (m *MockClient) ListTemplateAMIs(ctx context.Context, templateName string) ([]AMIReferenceResponse, error) {
	return []AMIReferenceResponse{
		{
			AMIID:        "mock-ami-123",
			TemplateName: templateName,
			Status:       "available",
		},
	}, nil
}

// Mock control methods for testing

func (m *MockClient) SetPingError(err error) {
	m.pingError = err
}

func (m *MockClient) AddTemplate(name string, template types.Template) {
	m.templates[name] = template
}

func (m *MockClient) AddInstance(name string, instance *types.Instance) {
	m.instances[name] = instance
}
