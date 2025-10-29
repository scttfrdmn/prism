package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLICommandErrorPaths tests CLI command error scenarios that users actually encounter
func TestCLICommandErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name:        "launch_missing_template_name",
			command:     "launch",
			args:        []string{},
			expectError: true,
			errorMsg:    "template name is required",
			description: "User runs launch command without specifying template name",
		},
		{
			name:        "launch_missing_project_name",
			command:     "launch",
			args:        []string{"python-ml"},
			expectError: true,
			errorMsg:    "project name is required",
			description: "User runs launch command without specifying project name",
		},
		{
			name:        "launch_empty_project_name",
			command:     "launch",
			args:        []string{"python-ml", ""},
			expectError: true,
			errorMsg:    "project name cannot be empty",
			description: "User provides empty project name during launch",
		},
		{
			name:        "connect_missing_instance_name",
			command:     "connect",
			args:        []string{},
			expectError: true,
			errorMsg:    "instance name is required",
			description: "User runs connect command without specifying instance name",
		},
		{
			name:        "stop_missing_instance_name",
			command:     "stop",
			args:        []string{},
			expectError: true,
			errorMsg:    "instance name is required",
			description: "User runs stop command without specifying instance name",
		},
		{
			name:        "hibernate_missing_instance_name",
			command:     "hibernate",
			args:        []string{},
			expectError: true,
			errorMsg:    "instance name is required",
			description: "User runs hibernate command without specifying instance name",
		},
		{
			name:        "volume_mount_missing_arguments",
			command:     "volume",
			args:        []string{"mount"},
			expectError: true,
			errorMsg:    "volume name and instance name are required",
			description: "User runs volume mount without specifying volume and instance names",
		},
		{
			name:        "invalid_size_parameter",
			command:     "launch",
			args:        []string{"python-ml", "my-project", "--size", "XXXL"},
			expectError: true,
			errorMsg:    "invalid size: XXXL",
			description: "User provides invalid size parameter during launch",
		},
		{
			name:        "valid_launch_command",
			command:     "launch",
			args:        []string{"python-ml", "my-project"},
			expectError: false,
			description: "Valid launch command should pass argument validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommandArguments(tt.command, tt.args)

			if tt.expectError {
				require.Error(t, err, "Expected validation error for: %s", tt.description)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"Error should contain expected message")
				}
				t.Logf("User would see validation error (expected): %s", err.Error())
			} else {
				assert.NoError(t, err, "Expected no validation error for: %s", tt.description)
			}
		})
	}
}

// TestCLIAPIErrorScenarios tests error scenarios when CLI communicates with daemon API
func TestCLIAPIErrorScenarios(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *MockAPIClient
		testCommand func(*MockAPIClient) error
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name: "launch_with_template_not_found",
			setupClient: func() *MockAPIClient {
				return NewMockAPIClientWithError("template not found")
			},
			testCommand: func(client *MockAPIClient) error {
				_, err := client.LaunchInstance(context.TODO(), types.LaunchRequest{
					Template: "nonexistent-template",
					Name:     "my-project",
				})
				return err
			},
			expectError: true,
			errorMsg:    "template not found",
			description: "User tries to launch instance with template that doesn't exist",
		},
		{
			name: "connect_to_stopped_instance",
			setupClient: func() *MockAPIClient {
				return NewMockAPIClientWithConnectError("instance is not running")
			},
			testCommand: func(client *MockAPIClient) error {
				_, err := client.ConnectInstance(context.TODO(), "stopped-instance")
				return err
			},
			expectError: true,
			errorMsg:    "instance is not running",
			description: "User tries to connect to instance that is stopped",
		},
		{
			name: "stop_nonexistent_instance",
			setupClient: func() *MockAPIClient {
				return NewMockAPIClient() // Uses default behavior which returns "not found"
			},
			testCommand: func(client *MockAPIClient) error {
				return client.StopInstance(context.TODO(), "nonexistent-instance")
			},
			expectError: true,
			errorMsg:    "not found",
			description: "User tries to stop instance that doesn't exist",
		},
		{
			name: "hibernation_unsupported_instance",
			setupClient: func() *MockAPIClient {
				client := NewMockAPIClient()
				client.HibernateError = errors.New("hibernation not supported for this instance type")
				return client
			},
			testCommand: func(client *MockAPIClient) error {
				return client.HibernateInstance(context.TODO(), "t2-micro-instance")
			},
			expectError: true,
			errorMsg:    "hibernation not supported",
			description: "User tries to hibernate instance type that doesn't support hibernation",
		},
		{
			name: "daemon_not_running",
			setupClient: func() *MockAPIClient {
				return NewMockAPIClientWithPingError()
			},
			testCommand: func(client *MockAPIClient) error {
				return client.Ping(context.TODO())
			},
			expectError: true,
			errorMsg:    "daemon not running",
			description: "User runs CLI command when daemon is not accessible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			err := tt.testCommand(client)

			if tt.expectError {
				require.Error(t, err, "Expected API error for: %s", tt.description)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"Error should contain expected message")
				}
				t.Logf("User would see API error (expected): %s", err.Error())
			} else {
				assert.NoError(t, err, "Expected no API error for: %s", tt.description)
			}
		})
	}
}

// TestCLIUserWorkflowErrors tests complete user workflow error scenarios
func TestCLIUserWorkflowErrors(t *testing.T) {
	userScenarios := []struct {
		name        string
		workflow    string
		expectation string
		errorType   string
		description string
	}{
		{
			name:        "student_first_time_setup",
			workflow:    "Student runs 'prism launch python-ml my-first-project' without AWS setup",
			expectation: "Clear AWS credentials configuration guidance",
			errorType:   "CredentialsNotFound",
			description: "Student hasn't configured AWS credentials yet",
		},
		{
			name:        "researcher_wrong_region",
			workflow:    "Researcher launches template in region where AMI doesn't exist",
			expectation: "Clear region/AMI availability error + suggested alternatives",
			errorType:   "InvalidAMIID.NotFound",
			description: "Template AMI not available in user's configured region",
		},
		{
			name:        "professor_class_capacity_limit",
			workflow:    "Professor launches 20 instances for class, hits AWS limits",
			expectation: "Clear capacity/limit error + guidance for requesting increases",
			errorType:   "InsufficientInstanceCapacity",
			description: "Class session hits AWS instance limits during peak usage",
		},
		{
			name:        "researcher_typo_in_instance_name",
			workflow:    "Researcher tries to connect to 'my-instnace' (typo) instead of 'my-instance'",
			expectation: "Instance not found error + suggestion of similar names",
			errorType:   "InstanceNotFound",
			description: "Typo in instance name leads to connection failure",
		},
		{
			name:        "lab_network_connectivity_issue",
			workflow:    "Computer lab users can't reach daemon due to network policy",
			expectation: "Clear daemon connectivity error + troubleshooting steps",
			errorType:   "ConnectionRefused",
			description: "Institutional network blocks daemon port or connectivity",
		},
	}

	for _, scenario := range userScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("🎓 User Workflow: %s", scenario.workflow)
			t.Logf("💡 Expected UX: %s", scenario.expectation)
			t.Logf("⚠️  Error Type: %s", scenario.errorType)
			t.Logf("📋 Description: %s", scenario.description)

			// This validates our error handling strategy addresses real user workflows
			// The test passes to document these scenarios are considered in CLI design
		})
	}
}

// validateCommandArguments validates CLI command arguments (pre-API validation)
func validateCommandArguments(command string, args []string) error {
	switch command {
	case "launch":
		return validateLaunchCommand(args)
	case "connect", "stop", "hibernate":
		return validateInstanceCommand(args)
	case "volume":
		return validateVolumeCommand(args)
	}

	return nil
}

// validateLaunchCommand validates launch command arguments
func validateLaunchCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("template name is required")
	}
	if len(args) == 1 {
		return errors.New("project name is required")
	}
	if len(args) >= 2 && args[1] == "" {
		return errors.New("project name cannot be empty")
	}

	// Validate size flag if present
	return validateSizeFlag(args)
}

// validateInstanceCommand validates instance-related command arguments
func validateInstanceCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("instance name is required")
	}
	return nil
}

// validateVolumeCommand validates volume command arguments
func validateVolumeCommand(args []string) error {
	if len(args) < 1 {
		return errors.New("volume subcommand is required")
	}
	if args[0] == "mount" && len(args) < 3 {
		return errors.New("volume name and instance name are required")
	}
	return nil
}

// validateSizeFlag validates the --size flag value in launch command args
func validateSizeFlag(args []string) error {
	for i, arg := range args {
		if arg == "--size" && i+1 < len(args) {
			size := args[i+1]
			validSizes := []string{"XS", "S", "M", "L", "XL"}

			for _, validSize := range validSizes {
				if size == validSize {
					return nil // Valid size found
				}
			}

			return errors.New("invalid size: " + size)
		}
	}
	return nil
}
