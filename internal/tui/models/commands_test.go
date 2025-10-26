package models

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/prism/internal/tui/api"
	"github.com/scttfrdmn/prism/internal/tui/components"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test CommandDispatcher Core Functionality
func TestCommandDispatcher(t *testing.T) {
	dispatcher := NewCommandDispatcher()

	// Test initialization
	assert.NotNil(t, dispatcher)
	assert.Empty(t, dispatcher.commands)

	// Test command registration
	testCmd := &WindowResizeCommand{}
	dispatcher.RegisterCommand(testCmd)
	assert.Len(t, dispatcher.commands, 1)

	// Test dispatching with no matching commands
	mockModel := createMockRepositoriesModel()
	newModel, cmd := dispatcher.Dispatch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}, mockModel)
	// Basic validation - not using deep equality due to complex BubbleTea internals
	assert.NotNil(t, newModel)
	assert.Nil(t, cmd)
}

// Test WindowResizeCommand
func TestWindowResizeCommand(t *testing.T) {
	cmd := &WindowResizeCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.WindowSizeMsg{Width: 100, Height: 30}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{}))

	// Test Execute with RepositoriesModel
	model := createMockRepositoriesModel()
	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}

	newModel, newCmd := cmd.Execute(model, resizeMsg)
	assert.Nil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	assert.Equal(t, 120, repoModel.width)
	assert.Equal(t, 40, repoModel.height)
}

// Test KeyCommand
func TestKeyCommand(t *testing.T) {
	handlerCalled := false
	handler := func(model tea.Model) (tea.Model, tea.Cmd) {
		handlerCalled = true
		return model, nil
	}

	cmd := NewKeyCommand("x", handler)
	require.NotNil(t, cmd)
	assert.Equal(t, "x", cmd.Key)

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}))

	// Test Execute
	model := createMockRepositoriesModel()
	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{})
	// Verify model type is preserved (avoid deep equality comparison)
	_, ok := newModel.(RepositoriesModel)
	assert.True(t, ok)
	assert.Nil(t, newCmd)
	assert.True(t, handlerCalled)
}

// Test RepositoryAddCommand
func TestRepositoryAddCommand(t *testing.T) {
	cmd := &RepositoryAddCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}))

	// Test Execute in view mode
	model := createMockRepositoriesModel()
	model.mode = "view"

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Nil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	assert.Equal(t, "add", repoModel.mode)
	assert.Equal(t, 0, repoModel.focusIndex)
	assert.Equal(t, "50", repoModel.priorityInput.Value())
	assert.Equal(t, "true", repoModel.enabledInput.Value())

	// Test Execute in non-view mode (should not change)
	model.mode = "edit"
	newModel2, _ := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	repoModel2 := newModel2.(RepositoriesModel)
	assert.Equal(t, "edit", repoModel2.mode)
}

// Test RepositoryEditCommand
func TestRepositoryEditCommand(t *testing.T) {
	cmd := &RepositoryEditCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute with selected item
	model := createMockRepositoriesModel()
	model.mode = "view"
	model.repos = []types.TemplateRepository{
		{
			Name:     "test-repo",
			URL:      "https://github.com/test/repo",
			Priority: 100,
			Enabled:  true,
		},
	}
	model.refreshRepositoryList() // This should populate the list

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	assert.Nil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	assert.Equal(t, "edit", repoModel.mode)
	assert.Equal(t, "test-repo", repoModel.selected)
	assert.Equal(t, 0, repoModel.focusIndex)
}

// Test RepositoryDeleteCommand
func TestRepositoryDeleteCommand(t *testing.T) {
	cmd := &RepositoryDeleteCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute with repositories
	model := createMockRepositoriesModel()
	model.mode = "view"
	model.repos = []types.TemplateRepository{
		{
			Name:     "repo1",
			URL:      "https://github.com/test/repo1",
			Priority: 100,
			Enabled:  true,
		},
		{
			Name:     "repo2",
			URL:      "https://github.com/test/repo2",
			Priority: 50,
			Enabled:  false,
		},
	}
	model.refreshRepositoryList()

	// Count before deletion
	initialCount := len(model.repos)

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Nil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	// Should have one less repository
	assert.Len(t, repoModel.repos, initialCount-1)
}

// Test RepositoryRefreshCommand
func TestRepositoryRefreshCommand(t *testing.T) {
	cmd := &RepositoryRefreshCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute
	model := createMockRepositoriesModel()
	model.mode = "view"
	model.loading = false

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	assert.NotNil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	assert.True(t, repoModel.loading)
}

// Test RepositorySyncCommand
func TestRepositorySyncCommand(t *testing.T) {
	cmd := &RepositorySyncCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute
	model := createMockRepositoriesModel()
	model.mode = "view"
	model.loading = false

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	assert.NotNil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	assert.True(t, repoModel.loading)
}

// Test FormCancelCommand
func TestFormCancelCommand(t *testing.T) {
	cmd := &FormCancelCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyEscape}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute from add mode
	model := createMockRepositoriesModel()
	model.mode = "add"

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyEscape})
	assert.Nil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	assert.Equal(t, "view", repoModel.mode)

	// Test Execute from edit mode
	model.mode = "edit"
	newModel2, _ := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyEscape})
	repoModel2 := newModel2.(RepositoriesModel)
	assert.Equal(t, "view", repoModel2.mode)

	// Test Execute from view mode (should not change)
	model.mode = "view"
	newModel3, _ := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyEscape})
	repoModel3 := newModel3.(RepositoriesModel)
	assert.Equal(t, "view", repoModel3.mode)
}

// Test FormSubmitCommand
func TestFormSubmitCommand(t *testing.T) {
	cmd := &FormSubmitCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyEnter}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute in add mode
	model := createMockRepositoriesModel()
	model.mode = "add"
	model.nameInput.SetValue("new-repo")
	model.urlInput.SetValue("https://github.com/new/repo")
	model.priorityInput.SetValue("75")
	model.enabledInput.SetValue("true")

	initialCount := len(model.repos)
	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	assert.Equal(t, "view", repoModel.mode)
	assert.Len(t, repoModel.repos, initialCount+1)

	// Check that the new repository was added correctly
	found := false
	for _, repo := range repoModel.repos {
		if repo.Name == "new-repo" {
			found = true
			assert.Equal(t, "https://github.com/new/repo", repo.URL)
			assert.Equal(t, 75, repo.Priority)
			assert.True(t, repo.Enabled)
			break
		}
	}
	assert.True(t, found, "New repository should be added")

	// Test Execute in edit mode
	model2 := createMockRepositoriesModel()
	model2.mode = "edit"
	model2.selected = "existing-repo"
	model2.repos = []types.TemplateRepository{
		{Name: "existing-repo", URL: "https://old.com", Priority: 10, Enabled: false},
	}
	model2.nameInput.SetValue("updated-repo")
	model2.urlInput.SetValue("https://new.com")
	model2.priorityInput.SetValue("90")
	model2.enabledInput.SetValue("false")

	newModel2, _ := cmd.Execute(model2, tea.KeyMsg{Type: tea.KeyEnter})
	repoModel2 := newModel2.(RepositoriesModel)
	assert.Equal(t, "view", repoModel2.mode)
	assert.Len(t, repoModel2.repos, 1)

	// Check that the repository was updated
	repo := repoModel2.repos[0]
	assert.Equal(t, "updated-repo", repo.Name)
	assert.Equal(t, "https://new.com", repo.URL)
	assert.Equal(t, 90, repo.Priority)
	assert.False(t, repo.Enabled)
}

// Test FormNavigationCommand
func TestFormNavigationCommand(t *testing.T) {
	cmd := &FormNavigationCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyTab}))
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyShiftTab}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute - forward navigation
	model := createMockRepositoriesModel()
	model.mode = "add"
	model.focusIndex = 0

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyTab})
	assert.Nil(t, newCmd)

	repoModel := newModel.(RepositoriesModel)
	assert.Equal(t, 1, repoModel.focusIndex)

	// Test Execute - backward navigation
	model.focusIndex = 1
	newModel2, _ := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyShiftTab})
	repoModel2 := newModel2.(RepositoriesModel)
	assert.Equal(t, 0, repoModel2.focusIndex)
}

// Test FormInputCommand
func TestFormInputCommand(t *testing.T) {
	cmd := &FormInputCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}))
	assert.False(t, cmd.CanExecute(tea.WindowSizeMsg{}))

	// Test Execute with different focus indices
	model := createMockRepositoriesModel()
	model.mode = "add"

	// Test focus on name input (index 0)
	model.focusIndex = 0
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	newModel, _ := cmd.Execute(model, keyMsg)
	repoModel := newModel.(RepositoriesModel)
	assert.Equal(t, "t", repoModel.nameInput.Value())

	// Test focus on URL input (index 1)
	model.focusIndex = 1
	model.nameInput.Blur()
	model.urlInput.Focus()
	newModel2, _ := cmd.Execute(model, keyMsg)
	repoModel2 := newModel2.(RepositoriesModel)
	assert.Equal(t, "t", repoModel2.urlInput.Value())
}

// Test Instance Commands

// Test InstanceWindowResizeCommand
func TestInstanceWindowResizeCommand(t *testing.T) {
	cmd := &InstanceWindowResizeCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.WindowSizeMsg{Width: 100, Height: 30}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{}))

	// Test Execute
	model := createMockInstancesModel()
	resizeMsg := tea.WindowSizeMsg{Width: 150, Height: 50}

	newModel, newCmd := cmd.Execute(model, resizeMsg)
	assert.Nil(t, newCmd)

	instanceModel := newModel.(InstancesModel)
	assert.Equal(t, 150, instanceModel.width)
	assert.Equal(t, 50, instanceModel.height)
}

// Test InstanceRefreshCommand
func TestInstanceRefreshCommand(t *testing.T) {
	cmd := &InstanceRefreshCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute when not showing actions
	model := createMockInstancesModel()
	model.showingActions = false
	model.loading = false

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	assert.NotNil(t, newCmd)

	instanceModel := newModel.(InstancesModel)
	assert.True(t, instanceModel.loading)
	assert.Equal(t, "", instanceModel.error)

	// Test Execute when showing actions (should not refresh)
	model.showingActions = true
	newModel2, newCmd2 := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	assert.Equal(t, model, newModel2)
	assert.Nil(t, newCmd2)
}

// Test InstanceActionsCommand
func TestInstanceActionsCommand(t *testing.T) {
	cmd := &InstanceActionsCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute with instances available
	model := createMockInstancesModel()
	model.showingActions = false
	model.instances = []api.InstanceResponse{
		{Name: "test-instance", State: "running"},
	}

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Nil(t, newCmd)

	instanceModel := newModel.(InstancesModel)
	assert.True(t, instanceModel.showingActions)

	// Test Execute when already showing actions (should not change)
	model.showingActions = true
	newModel2, _ := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, model, newModel2)

	// Test Execute with no instances (should not change)
	model.showingActions = false
	model.instances = []api.InstanceResponse{}
	newModel3, _ := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, model, newModel3)
}

// Test InstanceConnectionCommand
func TestInstanceConnectionCommand(t *testing.T) {
	cmd := &InstanceConnectionCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute with instances available
	model := createMockInstancesModel()
	model.showingActions = false
	model.selected = 0
	model.instances = []api.InstanceResponse{
		{Name: "test-instance", State: "running", PublicIP: "1.2.3.4"},
	}

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	assert.Nil(t, newCmd)

	instanceModel := newModel.(InstancesModel)
	assert.Contains(t, instanceModel.actionMessage, "ssh ubuntu@1.2.3.4")
}

// Test InstanceActionExecuteCommand
func TestInstanceActionExecuteCommand(t *testing.T) {
	cmd := &InstanceActionExecuteCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}))
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}))
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute when showing actions
	model := createMockInstancesModel()
	model.showingActions = true

	// Test start action
	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	assert.NotNil(t, newCmd)

	instanceModel := newModel.(InstancesModel)
	assert.False(t, instanceModel.showingActions)

	// Test when not showing actions (should not change)
	model.showingActions = false
	newModel2, newCmd2 := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	assert.Equal(t, model, newModel2)
	assert.Nil(t, newCmd2)
}

// Test InstanceActionCancelCommand
func TestInstanceActionCancelCommand(t *testing.T) {
	cmd := &InstanceActionCancelCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyEscape}))
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}))
	assert.False(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}))

	// Test Execute when showing actions
	model := createMockInstancesModel()
	model.showingActions = true

	newModel, newCmd := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyEscape})
	assert.Nil(t, newCmd)

	instanceModel := newModel.(InstancesModel)
	assert.False(t, instanceModel.showingActions)

	// Test Execute when not showing actions (should not change)
	model.showingActions = false
	newModel2, _ := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, model, newModel2)
}

// Test InstanceTableNavigationCommand
func TestInstanceTableNavigationCommand(t *testing.T) {
	cmd := &InstanceTableNavigationCommand{}

	// Test CanExecute
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}))
	assert.True(t, cmd.CanExecute(tea.KeyMsg{Type: tea.KeyUp}))
	assert.False(t, cmd.CanExecute(tea.WindowSizeMsg{}))

	// Test Execute conditions
	model := createMockInstancesModel()
	model.loading = false
	model.showingActions = false
	model.instances = []api.InstanceResponse{
		{Name: "test1", State: "running"},
		{Name: "test2", State: "stopped"},
	}

	newModel, _ := cmd.Execute(model, tea.KeyMsg{Type: tea.KeyDown})
	// Since we can't easily test table navigation without mocking the table,
	// we just verify the model is processed
	assert.IsType(t, InstancesModel{}, newModel)
}

// Helper functions to create mock models

func createMockRepositoriesModel() RepositoriesModel {
	// Create and focus inputs for testing
	nameInput := textinput.New()
	nameInput.Focus()
	urlInput := textinput.New()
	priorityInput := textinput.New()
	enabledInput := textinput.New()

	model := RepositoriesModel{
		nameInput:     nameInput,
		urlInput:      urlInput,
		priorityInput: priorityInput,
		enabledInput:  enabledInput,
		statusBar:     components.NewStatusBar("Repositories", ""),
		repos:         []types.TemplateRepository{},
		mode:          "view",
		focusIndex:    0,
		width:         80,
		height:        24,
	}
	// Create a proper repository list to avoid nil pointer issues
	model.repoList = list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	return model
}

func createMockInstancesModel() InstancesModel {
	// Create a mock instance for testing actions
	mockInstance := api.InstanceResponse{
		Name:     "test-instance",
		State:    "running",
		ID:       "i-1234567890abcdef0",
		PublicIP: "54.123.45.67",
	}

	return InstancesModel{
		apiClient:      &mockAPIClient{}, // Add mock API client for action calls
		statusBar:      components.NewStatusBar("Instances", ""),
		instances:      []api.InstanceResponse{mockInstance},
		loading:        false,
		error:          "",
		showingActions: false,
		selected:       0,
		actionMessage:  "",
		width:          80,
		height:         24,
	}
}

// Integration test - CommandDispatcher with multiple commands
func TestCommandDispatcherIntegration(t *testing.T) {
	dispatcher := NewCommandDispatcher()

	// Register multiple commands
	dispatcher.RegisterCommand(&WindowResizeCommand{})
	dispatcher.RegisterCommand(&RepositoryAddCommand{})
	dispatcher.RegisterCommand(&FormCancelCommand{})

	model := createMockRepositoriesModel()
	model.mode = "view"

	// Test resize command
	newModel, cmd := dispatcher.Dispatch(tea.WindowSizeMsg{Width: 100, Height: 30}, model)
	repoModel := newModel.(RepositoriesModel)
	assert.Equal(t, 100, repoModel.width)
	assert.Equal(t, 30, repoModel.height)
	assert.Nil(t, cmd)

	// Test add command
	newModel2, cmd2 := dispatcher.Dispatch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, model)
	repoModel2 := newModel2.(RepositoriesModel)
	assert.Equal(t, "add", repoModel2.mode)
	assert.Nil(t, cmd2)

	// Test cancel command (needs to be in form mode)
	model.mode = "add"
	newModel3, cmd3 := dispatcher.Dispatch(tea.KeyMsg{Type: tea.KeyEscape}, model)
	repoModel3 := newModel3.(RepositoriesModel)
	assert.Equal(t, "view", repoModel3.mode)
	assert.Nil(t, cmd3)
}

// Performance test - CommandDispatcher with many commands
func TestCommandDispatcherPerformance(t *testing.T) {
	dispatcher := NewCommandDispatcher()

	// Register many commands
	for i := 0; i < 100; i++ {
		key := string(rune('a' + (i % 26)))
		handler := func(model tea.Model) (tea.Model, tea.Cmd) { return model, nil }
		dispatcher.RegisterCommand(NewKeyCommand(key, handler))
	}

	model := createMockRepositoriesModel()

	// Test that dispatch still works efficiently
	newModel, cmd := dispatcher.Dispatch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, model)
	// Verify model type is preserved (meaningful behavior test)
	_, ok := newModel.(RepositoriesModel)
	assert.True(t, ok)
	assert.Nil(t, cmd)
}
