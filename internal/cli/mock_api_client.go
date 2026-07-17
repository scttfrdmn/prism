// Package cli provides mock API client implementation for testing
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/idle"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/storage"
	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/throttle"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/pkg/version"
)

// MockAPIClient implements the PrismAPI interface for testing
type MockAPIClient struct {
	// Response configuration
	ShouldReturnError      bool
	ErrorMessage           string
	PingError              error
	ConnectError           error // Specific error for ConnectInstance method
	StopError              error // Specific error for StopInstance method
	StartError             error // Specific error for StartInstance method
	DeleteError            error // Specific error for DeleteInstance method
	HibernateError         error // Specific error for HibernateInstance method
	ResumeError            error // Specific error for ResumeInstance method
	ListInstancesError     error // Specific error for ListInstances method
	HibernationStatusError error // Specific error for GetInstanceHibernationStatus method
	LaunchError            error // Specific error for LaunchInstance method

	// Mock data
	Instances      []types.Instance
	Templates      map[string]types.Template
	StorageVolumes []types.StorageVolume // Unified storage (EFS, EBS, S3)
	Projects       []types.Project
	Approvals      []*project.ApprovalRequest // Configurable list for ListApprovals/ListAllApprovals
	// Legacy idle fields removed - using new hibernation policy system
	DaemonStatus      *types.DaemonStatus
	HibernationStatus *types.HibernationStatus

	// Call tracking
	LaunchCalls        []types.LaunchRequest
	LaunchArrayCalls   []types.LaunchArrayRequest
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

// Helper function to create int32 pointer
func intPtr(i int32) *int32 {
	return &i
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
				AMI: map[string]map[string]string{
					"us-west-2": {
						"x86_64": "ami-test123456", // Mock AMI to prevent progress monitoring
					},
				},
			},
			"Python Machine Learning (Simplified)": {
				Name:        "Python Machine Learning (Simplified)",
				Description: "Simplified Python ML environment",
				AMI: map[string]map[string]string{
					"us-west-2": {
						"x86_64": "ami-test234567", // Mock AMI to prevent progress monitoring
					},
				},
			},
			"r-research": {
				Name:        "r-research",
				Description: "R Research environment",
				AMI: map[string]map[string]string{
					"us-west-2": {
						"x86_64": "ami-test345678", // Mock AMI to prevent progress monitoring
					},
				},
			},
			"Rocky Linux 9 + Conda Stack": {
				Name:        "Rocky Linux 9 + Conda Stack",
				Description: "Rocky Linux 9 with Conda stack",
				AMI: map[string]map[string]string{
					"us-west-2": {
						"x86_64": "ami-test456789", // Mock AMI to prevent progress monitoring
					},
				},
			},
		},
		StorageVolumes: []types.StorageVolume{
			// EFS (Shared) volume
			{
				Name:         "test-volume",
				Type:         types.StorageTypeShared,
				AWSService:   types.AWSServiceEFS,
				FileSystemID: "fs-1234567890abcdef0",
				State:        "available",
				CreationTime: time.Now().Add(-24 * time.Hour),
			},
			// EBS (Workspace) volume
			{
				Name:         "test-storage",
				Type:         types.StorageTypeWorkspace,
				AWSService:   types.AWSServiceEBS,
				VolumeID:     "vol-1234567890abcdef0",
				State:        "available",
				SizeGB:       intPtr(100),
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
		ConnectionInfo: fmt.Sprintf("prism workspace connect %s", req.Name),
	}, nil
}

func (m *MockAPIClient) LaunchArray(ctx context.Context, req types.LaunchArrayRequest) (*types.LaunchArrayResponse, error) {
	m.LaunchArrayCalls = append(m.LaunchArrayCalls, req)

	if m.LaunchError != nil {
		return nil, m.LaunchError
	}
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	resp := &types.LaunchArrayResponse{
		JobArrayID: fmt.Sprintf("%s-mock", req.Name),
		Requested:  req.Count,
	}
	for i := 0; i < req.Count; i++ {
		inst := types.Instance{
			ID:            fmt.Sprintf("i-%d-%d", time.Now().Unix(), i),
			Name:          fmt.Sprintf("%s-%d", req.Name, i),
			Template:      req.Template,
			State:         "running",
			LaunchTime:    time.Now(),
			ProjectID:     req.ProjectID,
			JobArrayID:    resp.JobArrayID,
			JobArrayIndex: i,
		}
		m.Instances = append(m.Instances, inst)
		resp.Instances = append(resp.Instances, inst)
		resp.Launched++
	}
	return resp, nil
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

// GetBilledCost returns mock AWS-billed cost for an instance.
func (m *MockAPIClient) GetBilledCost(ctx context.Context, name string) (*types.BilledCostResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	for _, instance := range m.Instances {
		if instance.Name == name {
			return &types.BilledCostResult{
				Name:        name,
				BilledTotal: instance.CurrentSpend,
				Currency:    "USD",
				Region:      instance.Region,
				Source:      "mock",
				TagActive:   true,
			}, nil
		}
	}

	return nil, fmt.Errorf("instance %s not found", name)
}

// GetProgress returns mock progress for an instance (v0.7.2 - Issue #453)
func (m *MockAPIClient) GetProgress(ctx context.Context, name string) (*types.ProgressResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	// Return mock progress response
	return &types.ProgressResponse{
		InstanceName:      name,
		OverallProgress:   100.0,
		CurrentStage:      "Complete",
		CurrentStageIndex: 3,
		IsComplete:        true,
		EstimatedTimeLeft: "0 seconds",
		Stages: []types.ProgressStage{
			{Name: "Launching", DisplayName: "Launching instance", Status: "complete"},
			{Name: "Cloud-init", DisplayName: "Running cloud-init", Status: "complete"},
			{Name: "Ready", DisplayName: "Ready", Status: "complete"},
		},
	}, nil
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
			status.PossiblyHibernated = instance.State == "hibernated"
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

// Legacy idle detection operations removed - using new hibernation policy system

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
func (m *MockAPIClient) CreateVolume(ctx context.Context, req types.VolumeCreateRequest) (*types.StorageVolume, error) {
	m.CreateVolumeCalls = append(m.CreateVolumeCalls, req)

	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	sizeBytes := int64(0)
	volume := &types.StorageVolume{
		Name:            req.Name,
		Type:            types.StorageTypeShared,
		AWSService:      types.AWSServiceEFS,
		FileSystemID:    fmt.Sprintf("fs-%d", time.Now().Unix()),
		State:           "creating",
		CreationTime:    time.Now(),
		SizeBytes:       &sizeBytes,
		PerformanceMode: req.PerformanceMode,
		ThroughputMode:  req.ThroughputMode,
		MountTargets:    []string{},
	}

	return volume, nil
}

func (m *MockAPIClient) ListVolumes(ctx context.Context) ([]*types.StorageVolume, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	// Filter for shared storage (EFS) volumes
	result := make([]*types.StorageVolume, 0)
	for i := range m.StorageVolumes {
		if m.StorageVolumes[i].IsShared() {
			result = append(result, &m.StorageVolumes[i])
		}
	}
	return result, nil
}

func (m *MockAPIClient) GetVolume(ctx context.Context, name string) (*types.StorageVolume, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	for i := range m.StorageVolumes {
		if m.StorageVolumes[i].Name == name && m.StorageVolumes[i].IsShared() {
			return &m.StorageVolumes[i], nil
		}
	}

	return nil, fmt.Errorf("volume %s not found", name)
}

func (m *MockAPIClient) DeleteVolume(ctx context.Context, name string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}

	for i, volume := range m.StorageVolumes {
		if volume.Name == name && volume.IsShared() {
			m.StorageVolumes = append(m.StorageVolumes[:i], m.StorageVolumes[i+1:]...)
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
func (m *MockAPIClient) CreateStorage(ctx context.Context, req types.StorageCreateRequest) (*types.StorageVolume, error) {
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

	iops := int32(3000)
	throughput := int32(125)

	volume := &types.StorageVolume{
		Name:         req.Name,
		Type:         types.StorageTypeWorkspace,
		AWSService:   types.AWSServiceEBS,
		VolumeID:     fmt.Sprintf("vol-%d", time.Now().Unix()),
		State:        "creating",
		SizeGB:       &sizeGB,
		VolumeType:   req.VolumeType,
		IOPS:         &iops,
		Throughput:   &throughput,
		CreationTime: time.Now(),
	}

	return volume, nil
}

func (m *MockAPIClient) ListStorage(ctx context.Context) ([]*types.StorageVolume, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	// Filter for workspace storage (EBS) volumes
	result := make([]*types.StorageVolume, 0)
	for i := range m.StorageVolumes {
		if m.StorageVolumes[i].IsWorkspace() {
			result = append(result, &m.StorageVolumes[i])
		}
	}
	return result, nil
}

func (m *MockAPIClient) GetStorage(ctx context.Context, name string) (*types.StorageVolume, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	for i := range m.StorageVolumes {
		if m.StorageVolumes[i].Name == name && m.StorageVolumes[i].IsWorkspace() {
			return &m.StorageVolumes[i], nil
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

func (m *MockAPIClient) GetProjectBudgetHistory(ctx context.Context, projectID string, days int) ([]float64, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []float64{}, nil
}

func (m *MockAPIClient) GetCostTrends(ctx context.Context, projectID, period string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}

	// Generate mock cost trend data based on period
	dailyData := []map[string]interface{}{
		{"date": "2025-10-01", "cost": 45.50, "instances": 3},
		{"date": "2025-10-02", "cost": 52.30, "instances": 4},
		{"date": "2025-10-03", "cost": 48.75, "instances": 3},
		{"date": "2025-10-04", "cost": 51.20, "instances": 4},
		{"date": "2025-10-05", "cost": 44.80, "instances": 3},
		{"date": "2025-10-06", "cost": 47.90, "instances": 3},
		{"date": "2025-10-07", "cost": 53.40, "instances": 4},
	}

	weeklyData := []map[string]interface{}{
		{"week": "Week 1", "cost": 320.50, "instances": 3},
		{"week": "Week 2", "cost": 355.30, "instances": 4},
		{"week": "Week 3", "cost": 298.75, "instances": 3},
		{"week": "Week 4", "cost": 340.20, "instances": 4},
	}

	monthlyData := []map[string]interface{}{
		{"month": "July 2025", "cost": 1250.00, "instances": 3},
		{"month": "August 2025", "cost": 1420.50, "instances": 4},
		{"month": "September 2025", "cost": 1380.75, "instances": 3},
		{"month": "October 2025", "cost": 1510.30, "instances": 4},
	}

	var trendsData []map[string]interface{}
	switch period {
	case "daily":
		trendsData = dailyData
	case "weekly":
		trendsData = weeklyData
	case "monthly":
		trendsData = monthlyData
	default:
		trendsData = dailyData
	}

	return map[string]interface{}{
		"project_id":     projectID,
		"period":         period,
		"trends":         trendsData,
		"total_cost":     1510.30,
		"average_cost":   377.58,
		"trend":          "increasing",
		"percent_change": 3.2,
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

// Idle policy operations

func (m *MockAPIClient) ListIdlePolicies(ctx context.Context) ([]*idle.PolicyTemplate, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	// Return empty list for mock
	return []*idle.PolicyTemplate{}, nil
}

func (m *MockAPIClient) GetIdlePolicy(ctx context.Context, policyID string) (*idle.PolicyTemplate, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &idle.PolicyTemplate{
		ID:   policyID,
		Name: "Test Policy",
	}, nil
}

func (m *MockAPIClient) ApplyIdlePolicy(ctx context.Context, instanceName, policyID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) RemoveIdlePolicy(ctx context.Context, instanceName, policyID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) GetInstanceIdlePolicies(ctx context.Context, instanceName string) ([]*idle.PolicyTemplate, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []*idle.PolicyTemplate{}, nil
}

func (m *MockAPIClient) RecommendIdlePolicy(ctx context.Context, instanceName string) (*idle.PolicyTemplate, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &idle.PolicyTemplate{
		ID:   "balanced",
		Name: "Balanced",
	}, nil
}

func (m *MockAPIClient) GetIdleSavingsReport(ctx context.Context, period string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"total_saved": 100.0,
	}, nil
}

// Template Marketplace operations - Mock implementations

func (m *MockAPIClient) SearchMarketplace(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"templates":     []map[string]interface{}{},
		"total_results": 0,
		"query":         params,
	}, nil
}

func (m *MockAPIClient) GetMarketplaceTemplate(ctx context.Context, templateID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"id":           templateID,
		"name":         "Mock Template",
		"description":  "Mock marketplace template",
		"category":     "Testing",
		"author":       "mock-user",
		"downloads":    0,
		"rating":       5.0,
		"last_updated": "2024-01-01",
		"verified":     false,
	}, nil
}

func (m *MockAPIClient) PublishMarketplaceTemplate(ctx context.Context, template map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success":     true,
		"template_id": "mock-template-123",
		"message":     "Template published successfully (mock)",
		"status":      "pending_review",
	}, nil
}

func (m *MockAPIClient) AddMarketplaceReview(ctx context.Context, templateID string, review map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success":   true,
		"review_id": "mock-review-123",
		"message":   "Review added successfully (mock)",
		"rating":    review["rating"],
	}, nil
}

func (m *MockAPIClient) ForkMarketplaceTemplate(ctx context.Context, templateID string, options map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success":           true,
		"forked_template":   "mock-forked-template-456",
		"message":           "Template forked successfully (mock)",
		"original_template": templateID,
	}, nil
}

func (m *MockAPIClient) GetMarketplaceFeatured(ctx context.Context) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"featured_templates": []map[string]interface{}{},
		"total_count":        0,
	}, nil
}

func (m *MockAPIClient) GetMarketplaceTrending(ctx context.Context) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"trending_templates": []map[string]interface{}{},
		"total_count":        0,
	}, nil
}

// Policy management operations - Mock implementations

func (m *MockAPIClient) GetPolicyStatus(ctx context.Context) (*client.PolicyStatusResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.PolicyStatusResponse{
		Enabled:          true,
		Status:           "active",
		StatusIcon:       "✅",
		AssignedPolicies: []string{"default"},
		Message:          "Policy enforcement active (mock)",
	}, nil
}

func (m *MockAPIClient) ListPolicySets(ctx context.Context) (*client.PolicySetsResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.PolicySetsResponse{
		PolicySets: map[string]client.PolicySetInfo{
			"default": {
				ID:          "default",
				Name:        "Default Policy Set",
				Description: "Mock default policy set",
				Policies:    5,
				Status:      "active",
			},
		},
	}, nil
}

func (m *MockAPIClient) AssignPolicySet(ctx context.Context, policySet string) (*client.PolicyAssignResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.PolicyAssignResponse{
		Success:           true,
		Message:           "Policy set assigned successfully (mock)",
		AssignedPolicySet: policySet,
		EnforcementStatus: "active",
	}, nil
}

func (m *MockAPIClient) SetPolicyEnforcement(ctx context.Context, enabled bool) (*client.PolicyEnforcementResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	return &client.PolicyEnforcementResponse{
		Success: true,
		Message: fmt.Sprintf("Policy enforcement %s (mock)", status),
		Enabled: enabled,
		Status:  status,
	}, nil
}

func (m *MockAPIClient) CheckTemplateAccess(ctx context.Context, templateName string) (*client.PolicyCheckResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.PolicyCheckResponse{
		Allowed:         true,
		TemplateName:    templateName,
		Reason:          "Template access allowed (mock)",
		MatchedPolicies: []string{"default"},
		Suggestions:     []string{},
	}, nil
}

// Universal AMI System operations - Mock implementations

func (m *MockAPIClient) ResolveAMI(ctx context.Context, templateName string, params map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"template_name":                templateName,
		"target_region":                "us-east-1",
		"resolution_method":            "fallback_script",
		"ami_id":                       "",
		"launch_time_estimate_seconds": 355,
		"cost_savings":                 0.0,
		"warning":                      "No AMI configuration found, using script provisioning (mock)",
	}, nil
}

func (m *MockAPIClient) TestAMIAvailability(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	templateName := "mock-template"
	if name, ok := request["template_name"].(string); ok {
		templateName = name
	}
	return map[string]interface{}{
		"template_name":     templateName,
		"overall_status":    "passed",
		"tested_at":         time.Now(),
		"total_regions":     4,
		"available_regions": 4,
		"region_results": map[string]interface{}{
			"us-east-1":  map[string]interface{}{"status": "passed"},
			"us-west-2":  map[string]interface{}{"status": "passed"},
			"eu-west-1":  map[string]interface{}{"status": "passed"},
			"ap-south-1": map[string]interface{}{"status": "passed"},
		},
	}, nil
}

func (m *MockAPIClient) GetAMICosts(ctx context.Context, templateName string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"template_name":      templateName,
		"region":             "us-east-1",
		"recommendation":     "neutral",
		"reasoning":          "Both AMI and script provisioning have similar cost/benefit profiles (mock)",
		"ami_launch_cost":    0.0336,
		"ami_storage_cost":   0.8000,
		"ami_setup_cost":     0.0003,
		"script_launch_cost": 0.0336,
		"script_setup_cost":  0.0033,
		"script_setup_time":  5,
		"break_even_point":   2.7,
		"cost_savings_1h":    0.0000,
		"cost_savings_8h":    0.0000,
		"time_savings":       5,
	}, nil
}

func (m *MockAPIClient) PreviewAMIResolution(ctx context.Context, templateName string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"template_name":                templateName,
		"target_region":                "us-east-1",
		"resolution_method":            "fallback_script",
		"launch_time_estimate_seconds": 355,
		"fallback_chain":               []string{"no_ami_config", "script_fallback"},
		"warning":                      "No AMI available, would use script provisioning (mock)",
	}, nil
}

// Rightsizing operations - Mock implementations

func (m *MockAPIClient) AnalyzeRightsizing(ctx context.Context, req types.RightsizingAnalysisRequest) (*types.RightsizingAnalysisResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.RightsizingAnalysisResponse{
		Recommendation: &types.RightsizingRecommendation{
			InstanceName:            req.InstanceName,
			CurrentInstanceType:     "t3.medium",
			RecommendedInstanceType: "t3.small",
			Reasoning:               "Instance underutilized (mock)",
			CostImpact: types.CostImpact{
				CurrentDailyCost:     1.50,
				RecommendedDailyCost: 0.98,
				DailyDifference:      0.52,
				PercentageChange:     34.4,
				MonthlySavings:       15.50,
				AnnualSavings:        186.00,
				IsIncrease:           false,
			},
			CreatedAt:           time.Now(),
			DataPointsAnalyzed:  100,
			AnalysisPeriodHours: 24.0,
		},
		MetricsAvailable:    true,
		DataPointsCount:     100,
		AnalysisPeriodHours: 24.0,
		LastUpdated:         time.Now(),
	}, nil
}

func (m *MockAPIClient) GetRightsizingRecommendations(ctx context.Context) (*types.RightsizingRecommendationsResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.RightsizingRecommendationsResponse{
		Recommendations:  []types.RightsizingRecommendation{},
		TotalInstances:   0,
		ActiveInstances:  0,
		PotentialSavings: 0.0,
		GeneratedAt:      time.Now(),
	}, nil
}

func (m *MockAPIClient) GetRightsizingStats(ctx context.Context, instanceName string) (*types.RightsizingStatsResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.RightsizingStatsResponse{
		InstanceName: instanceName,
		CurrentConfiguration: types.InstanceConfiguration{
			InstanceType: "t3.medium",
			VCPUs:        2,
			MemoryGB:     4.0,
		},
		MetricsSummary:   types.MetricsSummary{},
		RecentMetrics:    []types.InstanceMetrics{},
		CollectionStatus: types.MetricsCollectionStatus{},
	}, nil
}

func (m *MockAPIClient) ExportRightsizingData(ctx context.Context, instanceName string) ([]types.InstanceMetrics, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []types.InstanceMetrics{}, nil
}

func (m *MockAPIClient) GetRightsizingSummary(ctx context.Context) (*types.RightsizingSummaryResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.RightsizingSummaryResponse{
		FleetOverview: types.FleetOverview{
			TotalInstances:       2,
			RunningInstances:     1,
			InstancesWithMetrics: 2,
		},
		CostOptimization: types.CostOptimizationSummary{
			PotentialMonthlySavings: 25.00,
		},
		ResourceUtilization: types.ResourceUtilizationSummary{},
		Recommendations:     types.RecommendationsSummary{},
		GeneratedAt:         time.Now(),
	}, nil
}

func (m *MockAPIClient) GetInstanceMetrics(ctx context.Context, instanceName string, limit int) ([]types.InstanceMetrics, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []types.InstanceMetrics{}, nil
}

// Instance execution and logs

func (m *MockAPIClient) ExecInstance(ctx context.Context, instanceName string, req types.ExecRequest) (*types.ExecResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.ExecResult{
		ExitCode: 0,
		StdOut:   "mock output",
		StdErr:   "",
		Status:   "success",
	}, nil
}

func (m *MockAPIClient) ResizeInstance(ctx context.Context, req types.ResizeRequest) (*types.ResizeResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.ResizeResponse{
		Success: true,
		Message: "Instance resized successfully (mock)",
	}, nil
}

func (m *MockAPIClient) GetInstanceLogs(ctx context.Context, instanceName string, req types.LogRequest) (*types.LogResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.LogResponse{
		InstanceName: instanceName,
		LogType:      req.LogType,
		Lines:        []string{"mock log line 1", "mock log line 2"},
	}, nil
}

func (m *MockAPIClient) GetInstanceLogTypes(ctx context.Context, instanceName string) (*types.LogTypesResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.LogTypesResponse{
		InstanceName:      instanceName,
		AvailableLogTypes: []string{"system", "user-data", "cloud-init"},
		SSMEnabled:        true,
	}, nil
}

func (m *MockAPIClient) GetLogsSummary(ctx context.Context) (*types.LogSummaryResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.LogSummaryResponse{
		Instances:         []types.InstanceLogSummary{},
		AvailableLogTypes: []string{"system", "user-data", "cloud-init"},
	}, nil
}

// Project budget operations

func (m *MockAPIClient) SetProjectBudget(ctx context.Context, projectID string, req client.SetProjectBudgetRequest) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success": true,
		"message": "Budget set successfully (mock)",
	}, nil
}

func (m *MockAPIClient) UpdateProjectBudget(ctx context.Context, projectID string, req client.UpdateProjectBudgetRequest) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success": true,
		"message": "Budget updated successfully (mock)",
	}, nil
}

func (m *MockAPIClient) DisableProjectBudget(ctx context.Context, projectID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success": true,
		"message": "Budget disabled successfully (mock)",
	}, nil
}

// Snapshot operations

func (m *MockAPIClient) CreateInstanceSnapshot(ctx context.Context, req types.InstanceSnapshotRequest) (*types.InstanceSnapshotResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.InstanceSnapshotResult{
		SnapshotID:                 "snap-mock123",
		SnapshotName:               req.SnapshotName,
		SourceInstance:             req.InstanceName,
		SourceInstanceId:           "i-mock123",
		Description:                req.Description,
		State:                      "creating",
		EstimatedCompletionMinutes: 10,
		StorageCostMonthly:         5.00,
		CreatedAt:                  time.Now(),
		NoReboot:                   req.NoReboot,
	}, nil
}

func (m *MockAPIClient) ListInstanceSnapshots(ctx context.Context) (*types.InstanceSnapshotListResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.InstanceSnapshotListResponse{
		Snapshots: []types.InstanceSnapshotInfo{},
		Count:     0,
	}, nil
}

func (m *MockAPIClient) GetInstanceSnapshot(ctx context.Context, snapshotID string) (*types.InstanceSnapshotInfo, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.InstanceSnapshotInfo{
		SnapshotID:         snapshotID,
		SnapshotName:       "mock-snapshot",
		SourceInstance:     "test-instance",
		SourceInstanceId:   "i-mock123",
		SourceTemplate:     "python-ml",
		Description:        "Mock snapshot",
		State:              "available",
		Architecture:       "x86_64",
		StorageCostMonthly: 5.00,
		CreatedAt:          time.Now(),
	}, nil
}

func (m *MockAPIClient) DeleteInstanceSnapshot(ctx context.Context, snapshotID string) (*types.InstanceSnapshotDeleteResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.InstanceSnapshotDeleteResult{
		SnapshotName:          "mock-snapshot",
		SnapshotID:            snapshotID,
		DeletedSnapshots:      []string{snapshotID},
		StorageSavingsMonthly: 5.00,
		DeletedAt:             time.Now(),
	}, nil
}

func (m *MockAPIClient) RestoreInstanceFromSnapshot(ctx context.Context, snapshotID string, req types.InstanceRestoreRequest) (*types.InstanceRestoreResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.InstanceRestoreResult{
		NewInstanceName: req.NewInstanceName,
		InstanceID:      "i-restored123",
		SnapshotName:    req.SnapshotName,
		SnapshotID:      snapshotID,
		SourceTemplate:  "python-ml",
		State:           "pending",
		Message:         "Instance restored successfully (mock)",
		RestoredAt:      time.Now(),
	}, nil
}

// Backup operations

func (m *MockAPIClient) CreateBackup(ctx context.Context, req types.BackupCreateRequest) (*types.BackupCreateResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	backupType := "full"
	if req.Incremental {
		backupType = "incremental"
	}
	return &types.BackupCreateResult{
		BackupName:                 req.BackupName,
		BackupID:                   "backup-mock123",
		SourceInstance:             req.InstanceName,
		BackupType:                 backupType,
		StorageType:                req.StorageType,
		StorageLocation:            "s3://mock-bucket",
		EstimatedCompletionMinutes: 15,
		EstimatedSizeBytes:         1024 * 1024 * 1024, // 1GB
		StorageCostMonthly:         2.50,
		CreatedAt:                  time.Now(),
	}, nil
}

func (m *MockAPIClient) ListBackups(ctx context.Context) (*types.BackupListResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.BackupListResponse{
		Backups: []types.BackupInfo{},
		Count:   0,
	}, nil
}

func (m *MockAPIClient) GetBackup(ctx context.Context, backupID string) (*types.BackupInfo, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.BackupInfo{
		BackupName:       "mock-backup",
		BackupID:         backupID,
		SourceInstance:   "test-instance",
		SourceInstanceId: "i-mock123",
		Description:      "Mock backup",
		BackupType:       "full",
		StorageType:      "s3",
		StorageLocation:  "s3://mock-bucket",
		State:            "available",
		SizeBytes:        1024 * 1024 * 1024, // 1GB
		CompressedBytes:  512 * 1024 * 1024,  // 512MB
		FileCount:        1000,
		IncludedPaths:    []string{"/home"},
		ExcludedPaths:    []string{},
		Encrypted:        true,
	}, nil
}

func (m *MockAPIClient) DeleteBackup(ctx context.Context, backupID string) (*types.BackupDeleteResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.BackupDeleteResult{
		BackupName:            "mock-backup",
		BackupID:              backupID,
		StorageType:           "s3",
		StorageLocation:       "s3://mock-bucket",
		DeletedSizeBytes:      1024 * 1024 * 1024, // 1GB
		StorageSavingsMonthly: 2.50,
		DeletedAt:             time.Now(),
	}, nil
}

func (m *MockAPIClient) GetBackupContents(ctx context.Context, req types.BackupContentsRequest) (*types.BackupContentsResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.BackupContentsResponse{
		BackupName: "mock-backup",
		Path:       req.Path,
		Files:      []types.BackupFileInfo{},
		Count:      0,
		TotalSize:  0,
	}, nil
}

func (m *MockAPIClient) VerifyBackup(ctx context.Context, req types.BackupVerifyRequest) (*types.BackupVerifyResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	now := time.Now()
	return &types.BackupVerifyResult{
		BackupName:            "mock-backup",
		VerificationState:     "valid",
		CheckedFileCount:      1000,
		CorruptFileCount:      0,
		MissingFileCount:      0,
		VerifiedBytes:         1024 * 1024 * 1024, // 1GB
		VerificationStarted:   now.Add(-5 * time.Minute),
		VerificationCompleted: &now,
		CorruptFiles:          []string{},
		MissingFiles:          []string{},
	}, nil
}

func (m *MockAPIClient) RestoreBackup(ctx context.Context, req types.RestoreRequest) (*types.RestoreResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.RestoreResult{
		RestoreID:         "restore-mock123",
		BackupName:        req.BackupName,
		TargetInstance:    req.TargetInstance,
		RestorePath:       req.RestorePath,
		SelectivePaths:    req.SelectivePaths,
		State:             "running",
		RestoredFileCount: 0,
		RestoredBytes:     0,
		SkippedFileCount:  0,
		ErrorCount:        0,
	}, nil
}

func (m *MockAPIClient) GetRestoreStatus(ctx context.Context, restoreID string) (*types.RestoreResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.RestoreResult{
		RestoreID:         restoreID,
		BackupName:        "mock-backup",
		TargetInstance:    "test-instance",
		RestorePath:       "/home",
		SelectivePaths:    []string{},
		State:             "completed",
		RestoredFileCount: 1000,
		RestoredBytes:     1024 * 1024 * 1024, // 1GB
		SkippedFileCount:  0,
		ErrorCount:        0,
	}, nil
}

func (m *MockAPIClient) ListRestoreOperations(ctx context.Context) ([]types.RestoreResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []types.RestoreResult{}, nil
}

// Version compatibility

func (m *MockAPIClient) CheckVersionCompatibility(ctx context.Context, clientVersion string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// AMI lifecycle operations

func (m *MockAPIClient) CleanupAMIs(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"deleted_count": 0,
		"message":       "No AMIs to cleanup (mock)",
	}, nil
}

func (m *MockAPIClient) DeleteAMI(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success": true,
		"message": "AMI deleted successfully (mock)",
	}, nil
}

func (m *MockAPIClient) ListAMISnapshots(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"snapshots": []map[string]interface{}{},
		"count":     0,
	}, nil
}

func (m *MockAPIClient) CreateAMISnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"snapshot_id": "snap-mock123",
		"status":      "creating",
		"message":     "AMI snapshot creation initiated (mock)",
	}, nil
}

func (m *MockAPIClient) RestoreAMIFromSnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"ami_id":  "ami-mock123",
		"status":  "creating",
		"message": "AMI restoration initiated (mock)",
	}, nil
}

func (m *MockAPIClient) DeleteAMISnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success": true,
		"message": "AMI snapshot deleted successfully (mock)",
	}, nil
}

// AMI Creation operations - Mock implementations

func (m *MockAPIClient) CreateAMI(ctx context.Context, request types.AMICreationRequest) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"creation_id":                  fmt.Sprintf("ami-creation-%s-mock", request.TemplateName),
		"ami_id":                       "ami-mocktest123456789",
		"template_name":                request.TemplateName,
		"instance_id":                  request.InstanceID,
		"target_regions":               request.MultiRegion,
		"status":                       "pending",
		"message":                      "AMI creation initiated successfully (mock)",
		"estimated_completion_minutes": 12,
		"storage_cost":                 8.50,
		"creation_cost":                0.025,
	}, nil
}

func (m *MockAPIClient) GetAMIStatus(ctx context.Context, creationID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"creation_id":                  creationID,
		"ami_id":                       "ami-mocktest123456789",
		"status":                       "in_progress",
		"progress":                     75,
		"message":                      "AMI creation in progress - creating snapshot (mock)",
		"estimated_completion_minutes": 3,
		"elapsed_time_minutes":         9,
		"storage_cost":                 8.50,
		"creation_cost":                0.025,
	}, nil
}

func (m *MockAPIClient) ListUserAMIs(ctx context.Context) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"user_amis": []map[string]interface{}{
			{
				"ami_id":        "ami-mocktest123456789",
				"name":          "mock-custom-env",
				"description":   "Mock custom environment",
				"architecture":  "x86_64",
				"owner":         "123456789012",
				"creation_date": "2024-12-01T15:30:00Z",
				"public":        false,
				"tags": map[string]string{
					"Prism":    "true",
					"Template": "mock-template",
					"Creator":  "mock-user",
				},
			},
		},
		"total_count":  1,
		"storage_cost": 8.50,
	}, nil
}

// CheckAMIFreshness checks the freshness of AMIs for all templates
func (m *MockAPIClient) CheckAMIFreshness(ctx context.Context) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"templates": []map[string]interface{}{
			{
				"name":         "python-ml",
				"status":       "fresh",
				"days_old":     5,
				"max_age_days": 30,
				"message":      "AMI is up to date (mock)",
			},
		},
	}, nil
}

// CloseInstanceTunnels closes all tunnels for a given instance
func (m *MockAPIClient) CloseInstanceTunnels(_ context.Context, _ string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// CreateTunnels creates tunnels for the specified services
func (m *MockAPIClient) CreateTunnels(_ context.Context, _ string, _ []string) (*client.CreateTunnelsResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.CreateTunnelsResponse{}, nil
}

// ListTunnels lists all tunnels for an instance
func (m *MockAPIClient) ListTunnels(_ context.Context, _ string) (*client.ListTunnelsResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.ListTunnelsResponse{}, nil
}

// CloseTunnel closes a specific tunnel
func (m *MockAPIClient) CloseTunnel(_ context.Context, _ string, _ string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// ListInstancesWithRefresh lists instances with optional refresh
func (m *MockAPIClient) ListInstancesWithRefresh(_ context.Context, _ bool) (*types.ListResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.ListResponse{
		Instances: m.Instances,
	}, nil
}

// PreventProjectLaunches prevents new instance launches for a project (mock)
func (m *MockAPIClient) PreventProjectLaunches(ctx context.Context, projectID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Project '%s' launch prevention enabled successfully (mock)", projectID),
		"status":  "launches_prevented",
	}, nil
}

// AllowProjectLaunches allows new instance launches for a project (mock)
func (m *MockAPIClient) AllowProjectLaunches(ctx context.Context, projectID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Project '%s' launch prevention disabled successfully (mock)", projectID),
		"status":  "launches_allowed",
	}, nil
}

// GetProjectCushion returns mock cushion config.
func (m *MockAPIClient) GetProjectCushion(_ context.Context, _ string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"enabled": false, "headroom_percent": 0.10, "mode": "hibernate"}, nil
}

// SetProjectCushion saves mock cushion config.
func (m *MockAPIClient) SetProjectCushion(_ context.Context, _ string, cfg map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return cfg, nil
}

// DeleteProjectCushion removes mock cushion config.
func (m *MockAPIClient) DeleteProjectCushion(_ context.Context, _ string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// Invitation operations
func (m *MockAPIClient) GetInvitationByToken(ctx context.Context, token string) (*client.GetInvitationResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.GetInvitationResponse{
		Invitation: &types.Invitation{
			ID:        "inv-12345",
			ProjectID: "test-project",
			Email:     "user@example.com",
			Role:      types.ProjectRoleMember,
			Token:     token,
			InvitedBy: "admin",
			InvitedAt: time.Now().Add(-24 * time.Hour),
			ExpiresAt: time.Now().Add(6 * 24 * time.Hour),
			Status:    types.InvitationPending,
		},
		Project: &types.Project{
			ID:          "test-project",
			Name:        "Test Project",
			Description: "A test project",
			Owner:       "admin",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		},
	}, nil
}

func (m *MockAPIClient) AcceptInvitation(ctx context.Context, token string) (*client.InvitationActionResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	now := time.Now()
	return &client.InvitationActionResponse{
		Invitation: &types.Invitation{
			ID:         "inv-12345",
			ProjectID:  "test-project",
			Email:      "user@example.com",
			Role:       types.ProjectRoleMember,
			Token:      token,
			InvitedBy:  "admin",
			InvitedAt:  time.Now().Add(-24 * time.Hour),
			ExpiresAt:  time.Now().Add(6 * 24 * time.Hour),
			Status:     types.InvitationAccepted,
			AcceptedAt: &now,
		},
		Project: &types.Project{
			ID:          "test-project",
			Name:        "Test Project",
			Description: "A test project",
			Owner:       "admin",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		},
		Message: "Invitation accepted successfully",
	}, nil
}

func (m *MockAPIClient) DeclineInvitation(ctx context.Context, token string, reason string) (*client.InvitationActionResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	now := time.Now()
	return &client.InvitationActionResponse{
		Invitation: &types.Invitation{
			ID:            "inv-12345",
			ProjectID:     "test-project",
			Email:         "user@example.com",
			Role:          types.ProjectRoleMember,
			Token:         token,
			InvitedBy:     "admin",
			InvitedAt:     time.Now().Add(-24 * time.Hour),
			ExpiresAt:     time.Now().Add(6 * 24 * time.Hour),
			Status:        types.InvitationDeclined,
			DeclinedAt:    &now,
			DeclineReason: reason,
		},
		Message: "Invitation declined",
	}, nil
}

func (m *MockAPIClient) SendInvitation(ctx context.Context, projectID string, req client.SendInvitationRequest) (*client.SendInvitationResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.SendInvitationResponse{
		Invitation: &types.Invitation{
			ID:        "inv-67890",
			ProjectID: projectID,
			Email:     req.Email,
			Role:      req.Role,
			Token:     "mock-invitation-token",
			InvitedBy: "current-user",
			InvitedAt: time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			Status:    types.InvitationPending,
		},
		Project: &types.Project{
			ID:          projectID,
			Name:        "Test Project",
			Description: "A test project",
			Owner:       "admin",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		},
		Message: "Invitation sent successfully",
	}, nil
}

func (m *MockAPIClient) SendBulkInvitation(ctx context.Context, projectID string, req *types.BulkInvitationRequest) (*types.BulkInvitationResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	var results []types.BulkInvitationResult
	for _, entry := range req.Invitations {
		results = append(results, types.BulkInvitationResult{
			Email:        entry.Email,
			Status:       "sent",
			InvitationID: "mock-inv-" + entry.Email,
		})
	}
	return &types.BulkInvitationResponse{
		Summary: types.BulkInvitationSummary{
			Total:   len(req.Invitations),
			Sent:    len(req.Invitations),
			Failed:  0,
			Skipped: 0,
		},
		Results: results,
		Message: "All invitations sent successfully",
	}, nil
}

// Shared token operations
func (m *MockAPIClient) CreateSharedToken(ctx context.Context, projectID string, req *types.CreateSharedTokenRequest) (*types.SharedInvitationToken, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	return &types.SharedInvitationToken{
		ID:              "token-12345",
		Token:           "WORKSHOP-2025",
		Name:            req.Name,
		ProjectID:       projectID,
		Role:            req.Role,
		Message:         req.Message,
		CreatedBy:       req.CreatedBy,
		CreatedAt:       time.Now(),
		ExpiresAt:       &expiresAt,
		RedemptionLimit: req.RedemptionLimit,
		RedemptionCount: 0,
		Status:          types.SharedTokenActive,
		Redemptions:     []types.SharedTokenRedemption{},
	}, nil
}

func (m *MockAPIClient) RedeemSharedToken(ctx context.Context, req *types.RedeemSharedTokenRequest) (*types.RedeemSharedTokenResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.RedeemSharedTokenResponse{
		Success:              true,
		ProjectID:            "test-project",
		Role:                 types.ProjectRoleMember,
		RedemptionPosition:   1,
		RemainingRedemptions: 49,
		Message:              "Successfully redeemed shared token",
	}, nil
}

func (m *MockAPIClient) GetSharedToken(ctx context.Context, tokenID string) (*types.SharedInvitationToken, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	return &types.SharedInvitationToken{
		ID:              tokenID,
		Token:           "WORKSHOP-2025",
		Name:            "Mock Token",
		ProjectID:       "test-project",
		Role:            types.ProjectRoleMember,
		Message:         "Welcome to the workshop",
		CreatedBy:       "admin",
		CreatedAt:       time.Now().Add(-24 * time.Hour),
		ExpiresAt:       &expiresAt,
		RedemptionLimit: 50,
		RedemptionCount: 10,
		Status:          types.SharedTokenActive,
		Redemptions:     []types.SharedTokenRedemption{},
	}, nil
}

func (m *MockAPIClient) ListSharedTokens(ctx context.Context, projectID string) ([]*types.SharedInvitationToken, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	return []*types.SharedInvitationToken{
		{
			ID:              "token-1",
			Token:           "WORKSHOP-2025",
			Name:            "Workshop Token",
			ProjectID:       projectID,
			Role:            types.ProjectRoleMember,
			Message:         "Welcome",
			CreatedBy:       "admin",
			CreatedAt:       time.Now().Add(-24 * time.Hour),
			ExpiresAt:       &expiresAt,
			RedemptionLimit: 50,
			RedemptionCount: 23,
			Status:          types.SharedTokenActive,
			Redemptions:     []types.SharedTokenRedemption{},
		},
	}, nil
}

func (m *MockAPIClient) ExtendSharedToken(ctx context.Context, tokenID string, daysToAdd int) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) RevokeSharedToken(ctx context.Context, tokenID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) GetSharedTokenQR(ctx context.Context, tokenCode string) ([]byte, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []byte("mock-qr-png"), nil
}

// Throttling operations
func (m *MockAPIClient) ConfigureThrottling(ctx context.Context, config throttle.Config) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) GetThrottlingStatus(ctx context.Context, scope string) (*throttle.Status, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &throttle.Status{
		Enabled: true,
		Scope:   scope,
	}, nil
}

func (m *MockAPIClient) GetThrottlingRemaining(ctx context.Context, scope string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"scope":     scope,
		"remaining": 100,
		"total":     1000,
	}, nil
}

func (m *MockAPIClient) DisableThrottling(ctx context.Context, scope string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) ListProjectThrottleOverrides(ctx context.Context) ([]throttle.Override, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []throttle.Override{}, nil
}

func (m *MockAPIClient) SetProjectThrottleOverride(ctx context.Context, projectID string, override throttle.Override) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) RemoveProjectThrottleOverride(ctx context.Context, projectID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// Transfer operations (S3-backed file transfer)
func (m *MockAPIClient) StartTransfer(ctx context.Context, req client.TransferRequest) (*client.TransferResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.TransferResponse{
		TransferID: "mock-transfer-id",
		Status:     "in_progress",
	}, nil
}

func (m *MockAPIClient) GetTransferStatus(ctx context.Context, transferID string) (*storage.TransferProgress, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &storage.TransferProgress{
		TransferID:       transferID,
		Status:           "in_progress",
		PercentComplete:  50.0,
		TransferredBytes: 500,
		TotalBytes:       1000,
	}, nil
}

func (m *MockAPIClient) ListTransfers(ctx context.Context) ([]*storage.TransferProgress, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []*storage.TransferProgress{}, nil
}

func (m *MockAPIClient) CancelTransfer(ctx context.Context, transferID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// SSM file operations (#30a)
func (m *MockAPIClient) ListInstanceFiles(ctx context.Context, instanceName, path string) ([]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []interface{}{}, nil
}

func (m *MockAPIClient) PushFileToInstance(ctx context.Context, instanceName, localPath, remotePath string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"status": "ok", "remote_path": remotePath}, nil
}

func (m *MockAPIClient) PullFileFromInstance(ctx context.Context, instanceName, remotePath, localPath string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"status": "ok", "local_path": localPath}, nil
}

// CheckAllocationStatus checks if an allocation is exhausted (mock)
func (m *MockAPIClient) CheckAllocationStatus(ctx context.Context, allocationID string) (*client.AllocationStatus, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.AllocationStatus{
		Exhausted: false,
		Remaining: 1000.0,
	}, nil
}

// CreateAllocation creates a budget allocation for a project (mock)
func (m *MockAPIClient) CreateAllocation(ctx context.Context, req project.CreateAllocationRequest) (*types.ProjectBudgetAllocation, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.ProjectBudgetAllocation{
		ID:              "alloc-12345",
		BudgetID:        req.BudgetID,
		ProjectID:       req.ProjectID,
		AllocatedAmount: req.AllocatedAmount,
		SpentAmount:     0.0,
	}, nil
}

// CreateBudget creates a new budget pool (mock)
func (m *MockAPIClient) CreateBudget(ctx context.Context, req project.CreateBudgetRequest) (*types.Budget, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.Budget{
		ID:              "budget-12345",
		Name:            req.Name,
		Description:     req.Description,
		TotalAmount:     req.TotalAmount,
		AllocatedAmount: 0.0,
		SpentAmount:     0.0,
		Period:          types.BudgetPeriodMonthly,
		StartDate:       time.Now(),
		AlertThreshold:  0.8,
	}, nil
}

// DeleteAllocation deletes a budget allocation (mock)
func (m *MockAPIClient) DeleteAllocation(ctx context.Context, allocationID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// DeleteBudget deletes a budget pool (mock)
func (m *MockAPIClient) DeleteBudget(ctx context.Context, budgetID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// GetBudget gets a budget by ID (mock)
func (m *MockAPIClient) GetBudget(ctx context.Context, budgetID string) (*types.Budget, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.Budget{
		ID:              budgetID,
		Name:            "Mock Budget",
		Description:     "A mock budget",
		TotalAmount:     10000.0,
		AllocatedAmount: 5000.0,
		SpentAmount:     2000.0,
		Period:          types.BudgetPeriodMonthly,
		StartDate:       time.Now().Add(-30 * 24 * time.Hour),
		AlertThreshold:  0.8,
	}, nil
}

// ListBudgets lists all budgets (mock)
func (m *MockAPIClient) ListBudgets(ctx context.Context) ([]*types.Budget, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []*types.Budget{
		{
			ID:              "budget-1",
			Name:            "Mock Budget 1",
			Description:     "First mock budget",
			TotalAmount:     10000.0,
			AllocatedAmount: 5000.0,
			SpentAmount:     2000.0,
			Period:          types.BudgetPeriodMonthly,
			StartDate:       time.Now().Add(-30 * 24 * time.Hour),
			AlertThreshold:  0.8,
		},
	}, nil
}

// UpdateBudget updates a budget (mock)
func (m *MockAPIClient) UpdateBudget(ctx context.Context, budgetID string, req project.UpdateBudgetRequest) (*types.Budget, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.Budget{
		ID:              budgetID,
		Name:            "Updated Budget",
		Description:     "An updated mock budget",
		TotalAmount:     15000.0,
		AllocatedAmount: 5000.0,
		SpentAmount:     2000.0,
		Period:          types.BudgetPeriodMonthly,
		StartDate:       time.Now().Add(-30 * 24 * time.Hour),
		AlertThreshold:  0.8,
	}, nil
}

// GetBudgetSummary gets a budget summary (mock)
func (m *MockAPIClient) GetBudgetSummary(ctx context.Context, budgetID string) (*types.BudgetSummary, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.BudgetSummary{
		Budget: types.Budget{
			ID:              budgetID,
			Name:            "Mock Budget",
			Description:     "A mock budget",
			TotalAmount:     10000.0,
			AllocatedAmount: 5000.0,
			SpentAmount:     2000.0,
			Period:          types.BudgetPeriodMonthly,
			StartDate:       time.Now().Add(-30 * 24 * time.Hour),
			AlertThreshold:  0.8,
		},
		Allocations:     []types.ProjectBudgetAllocation{},
		ProjectNames:    map[string]string{},
		RemainingAmount: 5000.0,
		UtilizationRate: 0.4,
	}, nil
}

// GetInvitationByID gets an invitation by ID (mock)
func (m *MockAPIClient) GetInvitationByID(ctx context.Context, invitationID string) (*client.GetInvitationResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.GetInvitationResponse{
		Invitation: &types.Invitation{
			ID:        invitationID,
			ProjectID: "test-project",
			Email:     "user@example.com",
			Role:      types.ProjectRoleMember,
			Token:     "mock-token",
			InvitedBy: "admin",
			InvitedAt: time.Now().Add(-24 * time.Hour),
			ExpiresAt: time.Now().Add(6 * 24 * time.Hour),
			Status:    types.InvitationPending,
		},
		Project: &types.Project{
			ID:          "test-project",
			Name:        "Test Project",
			Description: "A test project",
			Owner:       "admin",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		},
	}, nil
}

// GetBudgetAllocations gets all allocations for a budget (mock)
func (m *MockAPIClient) GetBudgetAllocations(ctx context.Context, budgetID string) ([]*types.ProjectBudgetAllocation, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []*types.ProjectBudgetAllocation{
		{
			ID:              "alloc-1",
			BudgetID:        budgetID,
			ProjectID:       "project-1",
			AllocatedAmount: 2000.0,
			SpentAmount:     500.0,
		},
	}, nil
}

// GetAllocation gets an allocation by ID (mock)
func (m *MockAPIClient) GetAllocation(ctx context.Context, allocationID string) (*types.ProjectBudgetAllocation, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.ProjectBudgetAllocation{
		ID:              allocationID,
		BudgetID:        "budget-123",
		ProjectID:       "project-456",
		AllocatedAmount: 2000.0,
		SpentAmount:     500.0,
	}, nil
}

// UpdateAllocation updates an allocation (mock)
func (m *MockAPIClient) UpdateAllocation(ctx context.Context, allocationID string, req project.UpdateAllocationRequest) (*types.ProjectBudgetAllocation, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.ProjectBudgetAllocation{
		ID:              allocationID,
		BudgetID:        "budget-123",
		ProjectID:       "project-456",
		AllocatedAmount: 3000.0,
		SpentAmount:     500.0,
	}, nil
}

// GetProjectAllocations gets all allocations for a project (mock)
func (m *MockAPIClient) GetProjectAllocations(ctx context.Context, projectID string) ([]*types.ProjectBudgetAllocation, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []*types.ProjectBudgetAllocation{
		{
			ID:              "alloc-1",
			BudgetID:        "budget-123",
			ProjectID:       projectID,
			AllocatedAmount: 2000.0,
			SpentAmount:     500.0,
		},
	}, nil
}

// GetProjectFundingSummary gets the funding summary for a project (mock)
func (m *MockAPIClient) GetProjectFundingSummary(ctx context.Context, projectID string) (*types.ProjectFundingSummary, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.ProjectFundingSummary{
		ProjectID:   projectID,
		ProjectName: "Mock Project",
		Allocations: []types.ProjectBudgetAllocation{
			{
				ID:              "alloc-1",
				BudgetID:        "budget-123",
				ProjectID:       projectID,
				AllocatedAmount: 2000.0,
				SpentAmount:     500.0,
			},
		},
		BudgetNames:    map[string]string{"budget-123": "Mock Budget"},
		TotalAllocated: 2000.0,
		TotalSpent:     500.0,
	}, nil
}

// RecordSpending records spending against an allocation (mock)
func (m *MockAPIClient) RecordSpending(ctx context.Context, allocationID string, amount float64) (*project.SpendingResult, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &project.SpendingResult{
		AllocationID:        allocationID,
		AllocationExhausted: false,
		BackupActivated:     false,
		BackupAllocationID:  "",
		WarningMessage:      "",
	}, nil
}

// ResendInvitation resends an invitation (mock)
func (m *MockAPIClient) ResendInvitation(ctx context.Context, invitationID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// RevokeInvitation revokes an invitation (mock)
func (m *MockAPIClient) RevokeInvitation(ctx context.Context, invitationID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// ListProjectInvitations lists invitations for a project (mock)
func (m *MockAPIClient) ListProjectInvitations(ctx context.Context, projectID string, status string) (*client.ListProjectInvitationsResponse, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &client.ListProjectInvitationsResponse{
		Invitations: []*types.Invitation{},
		Total:       0,
	}, nil
}

// GetProjectGDEW returns mock GDEW status (Issue #206).
func (m *MockAPIClient) GetProjectGDEW(ctx context.Context, projectID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"project_id":         projectID,
		"total_spend_mtd":    0.0,
		"egress_charges_mtd": 0.0,
		"available_credit":   0.0,
		"used_credit":        0.0,
		"remaining_credit":   0.0,
		"net_egress_cost":    0.0,
		"coverage_percent":   0.0,
		"fully_covered":      false,
		"nearing_limit":      false,
		"status_message":     "No data tracked yet.",
	}, nil
}

// UpdateProjectGDEW updates GDEW spend/egress figures (mock, Issue #206).
func (m *MockAPIClient) UpdateProjectGDEW(ctx context.Context, projectID string, totalSpend, egressCharges float64) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	available := totalSpend * 0.15
	used := egressCharges
	if used > available {
		used = available
	}
	return map[string]interface{}{
		"project_id":         projectID,
		"total_spend_mtd":    totalSpend,
		"egress_charges_mtd": egressCharges,
		"available_credit":   available,
		"used_credit":        used,
		"remaining_credit":   available - used,
		"net_egress_cost":    egressCharges - used,
	}, nil
}

// GetProjectDiscounts returns mock discount discovery results (Issue #207).
func (m *MockAPIClient) GetProjectDiscounts(ctx context.Context, projectID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"project_id":            projectID,
		"discounts":             []interface{}{},
		"credits":               []interface{}{},
		"total_monthly_savings": 0.0,
		"total_credit_balance":  0.0,
		"effective_cost_rate":   1.0,
	}, nil
}

// GetProjectCredits returns mock AWS credit balances (Issue #207).
func (m *MockAPIClient) GetProjectCredits(ctx context.Context, projectID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{
		"project_id":            projectID,
		"discounts":             []interface{}{},
		"credits":               []interface{}{},
		"total_monthly_savings": 0.0,
		"total_credit_balance":  0.0,
		"effective_cost_rate":   1.0,
	}, nil
}

// v0.12.0 — Budget Enterprise & Governance mocks

func (m *MockAPIClient) GetProjectQuotas(ctx context.Context, projectID string) ([]types.RoleQuota, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []types.RoleQuota{}, nil
}

func (m *MockAPIClient) SetProjectQuota(ctx context.Context, projectID string, quota types.RoleQuota) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) GetGrantPeriod(ctx context.Context, projectID string) (*types.GrantPeriod, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil, nil
}

func (m *MockAPIClient) SetGrantPeriod(ctx context.Context, projectID string, gp types.GrantPeriod) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) DeleteGrantPeriod(ctx context.Context, projectID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) ShareProjectBudget(ctx context.Context, req types.BudgetShareRequest) (*types.BudgetShareRecord, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &types.BudgetShareRecord{ID: "mock-share-id"}, nil
}

func (m *MockAPIClient) ListProjectBudgetShares(ctx context.Context, projectID string) ([]*types.BudgetShareRecord, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []*types.BudgetShareRecord{}, nil
}

func (m *MockAPIClient) DeleteProjectBudgetShare(ctx context.Context, projectID, shareID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) SubmitApproval(ctx context.Context, projectID string, approvalType project.ApprovalType, details map[string]interface{}, reason string) (*project.ApprovalRequest, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &project.ApprovalRequest{
		ID:        "mock-approval-id",
		ProjectID: projectID,
		Type:      approvalType,
		Status:    project.ApprovalStatusPending,
		Reason:    reason,
	}, nil
}

func (m *MockAPIClient) ListApprovals(ctx context.Context, projectID string, status project.ApprovalStatus) ([]*project.ApprovalRequest, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	var result []*project.ApprovalRequest
	for _, r := range m.Approvals {
		if r.ProjectID != projectID {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		result = append(result, r)
	}
	if result == nil {
		return []*project.ApprovalRequest{}, nil
	}
	return result, nil
}

func (m *MockAPIClient) ListAllApprovals(ctx context.Context, status project.ApprovalStatus) ([]*project.ApprovalRequest, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	var result []*project.ApprovalRequest
	for _, r := range m.Approvals {
		if status != "" && r.Status != status {
			continue
		}
		result = append(result, r)
	}
	if result == nil {
		return []*project.ApprovalRequest{}, nil
	}
	return result, nil
}

func (m *MockAPIClient) ApproveRequest(ctx context.Context, projectID, requestID, note string) (*project.ApprovalRequest, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &project.ApprovalRequest{
		ID:        requestID,
		ProjectID: projectID,
		Status:    project.ApprovalStatusApproved,
	}, nil
}

func (m *MockAPIClient) DenyRequest(ctx context.Context, projectID, requestID, note string) (*project.ApprovalRequest, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &project.ApprovalRequest{
		ID:        requestID,
		ProjectID: projectID,
		Status:    project.ApprovalStatusDenied,
	}, nil
}

func (m *MockAPIClient) GetApproval(ctx context.Context, projectID, approvalID string) (*project.ApprovalRequest, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return &project.ApprovalRequest{
		ID:        approvalID,
		ProjectID: projectID,
		Status:    project.ApprovalStatusPending,
	}, nil
}

func (m *MockAPIClient) GetMonthlyReport(ctx context.Context, projectID, month, format string) (string, error) {
	if m.ShouldReturnError {
		return "", fmt.Errorf("%s", m.ErrorMessage)
	}
	return "Monthly Report: " + projectID + " " + month, nil
}

func (m *MockAPIClient) ListOnboardingTemplates(ctx context.Context, projectID string) ([]types.OnboardingTemplate, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return []types.OnboardingTemplate{}, nil
}

func (m *MockAPIClient) AddOnboardingTemplate(ctx context.Context, projectID string, tmpl types.OnboardingTemplate) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) DeleteOnboardingTemplate(ctx context.Context, projectID, nameOrID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// v0.14.0 — University Education System (interface methods for fixture cleanup)
func (m *MockAPIClient) DeleteCourse(ctx context.Context, courseID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

// v0.18.0 — Workshop & Event Management stubs

func (m *MockAPIClient) ListWorkshops(ctx context.Context, params string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"workshops": []interface{}{}}, nil
}

func (m *MockAPIClient) CreateWorkshop(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"id": "mock-ws-id", "title": req["title"], "status": "draft", "join_token": "WS-MOCKTOKEN"}, nil
}

func (m *MockAPIClient) GetWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"id": workshopID, "title": "Mock Workshop", "status": "draft"}, nil
}

func (m *MockAPIClient) UpdateWorkshop(ctx context.Context, workshopID string, req map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"id": workshopID}, nil
}

func (m *MockAPIClient) DeleteWorkshop(ctx context.Context, workshopID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}

func (m *MockAPIClient) ProvisionWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"provisioned": 0, "skipped": 0, "errors": []interface{}{}}, nil
}

func (m *MockAPIClient) GetWorkshopDashboard(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"workshop_id": workshopID, "title": "Mock Workshop", "status": "draft", "total_participants": 0, "active_instances": 0, "stopped_instances": 0, "pending_instances": 0, "time_remaining": "N/A", "total_spent": 0.0, "participants": []interface{}{}}, nil
}

func (m *MockAPIClient) EndWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"stopped": 0, "errors": []interface{}{}}, nil
}

func (m *MockAPIClient) GetWorkshopDownload(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"workshop_id": workshopID, "participants": []interface{}{}}, nil
}

func (m *MockAPIClient) ListWorkshopConfigs(ctx context.Context) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"configs": []interface{}{}}, nil
}

func (m *MockAPIClient) SaveWorkshopConfig(ctx context.Context, workshopID, name string) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"name": name, "template": "mock-template", "duration_hours": 6, "max_participants": 0}, nil
}

func (m *MockAPIClient) CreateWorkshopFromConfig(ctx context.Context, configName string, req map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"id": "mock-ws-from-config", "title": req["title"], "join_token": "WS-MOCKTOKEN"}, nil
}

func (m *MockAPIClient) AddWorkshopParticipant(ctx context.Context, workshopID string, req map[string]interface{}) (map[string]interface{}, error) {
	if m.ShouldReturnError {
		return nil, fmt.Errorf("%s", m.ErrorMessage)
	}
	return map[string]interface{}{"user_id": req["user_id"], "status": "pending"}, nil
}

func (m *MockAPIClient) RemoveWorkshopParticipant(ctx context.Context, workshopID, userID string) error {
	if m.ShouldReturnError {
		return fmt.Errorf("%s", m.ErrorMessage)
	}
	return nil
}
