// Package models provides comprehensive test coverage for settings display functionality
package models

import (
	"context"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
)

// Mock API client for settings testing - implements complete apiClient interface
type mockAPIClientSettings struct {
	shouldError  bool
	errorMessage string
	callLog      []string
}

func (m *mockAPIClientSettings) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	m.callLog = append(m.callLog, "ListInstances")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListInstancesResponse{}, nil
}

func (m *mockAPIClientSettings) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstance:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.InstanceResponse{}, nil
}

func (m *mockAPIClientSettings) LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error) {
	m.callLog = append(m.callLog, "LaunchInstance")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.LaunchInstanceResponse{}, nil
}

func (m *mockAPIClientSettings) StartInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StartInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientSettings) StopInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StopInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientSettings) DeleteInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientSettings) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	m.callLog = append(m.callLog, "ListTemplates")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListTemplatesResponse{}, nil
}

func (m *mockAPIClientSettings) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetTemplate:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.TemplateResponse{}, nil
}

func (m *mockAPIClientSettings) ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error) {
	m.callLog = append(m.callLog, "ListVolumes")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListVolumesResponse{}, nil
}

func (m *mockAPIClientSettings) ListStorage(ctx context.Context) (*api.ListStorageResponse, error) {
	m.callLog = append(m.callLog, "ListStorage")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListStorageResponse{}, nil
}

func (m *mockAPIClientSettings) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MountVolume:%s:%s:%s", volumeName, instanceName, mountPoint))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientSettings) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("UnmountVolume:%s:%s", volumeName, instanceName))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientSettings) ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error) {
	m.callLog = append(m.callLog, "ListIdlePolicies")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListIdlePoliciesResponse{}, nil
}

func (m *mockAPIClientSettings) UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error {
	m.callLog = append(m.callLog, "UpdateIdlePolicy")
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientSettings) GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstanceIdleStatus:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.IdleDetectionResponse{}, nil
}

func (m *mockAPIClientSettings) EnableIdleDetection(ctx context.Context, name, policy string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("EnableIdleDetection:%s:%s", name, policy))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientSettings) DisableIdleDetection(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DisableIdleDetection:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientSettings) GetStatus(ctx context.Context) (*api.SystemStatusResponse, error) {
	m.callLog = append(m.callLog, "GetStatus")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.SystemStatusResponse{}, nil
}

func (m *mockAPIClientSettings) ListProjects(ctx context.Context, filter *api.ProjectFilter) (*api.ListProjectsResponse, error) {
	return &api.ListProjectsResponse{}, nil
}

func (m *mockAPIClientSettings) GetPolicyStatus(ctx context.Context) (*api.PolicyStatusResponse, error) {
	return &api.PolicyStatusResponse{}, nil
}

func (m *mockAPIClientSettings) ListPolicySets(ctx context.Context) (*api.ListPolicySetsResponse, error) {
	return &api.ListPolicySetsResponse{}, nil
}

func (m *mockAPIClientSettings) AssignPolicySet(ctx context.Context, policySetID string) error {
	return nil
}

func (m *mockAPIClientSettings) SetPolicyEnforcement(ctx context.Context, enabled bool) error {
	return nil
}

func (m *mockAPIClientSettings) CheckTemplateAccess(ctx context.Context, templateName string) (*api.TemplateAccessResponse, error) {
	return &api.TemplateAccessResponse{}, nil
}

func (m *mockAPIClientSettings) ListMarketplaceTemplates(ctx context.Context, filter *api.MarketplaceFilter) (*api.ListMarketplaceTemplatesResponse, error) {
	return &api.ListMarketplaceTemplatesResponse{}, nil
}

func (m *mockAPIClientSettings) ListMarketplaceCategories(ctx context.Context) (*api.ListCategoriesResponse, error) {
	return &api.ListCategoriesResponse{}, nil
}

func (m *mockAPIClientSettings) ListMarketplaceRegistries(ctx context.Context) (*api.ListRegistriesResponse, error) {
	return &api.ListRegistriesResponse{}, nil
}

func (m *mockAPIClientSettings) InstallMarketplaceTemplate(ctx context.Context, templateName string) error {
	return nil
}

func (m *mockAPIClientSettings) ListAMIs(ctx context.Context) (*api.ListAMIsResponse, error) {
	return &api.ListAMIsResponse{}, nil
}

func (m *mockAPIClientSettings) ListAMIBuilds(ctx context.Context) (*api.ListAMIBuildsResponse, error) {
	return &api.ListAMIBuildsResponse{}, nil
}

func (m *mockAPIClientSettings) ListAMIRegions(ctx context.Context) (*api.ListAMIRegionsResponse, error) {
	return &api.ListAMIRegionsResponse{}, nil
}

func (m *mockAPIClientSettings) DeleteAMI(ctx context.Context, amiID string) error {
	return nil
}

func (m *mockAPIClientSettings) GetRightsizingRecommendations(ctx context.Context) (*api.GetRightsizingRecommendationsResponse, error) {
	return &api.GetRightsizingRecommendationsResponse{}, nil
}

func (m *mockAPIClientSettings) ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error {
	return nil
}

func (m *mockAPIClientSettings) GetLogs(ctx context.Context, instanceName, logType string) (*api.LogsResponse, error) {
	return &api.LogsResponse{}, nil
}

// TestNewSettingsModel tests settings model creation
func TestNewSettingsModel(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}

	model := NewSettingsModel(mockAPIClient)

	assert.NotNil(t, model.apiClient)
	assert.NotNil(t, model.statusBar)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.Equal(t, "unknown", model.daemonStatus)
	assert.Empty(t, model.error)
}

// TestSettingsModelInit tests model initialization
func TestSettingsModelInit(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}
	model := NewSettingsModel(mockAPIClient)

	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Execute the command to get daemon status check
	msg := cmd()
	assert.IsType(t, DaemonStatusMsg{}, msg)

	// Verify that ListIdlePolicies was called for daemon status check
	assert.Contains(t, mockAPIClient.callLog, "ListIdlePolicies")
}

// TestSettingsModelCheckDaemonStatus tests daemon status checking
func TestSettingsModelCheckDaemonStatus(t *testing.T) {
	tests := []struct {
		name           string
		shouldError    bool
		errorMessage   string
		expectedStatus string
		expectError    bool
		description    string
	}{
		{
			name:           "daemon_connected",
			shouldError:    false,
			expectedStatus: "connected",
			expectError:    false,
			description:    "Should show connected when daemon responds",
		},
		{
			name:           "daemon_disconnected",
			shouldError:    true,
			errorMessage:   "connection refused",
			expectedStatus: "disconnected",
			expectError:    true,
			description:    "Should show disconnected when daemon fails to respond",
		},
		{
			name:           "daemon_timeout",
			shouldError:    true,
			errorMessage:   "timeout",
			expectedStatus: "disconnected",
			expectError:    true,
			description:    "Should handle timeout errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPIClient := &mockAPIClientSettings{
				shouldError:  tt.shouldError,
				errorMessage: tt.errorMessage,
			}
			model := NewSettingsModel(mockAPIClient)

			cmd := model.checkDaemonStatus
			msg := cmd()

			statusMsg, ok := msg.(DaemonStatusMsg)
			assert.True(t, ok, "Should return DaemonStatusMsg")
			assert.Equal(t, tt.expectedStatus, statusMsg.Status, tt.description)

			if tt.expectError {
				assert.NotNil(t, statusMsg.Error)
				assert.Contains(t, statusMsg.Error.Error(), tt.errorMessage)
			} else {
				assert.Nil(t, statusMsg.Error)
			}
		})
	}
}

// TestSettingsModelWindowResize tests window resize handling
func TestSettingsModelWindowResize(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}
	model := NewSettingsModel(mockAPIClient)

	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, cmd := model.Update(resizeMsg)

	assert.Equal(t, 120, updatedModel.(SettingsModel).width)
	assert.Equal(t, 40, updatedModel.(SettingsModel).height)
	assert.Nil(t, cmd)
}

// TestSettingsModelDaemonStatusMsg tests daemon status message handling
func TestSettingsModelDaemonStatusMsg(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}
	model := NewSettingsModel(mockAPIClient)

	tests := []struct {
		name           string
		statusMsg      DaemonStatusMsg
		expectedStatus string
		expectError    bool
		description    string
	}{
		{
			name: "connected_status",
			statusMsg: DaemonStatusMsg{
				Status: "connected",
				Error:  nil,
			},
			expectedStatus: "connected",
			expectError:    false,
			description:    "Should handle connected status",
		},
		{
			name: "disconnected_status",
			statusMsg: DaemonStatusMsg{
				Status: "disconnected",
				Error:  fmt.Errorf("connection failed"),
			},
			expectedStatus: "disconnected",
			expectError:    true,
			description:    "Should handle disconnected status with error",
		},
		{
			name: "unknown_status",
			statusMsg: DaemonStatusMsg{
				Status: "timeout",
				Error:  fmt.Errorf("request timeout"),
			},
			expectedStatus: "timeout",
			expectError:    true,
			description:    "Should handle any status with error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedModel, cmd := model.Update(tt.statusMsg)

			result := updatedModel.(SettingsModel)
			assert.Equal(t, tt.expectedStatus, result.daemonStatus, tt.description)
			assert.Nil(t, cmd)

			if tt.expectError {
				assert.NotEmpty(t, result.error)
				assert.Contains(t, result.error, tt.statusMsg.Error.Error())
			} else {
				assert.Empty(t, result.error)
			}
		})
	}
}

// TestSettingsModelKeyboardShortcuts tests keyboard shortcuts
func TestSettingsModelKeyboardShortcuts(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}
	model := NewSettingsModel(mockAPIClient)
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
					// Execute the command to check if it's refresh
					msg := cmd()
					assert.IsType(t, DaemonStatusMsg{}, msg, "Should return daemon status message")
				}
			} else {
				assert.Nil(t, cmd, tt.description)
			}

			if tt.clearError {
				assert.Empty(t, updatedModel.(SettingsModel).error, "Should clear error on refresh")
			}
		})
	}
}

// TestSettingsModelView tests view rendering in different states
func TestSettingsModelView(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}

	tests := []struct {
		name             string
		setupModel       func(m SettingsModel) SettingsModel
		expectedContains []string
		description      string
	}{
		{
			name: "default_state",
			setupModel: func(m SettingsModel) SettingsModel {
				return m
			},
			expectedContains: []string{
				"CloudWorkstation Settings",
				"System Information:",
				"Version:",
				"Daemon Status: unknown",
				"API Endpoint: http://localhost:8947",
				"Configuration:",
				"cws config profile",
				"cws config region",
				"cws config show",
				"Daemon Management:",
				"cws daemon start",
				"cws daemon stop",
				"cws daemon status",
				"TUI Navigation:",
				"1: Dashboard",
				"2: Instances",
				"3: Templates",
				"4: Storage",
				"5: Settings",
				"6: Profiles",
				"r: refresh",
				"q: quit",
			},
			description: "Should show all system information and commands in default state",
		},
		{
			name: "connected_daemon",
			setupModel: func(m SettingsModel) SettingsModel {
				m.daemonStatus = "connected"
				return m
			},
			expectedContains: []string{
				"Daemon Status: connected",
				"System Information:",
			},
			description: "Should show connected daemon status",
		},
		{
			name: "disconnected_daemon",
			setupModel: func(m SettingsModel) SettingsModel {
				m.daemonStatus = "disconnected"
				return m
			},
			expectedContains: []string{
				"Daemon Status: disconnected",
			},
			description: "Should show disconnected daemon status",
		},
		{
			name: "error_state",
			setupModel: func(m SettingsModel) SettingsModel {
				m.error = "Connection timeout"
				return m
			},
			expectedContains: []string{
				"CloudWorkstation Settings",
				"Connection Error:",
				"Connection timeout",
			},
			description: "Should show error information when error is set",
		},
		{
			name: "connected_with_error_cleared",
			setupModel: func(m SettingsModel) SettingsModel {
				m.daemonStatus = "connected"
				m.error = "" // Error cleared after successful connection
				return m
			},
			expectedContains: []string{
				"Daemon Status: connected",
			},
			description: "Should show connected status without error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := tt.setupModel(NewSettingsModel(mockAPIClient))
			view := testModel.View()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, view, expected, tt.description)
			}
		})
	}
}

// TestSettingsModelCompleteWorkflow tests complete user interaction workflow
func TestSettingsModelCompleteWorkflow(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}
	model := NewSettingsModel(mockAPIClient)

	// Step 1: Initialize and check daemon status
	initCmd := model.Init()
	assert.NotNil(t, initCmd)

	initMsg := initCmd()
	updatedModel, _ := model.Update(initMsg)

	// Step 2: Resize window
	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ = updatedModel.Update(resizeMsg)
	assert.Equal(t, 120, updatedModel.(SettingsModel).width)

	// Step 3: Trigger refresh
	refreshKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updatedModel, refreshCmd := updatedModel.Update(refreshKey)
	assert.NotNil(t, refreshCmd)
	assert.Empty(t, updatedModel.(SettingsModel).error) // Error should be cleared

	// Step 4: Process refresh result
	refreshMsg := refreshCmd()
	finalModel, _ := updatedModel.Update(refreshMsg)

	// Verify final state
	result := finalModel.(SettingsModel)
	assert.Equal(t, 120, result.width)
	assert.Equal(t, 40, result.height)
	assert.NotEqual(t, "unknown", result.daemonStatus) // Should be connected or disconnected
}

// TestSettingsModelRefreshFlow tests the complete refresh workflow
func TestSettingsModelRefreshFlow(t *testing.T) {
	tests := []struct {
		name           string
		shouldError    bool
		errorMessage   string
		expectedStatus string
		description    string
	}{
		{
			name:           "successful_refresh",
			shouldError:    false,
			expectedStatus: "connected",
			description:    "Should refresh successfully when daemon is available",
		},
		{
			name:           "failed_refresh",
			shouldError:    true,
			errorMessage:   "daemon unavailable",
			expectedStatus: "disconnected",
			description:    "Should handle refresh failure gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPIClient := &mockAPIClientSettings{
				shouldError:  tt.shouldError,
				errorMessage: tt.errorMessage,
			}
			model := NewSettingsModel(mockAPIClient)
			model.error = "old error"

			// Trigger refresh
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
			updatedModel, cmd := model.Update(keyMsg)

			assert.NotNil(t, cmd)
			assert.Empty(t, updatedModel.(SettingsModel).error) // Error cleared on refresh

			// Execute refresh command
			msg := cmd()
			finalModel, _ := updatedModel.Update(msg)

			result := finalModel.(SettingsModel)
			assert.Equal(t, tt.expectedStatus, result.daemonStatus, tt.description)

			if tt.shouldError {
				assert.NotEmpty(t, result.error)
				assert.Contains(t, result.error, tt.errorMessage)
			} else {
				assert.Empty(t, result.error)
			}
		})
	}
}

// TestSettingsModelEdgeCases tests edge cases and boundary conditions
func TestSettingsModelEdgeCases(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}

	tests := []struct {
		name        string
		setupModel  func() SettingsModel
		testAction  func(SettingsModel) (tea.Model, tea.Cmd)
		description string
	}{
		{
			name: "very_small_window",
			setupModel: func() SettingsModel {
				model := NewSettingsModel(mockAPIClient)
				resizeMsg := tea.WindowSizeMsg{Width: 10, Height: 5}
				updatedModel, _ := model.Update(resizeMsg)
				return updatedModel.(SettingsModel)
			},
			testAction: func(m SettingsModel) (tea.Model, tea.Cmd) {
				_ = m.View() // Should not panic with very small window
				return m, nil
			},
			description: "Should handle very small window sizes gracefully",
		},
		{
			name: "empty_daemon_status",
			setupModel: func() SettingsModel {
				model := NewSettingsModel(mockAPIClient)
				model.daemonStatus = ""
				return model
			},
			testAction: func(m SettingsModel) (tea.Model, tea.Cmd) {
				view := m.View()
				assert.Contains(t, view, "Daemon Status:")
				return m, nil
			},
			description: "Should handle empty daemon status",
		},
		{
			name: "long_error_message",
			setupModel: func() SettingsModel {
				model := NewSettingsModel(mockAPIClient)
				model.error = "This is a very long error message that might wrap across multiple lines and should be handled gracefully by the settings view without breaking the layout"
				return model
			},
			testAction: func(m SettingsModel) (tea.Model, tea.Cmd) {
				view := m.View()
				assert.Contains(t, view, "Connection Error:")
				return m, nil
			},
			description: "Should handle long error messages",
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

// TestSettingsModelMessageHandling tests proper message type handling
func TestSettingsModelMessageHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientSettings{}
	model := NewSettingsModel(mockAPIClient)

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
			name: "daemon_status_message",
			message: DaemonStatusMsg{
				Status: "connected",
				Error:  nil,
			},
			expectPanic: false,
			description: "Should handle daemon status messages properly",
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
					// cmd can be nil for unhandled messages
					_ = cmd
				}, tt.description)
			}
		})
	}
}
