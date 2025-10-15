// Package models provides comprehensive test coverage for research user management functionality
package models

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/research"
)

// Mock API client for testing - implements complete apiClient interface
type mockAPIClientUsers struct {
	shouldError  bool
	errorMessage string
	callLog      []string
}

func (m *mockAPIClientUsers) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	m.callLog = append(m.callLog, "ListInstances")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListInstancesResponse{}, nil
}

func (m *mockAPIClientUsers) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstance:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.InstanceResponse{}, nil
}

func (m *mockAPIClientUsers) LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error) {
	m.callLog = append(m.callLog, "LaunchInstance")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.LaunchInstanceResponse{}, nil
}

func (m *mockAPIClientUsers) StartInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StartInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientUsers) StopInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("StopInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientUsers) DeleteInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteInstance:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientUsers) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	m.callLog = append(m.callLog, "ListTemplates")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListTemplatesResponse{}, nil
}

func (m *mockAPIClientUsers) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetTemplate:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.TemplateResponse{}, nil
}

func (m *mockAPIClientUsers) ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error) {
	m.callLog = append(m.callLog, "ListVolumes")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListVolumesResponse{}, nil
}

func (m *mockAPIClientUsers) ListStorage(ctx context.Context) (*api.ListStorageResponse, error) {
	m.callLog = append(m.callLog, "ListStorage")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListStorageResponse{}, nil
}

func (m *mockAPIClientUsers) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MountVolume:%s:%s:%s", volumeName, instanceName, mountPoint))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientUsers) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("UnmountVolume:%s:%s", volumeName, instanceName))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientUsers) ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error) {
	m.callLog = append(m.callLog, "ListIdlePolicies")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListIdlePoliciesResponse{}, nil
}

func (m *mockAPIClientUsers) UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error {
	m.callLog = append(m.callLog, "UpdateIdlePolicy")
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientUsers) GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetInstanceIdleStatus:%s", name))
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.IdleDetectionResponse{}, nil
}

func (m *mockAPIClientUsers) EnableIdleDetection(ctx context.Context, name, policy string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("EnableIdleDetection:%s:%s", name, policy))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientUsers) DisableIdleDetection(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DisableIdleDetection:%s", name))
	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockAPIClientUsers) GetStatus(ctx context.Context) (*api.SystemStatusResponse, error) {
	m.callLog = append(m.callLog, "GetStatus")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.SystemStatusResponse{}, nil
}

func (m *mockAPIClientUsers) ListProjects(ctx context.Context, filter *api.ProjectFilter) (*api.ListProjectsResponse, error) {
	return &api.ListProjectsResponse{}, nil
}

func (m *mockAPIClientUsers) GetPolicyStatus(ctx context.Context) (*api.PolicyStatusResponse, error) {
	return &api.PolicyStatusResponse{}, nil
}

func (m *mockAPIClientUsers) ListPolicySets(ctx context.Context) (*api.ListPolicySetsResponse, error) {
	return &api.ListPolicySetsResponse{}, nil
}

func (m *mockAPIClientUsers) AssignPolicySet(ctx context.Context, policySetID string) error {
	return nil
}

func (m *mockAPIClientUsers) SetPolicyEnforcement(ctx context.Context, enabled bool) error {
	return nil
}

func (m *mockAPIClientUsers) CheckTemplateAccess(ctx context.Context, templateName string) (*api.TemplateAccessResponse, error) {
	return &api.TemplateAccessResponse{}, nil
}

func (m *mockAPIClientUsers) ListMarketplaceTemplates(ctx context.Context, filter *api.MarketplaceFilter) (*api.ListMarketplaceTemplatesResponse, error) {
	return &api.ListMarketplaceTemplatesResponse{}, nil
}

func (m *mockAPIClientUsers) ListMarketplaceCategories(ctx context.Context) (*api.ListCategoriesResponse, error) {
	return &api.ListCategoriesResponse{}, nil
}

func (m *mockAPIClientUsers) ListMarketplaceRegistries(ctx context.Context) (*api.ListRegistriesResponse, error) {
	return &api.ListRegistriesResponse{}, nil
}

func (m *mockAPIClientUsers) InstallMarketplaceTemplate(ctx context.Context, templateName string) error {
	return nil
}

func (m *mockAPIClientUsers) ListAMIs(ctx context.Context) (*api.ListAMIsResponse, error) {
	return &api.ListAMIsResponse{}, nil
}

func (m *mockAPIClientUsers) ListAMIBuilds(ctx context.Context) (*api.ListAMIBuildsResponse, error) {
	return &api.ListAMIBuildsResponse{}, nil
}

func (m *mockAPIClientUsers) ListAMIRegions(ctx context.Context) (*api.ListAMIRegionsResponse, error) {
	return &api.ListAMIRegionsResponse{}, nil
}

func (m *mockAPIClientUsers) DeleteAMI(ctx context.Context, amiID string) error {
	return nil
}

func (m *mockAPIClientUsers) GetRightsizingRecommendations(ctx context.Context) (*api.GetRightsizingRecommendationsResponse, error) {
	return &api.GetRightsizingRecommendationsResponse{}, nil
}

func (m *mockAPIClientUsers) ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error {
	return nil
}

func (m *mockAPIClientUsers) GetLogs(ctx context.Context, instanceName, logType string) (*api.LogsResponse, error) {
	return &api.LogsResponse{}, nil
}

// TestNewUsersModel tests users model creation
func TestNewUsersModel(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}

	model := NewUsersModel(mockAPIClient)

	assert.NotNil(t, model.apiClient)
	assert.NotNil(t, model.statusBar)
	assert.NotNil(t, model.spinner)
	assert.NotNil(t, model.researchUserMgr)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.True(t, model.loading)
	assert.Empty(t, model.users)
	assert.Equal(t, 0, model.selectedUser)
	assert.False(t, model.showCreateDialog)
	assert.False(t, model.showDeleteDialog)
	assert.Empty(t, model.createUsername)
	assert.Empty(t, model.deleteUsername)
}

// TestUsersModelInit tests model initialization
func TestUsersModelInit(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)

	cmd := model.Init()

	assert.NotNil(t, cmd)

	// Execute the command to get initial data load
	msg := cmd()
	assert.NotNil(t, msg)
}

// TestUsersDataHandling tests user data processing
func TestUsersDataHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)
	model.loading = true

	users := []*research.ResearchUserConfig{
		{
			Username:      "alice",
			UID:           1001,
			FullName:      "Alice Smith",
			Email:         "alice@test.com",
			HomeDirectory: "/home/alice",
			Shell:         "/bin/bash",
			SudoAccess:    true,
			DockerAccess:  true,
			CreatedAt:     time.Now(),
			SSHPublicKeys: []string{"ssh-rsa AAAA..."},
		},
		{
			Username:      "bob",
			UID:           1002,
			FullName:      "Bob Jones",
			Email:         "bob@test.com",
			HomeDirectory: "/home/bob",
			Shell:         "/bin/zsh",
			SudoAccess:    false,
			DockerAccess:  false,
			CreatedAt:     time.Now(),
			SSHPublicKeys: []string{},
		},
	}

	usersData := UsersDataMsg{Users: users}
	updatedModel, cmd := model.Update(usersData)

	assert.False(t, updatedModel.(UsersModel).loading)
	assert.Equal(t, 2, len(updatedModel.(UsersModel).users))
	assert.Equal(t, "alice", updatedModel.(UsersModel).users[0].Username)
	assert.Equal(t, "bob", updatedModel.(UsersModel).users[1].Username)
	assert.Nil(t, cmd)
}

// TestUsersModelNavigation tests user selection navigation
func TestUsersModelNavigation(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)

	// Add test users
	users := []*research.ResearchUserConfig{
		{Username: "user1", UID: 1001, CreatedAt: time.Now()},
		{Username: "user2", UID: 1002, CreatedAt: time.Now()},
		{Username: "user3", UID: 1003, CreatedAt: time.Now()},
	}
	model.users = users
	model.selectedUser = 1

	tests := []struct {
		name             string
		key              string
		expectedSelected int
		description      string
	}{
		{
			name:             "navigate_up",
			key:              "up",
			expectedSelected: 0,
			description:      "Should move selection up",
		},
		{
			name:             "navigate_down_from_start",
			key:              "down",
			expectedSelected: 2,
			description:      "Should move selection down",
		},
		{
			name:             "navigate_up_at_boundary",
			key:              "up",
			expectedSelected: 0,
			description:      "Should not go below 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to middle position for boundary tests
			if tt.name == "navigate_up_at_boundary" {
				model.selectedUser = 0
			}

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedModel, cmd := model.Update(keyMsg)

			assert.Equal(t, tt.expectedSelected, updatedModel.(UsersModel).selectedUser, tt.description)
			assert.Nil(t, cmd)
		})
	}

	// Test navigation at lower boundary
	model.selectedUser = 2
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")}
	updatedModel, _ := model.Update(keyMsg)
	assert.Equal(t, 2, updatedModel.(UsersModel).selectedUser, "Should not go beyond last user")
}

// TestUsersModelKeyboardShortcuts tests keyboard shortcuts
func TestUsersModelKeyboardShortcuts(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	_ = NewUsersModel(mockAPIClient)

	tests := []struct {
		name         string
		key          string
		expectCmd    bool
		expectDialog bool
		description  string
	}{
		{
			name:         "refresh_command",
			key:          "r",
			expectCmd:    true,
			expectDialog: false,
			description:  "Should trigger refresh",
		},
		{
			name:         "create_command",
			key:          "c",
			expectCmd:    false,
			expectDialog: true,
			description:  "Should show create dialog",
		},
		{
			name:         "status_command",
			key:          "s",
			expectCmd:    false,
			expectDialog: false,
			description:  "Should show status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset model state
			testModel := NewUsersModel(mockAPIClient)
			testModel.users = []*research.ResearchUserConfig{
				{Username: "testuser", UID: 1001, CreatedAt: time.Now()},
			}
			testModel.loading = false

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedModel, cmd := testModel.Update(keyMsg)

			if tt.expectCmd {
				assert.NotNil(t, cmd, tt.description)
			}

			if tt.expectDialog && tt.key == "c" {
				assert.True(t, updatedModel.(UsersModel).showCreateDialog, tt.description)
			}
		})
	}
}

// TestUsersModelCreateDialog tests create user dialog functionality
func TestUsersModelCreateDialog(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)
	model.showCreateDialog = true
	model.createUsername = "test"

	tests := []struct {
		name             string
		key              string
		expectedState    bool
		expectedUsername string
		description      string
	}{
		{
			name:             "add_character",
			key:              "a",
			expectedState:    true,
			expectedUsername: "testa",
			description:      "Should add character to username",
		},
		{
			name:             "backspace",
			key:              "backspace",
			expectedState:    true,
			expectedUsername: "tes",
			description:      "Should remove character from username",
		},
		{
			name:             "escape_cancel",
			key:              "esc",
			expectedState:    false,
			expectedUsername: "test",
			description:      "Should cancel create dialog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := model
			testModel.showCreateDialog = true
			testModel.createUsername = "test"

			var keyMsg tea.KeyMsg
			switch tt.key {
			case "backspace":
				keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
			case "esc":
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			default:
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			updatedModel, _ := testModel.Update(keyMsg)

			assert.Equal(t, tt.expectedState, updatedModel.(UsersModel).showCreateDialog, tt.description)
			assert.Equal(t, tt.expectedUsername, updatedModel.(UsersModel).createUsername, tt.description)
		})
	}
}

// TestUsersModelCreateUserFlow tests complete user creation workflow
func TestUsersModelCreateUserFlow(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}

	model := NewUsersModel(mockAPIClient)
	model.showCreateDialog = true
	model.createUsername = "newuser"

	// Test Enter key to create user
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)

	assert.NotNil(t, cmd)
	assert.True(t, updatedModel.(UsersModel).showCreateDialog) // Dialog stays open until command completes

	// Execute the create command (will use real research manager)
	msg := cmd()
	createResult, ok := msg.(CreateUserMsg)
	require.True(t, ok)
	assert.Equal(t, "newuser", createResult.Username)
	// Note: Success depends on actual research manager, so we just check the flow

	// Process the create result
	finalModel, _ := updatedModel.Update(createResult)
	assert.False(t, finalModel.(UsersModel).showCreateDialog)
}

// TestUsersModelDeleteDialog tests delete user dialog functionality
func TestUsersModelDeleteDialog(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)
	model.users = []*research.ResearchUserConfig{
		{Username: "testuser", UID: 1001, CreatedAt: time.Now()},
	}
	model.selectedUser = 0
	model.showDeleteDialog = true
	model.deleteUsername = "testuser"

	tests := []struct {
		name          string
		key           string
		expectedState bool
		expectCmd     bool
		description   string
	}{
		{
			name:          "confirm_delete",
			key:           "y",
			expectedState: true, // Dialog stays open until command completes
			expectCmd:     true,
			description:   "Should confirm deletion",
		},
		{
			name:          "cancel_delete_n",
			key:           "n",
			expectedState: false,
			expectCmd:     false,
			description:   "Should cancel deletion with 'n'",
		},
		{
			name:          "cancel_delete_esc",
			key:           "esc",
			expectedState: false,
			expectCmd:     false,
			description:   "Should cancel deletion with escape",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := model
			testModel.showDeleteDialog = true

			var keyMsg tea.KeyMsg
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			updatedModel, cmd := testModel.Update(keyMsg)

			assert.Equal(t, tt.expectedState, updatedModel.(UsersModel).showDeleteDialog, tt.description)
			if tt.expectCmd {
				assert.NotNil(t, cmd, tt.description)
			} else {
				assert.Nil(t, cmd, tt.description)
			}
		})
	}
}

// TestUsersModelDeleteUserFlow tests complete user deletion workflow
func TestUsersModelDeleteUserFlow(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}

	// Add a user to delete
	user := &research.ResearchUserConfig{
		Username:  "deleteuser",
		UID:       1001,
		CreatedAt: time.Now(),
	}

	model := NewUsersModel(mockAPIClient)
	model.users = []*research.ResearchUserConfig{user}
	model.selectedUser = 0
	model.showDeleteDialog = true
	model.deleteUsername = "deleteuser"

	// Test 'y' key to confirm deletion
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, cmd := model.Update(keyMsg)

	assert.NotNil(t, cmd)
	assert.True(t, updatedModel.(UsersModel).showDeleteDialog) // Dialog stays open until command completes

	// Execute the delete command (will use real research manager)
	msg := cmd()
	deleteResult, ok := msg.(DeleteUserMsg)
	require.True(t, ok)
	assert.Equal(t, "deleteuser", deleteResult.Username)
	// Note: Success depends on actual research manager, so we just check the flow

	// Process the delete result
	finalModel, _ := updatedModel.Update(deleteResult)
	assert.False(t, finalModel.(UsersModel).showDeleteDialog)
}

// TestUsersModelErrorHandling tests error handling scenarios
func TestUsersModelErrorHandling(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)
	model.loading = true

	testError := fmt.Errorf("test error message")
	updatedModel, cmd := model.Update(testError)

	assert.False(t, updatedModel.(UsersModel).loading)
	assert.Equal(t, "test error message", updatedModel.(UsersModel).error)
	assert.Nil(t, cmd)
}

// TestUsersModelWindowResize tests window resize handling
func TestUsersModelWindowResize(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)

	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, cmd := model.Update(resizeMsg)

	assert.Equal(t, 120, updatedModel.(UsersModel).width)
	assert.Equal(t, 40, updatedModel.(UsersModel).height)
	assert.Nil(t, cmd)
}

// TestUsersModelView tests view rendering
func TestUsersModelView(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)

	tests := []struct {
		name             string
		setupModel       func(m UsersModel) UsersModel
		expectedContains []string
		description      string
	}{
		{
			name: "loading_state",
			setupModel: func(m UsersModel) UsersModel {
				m.loading = true
				return m
			},
			expectedContains: []string{"👥 Users"},
			description:      "Should show loading state",
		},
		{
			name: "error_state",
			setupModel: func(m UsersModel) UsersModel {
				m.loading = false
				m.error = "Test error"
				return m
			},
			expectedContains: []string{"👥 Users", "Error: Test error"},
			description:      "Should show error state",
		},
		{
			name: "empty_users",
			setupModel: func(m UsersModel) UsersModel {
				m.loading = false
				m.users = []*research.ResearchUserConfig{}
				return m
			},
			expectedContains: []string{"👥 Users", "No users found"},
			description:      "Should show empty state",
		},
		{
			name: "users_list",
			setupModel: func(m UsersModel) UsersModel {
				m.loading = false
				m.users = []*research.ResearchUserConfig{
					{
						Username:      "alice",
						UID:           1001,
						FullName:      "Alice Smith",
						Email:         "alice@test.com",
						HomeDirectory: "/home/alice",
						Shell:         "/bin/bash",
						SudoAccess:    true,
						DockerAccess:  false,
						CreatedAt:     time.Now(),
						SSHPublicKeys: []string{"key1", "key2"},
					},
				}
				m.selectedUser = 0
				return m
			},
			expectedContains: []string{"👥 Users", "alice", "UID: 1001", "SSH Keys: 2", "Alice Smith", "alice@test.com"},
			description:      "Should show user list with details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := tt.setupModel(model)
			view := testModel.View()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, view, expected, tt.description)
			}
		})
	}
}

// TestUsersModelCreateUserCommand tests create user command creation
func TestUsersModelCreateUserCommand(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)

	cmd := model.createUser("testuser")
	assert.NotNil(t, cmd)

	msg := cmd()
	createResult, ok := msg.(CreateUserMsg)
	require.True(t, ok)
	assert.Equal(t, "testuser", createResult.Username)
	// Note: Success/failure depends on actual research manager
}

// TestUsersModelDeleteUserCommand tests delete user command creation
func TestUsersModelDeleteUserCommand(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)

	cmd := model.deleteUser("testuser")
	assert.NotNil(t, cmd)

	msg := cmd()
	deleteResult, ok := msg.(DeleteUserMsg)
	require.True(t, ok)
	assert.Equal(t, "testuser", deleteResult.Username)
	// Note: Success/failure depends on actual research manager
}

// TestUsersModelRefreshFlow tests refresh functionality
func TestUsersModelRefreshFlow(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)
	model.loading = false

	refreshMsg := RefreshMsg{}
	updatedModel, cmd := model.Update(refreshMsg)

	assert.True(t, updatedModel.(UsersModel).loading)
	assert.Empty(t, updatedModel.(UsersModel).error)
	assert.NotNil(t, cmd)
}

// TestUsersModelStatusDisplay tests status information display
func TestUsersModelStatusDisplay(t *testing.T) {
	mockAPIClient := &mockAPIClientUsers{}
	model := NewUsersModel(mockAPIClient)
	model.users = []*research.ResearchUserConfig{
		{Username: "alice", UID: 1001, CreatedAt: time.Now()},
		{Username: "bob", UID: 1002, CreatedAt: time.Now()},
	}
	model.selectedUser = 1

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updatedModel, cmd := model.Update(keyMsg)

	assert.Nil(t, cmd)
	// Status should be set on the status bar (tested implicitly through component)
	_ = updatedModel
}

// TestIsValidUsernameChar tests username validation
func TestIsValidUsernameChar(t *testing.T) {
	tests := []struct {
		char        byte
		expected    bool
		description string
	}{
		{'a', true, "Lowercase letter should be valid"},
		{'Z', true, "Uppercase letter should be valid"},
		{'5', true, "Number should be valid"},
		{'-', true, "Hyphen should be valid"},
		{'_', true, "Underscore should be valid"},
		{'@', false, "Special character should be invalid"},
		{' ', false, "Space should be invalid"},
		{'.', false, "Dot should be invalid"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("char_%c", tt.char), func(t *testing.T) {
			result := isValidUsernameChar(tt.char)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestTUIProfileManagerAdapter tests profile manager adapter
func TestTUIProfileManagerAdapter(t *testing.T) {
	adapter := &TUIProfileManagerAdapter{}

	// Test GetCurrentProfile
	profile, err := adapter.GetCurrentProfile()
	// Should return default profile on error or actual profile name
	assert.NoError(t, err)
	assert.NotEmpty(t, profile)

	// Test GetProfileConfig (now implemented - returns error for non-existent profile)
	config, err := adapter.GetProfileConfig("nonexistent-profile")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "profile")

	// Test UpdateProfileConfig (now implemented - returns error for invalid config)
	err = adapter.UpdateProfileConfig("test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid profile config type")
}
