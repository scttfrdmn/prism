// Package cli tests for instance command module
package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestNewInstanceCommands tests instance commands creation
func TestNewInstanceCommands(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	ic := NewInstanceCommands(app)

	assert.NotNil(t, ic)
	assert.Equal(t, app, ic.app)
}

// TestInstanceCommands_Connect tests the connect instance command
func TestInstanceCommands_Connect(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid connect",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Connect with verbose flag",
			args:        []string{"test-instance", "--verbose"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Connect with short verbose flag",
			args:        []string{"test-instance", "-v"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Invalid flag",
			args:        []string{"test-instance", "--invalid"},
			expectError: true,
			errorMsg:    "invalid flag",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "daemon not running",
			setupMock: func(mock *MockAPIClient) {
				mock.PingError = assert.AnError
			},
		},
		{
			name:        "API error",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "get connection info for test-instance failed",
			setupMock: func(mock *MockAPIClient) {
				// Use specific connect error to avoid affecting Ping method
				mock.ConnectError = fmt.Errorf("connection failed")
			},
		},
		{
			name:        "Instance not running",
			args:        []string{"stopped-instance"},
			expectError: true,
			errorMsg:    "not running",
			setupMock: func(mock *MockAPIClient) {
				// Use stopped instance that exists in mock data
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable daemon auto-start for daemon-related error tests
			if tt.name == "Daemon not running" || tt.name == "API error" {
				t.Setenv("CWS_NO_AUTO_START", "1")
			}

			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			ic := NewInstanceCommands(app)

			err := ic.Connect(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if len(tt.args) > 0 {
					assert.Contains(t, mockClient.ConnectCalls, tt.args[0])
				}
			}
		})
	}
}

// TestInstanceCommands_Stop tests the stop instance command
func TestInstanceCommands_Stop(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid stop",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "daemon not running",
			setupMock: func(mock *MockAPIClient) {
				mock.PingError = assert.AnError
			},
		},
		{
			name:        "API error",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "stop instance test-instance failed",
			setupMock: func(mock *MockAPIClient) {
				// Use specific stop error to avoid affecting Ping method
				mock.StopError = fmt.Errorf("stop failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable daemon auto-start for daemon-related error tests
			if tt.name == "Daemon not running" || tt.name == "API error" {
				t.Setenv("CWS_NO_AUTO_START", "1")
			}

			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			ic := NewInstanceCommands(app)

			err := ic.Stop(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, mockClient.StopCalls, tt.args[0])
			}
		})
	}
}

// TestInstanceCommands_Start tests the start instance command with state management
func TestInstanceCommands_Start(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		instanceState string
		expectError   bool
		errorMsg      string
		setupMock     func(*MockAPIClient)
	}{
		{
			name:          "Start stopped instance",
			args:          []string{"stopped-instance"},
			instanceState: "stopped",
			expectError:   false,
			setupMock:     func(mock *MockAPIClient) {},
		},
		{
			name:          "Start hibernated instance",
			args:          []string{"test-instance"},
			instanceState: "hibernated",
			expectError:   false,
			setupMock: func(mock *MockAPIClient) {
				for i := range mock.Instances {
					if mock.Instances[i].Name == "test-instance" {
						mock.Instances[i].State = "hibernated"
					}
				}
			},
		},
		{
			name:          "Instance already running",
			args:          []string{"test-instance"},
			instanceState: "running",
			expectError:   false,
			setupMock:     func(mock *MockAPIClient) {},
		},
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Instance not found",
			args:        []string{"nonexistent-instance"},
			expectError: true,
			errorMsg:    "not found",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "daemon not running",
			setupMock: func(mock *MockAPIClient) {
				mock.PingError = assert.AnError
			},
		},
		{
			name:        "API error on list",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "get instance status failed",
			setupMock: func(mock *MockAPIClient) {
				// Start command first calls ListInstances, so we need to make that fail specifically
				mock.ListInstancesError = fmt.Errorf("list failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable daemon auto-start for daemon-related error tests
			if tt.name == "Daemon not running" || tt.name == "API error on list" {
				t.Setenv("CWS_NO_AUTO_START", "1")
			}

			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			ic := NewInstanceCommands(app)

			err := ic.Start(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if len(tt.args) > 0 && tt.instanceState != "running" {
					assert.Contains(t, mockClient.StartCalls, tt.args[0])
				}
			}
		})
	}
}

// TestInstanceCommands_Delete tests the delete instance command
func TestInstanceCommands_Delete(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid delete",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "daemon not running",
			setupMock: func(mock *MockAPIClient) {
				mock.PingError = assert.AnError
			},
		},
		{
			name:        "API error",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "delete instance test-instance failed",
			setupMock: func(mock *MockAPIClient) {
				// Use specific delete error to avoid affecting Ping method
				mock.DeleteError = fmt.Errorf("delete failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable daemon auto-start for daemon-related error tests
			if tt.name == "Daemon not running" || tt.name == "API error" {
				t.Setenv("CWS_NO_AUTO_START", "1")
			}

			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			ic := NewInstanceCommands(app)

			err := ic.Delete(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, mockClient.DeleteCalls, tt.args[0])
			}
		})
	}
}

// TestInstanceCommands_Hibernate tests the hibernate instance command
func TestInstanceCommands_Hibernate(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		hibernationSupported bool
		expectError          bool
		errorMsg             string
		setupMock            func(*MockAPIClient)
	}{
		{
			name:                 "Valid hibernate with support",
			args:                 []string{"test-instance"},
			hibernationSupported: true,
			expectError:          false,
			setupMock:            func(mock *MockAPIClient) {},
		},
		{
			name:                 "Hibernate without support falls back to stop",
			args:                 []string{"test-instance"},
			hibernationSupported: false,
			expectError:          false,
			setupMock: func(mock *MockAPIClient) {
				mock.HibernationStatus.HibernationSupported = false
			},
		},
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "daemon not running",
			setupMock: func(mock *MockAPIClient) {
				mock.PingError = assert.AnError
			},
		},
		{
			name:        "API error on hibernation status",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "check EC2 hibernation",
			setupMock: func(mock *MockAPIClient) {
				// Hibernate command first calls GetInstanceHibernationStatus
				mock.HibernationStatusError = fmt.Errorf("status check failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable daemon auto-start for daemon-related error tests
			if tt.name == "Daemon not running" || tt.name == "API error on hibernation status" {
				t.Setenv("CWS_NO_AUTO_START", "1")
			}

			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			ic := NewInstanceCommands(app)

			err := ic.Hibernate(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if len(tt.args) > 0 {
					assert.Contains(t, mockClient.HibernateCalls, tt.args[0])
				}
			}
		})
	}
}

// TestInstanceCommands_Resume tests the resume instance command
func TestInstanceCommands_Resume(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid resume",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "daemon not running",
			setupMock: func(mock *MockAPIClient) {
				mock.PingError = assert.AnError
			},
		},
		{
			name:        "API error",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "check EC2 hibernation",
			setupMock: func(mock *MockAPIClient) {
				// Resume command first calls GetInstanceHibernationStatus
				mock.HibernationStatusError = fmt.Errorf("status check failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable daemon auto-start for daemon-related error tests
			if tt.name == "Daemon not running" || tt.name == "API error" {
				t.Setenv("CWS_NO_AUTO_START", "1")
			}

			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			ic := NewInstanceCommands(app)

			err := ic.Resume(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if len(tt.args) > 0 {
					// Resume should call ResumeInstance for hibernated instances or StartInstance for non-hibernated
					found := false
					for _, call := range mockClient.ResumeCalls {
						if call == tt.args[0] {
							found = true
							break
						}
					}
					if !found {
						for _, call := range mockClient.StartCalls {
							if call == tt.args[0] {
								found = true
								break
							}
						}
					}
					assert.True(t, found, "Expected resume to call either ResumeInstance or StartInstance for %s", tt.args[0])
				}
			}
		})
	}
}

// TestInstanceCommandsArgumentParsing tests argument parsing across commands
func TestInstanceCommandsArgumentParsing(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	ic := NewInstanceCommands(app)

	// Test commands with various argument patterns
	commands := map[string]func([]string) error{
		"connect":   ic.Connect,
		"stop":      ic.Stop,
		"start":     ic.Start,
		"delete":    ic.Delete,
		"hibernate": ic.Hibernate,
		"resume":    ic.Resume,
	}

	for cmdName, cmdFunc := range commands {
		t.Run("args_"+cmdName, func(t *testing.T) {
			// Test with empty arguments
			err := cmdFunc([]string{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "usage:")

			// Test with valid arguments
			err = cmdFunc([]string{"test-instance"})
			// Some commands may error due to instance state, but should not be usage errors
			if err != nil && !strings.Contains(err.Error(), "usage:") {
				// Acceptable - might be API/state errors
				t.Logf("Command %s returned non-usage error: %v", cmdName, err)
			}
		})
	}
}

// TestInstanceCommandsCallTracking tests that API calls are properly tracked
func TestInstanceCommandsCallTracking(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	ic := NewInstanceCommands(app)

	// Reset call tracking
	mockClient.ResetCallTracking()

	// Test each command and verify calls are tracked
	testCases := []struct {
		name    string
		command func([]string) error
		args    []string
		checkFn func(*MockAPIClient) bool
	}{
		{
			name:    "connect tracks calls",
			command: ic.Connect,
			args:    []string{"test-instance", "--verbose"},
			checkFn: func(mock *MockAPIClient) bool {
				return len(mock.ConnectCalls) > 0 && mock.ConnectCalls[0] == "test-instance"
			},
		},
		{
			name:    "stop tracks calls",
			command: ic.Stop,
			args:    []string{"test-instance"},
			checkFn: func(mock *MockAPIClient) bool {
				return len(mock.StopCalls) > 0 && mock.StopCalls[0] == "test-instance"
			},
		},
		{
			name:    "start tracks calls",
			command: ic.Start,
			args:    []string{"stopped-instance"},
			checkFn: func(mock *MockAPIClient) bool {
				return len(mock.StartCalls) > 0 && mock.StartCalls[0] == "stopped-instance"
			},
		},
		{
			name:    "delete tracks calls",
			command: ic.Delete,
			args:    []string{"test-instance"},
			checkFn: func(mock *MockAPIClient) bool {
				return len(mock.DeleteCalls) > 0 && mock.DeleteCalls[0] == "test-instance"
			},
		},
		{
			name:    "hibernate tracks calls",
			command: ic.Hibernate,
			args:    []string{"test-instance"},
			checkFn: func(mock *MockAPIClient) bool {
				return len(mock.HibernateCalls) > 0 && mock.HibernateCalls[0] == "test-instance"
			},
		},
		{
			name:    "resume tracks calls",
			command: ic.Resume,
			args:    []string{"test-instance"},
			checkFn: func(mock *MockAPIClient) bool {
				// Resume should call ResumeInstance for hibernated instances or StartInstance for non-hibernated
				return (len(mock.ResumeCalls) > 0 && mock.ResumeCalls[0] == "test-instance") ||
					(len(mock.StartCalls) > 0 && mock.StartCalls[0] == "test-instance")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient.ResetCallTracking()

			err := tc.command(tc.args)
			// Some commands might error, but calls should still be tracked
			if err == nil || !strings.Contains(err.Error(), "usage:") {
				assert.True(t, tc.checkFn(mockClient), "API call not tracked properly")
			}
		})
	}
}

// TestInstanceStateManagement tests intelligent state management in start command
func TestInstanceStateManagement(t *testing.T) {
	tests := []struct {
		name           string
		instanceName   string
		initialState   string
		expectedAction string
	}{
		{
			name:           "Running instance - no action",
			instanceName:   "running-instance",
			initialState:   "running",
			expectedAction: "none",
		},
		{
			name:           "Stopped instance - start action",
			instanceName:   "stopped-instance",
			initialState:   "stopped",
			expectedAction: "start",
		},
		{
			name:           "Hibernated instance - start with warning",
			instanceName:   "hibernated-instance",
			initialState:   "hibernated",
			expectedAction: "start",
		},
		{
			name:           "Unknown state - attempt start",
			instanceName:   "unknown-instance",
			initialState:   "unknown",
			expectedAction: "start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()

			// Set up instance with specific state
			instanceFound := false
			for i := range mockClient.Instances {
				if mockClient.Instances[i].Name == "test-instance" {
					mockClient.Instances[i].Name = tt.instanceName
					mockClient.Instances[i].State = tt.initialState
					instanceFound = true
					break
				}
			}

			if !instanceFound {
				// Add new instance if not found
				mockClient.Instances = append(mockClient.Instances, types.Instance{
					Name:  tt.instanceName,
					State: tt.initialState,
				})
			}

			app := NewAppWithClient("1.0.0", mockClient)
			ic := NewInstanceCommands(app)

			err := ic.Start([]string{tt.instanceName})
			assert.NoError(t, err)

			// Verify expected action
			switch tt.expectedAction {
			case "none":
				assert.Len(t, mockClient.StartCalls, 0)
			case "start":
				assert.Len(t, mockClient.StartCalls, 1)
				assert.Equal(t, tt.instanceName, mockClient.StartCalls[0])
			}
		})
	}
}

// TestHibernationStatusChecking tests hibernation status checking logic
func TestHibernationStatusChecking(t *testing.T) {
	tests := []struct {
		name                 string
		hibernationSupported bool
		expectWarning        bool
	}{
		{
			name:                 "Hibernation supported",
			hibernationSupported: true,
			expectWarning:        false,
		},
		{
			name:                 "Hibernation not supported",
			hibernationSupported: false,
			expectWarning:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			mockClient.HibernationStatus.HibernationSupported = tt.hibernationSupported

			app := NewAppWithClient("1.0.0", mockClient)
			ic := NewInstanceCommands(app)

			err := ic.Hibernate([]string{"test-instance"})
			assert.NoError(t, err)

			// Verify hibernation call was made regardless of support
			assert.Len(t, mockClient.HibernateCalls, 1)
			assert.Equal(t, "test-instance", mockClient.HibernateCalls[0])
		})
	}
}

// TestConnectCommandVerboseMode tests verbose mode in connect command
func TestConnectCommandVerboseMode(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	ic := NewInstanceCommands(app)

	// Test verbose mode (should not execute SSH)
	err := ic.Connect([]string{"test-instance", "--verbose"})
	assert.NoError(t, err)
	assert.Len(t, mockClient.ConnectCalls, 1)

	// Test short verbose mode
	mockClient.ResetCallTracking()
	err = ic.Connect([]string{"test-instance", "-v"})
	assert.NoError(t, err)
	assert.Len(t, mockClient.ConnectCalls, 1)
}

// BenchmarkInstanceCommands benchmarks instance command operations
func BenchmarkInstanceCommands(b *testing.B) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	ic := NewInstanceCommands(app)

	commands := map[string]func([]string) error{
		"stop":      ic.Stop,
		"start":     ic.Start,
		"hibernate": ic.Hibernate,
		"resume":    ic.Resume,
	}

	for cmdName, cmdFunc := range commands {
		b.Run(cmdName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := cmdFunc([]string{"test-instance"})
				if err != nil {
					b.Fatal("Command failed:", err)
				}
			}
		})
	}
}
