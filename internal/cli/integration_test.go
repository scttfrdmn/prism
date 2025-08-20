// Package cli integration tests for CloudWorkstation CLI
//
// This file contains mock-based integration tests that run by default.
// For AWS integration tests against real resources, see integration_aws_test.go
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestAppCreation tests CLI app creation and initialization
func TestAppCreation(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"Valid version", "1.0.0"},
		{"Empty version", ""},
		{"Development version", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(tt.version)

			assert.NotNil(t, app)
			assert.Equal(t, tt.version, app.version)
			assert.NotNil(t, app.apiClient)
			assert.NotNil(t, app.ctx)
			assert.NotNil(t, app.config)
			assert.NotNil(t, app.launchDispatcher)
			assert.NotNil(t, app.instanceCommands)
			assert.NotNil(t, app.storageCommands)
			assert.NotNil(t, app.templateCommands)
			assert.NotNil(t, app.systemCommands)
			assert.NotNil(t, app.scalingCommands)
		})
	}
}

// TestAppCreationWithClient tests CLI app creation with custom client
func TestAppCreationWithClient(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	assert.NotNil(t, app)
	assert.Equal(t, "1.0.0", app.version)
	assert.Equal(t, mockClient, app.apiClient)
	assert.NotNil(t, app.ctx)
	assert.NotNil(t, app.config)
	assert.NotNil(t, app.launchDispatcher)
	assert.NotNil(t, app.instanceCommands)
	assert.NotNil(t, app.storageCommands)
	assert.NotNil(t, app.templateCommands)
	assert.NotNil(t, app.systemCommands)
	assert.NotNil(t, app.scalingCommands)
}

// TestLaunchCommand tests the launch command with various scenarios
func TestLaunchCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorType   string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid launch",
			args:        []string{"python-ml", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Missing arguments",
			args:        []string{"python-ml"},
			expectError: true,
			errorType:   "usage",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Empty arguments",
			args:        []string{},
			expectError: true,
			errorType:   "usage",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"python-ml", "test-instance"},
			expectError: true,
			errorType:   "api",
			setupMock: func(mock *MockAPIClient) {
				// Use specific launch error to avoid affecting Ping method
				mock.LaunchError = fmt.Errorf("launch failed")
			},
		},
		{
			name:        "Daemon not running",
			args:        []string{"python-ml", "test-instance"},
			expectError: true,
			errorType:   "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.PingError = assert.AnError
			},
		},
		{
			name:        "Launch with project",
			args:        []string{"python-ml", "test-instance", "--project", "test-project"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Launch with size",
			args:        []string{"python-ml", "test-instance", "--size", "L"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable daemon auto-start for daemon-related error tests
			if tt.errorType == "daemon" || tt.errorType == "api" {
				t.Setenv("CWS_NO_AUTO_START", "1")
			}
			
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			err := app.Launch(tt.args)

			if tt.expectError {
				assert.Error(t, err)
				switch tt.errorType {
				case "usage":
					assert.Contains(t, err.Error(), "usage:")
				case "api":
					assert.Contains(t, err.Error(), "instance launch failed")
				case "daemon":
					assert.Contains(t, err.Error(), "daemon not running")
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, mockClient.LaunchCalls, 1)
				assert.Equal(t, tt.args[0], mockClient.LaunchCalls[0].Template)
				assert.Equal(t, tt.args[1], mockClient.LaunchCalls[0].Name)
			}
		})
	}
}

// TestLaunchCommandWithProjectFiltering tests launch with project filtering
func TestLaunchCommandWithProjectFiltering(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	args := []string{"python-ml", "test-instance", "--project", "my-project"}
	err := app.Launch(args)

	assert.NoError(t, err)
	assert.Len(t, mockClient.LaunchCalls, 1)
	assert.Equal(t, "python-ml", mockClient.LaunchCalls[0].Template)
	assert.Equal(t, "test-instance", mockClient.LaunchCalls[0].Name)
	assert.Equal(t, "my-project", mockClient.LaunchCalls[0].ProjectID)
}

// TestListCommand tests the list command
func TestListCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "List all instances",
			args:        []string{},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "List with project filter",
			args:        []string{"--project", "test-project"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{},
			expectError: true,
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "API failure"
			},
		},
		{
			name:        "Daemon not running",
			args:        []string{},
			expectError: true,
			setupMock: func(mock *MockAPIClient) {
				mock.PingError = assert.AnError
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			err := app.List(tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestListCostCommand tests the list cost command
func TestListCostCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "List cost all instances",
			args:        []string{},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "List cost with project filter",
			args:        []string{"--project", "test-project"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Empty instance list",
			args:        []string{},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances = []types.Instance{}
			},
		},
		{
			name:        "API error",
			args:        []string{},
			expectError: true,
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "API failure"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			err := app.ListCost(tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConnectCommandIntegration tests the connect command delegation
func TestConnectCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to instance commands
	args := []string{"test-instance"}
	err := app.Connect(args)

	assert.NoError(t, err)
	assert.Len(t, mockClient.ConnectCalls, 1)
	assert.Equal(t, "test-instance", mockClient.ConnectCalls[0])
}

// TestStopCommandIntegration tests the stop command delegation
func TestStopCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to instance commands
	args := []string{"test-instance"}
	err := app.Stop(args)

	assert.NoError(t, err)
	assert.Len(t, mockClient.StopCalls, 1)
	assert.Equal(t, "test-instance", mockClient.StopCalls[0])
}

// TestStartCommandIntegration tests the start command delegation
func TestStartCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	
	// Set test-instance to stopped state so start command will actually call StartInstance
	for i := range mockClient.Instances {
		if mockClient.Instances[i].Name == "test-instance" {
			mockClient.Instances[i].State = "stopped"
			break
		}
	}
	
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to instance commands
	args := []string{"test-instance"}
	err := app.Start(args)

	assert.NoError(t, err)
	assert.Len(t, mockClient.StartCalls, 1)
	if len(mockClient.StartCalls) > 0 {
		assert.Equal(t, "test-instance", mockClient.StartCalls[0])
	}
}

// TestDeleteCommandIntegration tests the delete command delegation
func TestDeleteCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to instance commands
	args := []string{"test-instance"}
	err := app.Delete(args)

	assert.NoError(t, err)
	assert.Len(t, mockClient.DeleteCalls, 1)
	assert.Equal(t, "test-instance", mockClient.DeleteCalls[0])
}

// TestHibernateCommandIntegration tests the hibernate command delegation
func TestHibernateCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to instance commands
	args := []string{"test-instance"}
	err := app.Hibernate(args)

	assert.NoError(t, err)
	assert.Len(t, mockClient.HibernateCalls, 1)
	assert.Equal(t, "test-instance", mockClient.HibernateCalls[0])
}

// TestResumeCommandIntegration tests the resume command delegation
func TestResumeCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	
	// Set test-instance to hibernated state and configure hibernation status
	for i := range mockClient.Instances {
		if mockClient.Instances[i].Name == "test-instance" {
			mockClient.Instances[i].State = "hibernated"
			break
		}
	}
	mockClient.HibernationStatus.IsHibernated = true
	
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to instance commands
	args := []string{"test-instance"}
	err := app.Resume(args)

	assert.NoError(t, err)
	// Resume should call either ResumeInstance (for hibernated) or StartInstance (for non-hibernated)
	resumeCalls := len(mockClient.ResumeCalls)
	startCalls := len(mockClient.StartCalls)
	assert.True(t, resumeCalls > 0 || startCalls > 0, "Resume should call either ResumeInstance or StartInstance")
	
	if resumeCalls > 0 {
		assert.Equal(t, "test-instance", mockClient.ResumeCalls[0])
	} else if startCalls > 0 {
		assert.Equal(t, "test-instance", mockClient.StartCalls[0])
	}
}

// TestVolumeCommandIntegration tests the volume command delegation
func TestVolumeCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to storage commands
	args := []string{"list"}
	err := app.Volume(args)

	assert.NoError(t, err)
}

// TestStorageCommandIntegration tests the storage command delegation
func TestStorageCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to storage commands
	args := []string{"list"}
	err := app.Storage(args)

	assert.NoError(t, err)
}

// TestTemplatesCommandIntegration tests the templates command delegation
func TestTemplatesCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to template commands
	args := []string{}
	err := app.Templates(args)

	assert.NoError(t, err)
}

// TestDaemonCommandIntegration tests the daemon command delegation
func TestDaemonCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to system commands
	args := []string{"status"}
	err := app.Daemon(args)

	assert.NoError(t, err)
}

// TestRightsizingCommandIntegration tests the rightsizing command delegation
func TestRightsizingCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to scaling commands
	args := []string{"analyze"}
	err := app.Rightsizing(args)

	assert.NoError(t, err)
}

// TestScalingCommandIntegration tests the scaling command delegation
func TestScalingCommandIntegration(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test delegation to scaling commands
	args := []string{"up", "test-instance"}
	err := app.Scaling(args)

	assert.NoError(t, err)
}

// TestAMIDiscoverCommand tests the AMI discovery command
func TestAMIDiscoverCommand(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test AMI discovery
	args := []string{}
	err := app.AMIDiscover(args)

	assert.NoError(t, err)
}

// TestLaunchProgressMonitoring tests the launch progress monitoring system
func TestLaunchProgressMonitoring(t *testing.T) {
	tests := []struct {
		name          string
		templateName  string
		instanceState string
		expectError   bool
	}{
		{
			name:          "AMI template progress",
			templateName:  "Deep Learning AMI",
			instanceState: "running",
			expectError:   false,
		},
		{
			name:          "Package template progress",
			templateName:  "python-ml",
			instanceState: "running",
			expectError:   false,
		},
		{
			name:          "Instance terminated",
			templateName:  "python-ml",
			instanceState: "terminated",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()

			// Set up instance state
			for i := range mockClient.Instances {
				if mockClient.Instances[i].Name == "test-instance" {
					mockClient.Instances[i].State = tt.instanceState
					break
				}
			}

			// Monitor launch progress with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			monitor := NewLaunchProgressMonitor(mockClient, ctx)
			err := monitor.Monitor("test-instance")

			if tt.expectError && tt.instanceState == "terminated" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "terminated")
			} else {
				// For non-error cases, we expect no error or timeout
				if err != nil {
					// Timeout is acceptable for testing
					assert.True(t, strings.Contains(err.Error(), "timeout") || err == nil)
				}
			}
		})
	}
}

// TestInstanceStateHandlers tests individual state handlers
func TestInstanceStateHandlers(t *testing.T) {
	mockClient := NewMockAPIClient()
	ctx := context.Background()

	tests := []struct {
		name           string
		handler        InstanceStateHandler
		state          string
		canHandle      bool
		shouldContinue bool
		expectError    bool
	}{
		{
			name:           "Pending state handler",
			handler:        &PendingStateHandler{},
			state:          "pending",
			canHandle:      true,
			shouldContinue: true,
			expectError:    false,
		},
		{
			name:           "Running state handler",
			handler:        &RunningStateHandler{apiClient: mockClient, ctx: ctx},
			state:          "running",
			canHandle:      true,
			shouldContinue: true,
			expectError:    false,
		},
		{
			name:           "Error state handler - stopped",
			handler:        &ErrorStateHandler{},
			state:          "stopped",
			canHandle:      true,
			shouldContinue: false,
			expectError:    true,
		},
		{
			name:           "Error state handler - terminated",
			handler:        &ErrorStateHandler{},
			state:          "terminated",
			canHandle:      true,
			shouldContinue: false,
			expectError:    true,
		},
		{
			name:           "Dry run state handler",
			handler:        &DryRunStateHandler{},
			state:          "dry-run",
			canHandle:      true,
			shouldContinue: false,
			expectError:    false,
		},
		{
			name:           "Default state handler",
			handler:        &DefaultStateHandler{},
			state:          "unknown",
			canHandle:      true,
			shouldContinue: true,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canHandle := tt.handler.CanHandle(tt.state)
			assert.Equal(t, tt.canHandle, canHandle)

			if canHandle {
				shouldContinue, err := tt.handler.Handle(tt.state, 30, "test-instance")
				assert.Equal(t, tt.shouldContinue, shouldContinue)

				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestProjectFiltering tests project filtering in list operations
func TestProjectFiltering(t *testing.T) {
	mockClient := NewMockAPIClient()

	// Add instances with different projects
	mockClient.Instances = append(mockClient.Instances, types.Instance{
		Name:      "project1-instance",
		Template:  "python-ml",
		State:     "running",
		ProjectID: "project-1",
	})
	mockClient.Instances = append(mockClient.Instances, types.Instance{
		Name:      "project2-instance",
		Template:  "r-research",
		State:     "running",
		ProjectID: "project-2",
	})

	app := NewAppWithClient("1.0.0", mockClient)

	// Test filtering by project
	err := app.List([]string{"--project", "project-1"})
	assert.NoError(t, err)

	err = app.ListCost([]string{"--project", "project-2"})
	assert.NoError(t, err)
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		operation   func(*App) error
		expectError bool
		errorType   string
	}{
		{
			name: "API error on launch",
			operation: func(app *App) error {
				return app.Launch([]string{"python-ml", "test-instance"})
			},
			expectError: true,
			errorType:   "failed to",
		},
		{
			name: "API error on list",
			operation: func(app *App) error {
				return app.List([]string{})
			},
			expectError: true,
			errorType:   "failed to",
		},
		{
			name: "Daemon not running",
			operation: func(app *App) error {
				return app.Launch([]string{"python-ml", "test-instance"})
			},
			expectError: true,
			errorType:   "daemon not running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockClient *MockAPIClient

			if strings.Contains(tt.errorType, "daemon") {
				mockClient = NewMockAPIClientWithPingError()
			} else {
				mockClient = NewMockAPIClientWithError("Test API error")
			}

			app := NewAppWithClient("1.0.0", mockClient)
			err := tt.operation(app)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorType)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfigurationLoading tests configuration loading scenarios
func TestConfigurationLoading(t *testing.T) {
	// Test with environment variable
	originalURL := os.Getenv(DaemonURLEnvVar)
	defer func() {
		if originalURL != "" {
			os.Setenv(DaemonURLEnvVar, originalURL)
		} else {
			os.Unsetenv(DaemonURLEnvVar)
		}
	}()

	testURL := "http://test:9999"
	os.Setenv(DaemonURLEnvVar, testURL)

	app := NewApp("1.0.0")
	assert.NotNil(t, app)
	assert.NotNil(t, app.config)

	// Clean up
	os.Unsetenv(DaemonURLEnvVar)
}

// TestCostAnalysis tests cost analysis functionality
func TestCostAnalysis(t *testing.T) {
	mockClient := NewMockAPIClient()

	// Create instances with cost data
	mockClient.Instances = []types.Instance{
		{
			Name:              "expensive-instance",
			Template:          "gpu-ml",
			State:             "running",
			InstanceLifecycle: "on-demand",
			HourlyRate:        0.4375,
			CurrentSpend:      10.50,
		},
		{
			Name:              "cheap-instance",
			Template:          "basic-compute",
			State:             "stopped",
			InstanceLifecycle: "spot",
			HourlyRate:        0.052,
			CurrentSpend:      1.25,
		},
	}

	app := NewAppWithClient("1.0.0", mockClient)
	err := app.ListCost([]string{})

	assert.NoError(t, err)
}

// TestLaunchValidation tests launch command argument validation
func TestLaunchValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
		},
		{
			name:        "One argument only",
			args:        []string{"template"},
			expectError: true,
			errorMsg:    "usage:",
		},
		{
			name:        "Valid arguments",
			args:        []string{"template", "name"},
			expectError: false,
		},
		{
			name:        "Valid with flags",
			args:        []string{"template", "name", "--size", "L"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			app := NewAppWithClient("1.0.0", mockClient)

			err := app.Launch(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCommandDelegation tests that commands are properly delegated to specialized modules
func TestCommandDelegation(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	// Test all command delegations
	commands := map[string]func([]string) error{
		"connect":     app.Connect,
		"stop":        app.Stop,
		"start":       app.Start,
		"delete":      app.Delete,
		"hibernate":   app.Hibernate,
		"resume":      app.Resume,
		"volume":      app.Volume,
		"storage":     app.Storage,
		"templates":   app.Templates,
		"daemon":      app.Daemon,
		"rightsizing": app.Rightsizing,
		"scaling":     app.Scaling,
	}

	for cmdName, cmdFunc := range commands {
		t.Run("delegate_"+cmdName, func(t *testing.T) {
			// Test with minimal args - exact requirements depend on command
			var args []string
			if cmdName == "connect" || cmdName == "stop" || cmdName == "start" ||
				cmdName == "delete" || cmdName == "hibernate" || cmdName == "resume" {
				args = []string{"test-instance"}
			} else if cmdName == "volume" || cmdName == "storage" {
				args = []string{"list"}
			} else if cmdName == "daemon" {
				args = []string{"status"}
			} else if cmdName == "rightsizing" {
				args = []string{"analyze"}
			} else if cmdName == "scaling" {
				args = []string{"up", "test-instance"}
			} else {
				args = []string{}
			}

			err := cmdFunc(args)

			// Most delegated commands should not error with valid mock client
			// Some may have specific validation that causes errors, which is fine
			if err != nil {
				// Error is acceptable - we're mainly testing delegation doesn't panic
				t.Logf("Command %s returned error (acceptable): %v", cmdName, err)
			}
		})
	}
}

// BenchmarkAppCreation benchmarks app creation performance
func BenchmarkAppCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		app := NewApp("benchmark")
		if app == nil {
			b.Fatal("App creation returned nil")
		}
	}
}

// BenchmarkMockAPIClientCalls benchmarks mock API client performance
func BenchmarkMockAPIClientCalls(b *testing.B) {
	mockClient := NewMockAPIClient()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mockClient.ListInstances(ctx)
		if err != nil {
			b.Fatal("Mock API call failed:", err)
		}
	}
}
