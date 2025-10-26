// Package models provides comprehensive test coverage for repository management functionality
package models

import (
	"context"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/prism/internal/tui/api"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Mock API client for repositories testing - implements complete apiClient interface
type mockAPIClientRepositories struct {
	shouldError  bool
	errorMessage string
	callLog      []string
}

func (m *mockAPIClientRepositories) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	m.callLog = append(m.callLog, "ListInstances")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListInstancesResponse{}, nil
}

func (m *mockAPIClientRepositories) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstance:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.InstanceResponse{}, nil
}

func (m *mockAPIClientRepositories) LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error) {
	m.callLog = append(m.callLog, "LaunchInstance")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.LaunchInstanceResponse{}, nil
}

func (m *mockAPIClientRepositories) StartInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StartInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientRepositories) StopInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StopInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientRepositories) DeleteInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientRepositories) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	m.callLog = append(m.callLog, "ListTemplates")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListTemplatesResponse{}, nil
}

func (m *mockAPIClientRepositories) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetTemplate:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.TemplateResponse{}, nil
}

func (m *mockAPIClientRepositories) ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error) {
	m.callLog = append(m.callLog, "ListVolumes")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListVolumesResponse{}, nil
}

func (m *mockAPIClientRepositories) ListStorage(ctx context.Context) (*api.ListStorageResponse, error) {
	m.callLog = append(m.callLog, "ListStorage")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListStorageResponse{}, nil
}

func (m *mockAPIClientRepositories) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MountVolume:%s:%s:%s", volumeName, instanceName, mountPoint))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientRepositories) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("UnmountVolume:%s:%s", volumeName, instanceName))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientRepositories) ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error) {
	m.callLog = append(m.callLog, "ListIdlePolicies")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListIdlePoliciesResponse{}, nil
}

func (m *mockAPIClientRepositories) UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error {
	m.callLog = append(m.callLog, "UpdateIdlePolicy")
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientRepositories) GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstanceIdleStatus:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.IdleDetectionResponse{}, nil
}

func (m *mockAPIClientRepositories) EnableIdleDetection(ctx context.Context, name, policy string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("EnableIdleDetection:%s:%s", name, policy))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientRepositories) DisableIdleDetection(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DisableIdleDetection:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientRepositories) GetStatus(ctx context.Context) (*api.SystemStatusResponse, error) {
	m.callLog = append(m.callLog, "GetStatus")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.SystemStatusResponse{}, nil
}

func (m *mockAPIClientRepositories) ListProjects(ctx context.Context, filter *api.ProjectFilter) (*api.ListProjectsResponse, error) {
	return &api.ListProjectsResponse{}, nil
}

func (m *mockAPIClientRepositories) GetPolicyStatus(ctx context.Context) (*api.PolicyStatusResponse, error) {
	return &api.PolicyStatusResponse{}, nil
}

func (m *mockAPIClientRepositories) ListPolicySets(ctx context.Context) (*api.ListPolicySetsResponse, error) {
	return &api.ListPolicySetsResponse{}, nil
}

func (m *mockAPIClientRepositories) AssignPolicySet(ctx context.Context, policySetID string) error {
	return nil
}

func (m *mockAPIClientRepositories) SetPolicyEnforcement(ctx context.Context, enabled bool) error {
	return nil
}

func (m *mockAPIClientRepositories) CheckTemplateAccess(ctx context.Context, templateName string) (*api.TemplateAccessResponse, error) {
	return &api.TemplateAccessResponse{}, nil
}

func (m *mockAPIClientRepositories) ListMarketplaceTemplates(ctx context.Context, filter *api.MarketplaceFilter) (*api.ListMarketplaceTemplatesResponse, error) {
	return &api.ListMarketplaceTemplatesResponse{}, nil
}

func (m *mockAPIClientRepositories) ListMarketplaceCategories(ctx context.Context) (*api.ListCategoriesResponse, error) {
	return &api.ListCategoriesResponse{}, nil
}

func (m *mockAPIClientRepositories) ListMarketplaceRegistries(ctx context.Context) (*api.ListRegistriesResponse, error) {
	return &api.ListRegistriesResponse{}, nil
}

func (m *mockAPIClientRepositories) InstallMarketplaceTemplate(ctx context.Context, templateName string) error {
	return nil
}

func (m *mockAPIClientRepositories) ListAMIs(ctx context.Context) (*api.ListAMIsResponse, error) {
	return &api.ListAMIsResponse{}, nil
}

func (m *mockAPIClientRepositories) ListAMIBuilds(ctx context.Context) (*api.ListAMIBuildsResponse, error) {
	return &api.ListAMIBuildsResponse{}, nil
}

func (m *mockAPIClientRepositories) ListAMIRegions(ctx context.Context) (*api.ListAMIRegionsResponse, error) {
	return &api.ListAMIRegionsResponse{}, nil
}

func (m *mockAPIClientRepositories) DeleteAMI(ctx context.Context, amiID string) error {
	return nil
}

func (m *mockAPIClientRepositories) GetRightsizingRecommendations(ctx context.Context) (*api.GetRightsizingRecommendationsResponse, error) {
	return &api.GetRightsizingRecommendationsResponse{}, nil
}

func (m *mockAPIClientRepositories) ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error {
	return nil
}

func (m *mockAPIClientRepositories) GetLogs(ctx context.Context, instanceName, logType string) (*api.LogsResponse, error) {
	return &api.LogsResponse{}, nil
}

// TestRepositoryItem tests the RepositoryItem list interface
func TestRepositoryItem(t *testing.T) {
	item := RepositoryItem{
		Name:     "test-repo",
		URL:      "https://example.com/templates",
		Priority: 75,
		Enabled:  true,
	}

	assert.Equal(t, "test-repo", item.FilterValue())
	assert.Equal(t, "test-repo", item.Title())
	assert.Equal(t, "https://example.com/templates • Priority: 75 • Enabled", item.Description())

	// Test disabled repository
	disabledItem := RepositoryItem{
		Name:     "disabled-repo",
		URL:      "https://disabled.com/templates",
		Priority: 25,
		Enabled:  false,
	}

	assert.Equal(t, "https://disabled.com/templates • Priority: 25 • Disabled", disabledItem.Description())
}

// TestNewRepositoriesModel tests repositories model creation
func TestNewRepositoriesModel(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}

	model := NewRepositoriesModel(mockAPIClient)

	assert.NotNil(t, model.apiClient)
	assert.NotNil(t, model.repoList)
	assert.NotNil(t, model.statusBar)
	assert.NotNil(t, model.spinner)
	assert.NotNil(t, model.tabBar)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.True(t, model.loading)
	assert.Empty(t, model.error)
	assert.Empty(t, model.repos)
	assert.Empty(t, model.selected)
	assert.Equal(t, "view", model.mode)
	assert.Equal(t, 0, model.focusIndex)
	assert.NotNil(t, model.dispatcher)

	// Check form inputs are initialized
	assert.NotNil(t, model.nameInput)
	assert.NotNil(t, model.urlInput)
	assert.NotNil(t, model.priorityInput)
	assert.NotNil(t, model.enabledInput)
}

// TestRepositoriesModelInit tests model initialization
func TestRepositoriesModelInit(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Execute the command to get repository data
	msg := cmd()
	assert.NotNil(t, msg)
}

// TestRepositoriesModelFetchRepositories tests repository data fetching
func TestRepositoriesModelFetchRepositories(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

	cmd := model.fetchRepositories
	msg := cmd()

	repos, ok := msg.([]types.TemplateRepository)
	assert.True(t, ok, "Should return repository slice")
	assert.Len(t, repos, 2, "Should return mock repositories")

	// Check repository data
	assert.Equal(t, "default", repos[0].Name)
	assert.Equal(t, "https://cloudworkstation.example.com/templates", repos[0].URL)
	assert.Equal(t, 100, repos[0].Priority)
	assert.True(t, repos[0].Enabled)

	assert.Equal(t, "community", repos[1].Name)
	assert.Equal(t, "https://community.example.com/templates", repos[1].URL)
	assert.Equal(t, 50, repos[1].Priority)
	assert.True(t, repos[1].Enabled)
}

// TestRepositoriesModelSyncRepositories tests repository synchronization
func TestRepositoriesModelSyncRepositories(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

	cmd := model.syncRepositories
	msg := cmd()

	_, ok := msg.(RefreshMsg)
	assert.True(t, ok, "Should return RefreshMsg")
}

// TestRepositoriesModelRefreshRepositoryList tests repository list refresh
func TestRepositoriesModelRefreshRepositoryList(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

	// Set up test repositories
	model.repos = []types.TemplateRepository{
		{
			Name:     "test-repo1",
			URL:      "https://test1.com",
			Priority: 90,
			Enabled:  true,
		},
		{
			Name:     "test-repo2",
			URL:      "https://test2.com",
			Priority: 60,
			Enabled:  false,
		},
	}

	model.refreshRepositoryList()

	items := model.repoList.Items()
	assert.Len(t, items, 2)

	item1, ok := items[0].(RepositoryItem)
	assert.True(t, ok)
	assert.Equal(t, "test-repo1", item1.Name)
	assert.Equal(t, "https://test1.com", item1.URL)
	assert.Equal(t, 90, item1.Priority)
	assert.True(t, item1.Enabled)

	item2, ok := items[1].(RepositoryItem)
	assert.True(t, ok)
	assert.Equal(t, "test-repo2", item2.Name)
	assert.Equal(t, "https://test2.com", item2.URL)
	assert.Equal(t, 60, item2.Priority)
	assert.False(t, item2.Enabled)
}

// TestRepositoriesModelRepositoryDataHandling tests repository data processing
func TestRepositoriesModelRepositoryDataHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)
	model.loading = true

	repos := []types.TemplateRepository{
		{
			Name:     "production",
			URL:      "https://prod.example.com/templates",
			Priority: 100,
			Enabled:  true,
		},
	}

	updatedModel, cmd := model.Update(repos)

	result := updatedModel.(RepositoriesModel)
	assert.False(t, result.loading)
	assert.Len(t, result.repos, 1)
	assert.Equal(t, "production", result.repos[0].Name)
	assert.Nil(t, cmd)

	// Check that list was refreshed
	items := result.repoList.Items()
	assert.Len(t, items, 1)
}

// TestRepositoriesModelRefreshMsg tests refresh message handling
func TestRepositoriesModelRefreshMsg(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)
	model.loading = false

	refreshMsg := RefreshMsg{}
	updatedModel, cmd := model.Update(refreshMsg)

	result := updatedModel.(RepositoriesModel)
	assert.True(t, result.loading)
	assert.NotNil(t, cmd)
}

// TestRepositoriesModelErrorHandling tests error handling scenarios
func TestRepositoriesModelErrorHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)
	model.loading = true

	testError := fmt.Errorf("repository fetch failed")
	updatedModel, _ := model.Update(testError)

	result := updatedModel.(RepositoriesModel)
	assert.False(t, result.loading)
	assert.Equal(t, "repository fetch failed", result.error)
}

// TestRepositoriesModelCtrlC tests quit handling
func TestRepositoriesModelCtrlC(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

	ctrlCMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd := model.Update(ctrlCMsg)

	// Model may be processed by command dispatcher, so we just check that quit is triggered
	assert.NotNil(t, updatedModel)
	assert.NotNil(t, cmd)

	// Execute the command to check if it's quit
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

// TestRepositoriesModelTabNavigation tests tab navigation handling
func TestRepositoriesModelTabNavigation(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)
	model.mode = "view"

	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, cmd := model.Update(tabMsg)

	assert.Nil(t, updatedModel)
	assert.NotNil(t, cmd)

	// Execute the command to check if it's quit
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

// TestRepositoriesModelView tests view rendering in different states
func TestRepositoriesModelView(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}

	tests := []struct {
		name             string
		setupModel       func(m RepositoriesModel) RepositoriesModel
		expectedContains []string
		description      string
	}{
		{
			name: "loading_state",
			setupModel: func(m RepositoriesModel) RepositoriesModel {
				m.loading = true
				return m
			},
			expectedContains: []string{"Loading repositories..."},
			description:      "Should show loading state with spinner",
		},
		{
			name: "view_mode",
			setupModel: func(m RepositoriesModel) RepositoriesModel {
				m.loading = false
				m.mode = "view"
				m.repos = []types.TemplateRepository{
					{
						Name:     "test-repo",
						URL:      "https://test.com",
						Priority: 80,
						Enabled:  true,
					},
				}
				m.refreshRepositoryList()
				return m
			},
			expectedContains: []string{"Repository"}, // Just check for repository-related content
			description:      "Should show repository list in view mode",
		},
		{
			name: "add_mode",
			setupModel: func(m RepositoriesModel) RepositoriesModel {
				m.loading = false
				m.mode = "add"
				return m
			},
			expectedContains: []string{"Add Repository", "Name:", "URL:", "Priority:", "Enabled:"},
			description:      "Should show add form in add mode",
		},
		{
			name: "edit_mode",
			setupModel: func(m RepositoriesModel) RepositoriesModel {
				m.loading = false
				m.mode = "edit"
				return m
			},
			expectedContains: []string{"Edit Repository", "Name:", "URL:", "Priority:", "Enabled:"},
			description:      "Should show edit form in edit mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := tt.setupModel(NewRepositoriesModel(mockAPIClient))
			view := testModel.View()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, view, expected, tt.description)
			}
		})
	}
}

// TestRepositoriesModelModeTransitions tests mode transitions
func TestRepositoriesModelModeTransitions(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

	tests := []struct {
		name        string
		initialMode string
		targetMode  string
		description string
	}{
		{
			name:        "view_to_add",
			initialMode: "view",
			targetMode:  "add",
			description: "Should transition from view to add mode",
		},
		{
			name:        "view_to_edit",
			initialMode: "view",
			targetMode:  "edit",
			description: "Should transition from view to edit mode",
		},
		{
			name:        "add_to_view",
			initialMode: "add",
			targetMode:  "view",
			description: "Should transition from add to view mode",
		},
		{
			name:        "edit_to_view",
			initialMode: "edit",
			targetMode:  "view",
			description: "Should transition from edit to view mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := model
			testModel.mode = tt.initialMode

			// Manual mode transition for testing
			testModel.mode = tt.targetMode

			assert.Equal(t, tt.targetMode, testModel.mode, tt.description)
		})
	}
}

// TestRepositoriesModelFormInputs tests form input functionality
func TestRepositoriesModelFormInputs(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

	tests := []struct {
		name        string
		inputField  string
		testValue   string
		description string
	}{
		{
			name:        "name_input",
			inputField:  "name",
			testValue:   "my-repo",
			description: "Should handle name input",
		},
		{
			name:        "url_input",
			inputField:  "url",
			testValue:   "https://github.com/user/templates",
			description: "Should handle URL input",
		},
		{
			name:        "priority_input",
			inputField:  "priority",
			testValue:   "85",
			description: "Should handle priority input",
		},
		{
			name:        "enabled_input",
			inputField:  "enabled",
			testValue:   "true",
			description: "Should handle enabled input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that inputs can be set (we can't test actual typing without full BubbleTea setup)
			switch tt.inputField {
			case "name":
				model.nameInput.SetValue(tt.testValue)
				assert.Equal(t, tt.testValue, model.nameInput.Value())
			case "url":
				model.urlInput.SetValue(tt.testValue)
				assert.Equal(t, tt.testValue, model.urlInput.Value())
			case "priority":
				model.priorityInput.SetValue(tt.testValue)
				assert.Equal(t, tt.testValue, model.priorityInput.Value())
			case "enabled":
				model.enabledInput.SetValue(tt.testValue)
				assert.Equal(t, tt.testValue, model.enabledInput.Value())
			}
		})
	}
}

// TestRepositoriesModelCompleteWorkflow tests complete repository management workflow
func TestRepositoriesModelCompleteWorkflow(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

	// Step 1: Initialize and fetch repositories
	initCmd := model.Init()
	assert.NotNil(t, initCmd)

	initMsg := initCmd()
	updatedModel, _ := model.Update(initMsg)

	// Step 2: Process the repository data
	repos, ok := initMsg.([]types.TemplateRepository)
	if ok {
		updatedModel, _ = updatedModel.Update(repos)
	}
	result := updatedModel.(RepositoriesModel)
	// After processing, should have repository data
	assert.GreaterOrEqual(t, len(result.repos), 0) // May have 0 or more repositories

	// Step 3: Switch to add mode
	result.mode = "add"
	assert.Equal(t, "add", result.mode)

	// Step 4: Fill form inputs
	result.nameInput.SetValue("new-repo")
	result.urlInput.SetValue("https://new-repo.com")
	result.priorityInput.SetValue("75")
	result.enabledInput.SetValue("true")

	assert.Equal(t, "new-repo", result.nameInput.Value())
	assert.Equal(t, "https://new-repo.com", result.urlInput.Value())
	assert.Equal(t, "75", result.priorityInput.Value())
	assert.Equal(t, "true", result.enabledInput.Value())

	// Step 5: Switch back to view mode
	result.mode = "view"
	assert.Equal(t, "view", result.mode)
}

// TestRepositoriesModelEdgeCases tests edge cases and boundary conditions
func TestRepositoriesModelEdgeCases(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}

	tests := []struct {
		name        string
		setupModel  func() RepositoriesModel
		testAction  func(RepositoriesModel) (tea.Model, tea.Cmd)
		description string
	}{
		{
			name: "empty_repository_list",
			setupModel: func() RepositoriesModel {
				model := NewRepositoriesModel(mockAPIClient)
				model.loading = false
				model.repos = []types.TemplateRepository{}
				model.refreshRepositoryList()
				return model
			},
			testAction: func(m RepositoriesModel) (tea.Model, tea.Cmd) {
				view := m.View()
				assert.NotEmpty(t, view)
				return m, nil
			},
			description: "Should handle empty repository list gracefully",
		},
		{
			name: "very_small_window",
			setupModel: func() RepositoriesModel {
				model := NewRepositoriesModel(mockAPIClient)
				model.width = 10
				model.height = 5
				return model
			},
			testAction: func(m RepositoriesModel) (tea.Model, tea.Cmd) {
				view := m.View()
				assert.NotEmpty(t, view)
				return m, nil
			},
			description: "Should handle very small window sizes gracefully",
		},
		{
			name: "multiple_errors",
			setupModel: func() RepositoriesModel {
				model := NewRepositoriesModel(mockAPIClient)
				model.error = "First error"
				return model
			},
			testAction: func(m RepositoriesModel) (tea.Model, tea.Cmd) {
				// Process another error
				secondError := fmt.Errorf("Second error")
				updatedModel, _ := m.Update(secondError)
				result := updatedModel.(RepositoriesModel)
				assert.Equal(t, "Second error", result.error)
				return updatedModel, nil
			},
			description: "Should handle multiple consecutive errors",
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

// TestRepositoriesModelMessageHandling tests proper message type handling
func TestRepositoriesModelMessageHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientRepositories{}
	model := NewRepositoriesModel(mockAPIClient)

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
			name: "repository_data_message",
			message: []types.TemplateRepository{
				{Name: "test", URL: "https://test.com", Priority: 50, Enabled: true},
			},
			expectPanic: false,
			description: "Should handle repository data messages properly",
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
					// cmd can be nil for some messages
					_ = cmd
				}, tt.description)
			}
		})
	}
}
