// Package cli provides mock API client implementation for testing
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

// MockAPIClient implements the CloudWorkstationAPI interface for testing
type MockAPIClient struct {
	// Response configuration
	ShouldReturnError bool
	ErrorMessage      string
	PingError         error
	ConnectError      error  // Specific error for ConnectInstance method
	StopError         error  // Specific error for StopInstance method
	StartError        error  // Specific error for StartInstance method
	DeleteError       error  // Specific error for DeleteInstance method
	HibernateError    error  // Specific error for HibernateInstance method
	ResumeError       error  // Specific error for ResumeInstance method
	ListInstancesError error  // Specific error for ListInstances method
	HibernationStatusError error  // Specific error for GetInstanceHibernationStatus method
	LaunchError            error  // Specific error for LaunchInstance method

	// Mock data
	Instances         []types.Instance
	Templates         map[string]types.Template
	Volumes           []types.EFSVolume
	StorageVolumes    []types.EBSVolume
	Projects          []types.Project
	IdleProfiles      map[string]types.IdleProfile
	DaemonStatus      *types.DaemonStatus
	HibernationStatus *types.HibernationStatus

	// Call tracking
	LaunchCalls        []types.LaunchRequest
	StartCalls         []string
	StopCalls          []string
	DeleteCalls        []string
	HibernateCalls     []string
	ResumeCalls        []string
	ConnectCalls       []string
	GetInstanceCalls   []string
	CreateVolumeCalls  []types.VolumeCreateRequest
	CreateStorageCalls []types.StorageCreateRequest

	// Configuration
	Options client.Options
}

// NewMockAPIClient creates a new mock API client with default test data
func NewMockAPIClient() *MockAPIClient {
	return &MockAPIClient{
		Instances: []types.Instance{
			{
				ID:                "i-1234567890abcdef0",
				Name:              "test-instance",
				Template:          "python-ml",
				State:             "running",
				PublicIP:          "54.123.45.67",
				LaunchTime:        time.Now().Add(-1 * time.Hour),
				ProjectID:         "test-project",
				InstanceLifecycle: "on-demand",
			},
			{
				ID:                "i-0987654321fedcba0",
				Name:              "stopped-instance",
				Template:          "r-research",
				State:             "stopped",
				PublicIP:          "",
				LaunchTime:        time.Now().Add(-2 * time.Hour),
				InstanceLifecycle: "spot",
			},
		},
		Templates: map[string]types.Template{
			"python-ml": {
				Name:        "python-ml",
				Description: "Python Machine Learning environment",
			},
			"Python Machine Learning (Simplified)": {
				Name:        "Python Machine Learning (Simplified)",
				Description: "Simplified Python ML environment",
			},
			"r-research": {
				Name:        "r-research",
				Description: "R Research environment",
			},
			"Rocky Linux 9 + Conda Stack": {
				Name:        "Rocky Linux 9 + Conda Stack",
				Description: "Rocky Linux 9 with Conda stack",
			},
		},
		Volumes: []types.EFSVolume{
			{
				Name:         "test-volume",
				FileSystemId: "fs-1234567890abcdef0",
				State:        "available",
				CreationTime: time.Now().Add(-24 * time.Hour),
			},
		},
		StorageVolumes: []types.EBSVolume{
			{
				Name:         "test-storage",
				VolumeID:     "vol-1234567890abcdef0",
				State:        "available",
				SizeGB:       100,
				VolumeType:   "gp3",
				CreationTime: time.Now().Add(-24 * time.Hour),
			},
		},
		Projects: []types.Project{
			{
				ID:          "test-project",
				Name:        "Test Project",
				Description: "Test project for CLI testing",
				Status:      "active",
				CreatedAt:   time.Now().Add(-48 * time.Hour),
			},
		},
		IdleProfiles: map[string]types.IdleProfile{
			"batch": {
				Name:        "batch",
				IdleMinutes: 60,
				Action:      "hibernate",
			},
			"gpu": {
				Name:        "gpu",
				IdleMinutes: 15,
				Action:      "stop",
			},
		},
		DaemonStatus: &types.DaemonStatus{
			Status:  "running",
			Version: version.GetVersion(),
		},
		HibernationStatus: &types.HibernationStatus{
			HibernationSupported: true,
			IsHibernated:         false,
			InstanceName:         "test-instance",
		},
	}
}

// NewMockAPIClientWithError creates a mock client that returns errors for testing error paths
func NewMockAPIClientWithError(errorMsg string) *MockAPIClient {
	mock := NewMockAPIClient()
	mock.ShouldReturnError = true
	mock.ErrorMessage = errorMsg
	return mock
}

// NewMockAPIClientWithPingError creates a mock client that fails on Ping for daemon connection testing
func NewMockAPIClientWithPingError() *MockAPIClient {
	mock := NewMockAPIClient()
	mock.PingError = fmt.Errorf("daemon not running")
	return mock
}

// NewMockAPIClientWithConnectError creates a mock client that fails only on ConnectInstance calls
func NewMockAPIClientWithConnectError(errorMsg string) *MockAPIClient {
	mock := NewMockAPIClient()
	mock.ConnectError = fmt.Errorf("%s", errorMsg)
	return mock
}

// SetOptions sets the API client options
func (m *MockAPIClient) SetOptions(opts client.Options) {
	m.Options = opts
}

// Instance operations
func (m *MockAPIClient) LaunchInstance(ctx context.Context, req types.LaunchRequest) (*types.LaunchResponse, error) {
	m.LaunchCalls = append(m.LaunchCalls, req)

	// Check for specific launch error first
	if m.LaunchError != nil {
		return nil, m.LaunchError
	}

	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	instance := types.Instance{
		ID:         fmt.Sprintf("i-%d", time.Now().Unix()),
		Name:       req.Name,
		Template:   req.Template,
		State:      "running", // Set to running for immediate connection in tests
		PublicIP:   fmt.Sprintf("54.123.%d.%d", time.Now().Unix()%256, time.Now().Nanosecond()%256),
		LaunchTime: time.Now(),
		ProjectID:  req.ProjectID,
	}

	// Add to mock instances
	m.Instances = append(m.Instances, instance)

	return &types.LaunchResponse{
		Instance:       instance,
		Message:        fmt.Sprintf("Instance %s launched successfully", req.Name),
		EstimatedCost:  "$2.40/day",
		ConnectionInfo: fmt.Sprintf("cws connect %s", req.Name),
	}, nil
}

func (m *MockAPIClient) ListInstances(ctx context.Context) (*types.ListResponse, error) {
	// Check for specific list instances error first
	if m.ListInstancesError != nil {
		return nil, m.ListInstancesError
	}
	
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return &types.ListResponse{
		Instances: m.Instances,
	}, nil
}

func (m *MockAPIClient) GetInstance(ctx context.Context, name string) (*types.Instance, error) {
	m.GetInstanceCalls = append(m.GetInstanceCalls, name)

	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	for _, instance := range m.Instances {
		if instance.Name == name {
			return &instance, nil
		}
	}

	return nil, fmt.Errorf("instance %s not found", name)
}

func (m *MockAPIClient) StartInstance(ctx context.Context, name string) error {
	m.StartCalls = append(m.StartCalls, name)

	// Check for specific start error first
	if m.StartError != nil {
		return m.StartError
	}

	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	// Update instance state
	for i := range m.Instances {
		if m.Instances[i].Name == name {
			m.Instances[i].State = "pending"
			return nil
		}
	}

	return fmt.Errorf("instance %s not found", name)
}

func (m *MockAPIClient) StopInstance(ctx context.Context, name string) error {
	m.StopCalls = append(m.StopCalls, name)

	// Check for specific stop error first
	if m.StopError != nil {
		return m.StopError
	}

	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	// Update instance state
	for i := range m.Instances {
		if m.Instances[i].Name == name {
			m.Instances[i].State = "stopping"
			return nil
		}
	}

	return fmt.Errorf("instance %s not found", name)
}

func (m *MockAPIClient) HibernateInstance(ctx context.Context, name string) error {
	m.HibernateCalls = append(m.HibernateCalls, name)

	// Check for specific hibernate error first
	if m.HibernateError != nil {
		return m.HibernateError
	}

	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	// Update instance state
	for i := range m.Instances {
		if m.Instances[i].Name == name {
			m.Instances[i].State = "hibernated"
			return nil
		}
	}

	return fmt.Errorf("instance %s not found", name)
}

func (m *MockAPIClient) ResumeInstance(ctx context.Context, name string) error {
	m.ResumeCalls = append(m.ResumeCalls, name)

	// Check for specific resume error first
	if m.ResumeError != nil {
		return m.ResumeError
	}

	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	// Update instance state (resume from hibernation should go to running)
	for i := range m.Instances {
		if m.Instances[i].Name == name {
			m.Instances[i].State = "running"
			return nil
		}
	}

	return fmt.Errorf("instance %s not found", name)
}

func (m *MockAPIClient) GetInstanceHibernationStatus(ctx context.Context, name string) (*types.HibernationStatus, error) {
	// Check for specific hibernation status error first
	if m.HibernationStatusError != nil {
		return nil, m.HibernationStatusError
	}
	
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	status := *m.HibernationStatus
	status.InstanceName = name

	// Check if instance exists and is hibernated
	for _, instance := range m.Instances {
		if instance.Name == name {
			status.IsHibernated = instance.State == "hibernated"
			break
		}
	}

	return &status, nil
}

func (m *MockAPIClient) DeleteInstance(ctx context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	// Check for specific delete error first
	if m.DeleteError != nil {
		return m.DeleteError
	}

	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	// Remove instance from mock data
	for i, instance := range m.Instances {
		if instance.Name == name {
			m.Instances = append(m.Instances[:i], m.Instances[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("instance %s not found", name)
}

func (m *MockAPIClient) ConnectInstance(ctx context.Context, name string) (string, error) {
	m.ConnectCalls = append(m.ConnectCalls, name)

	// Check for specific connect error first
	if m.ConnectError != nil {
		return "", m.ConnectError
	}

	if m.ShouldReturnError {
		return "", fmt.Errorf("%s", m.ErrorMessage)
	}

	// Find instance and return SSH command
	for i, instance := range m.Instances {
		if instance.Name == name {
			// For test-instance, ensure it's running for connection tests
			if instance.Name == "test-instance" && instance.State != "running" {
				m.Instances[i].State = "running"
				instance.State = "running"
			}
			if instance.State != "running" {
				return "", fmt.Errorf("instance %s is not running", name)
			}
			return fmt.Sprintf("ssh user@%s", instance.PublicIP), nil
		}
	}

	return "", fmt.Errorf("instance %s not found", name)
}

// Template operations
func (m *MockAPIClient) ListTemplates(ctx context.Context) (map[string]types.Template, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return m.Templates, nil
}

func (m *MockAPIClient) GetTemplate(ctx context.Context, name string) (*types.Template, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	if template, exists := m.Templates[name]; exists {
		return &template, nil
	}

	return nil, fmt.Errorf("template %s not found", name)
}

// Template application operations
func (m *MockAPIClient) ApplyTemplate(ctx context.Context, req templates.ApplyRequest) (*templates.ApplyResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return &templates.ApplyResponse{
		Success: true,
		Message: "Template applied successfully",
	}, nil
}

func (m *MockAPIClient) DiffTemplate(ctx context.Context, req templates.DiffRequest) (*templates.TemplateDiff, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return &templates.TemplateDiff{
		PackagesToInstall: []templates.PackageDiff{
			{Name: "test-package", TargetVersion: "latest"},
		},
	}, nil
}

func (m *MockAPIClient) GetInstanceLayers(ctx context.Context, name string) ([]templates.AppliedTemplate, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return []templates.AppliedTemplate{
		{
			Name:      "base-template",
			AppliedAt: time.Now().Add(-1 * time.Hour),
		},
	}, nil
}

func (m *MockAPIClient) RollbackInstance(ctx context.Context, req types.RollbackRequest) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	return nil
}

// Idle detection operations
func (m *MockAPIClient) GetIdleStatus(ctx context.Context) (*types.IdleStatusResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return &types.IdleStatusResponse{
		Enabled:  true,
		Profiles: m.IdleProfiles,
	}, nil
}

func (m *MockAPIClient) EnableIdleDetection(ctx context.Context) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) DisableIdleDetection(ctx context.Context) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) GetIdleProfiles(ctx context.Context) (map[string]types.IdleProfile, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return m.IdleProfiles, nil
}

func (m *MockAPIClient) AddIdleProfile(ctx context.Context, profile types.IdleProfile) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	m.IdleProfiles[profile.Name] = profile
	return nil
}

func (m *MockAPIClient) GetIdlePendingActions(ctx context.Context) ([]types.IdleState, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []types.IdleState{
		{
			InstanceName: "test-instance",
			IsIdle:       true,
		},
	}, nil
}

func (m *MockAPIClient) ExecuteIdleActions(ctx context.Context) (*types.IdleExecutionResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.IdleExecutionResponse{
		Executed: 1,
		Errors:   []string{},
		Total:    1,
	}, nil
}

func (m *MockAPIClient) GetIdleHistory(ctx context.Context) ([]types.IdleHistoryEntry, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []types.IdleHistoryEntry{
		{
			InstanceName: "test-instance",
			Action:       "hibernate",
			Time:         time.Now().Add(-1 * time.Hour),
		},
	}, nil
}

// Volume operations (EFS)
func (m *MockAPIClient) CreateVolume(ctx context.Context, req types.VolumeCreateRequest) (*types.EFSVolume, error) {
	m.CreateVolumeCalls = append(m.CreateVolumeCalls, req)

	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	volume := types.EFSVolume{
		Name:         req.Name,
		FileSystemId: fmt.Sprintf("fs-%d", time.Now().Unix()),
		State:        "creating",
		CreationTime: time.Now(),
	}

	m.Volumes = append(m.Volumes, volume)
	return &volume, nil
}

func (m *MockAPIClient) ListVolumes(ctx context.Context) ([]types.EFSVolume, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return m.Volumes, nil
}

func (m *MockAPIClient) GetVolume(ctx context.Context, name string) (*types.EFSVolume, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	for _, volume := range m.Volumes {
		if volume.Name == name {
			return &volume, nil
		}
	}

	return nil, fmt.Errorf("volume %s not found", name)
}

func (m *MockAPIClient) DeleteVolume(ctx context.Context, name string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	for i, volume := range m.Volumes {
		if volume.Name == name {
			m.Volumes = append(m.Volumes[:i], m.Volumes[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("volume %s not found", name)
}

func (m *MockAPIClient) AttachVolume(ctx context.Context, volumeName, instanceName string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) DetachVolume(ctx context.Context, volumeName string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// Storage operations (EBS)
func (m *MockAPIClient) CreateStorage(ctx context.Context, req types.StorageCreateRequest) (*types.EBSVolume, error) {
	m.CreateStorageCalls = append(m.CreateStorageCalls, req)

	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	// Parse size if it's a preset size (XS, S, M, L, XL)
	sizeGB := int32(100) // default
	switch req.Size {
	case "XS":
		sizeGB = 50
	case "S":
		sizeGB = 100
	case "M":
		sizeGB = 250
	case "L":
		sizeGB = 500
	case "XL":
		sizeGB = 1000
	}

	volume := types.EBSVolume{
		Name:         req.Name,
		VolumeID:     fmt.Sprintf("vol-%d", time.Now().Unix()),
		State:        "creating",
		SizeGB:       sizeGB,
		VolumeType:   req.VolumeType,
		CreationTime: time.Now(),
	}

	m.StorageVolumes = append(m.StorageVolumes, volume)
	return &volume, nil
}

func (m *MockAPIClient) ListStorage(ctx context.Context) ([]types.EBSVolume, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return m.StorageVolumes, nil
}

func (m *MockAPIClient) GetStorage(ctx context.Context, name string) (*types.EBSVolume, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	for _, volume := range m.StorageVolumes {
		if volume.Name == name {
			return &volume, nil
		}
	}

	return nil, fmt.Errorf("storage %s not found", name)
}

func (m *MockAPIClient) DeleteStorage(ctx context.Context, name string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	for i, volume := range m.StorageVolumes {
		if volume.Name == name {
			m.StorageVolumes = append(m.StorageVolumes[:i], m.StorageVolumes[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("storage %s not found", name)
}

func (m *MockAPIClient) AttachStorage(ctx context.Context, volumeName, instanceName string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) DetachStorage(ctx context.Context, volumeName string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// Project management operations
func (m *MockAPIClient) CreateProject(ctx context.Context, req project.CreateProjectRequest) (*types.Project, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	proj := types.Project{
		ID:          fmt.Sprintf("proj-%d", time.Now().Unix()),
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   time.Now(),
	}

	m.Projects = append(m.Projects, proj)
	return &proj, nil
}

func (m *MockAPIClient) ListProjects(ctx context.Context, filter *project.ProjectFilter) (*project.ProjectListResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	// Convert Project to ProjectSummary
	summaries := make([]project.ProjectSummary, len(m.Projects))
	for i, proj := range m.Projects {
		summaries[i] = project.ProjectSummary{
			ID:        proj.ID,
			Name:      proj.Name,
			Owner:     proj.Owner,
			Status:    proj.Status,
			CreatedAt: proj.CreatedAt,
		}
	}

	return &project.ProjectListResponse{
		Projects:   summaries,
		TotalCount: len(m.Projects),
	}, nil
}

func (m *MockAPIClient) GetProject(ctx context.Context, id string) (*types.Project, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	for _, proj := range m.Projects {
		if proj.ID == id {
			return &proj, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", id)
}

func (m *MockAPIClient) UpdateProject(ctx context.Context, id string, req project.UpdateProjectRequest) (*types.Project, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	for i, proj := range m.Projects {
		if proj.ID == id {
			if req.Name != nil {
				proj.Name = *req.Name
			}
			if req.Description != nil {
				proj.Description = *req.Description
			}
			m.Projects[i] = proj
			return &proj, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", id)
}

func (m *MockAPIClient) DeleteProject(ctx context.Context, id string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	for i, proj := range m.Projects {
		if proj.ID == id {
			m.Projects = append(m.Projects[:i], m.Projects[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("project %s not found", id)
}

func (m *MockAPIClient) AddProjectMember(ctx context.Context, projectID string, req project.AddMemberRequest) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) UpdateProjectMember(ctx context.Context, projectID, userID string, req project.UpdateMemberRequest) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) GetProjectMembers(ctx context.Context, projectID string) ([]types.ProjectMember, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return []types.ProjectMember{
		{
			UserID:  "user-123",
			Role:    "owner",
			AddedAt: time.Now().Add(-24 * time.Hour),
		},
	}, nil
}

func (m *MockAPIClient) GetProjectBudgetStatus(ctx context.Context, projectID string) (*project.BudgetStatus, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return &project.BudgetStatus{
		TotalBudget: 1000.0,
		SpentAmount: 250.0,
	}, nil
}

func (m *MockAPIClient) GetProjectCostBreakdown(ctx context.Context, projectID string, startTime, endTime time.Time) (*types.ProjectCostBreakdown, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return &types.ProjectCostBreakdown{
		ProjectID:   projectID,
		TotalCost:   100.0,
		PeriodStart: startTime,
		PeriodEnd:   endTime,
		GeneratedAt: time.Now(),
	}, nil
}

func (m *MockAPIClient) GetProjectResourceUsage(ctx context.Context, projectID string, duration time.Duration) (*types.ProjectResourceUsage, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return &types.ProjectResourceUsage{
		ProjectID:       projectID,
		ActiveInstances: 2,
		TotalStorage:    100.0,
	}, nil
}

// Status operations
func (m *MockAPIClient) GetStatus(ctx context.Context) (*types.DaemonStatus, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return m.DaemonStatus, nil
}

func (m *MockAPIClient) Ping(ctx context.Context) error {
	if m.PingError != nil {
		return m.PingError
	}
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) Shutdown(ctx context.Context) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// Raw API request method for generic endpoint access
func (m *MockAPIClient) MakeRequest(method, path string, body interface{}) ([]byte, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []byte(`{"success": true}`), nil
}

// Registry operations
func (m *MockAPIClient) GetRegistryStatus(ctx context.Context) (*client.RegistryStatusResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	lastSync := time.Now().Add(-1 * time.Hour)
	return &client.RegistryStatusResponse{
		Active:        true,
		LastSync:      &lastSync,
		TemplateCount: len(m.Templates),
		AMICount:      5,
		Status:        "healthy",
	}, nil
}

func (m *MockAPIClient) SetRegistryStatus(ctx context.Context, active bool) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) LookupAMI(ctx context.Context, templateName, region, architecture string) (*client.AMIReferenceResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return &client.AMIReferenceResponse{
		AMIID:        "ami-12345678",
		Region:       region,
		Architecture: architecture,
		TemplateName: templateName,
		Version:      "1.0.0",
		BuildDate:    time.Now().Add(-24 * time.Hour),
		Status:       "available",
	}, nil
}

func (m *MockAPIClient) ListTemplateAMIs(ctx context.Context, templateName string) ([]client.AMIReferenceResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	return []client.AMIReferenceResponse{
		{
			AMIID:        "ami-12345678",
			Region:       "us-east-1",
			Architecture: "x86_64",
			TemplateName: templateName,
			Version:      "1.0.0",
			BuildDate:    time.Now().Add(-24 * time.Hour),
			Status:       "available",
		},
	}, nil
}

// Helper methods for test verification

// ResetCallTracking clears all call tracking arrays
func (m *MockAPIClient) ResetCallTracking() {
	m.LaunchCalls = nil
	m.StartCalls = nil
	m.StopCalls = nil
	m.DeleteCalls = nil
	m.HibernateCalls = nil
	m.ResumeCalls = nil
	m.ConnectCalls = nil
	m.GetInstanceCalls = nil
	m.CreateVolumeCalls = nil
	m.CreateStorageCalls = nil
}

// GetCallCount returns the total number of API calls made
func (m *MockAPIClient) GetCallCount() int {
	return len(m.LaunchCalls) + len(m.StartCalls) + len(m.StopCalls) +
		len(m.DeleteCalls) + len(m.HibernateCalls) + len(m.ResumeCalls) +
		len(m.ConnectCalls) + len(m.GetInstanceCalls) +
		len(m.CreateVolumeCalls) + len(m.CreateStorageCalls)
}
