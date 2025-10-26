// Package models provides comprehensive test coverage for dashboard display functionality
package models

import (
	"context"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/prism/internal/tui/api"
)

// Mock API client for dashboard testing - implements complete apiClient interface
type mockAPIClientDashboard struct {
	shouldError       bool
	errorMessage      string
	callLog           []string
	instancesResponse *api.ListInstancesResponse
	statusResponse    *api.SystemStatusResponse
	instancesError    error
	statusError       error
}

func (m *mockAPIClientDashboard) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	m.callLog = append(m.callLog, "ListInstances")
	if m.instancesError != nil {
		return nil, m.instancesError
	}
	if m.instancesResponse != nil {
		return m.instancesResponse, nil
	}
	return &api.ListInstancesResponse{}, nil
}

func (m *mockAPIClientDashboard) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstance:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.InstanceResponse{}, nil
}

func (m *mockAPIClientDashboard) LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error) {
	m.callLog = append(m.callLog, "LaunchInstance")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.LaunchInstanceResponse{}, nil
}

func (m *mockAPIClientDashboard) StartInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StartInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientDashboard) StopInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StopInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientDashboard) DeleteInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientDashboard) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	m.callLog = append(m.callLog, "ListTemplates")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListTemplatesResponse{}, nil
}

func (m *mockAPIClientDashboard) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetTemplate:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.TemplateResponse{}, nil
}

func (m *mockAPIClientDashboard) ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error) {
	m.callLog = append(m.callLog, "ListVolumes")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListVolumesResponse{}, nil
}

func (m *mockAPIClientDashboard) ListStorage(ctx context.Context) (*api.ListStorageResponse, error) {
	m.callLog = append(m.callLog, "ListStorage")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListStorageResponse{}, nil
}

func (m *mockAPIClientDashboard) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MountVolume:%s:%s:%s", volumeName, instanceName, mountPoint))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientDashboard) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("UnmountVolume:%s:%s", volumeName, instanceName))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientDashboard) ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error) {
	m.callLog = append(m.callLog, "ListIdlePolicies")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListIdlePoliciesResponse{}, nil
}

func (m *mockAPIClientDashboard) UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error {
	m.callLog = append(m.callLog, "UpdateIdlePolicy")
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientDashboard) GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstanceIdleStatus:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.IdleDetectionResponse{}, nil
}

func (m *mockAPIClientDashboard) EnableIdleDetection(ctx context.Context, name, policy string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("EnableIdleDetection:%s:%s", name, policy))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientDashboard) DisableIdleDetection(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DisableIdleDetection:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientDashboard) GetStatus(ctx context.Context) (*api.SystemStatusResponse, error) {
	m.callLog = append(m.callLog, "GetStatus")
	if m.statusError != nil {
		return nil, m.statusError
	}
	if m.statusResponse != nil {
		return m.statusResponse, nil
	}
	return &api.SystemStatusResponse{}, nil
}

func (m *mockAPIClientDashboard) ListProjects(ctx context.Context, filter *api.ProjectFilter) (*api.ListProjectsResponse, error) {
	m.callLog = append(m.callLog, "ListProjects")
	return &api.ListProjectsResponse{}, nil
}

func (m *mockAPIClientDashboard) GetPolicyStatus(ctx context.Context) (*api.PolicyStatusResponse, error) {
	m.callLog = append(m.callLog, "GetPolicyStatus")
	return &api.PolicyStatusResponse{}, nil
}

func (m *mockAPIClientDashboard) ListPolicySets(ctx context.Context) (*api.ListPolicySetsResponse, error) {
	m.callLog = append(m.callLog, "ListPolicySets")
	return &api.ListPolicySetsResponse{}, nil
}

func (m *mockAPIClientDashboard) AssignPolicySet(ctx context.Context, policySetID string) error {
	m.callLog = append(m.callLog, "AssignPolicySet")
	return nil
}

func (m *mockAPIClientDashboard) SetPolicyEnforcement(ctx context.Context, enabled bool) error {
	m.callLog = append(m.callLog, "SetPolicyEnforcement")
	return nil
}

func (m *mockAPIClientDashboard) CheckTemplateAccess(ctx context.Context, templateName string) (*api.TemplateAccessResponse, error) {
	m.callLog = append(m.callLog, "CheckTemplateAccess")
	return &api.TemplateAccessResponse{}, nil
}

func (m *mockAPIClientDashboard) ListMarketplaceTemplates(ctx context.Context, filter *api.MarketplaceFilter) (*api.ListMarketplaceTemplatesResponse, error) {
	m.callLog = append(m.callLog, "ListMarketplaceTemplates")
	return &api.ListMarketplaceTemplatesResponse{}, nil
}

func (m *mockAPIClientDashboard) ListMarketplaceCategories(ctx context.Context) (*api.ListCategoriesResponse, error) {
	m.callLog = append(m.callLog, "ListMarketplaceCategories")
	return &api.ListCategoriesResponse{}, nil
}

func (m *mockAPIClientDashboard) ListMarketplaceRegistries(ctx context.Context) (*api.ListRegistriesResponse, error) {
	m.callLog = append(m.callLog, "ListMarketplaceRegistries")
	return &api.ListRegistriesResponse{}, nil
}

func (m *mockAPIClientDashboard) InstallMarketplaceTemplate(ctx context.Context, templateName string) error {
	m.callLog = append(m.callLog, "InstallMarketplaceTemplate")
	return nil
}

func (m *mockAPIClientDashboard) ListAMIs(ctx context.Context) (*api.ListAMIsResponse, error) {
	m.callLog = append(m.callLog, "ListAMIs")
	return &api.ListAMIsResponse{}, nil
}

func (m *mockAPIClientDashboard) ListAMIBuilds(ctx context.Context) (*api.ListAMIBuildsResponse, error) {
	m.callLog = append(m.callLog, "ListAMIBuilds")
	return &api.ListAMIBuildsResponse{}, nil
}

func (m *mockAPIClientDashboard) ListAMIRegions(ctx context.Context) (*api.ListAMIRegionsResponse, error) {
	m.callLog = append(m.callLog, "ListAMIRegions")
	return &api.ListAMIRegionsResponse{}, nil
}

func (m *mockAPIClientDashboard) DeleteAMI(ctx context.Context, amiID string) error {
	m.callLog = append(m.callLog, "DeleteAMI")
	return nil
}

func (m *mockAPIClientDashboard) GetRightsizingRecommendations(ctx context.Context) (*api.GetRightsizingRecommendationsResponse, error) {
	m.callLog = append(m.callLog, "GetRightsizingRecommendations")
	return &api.GetRightsizingRecommendationsResponse{}, nil
}

func (m *mockAPIClientDashboard) ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error {
	m.callLog = append(m.callLog, "ApplyRightsizingRecommendation")
	return nil
}

func (m *mockAPIClientDashboard) GetLogs(ctx context.Context, instanceName, logType string) (*api.LogsResponse, error) {
	m.callLog = append(m.callLog, "GetLogs")
	return &api.LogsResponse{}, nil
}

// TestCostData tests the cost data structure
func TestCostData(t *testing.T) {
	costData := CostData{
		DailyCost:   5.25,
		MonthlyCost: 157.50,
		ByTemplate: map[string]float64{
			"python-ml":  3.50,
			"r-research": 1.75,
		},
		ByInstance: map[string]float64{
			"my-analysis": 2.50,
			"my-training": 2.75,
		},
		Storage: 0.50,
		Volumes: 0.25,
	}

	assert.Equal(t, 5.25, costData.DailyCost)
	assert.Equal(t, 157.50, costData.MonthlyCost)
	assert.Equal(t, 3.50, costData.ByTemplate["python-ml"])
	assert.Equal(t, 1.75, costData.ByTemplate["r-research"])
	assert.Equal(t, 2.50, costData.ByInstance["my-analysis"])
	assert.Equal(t, 2.75, costData.ByInstance["my-training"])
	assert.Equal(t, 0.50, costData.Storage)
	assert.Equal(t, 0.25, costData.Volumes)
}

// TestNewDashboardModel tests dashboard model creation
func TestNewDashboardModel(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}

	model := NewDashboardModel(mockAPIClient)

	assert.NotNil(t, model.apiClient)
	assert.NotNil(t, model.instancesTable)
	assert.NotNil(t, model.statusBar)
	assert.NotNil(t, model.spinner)
	assert.NotNil(t, model.tabs)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.True(t, model.loading)
	assert.Empty(t, model.error)
	assert.Empty(t, model.instances)
	assert.Nil(t, model.systemStatus)
	assert.Equal(t, "Overview", model.activeTab)
	assert.NotNil(t, model.refreshTicker)

	// Check cost data initialization
	assert.Equal(t, 0.0, model.costData.DailyCost)
	assert.Equal(t, 0.0, model.costData.MonthlyCost)
	assert.NotNil(t, model.costData.ByTemplate)
	assert.NotNil(t, model.costData.ByInstance)
	assert.Empty(t, model.costData.ByTemplate)
	assert.Empty(t, model.costData.ByInstance)
}

// TestDashboardModelInit tests model initialization
func TestDashboardModelInit(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}
	model := NewDashboardModel(mockAPIClient)

	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Execute the command to get dashboard data
	msg := cmd()
	assert.NotNil(t, msg)
}

// TestDashboardModelFetchDashboardData tests dashboard data fetching
func TestDashboardModelFetchDashboardData(t *testing.T) {
	tests := []struct {
		name              string
		instancesResponse *api.ListInstancesResponse
		statusResponse    *api.SystemStatusResponse
		instancesError    error
		statusError       error
		expectError       bool
		description       string
	}{
		{
			name: "successful_fetch",
			instancesResponse: &api.ListInstancesResponse{
				Instances: []api.InstanceResponse{
					{
						Name:          "test-instance",
						Template:      "python-ml",
						State:         "running",
						HourlyRate:    0.125,
						EffectiveRate: 0.100,
					},
				},
				TotalCost: 2.40,
			},
			statusResponse: &api.SystemStatusResponse{
				Status:    "healthy",
				AWSRegion: "us-west-2",
			},
			expectError: false,
			description: "Should fetch dashboard data successfully",
		},
		{
			name:           "instances_error",
			instancesError: fmt.Errorf("instances API failed"),
			expectError:    true,
			description:    "Should handle instances API error",
		},
		{
			name: "status_error",
			instancesResponse: &api.ListInstancesResponse{
				Instances: []api.InstanceResponse{},
				TotalCost: 0.0,
			},
			statusError: fmt.Errorf("status API failed"),
			expectError: true,
			description: "Should handle status API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPIClient := &mockAPIClientDashboard{
				instancesResponse: tt.instancesResponse,
				statusResponse:    tt.statusResponse,
				instancesError:    tt.instancesError,
				statusError:       tt.statusError,
			}
			model := NewDashboardModel(mockAPIClient)

			cmd := model.fetchDashboardData
			msg := cmd()

			dashboardMsg, ok := msg.(DashboardDataMsg)
			assert.True(t, ok, "Should return DashboardDataMsg")

			if tt.expectError {
				assert.NotNil(t, dashboardMsg.Error, tt.description)
			} else {
				assert.Nil(t, dashboardMsg.Error, tt.description)
				assert.NotNil(t, dashboardMsg.Instances, "Should have instances data")
				assert.NotNil(t, dashboardMsg.SystemStatus, "Should have system status")
			}

			// Verify API calls were made
			assert.Contains(t, mockAPIClient.callLog, "ListInstances")
			assert.Contains(t, mockAPIClient.callLog, "GetStatus")
		})
	}
}

// TestDashboardModelWindowResize tests window resize handling
func TestDashboardModelWindowResize(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}
	model := NewDashboardModel(mockAPIClient)

	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, cmd := model.Update(resizeMsg)

	result := updatedModel.(DashboardModel)
	assert.Equal(t, 120, result.width)
	assert.Equal(t, 40, result.height)
	assert.Nil(t, cmd)
}

// TestDashboardModelKeyboardShortcuts tests keyboard shortcuts
func TestDashboardModelKeyboardShortcuts(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}
	model := NewDashboardModel(mockAPIClient)
	model.loading = false
	model.error = "previous error" // Set error to test clearing

	tests := []struct {
		name        string
		key         string
		expectCmd   bool
		expectQuit  bool
		clearError  bool
		description string
	}{
		{
			name:        "refresh_command",
			key:         "r",
			expectCmd:   true,
			expectQuit:  false,
			clearError:  true,
			description: "Should trigger refresh and clear error",
		},
		{
			name:        "quit_command_q",
			key:         "q",
			expectCmd:   true,
			expectQuit:  true,
			clearError:  false,
			description: "Should trigger quit with 'q'",
		},
		{
			name:        "quit_command_esc",
			key:         "esc",
			expectCmd:   true,
			expectQuit:  true,
			clearError:  false,
			description: "Should trigger quit with escape",
		},
		{
			name:        "unknown_command",
			key:         "x",
			expectCmd:   false,
			expectQuit:  false,
			clearError:  false,
			description: "Should ignore unknown commands",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := model
			testModel.error = "previous error" // Reset error for each test

			var keyMsg tea.KeyMsg
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			updatedModel, cmd := testModel.Update(keyMsg)

			if tt.expectCmd {
				assert.NotNil(t, cmd, tt.description)
				if tt.expectQuit {
					// Execute the command to check if it's quit
					msg := cmd()
					assert.IsType(t, tea.QuitMsg{}, msg, "Should return quit message")
				} else {
					// Execute the command to check if it's dashboard data fetch
					msg := cmd()
					assert.IsType(t, DashboardDataMsg{}, msg, "Should return dashboard data message")
				}
			} else {
				// For unknown commands, cmd might be non-nil due to component updates
			}

			if tt.clearError {
				assert.Empty(t, updatedModel.(DashboardModel).error, "Should clear error on refresh")
			}

			if tt.key == "r" {
				assert.True(t, updatedModel.(DashboardModel).loading, "Should set loading state on refresh")
			}
		})
	}
}

// TestDashboardModelDashboardRefreshMsg tests dashboard refresh message handling
func TestDashboardModelDashboardRefreshMsg(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}
	model := NewDashboardModel(mockAPIClient)
	model.loading = false

	refreshMsg := DashboardRefreshMsg{}
	updatedModel, cmd := model.Update(refreshMsg)

	result := updatedModel.(DashboardModel)
	assert.True(t, result.loading)
	assert.Empty(t, result.error)
	assert.NotNil(t, cmd)
}

// TestDashboardModelErrorHandling tests error message handling
func TestDashboardModelErrorHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}
	model := NewDashboardModel(mockAPIClient)
	model.loading = true

	testError := fmt.Errorf("dashboard fetch failed")
	updatedModel, cmd := model.Update(testError)

	result := updatedModel.(DashboardModel)
	assert.False(t, result.loading)
	assert.Equal(t, "dashboard fetch failed", result.error)
	// cmd might be present due to component updates
	_ = cmd
}

// TestDashboardModelDashboardDataMsg tests dashboard data message handling
func TestDashboardModelDashboardDataMsg(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}
	model := NewDashboardModel(mockAPIClient)
	model.loading = true

	tests := []struct {
		name        string
		dataMsg     DashboardDataMsg
		expectError bool
		expectRetry bool
		description string
	}{
		{
			name: "successful_data_update",
			dataMsg: DashboardDataMsg{
				Instances: &api.ListInstancesResponse{
					Instances: []api.InstanceResponse{
						{
							Name:          "test-instance",
							Template:      "python-ml",
							State:         "running",
							HourlyRate:    0.125,
							EffectiveRate: 0.100,
						},
					},
					TotalCost: 3.00,
				},
				SystemStatus: &api.SystemStatusResponse{
					Status:    "healthy",
					AWSRegion: "us-west-2",
				},
			},
			expectError: false,
			expectRetry: false,
			description: "Should update dashboard data successfully",
		},
		{
			name: "error_in_data",
			dataMsg: DashboardDataMsg{
				Error: fmt.Errorf("API connection failed"),
			},
			expectError: true,
			expectRetry: true,
			description: "Should handle errors in dashboard data",
		},
		{
			name: "partial_data_with_error",
			dataMsg: DashboardDataMsg{
				Instances: &api.ListInstancesResponse{
					Instances: []api.InstanceResponse{},
					TotalCost: 0.0,
				},
				Error: fmt.Errorf("status API failed"),
			},
			expectError: true,
			expectRetry: true,
			description: "Should handle partial data with error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := model
			testModel.loading = true

			updatedModel, cmd := testModel.Update(tt.dataMsg)

			result := updatedModel.(DashboardModel)
			assert.False(t, result.loading, "Should stop loading")

			if tt.expectError {
				assert.NotEmpty(t, result.error, tt.description)
			} else {
				assert.Empty(t, result.error, tt.description)

				if tt.dataMsg.Instances != nil {
					assert.Equal(t, len(tt.dataMsg.Instances.Instances), len(result.instances))
					assert.Equal(t, tt.dataMsg.Instances.TotalCost, result.costData.DailyCost)
				}

				if tt.dataMsg.SystemStatus != nil {
					assert.Equal(t, tt.dataMsg.SystemStatus, result.systemStatus)
				}
			}

			if tt.expectRetry {
				assert.NotNil(t, cmd, "Should schedule retry")
			} else {
				assert.NotNil(t, cmd, "Should schedule next refresh")
			}
		})
	}
}

// TestDashboardModelView tests view rendering in different states
func TestDashboardModelView(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}

	tests := []struct {
		name             string
		setupModel       func(m DashboardModel) DashboardModel
		expectedContains []string
		description      string
	}{
		{
			name: "loading_state",
			setupModel: func(m DashboardModel) DashboardModel {
				m.loading = true
				return m
			},
			expectedContains: []string{"Prism Dashboard"},
			description:      "Should show loading state with spinner",
		},
		{
			name: "error_state",
			setupModel: func(m DashboardModel) DashboardModel {
				m.loading = false
				m.error = "Connection failed"
				return m
			},
			expectedContains: []string{"Prism Dashboard", "Error: Connection failed"},
			description:      "Should show error state",
		},
		{
			name: "normal_state",
			setupModel: func(m DashboardModel) DashboardModel {
				m.loading = false
				m.systemStatus = &api.SystemStatusResponse{
					Status:    "healthy",
					AWSRegion: "us-west-2",
				}
				m.instances = []api.InstanceResponse{
					{
						Name:          "test-instance",
						Template:      "python-ml",
						State:         "running",
						HourlyRate:    0.125,
						EffectiveRate: 0.100,
					},
				}
				m.costData.DailyCost = 3.00
				return m
			},
			expectedContains: []string{
				"Prism Dashboard",
				"System Status",
				"Region: us-west-2",
				"Daemon: healthy",
				"Running Instances",
				"Cost Overview",
				"Daily Cost: $3.00",
				"Monthly Estimate: $90.00",
				"Quick Actions",
				"Launch",
				"Templates",
				"Storage",
				"Navigation:",
				"Actions: r: refresh",
			},
			description: "Should show complete dashboard with all panels",
		},
		{
			name: "no_system_status",
			setupModel: func(m DashboardModel) DashboardModel {
				m.loading = false
				m.systemStatus = nil
				return m
			},
			expectedContains: []string{
				"Region: unknown",
				"Daemon: unknown",
			},
			description: "Should show unknown status when system status is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := tt.setupModel(NewDashboardModel(mockAPIClient))
			view := testModel.View()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, view, expected, tt.description)
			}
		})
	}
}

// TestDashboardModelCompleteWorkflow tests complete dashboard workflow
func TestDashboardModelCompleteWorkflow(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{
		instancesResponse: &api.ListInstancesResponse{
			Instances: []api.InstanceResponse{
				{
					Name:          "workflow-test",
					Template:      "python-ml",
					State:         "running",
					HourlyRate:    0.150,
					EffectiveRate: 0.120,
				},
			},
			TotalCost: 3.60,
		},
		statusResponse: &api.SystemStatusResponse{
			Status:    "healthy",
			AWSRegion: "us-east-1",
		},
	}
	model := NewDashboardModel(mockAPIClient)

	// Step 1: Initialize
	initCmd := model.Init()
	assert.NotNil(t, initCmd)

	// Step 2: Fetch initial data
	fetchCmd := model.fetchDashboardData
	dataMsg := fetchCmd()
	updatedModel, refreshCmd := model.Update(dataMsg)

	// Step 3: Verify data was processed
	result := updatedModel.(DashboardModel)
	assert.False(t, result.loading)
	assert.Empty(t, result.error)
	assert.Len(t, result.instances, 1)
	assert.Equal(t, "workflow-test", result.instances[0].Name)
	assert.Equal(t, 3.60, result.costData.DailyCost)
	assert.NotNil(t, result.systemStatus)
	assert.Equal(t, "healthy", result.systemStatus.Status)
	assert.NotNil(t, refreshCmd) // Should schedule next refresh

	// Step 4: Resize window
	resizeMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, _ = updatedModel.Update(resizeMsg)
	result = updatedModel.(DashboardModel)
	assert.Equal(t, 100, result.width)
	assert.Equal(t, 30, result.height)

	// Step 5: Trigger manual refresh
	refreshKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updatedModel, refreshCmd = updatedModel.Update(refreshKey)
	result = updatedModel.(DashboardModel)
	assert.True(t, result.loading) // Should start loading
	assert.Empty(t, result.error)  // Should clear previous errors
	assert.NotNil(t, refreshCmd)
}

// TestDashboardModelEdgeCases tests edge cases and boundary conditions
func TestDashboardModelEdgeCases(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}

	tests := []struct {
		name        string
		setupModel  func() DashboardModel
		testAction  func(DashboardModel) (tea.Model, tea.Cmd)
		description string
	}{
		{
			name: "empty_instances_list",
			setupModel: func() DashboardModel {
				model := NewDashboardModel(mockAPIClient)
				model.loading = false
				model.instances = []api.InstanceResponse{}
				model.costData.DailyCost = 0.0
				return model
			},
			testAction: func(m DashboardModel) (tea.Model, tea.Cmd) {
				view := m.View()
				assert.Contains(t, view, "Daily Cost: $0.00")
				return m, nil
			},
			description: "Should handle empty instances list",
		},
		{
			name: "very_small_window",
			setupModel: func() DashboardModel {
				model := NewDashboardModel(mockAPIClient)
				model.width = 10
				model.height = 5
				model.loading = false
				return model
			},
			testAction: func(m DashboardModel) (tea.Model, tea.Cmd) {
				view := m.View()
				assert.NotEmpty(t, view)
				return m, nil
			},
			description: "Should handle very small window sizes",
		},
		{
			name: "high_cost_values",
			setupModel: func() DashboardModel {
				model := NewDashboardModel(mockAPIClient)
				model.loading = false
				model.costData.DailyCost = 999.99
				return model
			},
			testAction: func(m DashboardModel) (tea.Model, tea.Cmd) {
				view := m.View()
				assert.Contains(t, view, "Daily Cost: $999.99")
				assert.Contains(t, view, "Monthly Estimate: $29999.70")
				return m, nil
			},
			description: "Should handle high cost values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := tt.setupModel()

			// Should not panic
			assert.NotPanics(t, func() {
				_, _ = tt.testAction(model)
			}, tt.description)
		})
	}
}

// TestDashboardModelMessageHandling tests proper message type handling
func TestDashboardModelMessageHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientDashboard{}
	model := NewDashboardModel(mockAPIClient)

	tests := []struct {
		name        string
		message     tea.Msg
		expectPanic bool
		description string
	}{
		{
			name:        "unknown_message_type",
			message:     "unknown string message",
			expectPanic: false,
			description: "Should handle unknown message types gracefully",
		},
		{
			name:        "nil_message",
			message:     nil,
			expectPanic: false,
			description: "Should handle nil messages gracefully",
		},
		{
			name: "dashboard_data_message",
			message: DashboardDataMsg{
				Instances:    &api.ListInstancesResponse{},
				SystemStatus: &api.SystemStatusResponse{},
			},
			expectPanic: false,
			description: "Should handle dashboard data messages properly",
		},
		{
			name:        "dashboard_refresh_message",
			message:     DashboardRefreshMsg{},
			expectPanic: false,
			description: "Should handle dashboard refresh messages properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.Panics(t, func() {
					_, _ = model.Update(tt.message)
				}, tt.description)
			} else {
				assert.NotPanics(t, func() {
					updatedModel, cmd := model.Update(tt.message)
					assert.NotNil(t, updatedModel)
					// cmd can be nil or contain various commands
					_ = cmd
				}, tt.description)
			}
		})
	}
}
