// Package models provides comprehensive test coverage for profile management display functionality
package models

import (
	"context"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// Mock API client for profiles testing - implements complete apiClient interface
type mockAPIClientProfiles struct {
	shouldError  bool
	errorMessage string
	callLog      []string
}

func (m *mockAPIClientProfiles) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	m.callLog = append(m.callLog, "ListInstances")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListInstancesResponse{}, nil
}

func (m *mockAPIClientProfiles) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstance:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.InstanceResponse{}, nil
}

func (m *mockAPIClientProfiles) LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error) {
	m.callLog = append(m.callLog, "LaunchInstance")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.LaunchInstanceResponse{}, nil
}

func (m *mockAPIClientProfiles) StartInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StartInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientProfiles) StopInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StopInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientProfiles) DeleteInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientProfiles) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	m.callLog = append(m.callLog, "ListTemplates")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListTemplatesResponse{}, nil
}

func (m *mockAPIClientProfiles) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetTemplate:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.TemplateResponse{}, nil
}

func (m *mockAPIClientProfiles) ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error) {
	m.callLog = append(m.callLog, "ListVolumes")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListVolumesResponse{}, nil
}

func (m *mockAPIClientProfiles) ListStorage(ctx context.Context) (*api.ListStorageResponse, error) {
	m.callLog = append(m.callLog, "ListStorage")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListStorageResponse{}, nil
}

func (m *mockAPIClientProfiles) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MountVolume:%s:%s:%s", volumeName, instanceName, mountPoint))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientProfiles) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("UnmountVolume:%s:%s", volumeName, instanceName))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientProfiles) ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error) {
	m.callLog = append(m.callLog, "ListIdlePolicies")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListIdlePoliciesResponse{}, nil
}

func (m *mockAPIClientProfiles) UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error {
	m.callLog = append(m.callLog, "UpdateIdlePolicy")
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientProfiles) GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstanceIdleStatus:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.IdleDetectionResponse{}, nil
}

func (m *mockAPIClientProfiles) EnableIdleDetection(ctx context.Context, name, policy string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("EnableIdleDetection:%s:%s", name, policy))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientProfiles) DisableIdleDetection(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DisableIdleDetection:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientProfiles) GetStatus(ctx context.Context) (*api.SystemStatusResponse, error) {
	m.callLog = append(m.callLog, "GetStatus")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.SystemStatusResponse{}, nil
}

// TestNewProfilesModel tests profiles model creation
func TestNewProfilesModel(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}

	model := NewProfilesModel(mockAPIClient)

	assert.NotNil(t, model.apiClient)
	assert.NotNil(t, model.statusBar)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.Nil(t, model.currentProfile)
	assert.Empty(t, model.error)
}

// TestProfilesModelInit tests model initialization
func TestProfilesModelInit(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}
	model := NewProfilesModel(mockAPIClient)

	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Execute the command to get profile init message
	msg := cmd()
	assert.IsType(t, ProfileInitMsg{}, msg)
}

// TestProfilesModelSetSize tests size setting functionality
func TestProfilesModelSetSize(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}
	model := NewProfilesModel(mockAPIClient)

	model.SetSize(120, 40)

	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
}

// TestProfilesModelWindowResize tests window resize handling
func TestProfilesModelWindowResize(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}
	model := NewProfilesModel(mockAPIClient)

	resizeMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd := model.Update(resizeMsg)

	assert.Equal(t, 100, updatedModel.(ProfilesModel).width)
	assert.Equal(t, 30, updatedModel.(ProfilesModel).height)
	assert.Nil(t, cmd)
}

// TestProfilesModelProfileInitMsg tests profile initialization message handling
func TestProfilesModelProfileInitMsg(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}
	model := NewProfilesModel(mockAPIClient)

	profileInitMsg := ProfileInitMsg{}
	updatedModel, cmd := model.Update(profileInitMsg)

	// The model will attempt to load profile using real profile manager
	// We just verify the message is handled without panicking
	assert.NotNil(t, updatedModel)
	assert.Nil(t, cmd)

	// Check that either profile loaded successfully or error is set
	result := updatedModel.(ProfilesModel)
	// Either we have a profile OR we have an error message
	hasProfileOrError := result.currentProfile != nil || result.error != ""
	assert.True(t, hasProfileOrError, "Should have either loaded a profile or set an error")
}

// TestProfilesModelKeyboardShortcuts tests keyboard shortcuts
func TestProfilesModelKeyboardShortcuts(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}
	model := NewProfilesModel(mockAPIClient)

	tests := []struct {
		name        string
		key         string
		expectCmd   bool
		expectQuit  bool
		description string
	}{
		{
			name:        "refresh_command",
			key:         "r",
			expectCmd:   true,
			expectQuit:  false,
			description: "Should trigger refresh",
		},
		{
			name:        "quit_command_q",
			key:         "q",
			expectCmd:   true,
			expectQuit:  true,
			description: "Should trigger quit with 'q'",
		},
		{
			name:        "quit_command_esc",
			key:         "esc",
			expectCmd:   true,
			expectQuit:  true,
			description: "Should trigger quit with escape",
		},
		{
			name:        "unknown_command",
			key:         "x",
			expectCmd:   false,
			expectQuit:  false,
			description: "Should ignore unknown commands",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var keyMsg tea.KeyMsg
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			updatedModel, cmd := model.Update(keyMsg)

			if tt.expectCmd {
				assert.NotNil(t, cmd, tt.description)
				if tt.expectQuit {
					// Execute the command to check if it's quit
					msg := cmd()
					assert.IsType(t, tea.QuitMsg{}, msg, "Should return quit message")
				} else {
					// Execute the command to check if it's refresh
					msg := cmd()
					assert.IsType(t, ProfileInitMsg{}, msg, "Should return profile init message")
				}
			} else {
				assert.Nil(t, cmd, tt.description)
			}

			// Model should be returned unchanged for non-quit commands
			if !tt.expectQuit {
				assert.Equal(t, model, updatedModel)
			}
		})
	}
}

// TestProfilesModelView tests view rendering in different states
func TestProfilesModelView(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}

	tests := []struct {
		name             string
		setupModel       func(m ProfilesModel) ProfilesModel
		expectedContains []string
		description      string
	}{
		{
			name: "loading_state",
			setupModel: func(m ProfilesModel) ProfilesModel {
				// Default state - no profile, no error
				return m
			},
			expectedContains: []string{"CloudWorkstation Profiles", "Loading profile information..."},
			description:      "Should show loading state when no profile or error",
		},
		{
			name: "error_state",
			setupModel: func(m ProfilesModel) ProfilesModel {
				m.error = "Failed to load profile"
				return m
			},
			expectedContains: []string{"CloudWorkstation Profiles", "Error: Failed to load profile"},
			description:      "Should show error state when error is set",
		},
		{
			name: "profile_loaded_state",
			setupModel: func(m ProfilesModel) ProfilesModel {
				m.currentProfile = &profile.Profile{
					Name:       "test-profile",
					AWSProfile: "my-aws-profile",
					Region:     "us-west-2",
				}
				return m
			},
			expectedContains: []string{
				"CloudWorkstation Profiles",
				"Current Profile:",
				"Name: test-profile",
				"AWS Profile: my-aws-profile",
				"Region: us-west-2",
				"Profile Management:",
				"cws config profile",
				"cws config region",
				"cws config show",
			},
			description: "Should show profile information and management commands",
		},
		{
			name: "profile_with_default_values",
			setupModel: func(m ProfilesModel) ProfilesModel {
				m.currentProfile = &profile.Profile{
					Name:       "default",
					AWSProfile: "default",
					Region:     "us-east-1",
				}
				return m
			},
			expectedContains: []string{
				"Name: default",
				"AWS Profile: default",
				"Region: us-east-1",
			},
			description: "Should handle default profile values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := tt.setupModel(NewProfilesModel(mockAPIClient))
			view := testModel.View()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, view, expected, tt.description)
			}

			// All views should contain help text
			assert.Contains(t, view, "r: refresh", "Should always show refresh help")
			assert.Contains(t, view, "q: quit", "Should always show quit help")
		})
	}
}

// TestProfilesModelRefreshFlow tests the complete refresh workflow
func TestProfilesModelRefreshFlow(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}
	model := NewProfilesModel(mockAPIClient)

	// Start refresh
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updatedModel, cmd := model.Update(keyMsg)

	assert.NotNil(t, cmd)
	assert.Equal(t, model, updatedModel) // Model unchanged by refresh command

	// Execute refresh command
	msg := cmd()
	assert.IsType(t, ProfileInitMsg{}, msg)

	// Process the profile init message
	finalModel, finalCmd := updatedModel.Update(msg)
	assert.NotNil(t, finalModel)
	assert.Nil(t, finalCmd)

	// Check that profile loading was attempted
	result := finalModel.(ProfilesModel)
	hasProfileOrError := result.currentProfile != nil || result.error != ""
	assert.True(t, hasProfileOrError, "Should have either loaded a profile or set an error after refresh")
}

// TestProfilesModelCompleteWorkflow tests complete user interaction workflow
func TestProfilesModelCompleteWorkflow(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}
	model := NewProfilesModel(mockAPIClient)

	// Step 1: Initialize
	initCmd := model.Init()
	assert.NotNil(t, initCmd)

	// Step 2: Process initialization
	initMsg := initCmd()
	updatedModel, _ := model.Update(initMsg)

	// Step 3: Resize window
	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ = updatedModel.Update(resizeMsg)
	assert.Equal(t, 120, updatedModel.(ProfilesModel).width)

	// Step 4: Refresh
	refreshKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updatedModel, refreshCmd := updatedModel.Update(refreshKey)
	assert.NotNil(t, refreshCmd)

	// Step 5: Process refresh
	refreshMsg := refreshCmd()
	finalModel, _ := updatedModel.Update(refreshMsg)

	// Verify final state
	result := finalModel.(ProfilesModel)
	assert.Equal(t, 120, result.width)
	assert.Equal(t, 40, result.height)
	hasProfileOrError := result.currentProfile != nil || result.error != ""
	assert.True(t, hasProfileOrError, "Should have either loaded a profile or set an error")
}

// TestProfilesModelEdgeCases tests edge cases and boundary conditions
func TestProfilesModelEdgeCases(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}

	tests := []struct {
		name        string
		setupModel  func() ProfilesModel
		testAction  func(ProfilesModel) (tea.Model, tea.Cmd)
		description string
	}{
		{
			name: "empty_profile_data",
			setupModel: func() ProfilesModel {
				model := NewProfilesModel(mockAPIClient)
				model.currentProfile = &profile.Profile{
					Name:       "",
					AWSProfile: "",
					Region:     "",
				}
				return model
			},
			testAction: func(m ProfilesModel) (tea.Model, tea.Cmd) {
				return m, nil
			},
			description: "Should handle empty profile data without crashing",
		},
		{
			name: "very_small_window",
			setupModel: func() ProfilesModel {
				model := NewProfilesModel(mockAPIClient)
				model.SetSize(10, 5)
				return model
			},
			testAction: func(m ProfilesModel) (tea.Model, tea.Cmd) {
				_ = m.View()
				// Should not panic with very small window
				return m, nil
			},
			description: "Should handle very small window sizes gracefully",
		},
		{
			name: "multiple_refreshes",
			setupModel: func() ProfilesModel {
				return NewProfilesModel(mockAPIClient)
			},
			testAction: func(m ProfilesModel) (tea.Model, tea.Cmd) {
				// Trigger multiple rapid refreshes
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
				m1, cmd1 := m.Update(keyMsg)
				assert.NotNil(t, cmd1)

				m2, cmd2 := m1.Update(keyMsg)
				assert.NotNil(t, cmd2)

				return m2, cmd2
			},
			description: "Should handle multiple refresh commands gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := tt.setupModel()

			// Should not panic
			assert.NotPanics(t, func() {
				_, _ = tt.testAction(model)
			}, tt.description)

			// Should be able to render view
			assert.NotPanics(t, func() {
				view := model.View()
				assert.NotEmpty(t, view)
			}, "Should render view without panicking")
		})
	}
}

// TestProfilesModelMessageHandling tests proper message type handling
func TestProfilesModelMessageHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientProfiles{}
	model := NewProfilesModel(mockAPIClient)

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
			name:        "profile_init_message",
			message:     ProfileInitMsg{},
			expectPanic: false,
			description: "Should handle profile init messages properly",
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
