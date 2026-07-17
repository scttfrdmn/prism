package client

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/prism/pkg/idle"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/storage"
	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
)

// MockClient provides a mock implementation of PrismAPI for testing
type MockClient struct {
	// Mock data
	instances map[string]*types.Instance
	templates map[string]types.Template
	volumes   []types.StorageVolume
	storage   []types.StorageVolume

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
		volumes:   make([]types.StorageVolume, 0),
		storage:   make([]types.StorageVolume, 0),
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

func (m *MockClient) LaunchArray(ctx context.Context, req types.LaunchArrayRequest) (*types.LaunchArrayResponse, error) {
	resp := &types.LaunchArrayResponse{
		JobArrayID: req.Name + "-mock",
		Requested:  req.Count,
	}
	for i := 0; i < req.Count; i++ {
		name := fmt.Sprintf("%s-%d", req.Name, i)
		inst := &types.Instance{
			ID:            "mock-" + name,
			Name:          name,
			Template:      req.Template,
			State:         "running",
			JobArrayID:    resp.JobArrayID,
			JobArrayIndex: i,
		}
		m.instances[name] = inst
		resp.Instances = append(resp.Instances, *inst)
		resp.Launched++
	}
	return resp, nil
}

func (m *MockClient) LaunchSweep(ctx context.Context, req types.LaunchSweepRequest) (*types.LaunchSweepResponse, error) {
	resp := &types.LaunchSweepResponse{
		SweepID:   req.Name + "-mock",
		Requested: len(req.ParamSets),
	}
	for i := range req.ParamSets {
		name := fmt.Sprintf("%s-%d", req.Name, i)
		inst := &types.Instance{
			ID:         "mock-" + name,
			Name:       name,
			Template:   req.Template,
			State:      "running",
			SweepID:    resp.SweepID,
			SweepIndex: i,
		}
		m.instances[name] = inst
		resp.Instances = append(resp.Instances, *inst)
		resp.Launched++
	}
	return resp, nil
}

func (m *MockClient) ListInstances(ctx context.Context) (*types.ListResponse, error) {
	return m.ListInstancesWithRefresh(ctx, false)
}

func (m *MockClient) ListInstancesWithRefresh(ctx context.Context, refresh bool) (*types.ListResponse, error) {
	// Mock implementation - refresh parameter is ignored since we don't have real AWS state
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

func (m *MockClient) GetBilledCost(ctx context.Context, name string) (*types.BilledCostResult, error) {
	if instance, exists := m.instances[name]; exists {
		return &types.BilledCostResult{
			Name:        name,
			BilledTotal: instance.CurrentSpend,
			Currency:    "USD",
			Region:      instance.Region,
			Source:      "mock",
			TagActive:   true,
		}, nil
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

func (m *MockClient) ExecInstance(ctx context.Context, instanceName string, execRequest types.ExecRequest) (*types.ExecResult, error) {
	return &types.ExecResult{
		Command:       execRequest.Command,
		ExitCode:      0,
		StdOut:        "Mock command executed successfully",
		Status:        "success",
		ExecutionTime: 100,
		CommandID:     "mock-command-id",
	}, nil
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
func (m *MockClient) CreateVolume(ctx context.Context, req types.VolumeCreateRequest) (*types.StorageVolume, error) {
	volume := types.StorageVolume{
		Name:         req.Name,
		Type:         types.StorageTypeShared,
		AWSService:   types.AWSServiceEFS,
		FileSystemID: "mock-fs-" + req.Name,
		State:        "available",
	}
	m.volumes = append(m.volumes, volume)
	return &volume, nil
}

func (m *MockClient) ListVolumes(ctx context.Context) ([]*types.StorageVolume, error) {
	result := make([]*types.StorageVolume, len(m.volumes))
	for i := range m.volumes {
		result[i] = &m.volumes[i]
	}
	return result, nil
}

func (m *MockClient) GetVolume(ctx context.Context, name string) (*types.StorageVolume, error) {
	for i := range m.volumes {
		if m.volumes[i].Name == name {
			return &m.volumes[i], nil
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
func (m *MockClient) CreateStorage(ctx context.Context, req types.StorageCreateRequest) (*types.StorageVolume, error) {
	volume := types.StorageVolume{
		Name:       req.Name,
		Type:       types.StorageTypeWorkspace,
		AWSService: types.AWSServiceEBS,
		VolumeID:   "mock-vol-" + req.Name,
		State:      "available",
	}
	m.storage = append(m.storage, volume)
	return &volume, nil
}

func (m *MockClient) ListStorage(ctx context.Context) ([]*types.StorageVolume, error) {
	result := make([]*types.StorageVolume, len(m.storage))
	for i := range m.storage {
		result[i] = &m.storage[i]
	}
	return result, nil
}

func (m *MockClient) GetStorage(ctx context.Context, name string) (*types.StorageVolume, error) {
	for i := range m.storage {
		if m.storage[i].Name == name {
			return &m.storage[i], nil
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

// Template Marketplace operations (Phase 5.2) - Mock implementations

func (m *MockClient) SearchMarketplace(ctx context.Context, query map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"templates": []map[string]interface{}{
			{
				"id":       "mock-template-1",
				"name":     "Mock ML Template",
				"author":   "mock-user",
				"rating":   4.5,
				"tags":     []string{"ml", "python", "jupyter"},
				"featured": true,
			},
		},
	}, nil
}

func (m *MockClient) GetMarketplaceTemplate(ctx context.Context, templateID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":          templateID,
		"name":        "Mock Template",
		"description": "A mock template for testing",
		"author":      "mock-user",
		"rating":      4.5,
		"downloads":   100,
		"tags":        []string{"test", "mock"},
	}, nil
}

func (m *MockClient) PublishMarketplaceTemplate(ctx context.Context, template map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"success":     true,
		"message":     "Template published successfully",
		"template_id": "mock-published-123",
	}, nil
}

func (m *MockClient) AddMarketplaceReview(ctx context.Context, templateID string, review map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"success":   true,
		"message":   "Review added successfully",
		"review_id": "mock-review-123",
	}, nil
}

func (m *MockClient) ForkMarketplaceTemplate(ctx context.Context, templateID string, fork map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"success":        true,
		"message":        "Template forked successfully",
		"forked_id":      "mock-fork-123",
		"original_id":    templateID,
		"customizations": fork,
	}, nil
}

func (m *MockClient) GetMarketplaceFeatured(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"templates": []map[string]interface{}{
			{
				"id":     "featured-1",
				"name":   "Featured ML Template",
				"author": "community",
				"rating": 4.8,
				"tags":   []string{"ml", "featured"},
				"reason": "Top rated ML template",
			},
		},
	}, nil
}

func (m *MockClient) GetMarketplaceTrending(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"templates": []map[string]interface{}{
			{
				"id":        "trending-1",
				"name":      "Trending AI Template",
				"author":    "ai-researcher",
				"rating":    4.6,
				"tags":      []string{"ai", "trending"},
				"downloads": 250,
				"reason":    "Most downloaded this week",
			},
		},
	}, nil
}

// Hibernation operations
func (m *MockClient) HibernateInstance(ctx context.Context, name string) error {
	if instance, exists := m.instances[name]; exists {
		instance.State = "hibernated"
		return nil
	}
	return fmt.Errorf("instance not found: %s", name)
}

func (m *MockClient) ResumeInstance(ctx context.Context, name string) error {
	if instance, exists := m.instances[name]; exists {
		instance.State = "running"
		return nil
	}
	return fmt.Errorf("instance not found: %s", name)
}

func (m *MockClient) GetInstanceHibernationStatus(ctx context.Context, name string) (*types.HibernationStatus, error) {
	return &types.HibernationStatus{
		HibernationSupported: true,
		PossiblyHibernated:   false,
		InstanceState:        "running",
	}, nil
}

// Resize operations
func (m *MockClient) ResizeInstance(ctx context.Context, req types.ResizeRequest) (*types.ResizeResponse, error) {
	return &types.ResizeResponse{
		Success: true,
		Message: "Instance resized successfully",
	}, nil
}

// Log operations
func (m *MockClient) GetInstanceLogs(ctx context.Context, name string, req types.LogRequest) (*types.LogResponse, error) {
	return &types.LogResponse{}, nil
}

func (m *MockClient) GetInstanceLogTypes(ctx context.Context, name string) (*types.LogTypesResponse, error) {
	return &types.LogTypesResponse{}, nil
}

func (m *MockClient) GetLogsSummary(ctx context.Context) (*types.LogSummaryResponse, error) {
	return &types.LogSummaryResponse{}, nil
}

// Template application operations
func (m *MockClient) ApplyTemplate(ctx context.Context, req templates.ApplyRequest) (*templates.ApplyResponse, error) {
	return &templates.ApplyResponse{}, nil
}

func (m *MockClient) DiffTemplate(ctx context.Context, req templates.DiffRequest) (*templates.TemplateDiff, error) {
	return &templates.TemplateDiff{}, nil
}

func (m *MockClient) GetInstanceLayers(ctx context.Context, name string) ([]templates.AppliedTemplate, error) {
	return []templates.AppliedTemplate{}, nil
}

func (m *MockClient) RollbackInstance(ctx context.Context, req types.RollbackRequest) error {
	return nil
}

// Idle detection operations
func (m *MockClient) GetIdlePendingActions(ctx context.Context) ([]types.IdleState, error) {
	return []types.IdleState{}, nil
}

func (m *MockClient) ExecuteIdleActions(ctx context.Context) (*types.IdleExecutionResponse, error) {
	return &types.IdleExecutionResponse{}, nil
}

func (m *MockClient) GetIdleHistory(ctx context.Context) ([]types.IdleHistoryEntry, error) {
	return []types.IdleHistoryEntry{}, nil
}

// Project management operations - Stubs
func (m *MockClient) CreateProject(ctx context.Context, req project.CreateProjectRequest) (*types.Project, error) {
	return &types.Project{
		ID:          "mock-project-123",
		Name:        req.Name,
		Description: req.Description,
	}, nil
}

func (m *MockClient) ListProjects(ctx context.Context, filter *project.ProjectFilter) (*project.ProjectListResponse, error) {
	return &project.ProjectListResponse{}, nil
}

func (m *MockClient) GetProject(ctx context.Context, id string) (*types.Project, error) {
	return &types.Project{
		ID:   id,
		Name: "Mock Project",
	}, nil
}

func (m *MockClient) UpdateProject(ctx context.Context, id string, req project.UpdateProjectRequest) (*types.Project, error) {
	return &types.Project{
		ID:   id,
		Name: "Updated Project",
	}, nil
}

func (m *MockClient) DeleteProject(ctx context.Context, id string) error {
	return nil
}

func (m *MockClient) AddProjectMember(ctx context.Context, projectID string, req project.AddMemberRequest) error {
	return nil
}

func (m *MockClient) UpdateProjectMember(ctx context.Context, projectID, userID string, req project.UpdateMemberRequest) error {
	return nil
}

func (m *MockClient) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	return nil
}

func (m *MockClient) GetProjectMembers(ctx context.Context, projectID string) ([]types.ProjectMember, error) {
	return []types.ProjectMember{}, nil
}

func (m *MockClient) GetProjectBudgetStatus(ctx context.Context, projectID string) (*project.BudgetStatus, error) {
	return &project.BudgetStatus{
		BudgetEnabled: true,
		TotalBudget:   1000.0,
	}, nil
}

func (m *MockClient) SetProjectBudget(ctx context.Context, projectID string, req SetProjectBudgetRequest) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

func (m *MockClient) UpdateProjectBudget(ctx context.Context, projectID string, req UpdateProjectBudgetRequest) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

func (m *MockClient) DisableProjectBudget(ctx context.Context, projectID string) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

func (m *MockClient) GetProjectCostBreakdown(ctx context.Context, projectID string, start, end time.Time) (*types.ProjectCostBreakdown, error) {
	return &types.ProjectCostBreakdown{}, nil
}

func (m *MockClient) GetProjectResourceUsage(ctx context.Context, projectID string, duration time.Duration) (*types.ProjectResourceUsage, error) {
	return &types.ProjectResourceUsage{}, nil
}

// Policy management operations
func (m *MockClient) GetPolicyStatus(ctx context.Context) (*PolicyStatusResponse, error) {
	return &PolicyStatusResponse{
		Enabled: false,
		Status:  "disabled",
	}, nil
}

func (m *MockClient) ListPolicySets(ctx context.Context) (*PolicySetsResponse, error) {
	return &PolicySetsResponse{
		PolicySets: map[string]PolicySetInfo{},
	}, nil
}

func (m *MockClient) AssignPolicySet(ctx context.Context, setID string) (*PolicyAssignResponse, error) {
	return &PolicyAssignResponse{
		Success: true,
		Message: "Policy set assigned",
	}, nil
}

func (m *MockClient) SetPolicyEnforcement(ctx context.Context, enabled bool) (*PolicyEnforcementResponse, error) {
	return &PolicyEnforcementResponse{
		Success: true,
		Enabled: enabled,
	}, nil
}

func (m *MockClient) CheckTemplateAccess(ctx context.Context, templateName string) (*PolicyCheckResponse, error) {
	return &PolicyCheckResponse{
		Allowed:      true,
		TemplateName: templateName,
	}, nil
}

// Idle policy operations
func (m *MockClient) ListIdlePolicies(ctx context.Context) ([]*idle.PolicyTemplate, error) {
	return []*idle.PolicyTemplate{}, nil
}

func (m *MockClient) GetIdlePolicy(ctx context.Context, id string) (*idle.PolicyTemplate, error) {
	return &idle.PolicyTemplate{
		ID:   id,
		Name: "Mock Policy",
	}, nil
}

func (m *MockClient) ApplyIdlePolicy(ctx context.Context, instanceID, policyID string) error {
	return nil
}

func (m *MockClient) RemoveIdlePolicy(ctx context.Context, instanceID, policyID string) error {
	return nil
}

func (m *MockClient) GetInstanceIdlePolicies(ctx context.Context, instanceID string) ([]*idle.PolicyTemplate, error) {
	return []*idle.PolicyTemplate{}, nil
}

func (m *MockClient) RecommendIdlePolicy(ctx context.Context, instanceID string) (*idle.PolicyTemplate, error) {
	return &idle.PolicyTemplate{
		ID:   "recommended",
		Name: "Recommended Policy",
	}, nil
}

func (m *MockClient) GetIdleSavingsReport(ctx context.Context, instanceID string) (map[string]interface{}, error) {
	return map[string]interface{}{"savings": 100.0}, nil
}

// Rightsizing analysis operations
func (m *MockClient) AnalyzeRightsizing(ctx context.Context, req types.RightsizingAnalysisRequest) (*types.RightsizingAnalysisResponse, error) {
	return &types.RightsizingAnalysisResponse{}, nil
}

func (m *MockClient) GetRightsizingRecommendations(ctx context.Context) (*types.RightsizingRecommendationsResponse, error) {
	return &types.RightsizingRecommendationsResponse{}, nil
}

func (m *MockClient) GetRightsizingStats(ctx context.Context, instanceID string) (*types.RightsizingStatsResponse, error) {
	return &types.RightsizingStatsResponse{}, nil
}

func (m *MockClient) ExportRightsizingData(ctx context.Context, format string) ([]types.InstanceMetrics, error) {
	return []types.InstanceMetrics{}, nil
}

func (m *MockClient) GetRightsizingSummary(ctx context.Context) (*types.RightsizingSummaryResponse, error) {
	return &types.RightsizingSummaryResponse{}, nil
}

func (m *MockClient) GetInstanceMetrics(ctx context.Context, instanceID string, days int) ([]types.InstanceMetrics, error) {
	return []types.InstanceMetrics{}, nil
}

// AMI operations
func (m *MockClient) ResolveAMI(ctx context.Context, templateName string, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"ami_id": "mock-ami-123"}, nil
}

func (m *MockClient) TestAMIAvailability(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"available": true}, nil
}

func (m *MockClient) GetAMICosts(ctx context.Context, amiID string) (map[string]interface{}, error) {
	return map[string]interface{}{"cost": 0.05}, nil
}

func (m *MockClient) PreviewAMIResolution(ctx context.Context, templateName string) (map[string]interface{}, error) {
	return map[string]interface{}{"ami_id": "mock-ami-preview"}, nil
}

func (m *MockClient) CreateAMI(ctx context.Context, req types.AMICreationRequest) (map[string]interface{}, error) {
	return map[string]interface{}{"ami_id": "mock-created-ami"}, nil
}

func (m *MockClient) GetAMIStatus(ctx context.Context, amiID string) (map[string]interface{}, error) {
	return map[string]interface{}{"status": "available"}, nil
}

func (m *MockClient) ListUserAMIs(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{"amis": []string{}}, nil
}

func (m *MockClient) CleanupAMIs(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"cleaned": 0}, nil
}

func (m *MockClient) DeleteAMI(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

func (m *MockClient) ListAMISnapshots(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"snapshots": []string{}}, nil
}

func (m *MockClient) CreateAMISnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"snapshot_id": "mock-snapshot"}, nil
}

func (m *MockClient) RestoreAMIFromSnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"ami_id": "mock-restored-ami"}, nil
}

func (m *MockClient) DeleteAMISnapshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

// Instance Snapshot operations
func (m *MockClient) CreateInstanceSnapshot(ctx context.Context, req types.InstanceSnapshotRequest) (*types.InstanceSnapshotResult, error) {
	return &types.InstanceSnapshotResult{}, nil
}

func (m *MockClient) ListInstanceSnapshots(ctx context.Context) (*types.InstanceSnapshotListResponse, error) {
	return &types.InstanceSnapshotListResponse{}, nil
}

func (m *MockClient) GetInstanceSnapshot(ctx context.Context, snapshotID string) (*types.InstanceSnapshotInfo, error) {
	return &types.InstanceSnapshotInfo{}, nil
}

func (m *MockClient) DeleteInstanceSnapshot(ctx context.Context, snapshotID string) (*types.InstanceSnapshotDeleteResult, error) {
	return &types.InstanceSnapshotDeleteResult{}, nil
}

func (m *MockClient) RestoreInstanceFromSnapshot(ctx context.Context, snapshotID string, req types.InstanceRestoreRequest) (*types.InstanceRestoreResult, error) {
	return &types.InstanceRestoreResult{}, nil
}

// Backup operations
func (m *MockClient) CreateBackup(ctx context.Context, req types.BackupCreateRequest) (*types.BackupCreateResult, error) {
	return &types.BackupCreateResult{}, nil
}

func (m *MockClient) ListBackups(ctx context.Context) (*types.BackupListResponse, error) {
	return &types.BackupListResponse{}, nil
}

func (m *MockClient) GetBackup(ctx context.Context, backupID string) (*types.BackupInfo, error) {
	return &types.BackupInfo{}, nil
}

func (m *MockClient) DeleteBackup(ctx context.Context, backupID string) (*types.BackupDeleteResult, error) {
	return &types.BackupDeleteResult{}, nil
}

func (m *MockClient) GetBackupContents(ctx context.Context, req types.BackupContentsRequest) (*types.BackupContentsResponse, error) {
	return &types.BackupContentsResponse{}, nil
}

func (m *MockClient) VerifyBackup(ctx context.Context, req types.BackupVerifyRequest) (*types.BackupVerifyResult, error) {
	return &types.BackupVerifyResult{}, nil
}

// Restore operations
func (m *MockClient) RestoreBackup(ctx context.Context, req types.RestoreRequest) (*types.RestoreResult, error) {
	return &types.RestoreResult{}, nil
}

func (m *MockClient) GetRestoreStatus(ctx context.Context, restoreID string) (*types.RestoreResult, error) {
	return &types.RestoreResult{}, nil
}

func (m *MockClient) ListRestoreOperations(ctx context.Context) ([]types.RestoreResult, error) {
	return []types.RestoreResult{}, nil
}

// Version compatibility
func (m *MockClient) CheckVersionCompatibility(ctx context.Context, version string) error {
	return nil
}

// Transfer operations (S3-backed file transfer)
func (m *MockClient) StartTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	return &TransferResponse{
		TransferID: "mock-transfer-id",
		Status:     "in_progress",
	}, nil
}

func (m *MockClient) GetTransferStatus(ctx context.Context, transferID string) (*storage.TransferProgress, error) {
	return &storage.TransferProgress{
		TransferID:       transferID,
		Status:           "in_progress",
		PercentComplete:  50.0,
		TransferredBytes: 500,
		TotalBytes:       1000,
	}, nil
}

func (m *MockClient) ListTransfers(ctx context.Context) ([]*storage.TransferProgress, error) {
	return []*storage.TransferProgress{}, nil
}

func (m *MockClient) CancelTransfer(ctx context.Context, transferID string) error {
	return nil
}

// MakeRequest for generic endpoint access
func (m *MockClient) MakeRequest(method, path string, body interface{}) ([]byte, error) {
	return []byte(`{"success": true}`), nil
}

// v0.12.0 — Budget Enterprise & Governance mocks

func (m *MockClient) GetProjectQuotas(ctx context.Context, projectID string) ([]types.RoleQuota, error) {
	return []types.RoleQuota{}, nil
}

func (m *MockClient) SetProjectQuota(ctx context.Context, projectID string, quota types.RoleQuota) error {
	return nil
}

func (m *MockClient) GetGrantPeriod(ctx context.Context, projectID string) (*types.GrantPeriod, error) {
	return nil, nil
}

func (m *MockClient) SetGrantPeriod(ctx context.Context, projectID string, gp types.GrantPeriod) error {
	return nil
}

func (m *MockClient) DeleteGrantPeriod(ctx context.Context, projectID string) error {
	return nil
}

func (m *MockClient) ShareProjectBudget(ctx context.Context, req types.BudgetShareRequest) (*types.BudgetShareRecord, error) {
	return &types.BudgetShareRecord{ID: "mock-share-id"}, nil
}

func (m *MockClient) ListProjectBudgetShares(ctx context.Context, projectID string) ([]*types.BudgetShareRecord, error) {
	return []*types.BudgetShareRecord{}, nil
}

func (m *MockClient) DeleteProjectBudgetShare(ctx context.Context, projectID, shareID string) error {
	return nil
}

func (m *MockClient) SubmitApproval(ctx context.Context, projectID string, approvalType project.ApprovalType, details map[string]interface{}, reason string) (*project.ApprovalRequest, error) {
	return &project.ApprovalRequest{
		ID:        "mock-approval-id",
		ProjectID: projectID,
		Type:      approvalType,
		Status:    project.ApprovalStatusPending,
		Reason:    reason,
	}, nil
}

func (m *MockClient) ListApprovals(ctx context.Context, projectID string, status project.ApprovalStatus) ([]*project.ApprovalRequest, error) {
	return []*project.ApprovalRequest{}, nil
}

func (m *MockClient) ListAllApprovals(ctx context.Context, status project.ApprovalStatus) ([]*project.ApprovalRequest, error) {
	return []*project.ApprovalRequest{}, nil
}

func (m *MockClient) ApproveRequest(ctx context.Context, projectID, requestID, note string) (*project.ApprovalRequest, error) {
	return &project.ApprovalRequest{
		ID:        requestID,
		ProjectID: projectID,
		Status:    project.ApprovalStatusApproved,
	}, nil
}

func (m *MockClient) DenyRequest(ctx context.Context, projectID, requestID, note string) (*project.ApprovalRequest, error) {
	return &project.ApprovalRequest{
		ID:        requestID,
		ProjectID: projectID,
		Status:    project.ApprovalStatusDenied,
	}, nil
}

func (m *MockClient) GetMonthlyReport(ctx context.Context, projectID, month, format string) (string, error) {
	return "Monthly Report: " + projectID + " " + month, nil
}

func (m *MockClient) ListOnboardingTemplates(ctx context.Context, projectID string) ([]types.OnboardingTemplate, error) {
	return []types.OnboardingTemplate{}, nil
}

func (m *MockClient) AddOnboardingTemplate(ctx context.Context, projectID string, tmpl types.OnboardingTemplate) error {
	return nil
}

func (m *MockClient) DeleteOnboardingTemplate(ctx context.Context, projectID, nameOrID string) error {
	return nil
}

// ─── v0.14.0/v0.16.0 University Education System stubs ───────────────────────

func (m *MockClient) CreateCourse(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) ListCourses(ctx context.Context, params string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) GetCourse(ctx context.Context, courseID string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) CloseCourse(ctx context.Context, courseID string) error {
	return nil
}

func (m *MockClient) DeleteCourse(ctx context.Context, courseID string) error {
	return nil
}

func (m *MockClient) EnrollCourseMember(ctx context.Context, courseID string, req map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) ListCourseMembers(ctx context.Context, courseID string, role string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) UnenrollCourseMember(ctx context.Context, courseID, userID string) error {
	return nil
}

func (m *MockClient) ListCourseTemplates(ctx context.Context, courseID string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) AddCourseTemplate(ctx context.Context, courseID, templateSlug string) error {
	return nil
}

func (m *MockClient) RemoveCourseTemplate(ctx context.Context, courseID, templateSlug string) error {
	return nil
}

func (m *MockClient) GetCourseBudget(ctx context.Context, courseID string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) DistributeCourseBudget(ctx context.Context, courseID string, amountPerStudent float64) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) TADebugStudent(ctx context.Context, courseID, studentID string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) TAResetStudent(ctx context.Context, courseID string, req map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) GetCourseOverview(ctx context.Context, courseID string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) GetCourseReport(ctx context.Context, courseID, format string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) GetCourseAuditLog(ctx context.Context, courseID, studentID, since string, limit int) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) ArchiveCourse(ctx context.Context, courseID string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) ImportCourseRoster(ctx context.Context, courseID string, csvBytes []byte, format string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockClient) ProvisionStudent(ctx context.Context, courseID, studentID string, req map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// ─── v0.18.0 Workshop & Event Management stubs ────────────────────────────────

func (m *MockClient) ListWorkshops(ctx context.Context, params string) (map[string]interface{}, error) {
	return map[string]interface{}{"workshops": []interface{}{}}, nil
}

func (m *MockClient) CreateWorkshop(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"id": "mock-workshop-id", "title": req["title"], "status": "draft", "join_token": "WS-MOCKTOKEN"}, nil
}

func (m *MockClient) GetWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": workshopID, "title": "Mock Workshop", "status": "draft"}, nil
}

func (m *MockClient) UpdateWorkshop(ctx context.Context, workshopID string, req map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"id": workshopID}, nil
}

func (m *MockClient) DeleteWorkshop(ctx context.Context, workshopID string) error {
	return nil
}

func (m *MockClient) ProvisionWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	return map[string]interface{}{"provisioned": 0, "skipped": 0, "errors": []interface{}{}}, nil
}

func (m *MockClient) GetWorkshopDashboard(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	return map[string]interface{}{"workshop_id": workshopID, "total_participants": 0, "active_instances": 0, "stopped_instances": 0, "total_spent": 0.0, "time_remaining": "N/A", "participants": []interface{}{}}, nil
}

func (m *MockClient) EndWorkshop(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	return map[string]interface{}{"stopped": 0, "errors": []interface{}{}}, nil
}

func (m *MockClient) GetWorkshopDownload(ctx context.Context, workshopID string) (map[string]interface{}, error) {
	return map[string]interface{}{"workshop_id": workshopID, "participants": []interface{}{}}, nil
}

func (m *MockClient) ListWorkshopConfigs(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{"configs": []interface{}{}}, nil
}

func (m *MockClient) SaveWorkshopConfig(ctx context.Context, workshopID, name string) (map[string]interface{}, error) {
	return map[string]interface{}{"name": name, "template": "mock-template", "duration_hours": 6, "max_participants": 0}, nil
}

func (m *MockClient) CreateWorkshopFromConfig(ctx context.Context, configName string, req map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"id": "mock-workshop-from-config", "title": req["title"], "join_token": "WS-MOCKTOKEN"}, nil
}

func (m *MockClient) AddWorkshopParticipant(ctx context.Context, workshopID string, req map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"user_id": req["user_id"], "status": "pending"}, nil
}

func (m *MockClient) RemoveWorkshopParticipant(ctx context.Context, workshopID, userID string) error {
	return nil
}
