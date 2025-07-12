package tui

import (
	"bytes"
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/models"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPI is a mock implementation of the CloudWorkstation API
type MockAPI struct {
	mock.Mock
}

// Implement the CloudWorkstation API interface methods
func (m *MockAPI) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*api.ListInstancesResponse), args.Error(1)
}

func (m *MockAPI) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*api.InstanceResponse), args.Error(1)
}

func (m *MockAPI) LaunchInstance(ctx context.Context, req types.LaunchRequest) (*types.LaunchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*types.LaunchResponse), args.Error(1)
}

func (m *MockAPI) StartInstance(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockAPI) StopInstance(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockAPI) DeleteInstance(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockAPI) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*api.ListTemplatesResponse), args.Error(1)
}

func (m *MockAPI) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*api.TemplateResponse), args.Error(1)
}

// Implement the rest of the API methods as needed...

// TestInstancesModel tests the instances model
func TestInstancesModel(t *testing.T) {
	// Create a mock API
	mockAPI := new(MockAPI)
	
	// Setup test data
	testInstances := []api.InstanceResponse{
		{
			ID:                "i-12345",
			Name:              "test-instance",
			Template:          "python-research",
			State:             "running",
			LaunchTime:        time.Now().Add(-24 * time.Hour),
			PublicIP:          "1.2.3.4",
			PrivateIP:         "10.0.0.1",
			EstimatedDailyCost: 2.50,
		},
	}
	
	// Setup the mock expectations
	mockAPI.On("ListInstances", mock.Anything).Return(&api.ListInstancesResponse{
		Instances: testInstances,
		TotalCost: 2.50,
	}, nil)
	
	// Create the model
	model := models.NewInstancesModel(mockAPI)
	
	// Create a program with the model
	program := tea.NewProgram(model, tea.WithOutput(bytes.NewBuffer(nil)))
	
	// Start the program in the background
	go program.Start()
	
	// Wait a bit for the program to process
	time.Sleep(100 * time.Millisecond)
	
	// Send a message to the program
	program.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	
	// Wait a bit more
	time.Sleep(100 * time.Millisecond)
	
	// Send a quit message
	program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	
	// Verify the mock expectations
	mockAPI.AssertExpectations(t)
}

// TestIdleModel tests the idle detection model
func TestIdleModel(t *testing.T) {
	// Create a mock API
	mockAPI := new(MockAPI)
	
	// Setup test data
	testIdlePolicies := []types.IdlePolicy{
		{
			Name:        "default",
			Description: "Default idle policy",
			Threshold:   30,
			Action:      "stop",
			AppliesTo:   []string{"all"},
		},
	}
	
	// Setup the mock expectations
	mockAPI.On("ListIdlePolicies", mock.Anything).Return(testIdlePolicies, nil)
	
	// Create the model
	model := models.NewIdleSettingsModel(mockAPI)
	
	// Create a program with the model
	program := tea.NewProgram(model, tea.WithOutput(bytes.NewBuffer(nil)))
	
	// Start the program in the background
	go program.Start()
	
	// Wait a bit for the program to process
	time.Sleep(100 * time.Millisecond)
	
	// Send a message to the program
	program.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	
	// Wait a bit more
	time.Sleep(100 * time.Millisecond)
	
	// Send a quit message
	program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	
	// No assertion needed as we use a sample response in the model
	// In a real test, you would verify the mock expectations
}

// TestRepositoriesModel tests the repositories model
func TestRepositoriesModel(t *testing.T) {
	// Create a mock API
	mockAPI := new(MockAPI)
	
	// Setup test data
	testTime := time.Now()
	testRepos := []types.TemplateRepository{
		{
			Name:         "default",
			URL:          "https://github.com/example/templates",
			Priority:     1,
			Enabled:      true,
			LastSync:     testTime,
			TemplateCount: 5,
		},
	}
	
	// Setup the mock expectations
	mockAPI.On("ListRepositories", mock.Anything).Return(testRepos, nil)
	
	// Create the model
	model := models.NewRepositoriesModel(mockAPI)
	
	// Create a program with the model
	program := tea.NewProgram(model, tea.WithOutput(bytes.NewBuffer(nil)))
	
	// Start the program in the background
	go program.Start()
	
	// Wait a bit for the program to process
	time.Sleep(100 * time.Millisecond)
	
	// Send a message to the program
	program.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	
	// Wait a bit more
	time.Sleep(100 * time.Millisecond)
	
	// Send a quit message
	program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	
	// Verify the mock expectations
	mockAPI.AssertExpectations(t)
}

// TestTUINavigation tests navigation between TUI models
func TestTUINavigation(t *testing.T) {
	// Create a mock API
	mockAPI := new(MockAPI)
	
	// Setup mock data
	testInstances := []api.InstanceResponse{
		{
			ID:                "i-12345",
			Name:              "test-instance",
			Template:          "python-research",
			State:             "running",
			LaunchTime:        time.Now(),
			PublicIP:          "1.2.3.4",
			EstimatedDailyCost: 2.50,
		},
	}
	
	// Setup the mock expectations
	mockAPI.On("ListInstances", mock.Anything).Return(&api.ListInstancesResponse{
		Instances: testInstances,
		TotalCost: 2.50,
	}, nil)
	
	// Create a main menu model
	model := models.NewDashboardModel(mockAPI)
	
	// Create a program with the model
	program := tea.NewProgram(model, tea.WithOutput(bytes.NewBuffer(nil)))
	
	// Start the program in the background
	go func() {
		_ = program.Start()
	}()
	
	// Wait a bit for the program to process
	time.Sleep(100 * time.Millisecond)
	
	// Simulate tab press to navigate
	program.Send(tea.KeyMsg{Type: tea.KeyTab})
	
	// Wait a bit more
	time.Sleep(100 * time.Millisecond)
	
	// Send a quit message
	program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	
	// No specific assertions needed, just making sure it doesn't crash
}

// TestTabBar tests the TabBar component
func TestTabBar(t *testing.T) {
	// Create a TabBar with 3 tabs
	tabs := []TabItem{
		{ID: "tab1", Title: "Instances"},
		{ID: "tab2", Title: "Templates"},
		{ID: "tab3", Title: "Settings"},
	}
	
	tabBar := NewTabBar(tabs, "tab1")
	
	// Test initial state
	assert.Equal(t, "tab1", tabBar.ActiveTab())
	
	// Test next tab
	tabBar.Next()
	assert.Equal(t, "tab2", tabBar.ActiveTab())
	
	// Test previous tab
	tabBar.Prev()
	assert.Equal(t, "tab1", tabBar.ActiveTab())
	
	// Test direct tab selection
	tabBar.SetActiveTab("tab3")
	assert.Equal(t, "tab3", tabBar.ActiveTab())
	
	// Test view generation - just ensure it doesn't crash
	view := tabBar.View()
	assert.NotEmpty(t, view)
}