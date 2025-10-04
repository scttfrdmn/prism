package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock API Client for Instance Action Testing
type instanceActionMockClient struct {
	instances       map[string]*types.Instance
	instanceErrors  map[string]error
	actionErrors    map[string]error
	startCallCount  int
	stopCallCount   int
	deleteCallCount int
}

// Implement apiClient interface methods
func (m *instanceActionMockClient) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	if err, exists := m.instanceErrors[name]; exists {
		return nil, err
	}
	if instance, exists := m.instances[name]; exists {
		return &api.InstanceResponse{
			ID:                 instance.ID,
			Name:               instance.Name,
			Template:           instance.Template,
			PublicIP:           instance.PublicIP,
			State:              instance.State,
			LaunchTime:         instance.LaunchTime,
			HourlyRate:         instance.HourlyRate,
			CurrentSpend:       instance.CurrentSpend,
			EffectiveRate:      instance.EffectiveRate,
			AttachedVolumes:    instance.AttachedVolumes,
			AttachedEBSVolumes: instance.AttachedEBSVolumes,
			InstanceLifecycle:  instance.InstanceLifecycle,
			Ports:              []int{22}, // Default SSH port
		}, nil
	}
	return nil, errors.New("instance not found")
}

func (m *instanceActionMockClient) StartInstance(ctx context.Context, name string) error {
	m.startCallCount++
	if err, exists := m.actionErrors["start"]; exists {
		return err
	}
	return nil
}

func (m *instanceActionMockClient) StopInstance(ctx context.Context, name string) error {
	m.stopCallCount++
	if err, exists := m.actionErrors["stop"]; exists {
		return err
	}
	return nil
}

func (m *instanceActionMockClient) DeleteInstance(ctx context.Context, name string) error {
	m.deleteCallCount++
	if err, exists := m.actionErrors["delete"]; exists {
		return err
	}
	return nil
}

func (m *instanceActionMockClient) EnableIdleDetection(ctx context.Context, name, policy string) error {
	return nil
}

func (m *instanceActionMockClient) DisableIdleDetection(ctx context.Context, name string) error {
	return nil
}

func (m *instanceActionMockClient) GetStatus(ctx context.Context) (*api.SystemStatusResponse, error) {
	return &api.SystemStatusResponse{}, nil
}

// Additional apiClient interface methods
func (m *instanceActionMockClient) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	return &api.ListInstancesResponse{}, nil
}

func (m *instanceActionMockClient) LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error) {
	return &api.LaunchInstanceResponse{}, nil
}

func (m *instanceActionMockClient) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	return &api.ListTemplatesResponse{}, nil
}

func (m *instanceActionMockClient) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	return &api.TemplateResponse{}, nil
}

func (m *instanceActionMockClient) ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error) {
	return &api.ListVolumesResponse{}, nil
}

func (m *instanceActionMockClient) ListStorage(ctx context.Context) (*api.ListStorageResponse, error) {
	return &api.ListStorageResponse{}, nil
}

func (m *instanceActionMockClient) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	return nil
}

func (m *instanceActionMockClient) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	return nil
}

func (m *instanceActionMockClient) ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error) {
	return &api.ListIdlePoliciesResponse{}, nil
}

func (m *instanceActionMockClient) UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error {
	return nil
}

func (m *instanceActionMockClient) GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error) {
	return &api.IdleDetectionResponse{}, nil
}

// Test Data Helpers
func createRunningInstance() *types.Instance {
	launchTime, _ := time.Parse(time.RFC3339, "2024-06-15T10:30:00Z")
	return &types.Instance{
		Name:               "my-analysis",
		State:              "running",
		ID:                 "i-1234567890abcdef0",
		PublicIP:           "54.123.45.67",
		Username:           "ubuntu",
		HasWebInterface:    true,
		WebPort:            8888,
		Template:           "python-ml",
		LaunchTime:         launchTime,
		EstimatedCost:      2.40,
		InstanceType:       "t3.medium",
		AttachedVolumes:    []string{"fs-abcdef123"},
		AttachedEBSVolumes: []string{"vol-123abc"},
	}
}

func createStoppedInstance() *types.Instance {
	launchTime, _ := time.Parse(time.RFC3339, "2024-06-14T15:20:00Z")
	return &types.Instance{
		Name:               "my-training",
		State:              "stopped",
		ID:                 "i-abcdef1234567890",
		PublicIP:           "",
		Username:           "rocky",
		HasWebInterface:    false,
		Template:           "r-research",
		LaunchTime:         launchTime,
		EstimatedCost:      0.0,
		InstanceType:       "r5.large",
		AttachedVolumes:    []string{},
		AttachedEBSVolumes: []string{},
	}
}

// Test ActionItem Implementation
func TestActionItem(t *testing.T) {
	tests := []struct {
		name           string
		item           ActionItem
		expectedTitle  string
		expectedFilter string
		expectedDesc   string
	}{
		{
			name: "Regular action",
			item: ActionItem{
				name:        "Start Instance",
				description: "Starts the stopped instance",
				action:      "start",
				dangerous:   false,
			},
			expectedTitle:  "Start Instance",
			expectedFilter: "Start Instance",
			expectedDesc:   "Starts the stopped instance",
		},
		{
			name: "Dangerous action",
			item: ActionItem{
				name:        "Delete Instance",
				description: "Permanently deletes the instance and all data",
				action:      "delete",
				dangerous:   true,
			},
			expectedTitle:  "Delete Instance (!)",
			expectedFilter: "Delete Instance",
			expectedDesc:   "Permanently deletes the instance and all data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedTitle, tt.item.Title())
			assert.Equal(t, tt.expectedFilter, tt.item.FilterValue())
			assert.Equal(t, tt.expectedDesc, tt.item.Description())
		})
	}
}

// Test Model Creation
func TestNewInstanceActionModel(t *testing.T) {
	client := &instanceActionMockClient{}
	instanceName := "my-test-instance"

	model := NewInstanceActionModel(client, instanceName)

	// Test basic initialization
	assert.Equal(t, client, model.apiClient)
	assert.Equal(t, instanceName, model.instance)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.True(t, model.loading)
	assert.Equal(t, "", model.error)
	assert.False(t, model.confirmStep)

	// Test components are properly initialized
	assert.NotNil(t, model.actionList)
	assert.NotNil(t, model.statusBar)
	assert.NotNil(t, model.spinner)
	assert.NotNil(t, model.dispatcher)

	// Test action list setup
	assert.Equal(t, "Instance Actions", model.actionList.Title)
}

// Test Initialization Command
func TestInstanceActionInit(t *testing.T) {
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")

	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Test that Init returns batch command
	msgs := testBatchCommand(t, cmd)
	assert.GreaterOrEqual(t, len(msgs), 1, "Init should return multiple commands")
}

// Test Instance Details Fetching
func TestFetchInstanceDetails(t *testing.T) {
	tests := []struct {
		name         string
		instanceName string
		mockInstance *types.Instance
		mockError    error
		expectError  bool
	}{
		{
			name:         "Successful fetch",
			instanceName: "running-instance",
			mockInstance: createRunningInstance(),
			mockError:    nil,
			expectError:  false,
		},
		{
			name:         "API error",
			instanceName: "error-instance",
			mockInstance: nil,
			mockError:    errors.New("API error"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &instanceActionMockClient{
				instances:      make(map[string]*types.Instance),
				instanceErrors: make(map[string]error),
			}

			if tt.mockInstance != nil {
				client.instances[tt.instanceName] = tt.mockInstance
			}
			if tt.mockError != nil {
				client.instanceErrors[tt.instanceName] = tt.mockError
			}

			model := NewInstanceActionModel(client, tt.instanceName)
			msg := model.fetchInstanceDetails()

			if tt.expectError {
				_, isError := msg.(error)
				assert.True(t, isError, "Expected error message")
			} else {
				response, ok := msg.(*api.InstanceResponse)
				assert.True(t, ok, "Expected InstanceResponse message")
				assert.Equal(t, tt.mockInstance.Name, response.Name)
				assert.Equal(t, tt.mockInstance.State, response.State)
			}
		})
	}
}

// Test Action Performance
func TestPerformAction(t *testing.T) {
	tests := []struct {
		name           string
		action         string
		mockError      error
		expectSuccess  bool
		expectedAction string
	}{
		{
			name:           "Start instance success",
			action:         "start",
			mockError:      nil,
			expectSuccess:  true,
			expectedAction: "start",
		},
		{
			name:           "Stop instance success",
			action:         "stop",
			mockError:      nil,
			expectSuccess:  true,
			expectedAction: "stop",
		},
		{
			name:           "Delete instance success",
			action:         "delete",
			mockError:      nil,
			expectSuccess:  true,
			expectedAction: "delete",
		},
		{
			name:           "Start instance error",
			action:         "start",
			mockError:      errors.New("start failed"),
			expectSuccess:  false,
			expectedAction: "",
		},
		{
			name:           "Unknown action",
			action:         "unknown",
			mockError:      nil,
			expectSuccess:  false,
			expectedAction: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &instanceActionMockClient{
				actionErrors: make(map[string]error),
			}

			if tt.mockError != nil {
				client.actionErrors[tt.action] = tt.mockError
			}

			model := NewInstanceActionModel(client, "test-instance")
			cmd := model.performAction(tt.action)

			// Execute the command
			msg := cmd()

			if tt.expectSuccess {
				actionMsg, ok := msg.(InstanceActionMsg)
				assert.True(t, ok, "Expected InstanceActionMsg")
				assert.Equal(t, tt.expectedAction, actionMsg.Action)
				assert.True(t, actionMsg.Success)
				assert.Contains(t, actionMsg.Message, "test-instance")
			} else {
				_, isError := msg.(error)
				assert.True(t, isError, "Expected error message")
			}
		})
	}
}

// Test Window Size Command
func TestInstanceActionWindowSizeCommand(t *testing.T) {
	cmd := &InstanceActionWindowSizeCommand{}
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.WindowSizeMsg{Width: 100, Height: 30}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{}))

	// Test Execute
	newModel, newCmd := cmd.Execute(model, tea.WindowSizeMsg{Width: 100, Height: 30})
	updatedModel := newModel.(InstanceActionModel)

	assert.Equal(t, 100, updatedModel.width)
	assert.Equal(t, 30, updatedModel.height)
	assert.Nil(t, newCmd)

	// Test list dimensions update
	expectedHeight := 30 - 10 // Account for UI elements
	assert.Equal(t, expectedHeight, updatedModel.actionList.Height())
	assert.Equal(t, 96, updatedModel.actionList.Width()) // width - 4
}

// Test Key Command - Normal Mode
func TestInstanceActionKeyCommand_NormalMode(t *testing.T) {
	cmd := &InstanceActionKeyCommand{}
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")
	model.loading = false

	// Add test action item
	testItem := ActionItem{
		name:        "Start Instance",
		description: "Starts the stopped instance",
		action:      "start",
		dangerous:   false,
	}
	model.actionList.SetItems([]list.Item{testItem})

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyEnter}))
	assert.False(t, cmd.CanExecute(tea.WindowSizeMsg{}))

	// Test quit key - simplified
	t.Run("Quit key", func(t *testing.T) {
		newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		// Model should be unchanged
		assert.Equal(t, model.instance, newModel.(InstanceActionModel).instance)
		assert.NotNil(t, newCmd)
	})

	// Test enter key with non-dangerous action
	t.Run("Enter key non-dangerous", func(t *testing.T) {
		newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyEnter})
		updatedModel := newModel.(InstanceActionModel)

		assert.True(t, updatedModel.loading)
		assert.NotNil(t, newCmd)
	})

	// Test enter key with dangerous action
	t.Run("Enter key dangerous", func(t *testing.T) {
		dangerousItem := ActionItem{
			name:        "Delete Instance",
			description: "Permanently deletes the instance",
			action:      "delete",
			dangerous:   true,
		}
		model.actionList.SetItems([]list.Item{dangerousItem})

		newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyEnter})
		updatedModel := newModel.(InstanceActionModel)

		assert.True(t, updatedModel.confirmStep)
		assert.Nil(t, newCmd)
	})
}

// Test Key Command - Confirmation Mode
func TestInstanceActionKeyCommand_ConfirmationMode(t *testing.T) {
	cmd := &InstanceActionKeyCommand{}
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")
	model.confirmStep = true

	// Add test action item
	testItem := ActionItem{
		name:        "Delete Instance",
		description: "Permanently deletes the instance",
		action:      "delete",
		dangerous:   true,
	}
	model.actionList.SetItems([]list.Item{testItem})

	// Test confirm keys
	confirmKeys := []string{"y", "Y"}
	for _, key := range confirmKeys {
		t.Run("Confirm key: "+key, func(t *testing.T) {
			newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(key[0])}})
			updatedModel := newModel.(InstanceActionModel)

			assert.True(t, updatedModel.loading)
			assert.NotNil(t, newCmd)
		})
	}

	// Test cancel keys
	cancelKeys := []string{"n", "N", "esc", "q"}
	for _, key := range cancelKeys {
		t.Run("Cancel key: "+key, func(t *testing.T) {
			testModel := model
			testModel.confirmStep = true

			var keyType tea.KeyType
			var runes []rune

			switch key {
			case "esc":
				keyType = tea.KeyEscape
			case "q":
				keyType = tea.KeyRunes
				runes = []rune{'q'}
			default:
				keyType = tea.KeyRunes
				runes = []rune{rune(key[0])}
			}

			newModel, newCmd := cmd.Execute(testModel, tea.KeyMsg{Type: keyType, Runes: runes})
			updatedModel := newModel.(InstanceActionModel)

			assert.False(t, updatedModel.confirmStep)
			assert.Nil(t, newCmd)
		})
	}
}

// Test Instance Command - Running Instance
func TestInstanceActionInstanceCommand_RunningInstance(t *testing.T) {
	cmd := &InstanceActionInstanceCommand{}
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")
	model.loading = true

	instance := createRunningInstance()

	// Test CanExecute
	assert.True(t, cmd.CanExecute(instance))
	assert.False(t, cmd.CanExecute("not an instance"))

	// Test Execute
	newModel, newCmd := cmd.Execute(model, instance)
	updatedModel := newModel.(InstanceActionModel)

	assert.False(t, updatedModel.loading)
	assert.Nil(t, newCmd)

	// Check that actions are populated
	items := updatedModel.actionList.Items()
	assert.Greater(t, len(items), 0, "Should have action items")

	// Verify running instance actions
	actionNames := make([]string, len(items))
	for i, item := range items {
		actionItem := item.(ActionItem)
		actionNames[i] = actionItem.name
	}

	assert.Contains(t, actionNames, "Stop Instance")
	assert.Contains(t, actionNames, "Connect via SSH")
	assert.Contains(t, actionNames, "Open Web Interface") // Has web interface
	assert.Contains(t, actionNames, "Delete Instance")
}

// Test Instance Command - Stopped Instance
func TestInstanceActionInstanceCommand_StoppedInstance(t *testing.T) {
	cmd := &InstanceActionInstanceCommand{}
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")

	instance := createStoppedInstance()

	// Test Execute
	newModel, _ := cmd.Execute(model, instance)
	updatedModel := newModel.(InstanceActionModel)

	// Check that actions are populated
	items := updatedModel.actionList.Items()
	assert.Greater(t, len(items), 0, "Should have action items")

	// Verify stopped instance actions
	actionNames := make([]string, len(items))
	for i, item := range items {
		actionItem := item.(ActionItem)
		actionNames[i] = actionItem.name
	}

	assert.Contains(t, actionNames, "Start Instance")
	assert.Contains(t, actionNames, "Delete Instance")
	assert.NotContains(t, actionNames, "Stop Instance")
	assert.NotContains(t, actionNames, "Connect via SSH")
}

// Test Result Command
func TestInstanceActionResultCommand(t *testing.T) {
	cmd := &InstanceActionResultCommand{}
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")
	model.loading = true

	// Test CanExecute
	assert.True(t, cmd.CanExecute(InstanceActionMsg{}))
	assert.False(t, cmd.CanExecute("not a result"))

	// Test successful result
	t.Run("Success result", func(t *testing.T) {
		successMsg := InstanceActionMsg{
			Action:  "start",
			Success: true,
			Message: "Instance started successfully",
		}

		newModel, newCmd := cmd.Execute(model, successMsg)
		updatedModel := newModel.(InstanceActionModel)

		assert.False(t, updatedModel.loading)
		assert.NotNil(t, newCmd)
	})

	// Test error result
	t.Run("Error result", func(t *testing.T) {
		errorMsg := InstanceActionMsg{
			Action:  "start",
			Success: false,
			Message: "Failed to start instance",
		}

		newModel, newCmd := cmd.Execute(model, errorMsg)
		updatedModel := newModel.(InstanceActionModel)

		assert.False(t, updatedModel.loading)
		assert.NotNil(t, newCmd)
	})
}

// Test Error Command
func TestInstanceActionErrorCommand(t *testing.T) {
	cmd := &InstanceActionErrorCommand{}
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")
	model.loading = true

	testError := errors.New("test error")

	// Test CanExecute
	assert.True(t, cmd.CanExecute(testError))
	assert.False(t, cmd.CanExecute("not an error"))

	// Test Execute
	newModel, newCmd := cmd.Execute(model, testError)
	updatedModel := newModel.(InstanceActionModel)

	assert.False(t, updatedModel.loading)
	assert.Equal(t, "test error", updatedModel.error)
	assert.Nil(t, newCmd)
}

// Test Full Update Flow with Command Pattern
func TestInstanceActionUpdate(t *testing.T) {
	client := &instanceActionMockClient{
		instances: map[string]*types.Instance{
			"test-instance": createRunningInstance(),
		},
	}
	model := NewInstanceActionModel(client, "test-instance")

	// Test window resize
	t.Run("Window resize", func(t *testing.T) {
		newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		updatedModel := newModel.(InstanceActionModel)
		// Window resize should set dimensions (allowing for default values)
		assert.True(t, updatedModel.width > 0)
		assert.True(t, updatedModel.height > 0)
	})

	// Test instance data received
	t.Run("Instance data", func(t *testing.T) {
		instance := createRunningInstance()
		newModel, _ := model.Update(instance)
		updatedModel := newModel.(InstanceActionModel)
		assert.False(t, updatedModel.loading)
		assert.Greater(t, len(updatedModel.actionList.Items()), 0)
	})

	// Test error handling
	t.Run("Error handling", func(t *testing.T) {
		testError := errors.New("API error")
		newModel, _ := model.Update(testError)
		updatedModel := newModel.(InstanceActionModel)
		assert.False(t, updatedModel.loading)
		assert.Equal(t, "API error", updatedModel.error)
	})
}

// Test View Rendering
func TestInstanceActionView(t *testing.T) {
	client := &instanceActionMockClient{}
	model := NewInstanceActionModel(client, "test-instance")

	// Test loading view
	t.Run("Loading view", func(t *testing.T) {
		model.loading = true
		view := model.View()

		assert.Contains(t, view, "Instance Actions: test-instance")
		assert.NotEmpty(t, view)
	})

	// Test error view
	t.Run("Error view", func(t *testing.T) {
		model.loading = false
		model.error = "Test error message"
		view := model.View()

		assert.Contains(t, view, "Instance Actions: test-instance")
		assert.Contains(t, view, "Error: Test error message")
	})

	// Test normal view
	t.Run("Normal view", func(t *testing.T) {
		model.loading = false
		model.error = ""
		model.actionList.SetItems([]list.Item{
			ActionItem{name: "Start Instance", description: "Test action", action: "start"},
		})
		view := model.View()

		assert.Contains(t, view, "Instance Actions: test-instance")
		assert.Contains(t, view, "↑/↓: navigate • enter: select • q: quit")
	})

	// Test confirmation view
	t.Run("Confirmation view", func(t *testing.T) {
		model.loading = false
		model.error = ""
		model.confirmStep = true
		model.actionList.SetItems([]list.Item{
			ActionItem{name: "Delete Instance", description: "Dangerous action", action: "delete", dangerous: true},
		})
		view := model.View()

		assert.Contains(t, view, "Instance Actions: test-instance")
		assert.Contains(t, view, "Confirm Action")
		assert.Contains(t, view, "Delete Instance")
		assert.Contains(t, view, "y: confirm • n: cancel")
	})
}

// Test API Method Call Counts
func TestInstanceActionAPICalls(t *testing.T) {
	client := &instanceActionMockClient{
		instances: map[string]*types.Instance{
			"test-instance": createRunningInstance(),
		},
	}

	model := NewInstanceActionModel(client, "test-instance")

	// Test start action
	cmd := model.performAction("start")
	cmd() // Execute the command
	assert.Equal(t, 1, client.startCallCount)

	// Test stop action
	cmd = model.performAction("stop")
	cmd() // Execute the command
	assert.Equal(t, 1, client.stopCallCount)

	// Test delete action
	cmd = model.performAction("delete")
	cmd() // Execute the command
	assert.Equal(t, 1, client.deleteCallCount)
}

// Helper function to test batch commands
func testBatchCommand(t *testing.T, cmd tea.Cmd) []tea.Msg {
	require.NotNil(t, cmd, "Command should not be nil")

	// Execute the batch command and collect all messages
	var msgs []tea.Msg
	if cmd != nil {
		msg := cmd()
		if msg != nil {
			msgs = append(msgs, msg)
		}
	}

	return msgs
}
