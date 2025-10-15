// Package cli tests for scaling command module
package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestNewScalingCommands tests scaling commands creation
func TestNewScalingCommands(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	sc := NewScalingCommands(app)

	assert.NotNil(t, sc)
	assert.Equal(t, app, sc.app)
}

// TestScalingCommands_Rightsizing tests the rightsizing command routing
func TestScalingCommands_Rightsizing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Analyze subcommand",
			args:        []string{"analyze", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Recommendations subcommand",
			args:        []string{"recommendations"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Stats subcommand",
			args:        []string{"stats", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Export subcommand",
			args:        []string{"export", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Summary subcommand",
			args:        []string{"summary"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Invalid subcommand",
			args:        []string{"invalid"},
			expectError: true,
			errorMsg:    "unknown rightsizing subcommand",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"analyze", "test-instance"},
			expectError: true,
			errorMsg:    "daemon",
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
			sc := NewScalingCommands(app)

			err := sc.Rightsizing(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRightsizingAnalyze tests rightsizing analysis functionality
func TestRightsizingAnalyze(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid analyze with running instance",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No instance name",
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
			name:        "Instance not running",
			args:        []string{"stopped-instance"},
			expectError: true,
			errorMsg:    "expected 'running'",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "daemon",
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
			sc := NewScalingCommands(app)

			err := sc.rightsizingAnalyze(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRightsizingRecommendations tests rightsizing recommendations
func TestRightsizingRecommendations(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "With running instances",
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No instances",
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances = []types.Instance{}
			},
		},
		{
			name:        "Only stopped instances",
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				for i := range mock.Instances {
					mock.Instances[i].State = "stopped"
				}
			},
		},
		{
			name:        "API error",
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
			sc := NewScalingCommands(app)

			err := sc.rightsizingRecommendations([]string{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRightsizingStats tests rightsizing statistics
func TestRightsizingStats(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid stats for running instance",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Stats for stopped instance",
			args:        []string{"stopped-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No instance name",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewScalingCommands(app)

			err := sc.rightsizingStats(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRightsizingExport tests rightsizing data export
func TestRightsizingExport(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid export",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No instance name",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewScalingCommands(app)

			err := sc.rightsizingExport(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRightsizingSummary tests rightsizing fleet summary
func TestRightsizingSummary(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "With mixed instances",
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No instances",
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances = []types.Instance{}
			},
		},
		{
			name:        "All running instances",
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				for i := range mock.Instances {
					mock.Instances[i].State = "running"
					mock.Instances[i].CurrentSpend = 2.5
				}
			},
		},
		{
			name:        "API error",
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
			sc := NewScalingCommands(app)

			err := sc.rightsizingSummary([]string{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestScalingCommands_Scaling tests the scaling command routing
func TestScalingCommands_Scaling(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "No arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Analyze subcommand",
			args:        []string{"analyze", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Scale subcommand",
			args:        []string{"scale", "test-instance", "L"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Preview subcommand",
			args:        []string{"preview", "test-instance", "L"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "History subcommand",
			args:        []string{"history", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Invalid subcommand",
			args:        []string{"invalid"},
			expectError: true,
			errorMsg:    "unknown scaling subcommand",
			setupMock:   func(mock *MockAPIClient) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewScalingCommands(app)

			err := sc.Scaling(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestScalingAnalyze tests scaling analysis functionality
func TestScalingAnalyze(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid analyze with running instance",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances[0].InstanceType = "t3.medium"
			},
		},
		{
			name:        "Analyze stopped instance",
			args:        []string{"stopped-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No instance name",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewScalingCommands(app)

			err := sc.scalingAnalyze(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestScalingScale tests scaling execution functionality
func TestScalingScale(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid scale operation",
			args:        []string{"test-instance", "L"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances[0].InstanceType = "t3.medium"
			},
		},
		{
			name:        "Scale to same size",
			args:        []string{"test-instance", "M"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances[0].InstanceType = "t3.medium"
			},
		},
		{
			name:        "Invalid size",
			args:        []string{"test-instance", "INVALID"},
			expectError: true,
			errorMsg:    "invalid size",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Missing arguments",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Instance not found",
			args:        []string{"nonexistent-instance", "L"},
			expectError: true,
			errorMsg:    "not found",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Instance in invalid state",
			args:        []string{"test-instance", "L"},
			expectError: true,
			errorMsg:    "expected 'running or stopped'",
			setupMock: func(mock *MockAPIClient) {
				mock.Instances[0].State = "pending"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewScalingCommands(app)

			err := sc.scalingScale(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestScalingPreview tests scaling preview functionality
func TestScalingPreview(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid preview",
			args:        []string{"test-instance", "L"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances[0].InstanceType = "t3.medium"
			},
		},
		{
			name:        "Preview with cost increase",
			args:        []string{"test-instance", "XL"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances[0].InstanceType = "t3.small"
				mock.Instances[0].CurrentSpend = 1.0
			},
		},
		{
			name:        "Preview with cost decrease",
			args:        []string{"test-instance", "S"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances[0].InstanceType = "t3.large"
				mock.Instances[0].CurrentSpend = 4.0
			},
		},
		{
			name:        "Invalid size",
			args:        []string{"test-instance", "INVALID"},
			expectError: true,
			errorMsg:    "invalid size",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Missing arguments",
			args:        []string{"test-instance"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewScalingCommands(app)

			err := sc.scalingPreview(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestScalingHistory tests scaling history functionality
func TestScalingHistory(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid history request",
			args:        []string{"test-instance"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Instances[0].InstanceType = "t3.medium"
				mock.Instances[0].LaunchTime = time.Now().Add(-24 * time.Hour)
			},
		},
		{
			name:        "No instance name",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewScalingCommands(app)

			err := sc.scalingHistory(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestScalingHelperFunctions tests scaling helper functions
func TestScalingHelperFunctions(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	sc := NewScalingCommands(app)

	// Test instance size parsing
	tests := []struct {
		instanceType string
		expectedSize string
	}{
		{"t3.small", "S"},
		{"t3.medium", "M"},
		{"t3.large", "L"},
		{"t3.xlarge", "XL"},
		{"t4g.small", "S"},
		{"unknown-type", "Unknown"},
	}

	for _, tt := range tests {
		t.Run("parse_"+tt.instanceType, func(t *testing.T) {
			result := sc.parseInstanceSize(tt.instanceType)
			assert.Equal(t, tt.expectedSize, result)
		})
	}

	// Test instance type for size
	sizeTests := []struct {
		size         string
		expectedType string
	}{
		{"XS", "t4g.nano"},
		{"S", "t4g.small"},
		{"M", "t4g.medium"},
		{"L", "t4g.large"},
		{"XL", "t4g.xlarge"},
		{"INVALID", "unknown"},
	}

	for _, tt := range sizeTests {
		t.Run("type_for_"+tt.size, func(t *testing.T) {
			result := sc.getInstanceTypeForSize(tt.size)
			assert.Equal(t, tt.expectedType, result)
		})
	}

	// Test cost estimation
	costTests := []struct {
		size         string
		expectedCost float64
	}{
		{"XS", 0.50},
		{"S", 1.00},
		{"M", 2.00},
		{"L", 4.00},
		{"XL", 8.00},
		{"INVALID", 0.0},
	}

	for _, tt := range costTests {
		t.Run("cost_"+tt.size, func(t *testing.T) {
			result := sc.estimateCostForSize(tt.size)
			assert.Equal(t, tt.expectedCost, result)
		})
	}

	// Test size specs
	specsTests := []struct {
		size     string
		expected SizeSpecs
	}{
		{"M", SizeSpecs{"2vCPU", "8GB", "1TB"}},
		{"L", SizeSpecs{"4vCPU", "16GB", "2TB"}},
		{"INVALID", SizeSpecs{"Unknown", "Unknown", "Unknown"}},
	}

	for _, tt := range specsTests {
		t.Run("specs_"+tt.size, func(t *testing.T) {
			result := sc.getSizeSpecs(tt.size)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestScalingCommandsArgumentValidation tests argument validation
func TestScalingCommandsArgumentValidation(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	sc := NewScalingCommands(app)

	// Test commands that require instance names
	instanceCommands := map[string]func([]string) error{
		"rightsizing analyze": sc.rightsizingAnalyze,
		"rightsizing stats":   sc.rightsizingStats,
		"rightsizing export":  sc.rightsizingExport,
		"scaling analyze":     sc.scalingAnalyze,
		"scaling history":     sc.scalingHistory,
	}

	for cmdName, cmdFunc := range instanceCommands {
		t.Run("no_args_"+cmdName, func(t *testing.T) {
			err := cmdFunc([]string{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "usage:")
		})

		t.Run("valid_args_"+cmdName, func(t *testing.T) {
			err := cmdFunc([]string{"test-instance"})
			// Command may succeed or fail due to mock state, but shouldn't be usage error
			if err != nil && strings.Contains(err.Error(), "usage:") {
				t.Errorf("Unexpected usage error for valid args in %s: %v", cmdName, err)
			}
		})
	}

	// Test scaling commands that require instance and size
	scalingCommands := []func([]string) error{
		sc.scalingScale,
		sc.scalingPreview,
	}

	for i, cmdFunc := range scalingCommands {
		cmdName := []string{"scaling scale", "scaling preview"}[i]

		t.Run("insufficient_args_"+cmdName, func(t *testing.T) {
			err := cmdFunc([]string{"test-instance"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "usage:")
		})

		t.Run("valid_args_"+cmdName, func(t *testing.T) {
			err := cmdFunc([]string{"test-instance", "L"})
			// Command may succeed or fail due to mock state, but shouldn't be usage error
			if err != nil && strings.Contains(err.Error(), "usage:") {
				t.Errorf("Unexpected usage error for valid args in %s: %v", cmdName, err)
			}
		})
	}
}

// TestScalingWithInstanceStates tests scaling operations with different instance states
func TestScalingWithInstanceStates(t *testing.T) {
	states := []string{"running", "stopped", "pending", "terminated"}

	for _, state := range states {
		t.Run("state_"+state, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			mockClient.Instances[0].State = state

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewScalingCommands(app)

			// Test scaling analyze with different states
			err := sc.scalingAnalyze([]string{"test-instance"})
			// Should not panic regardless of state
			if err != nil {
				t.Logf("Scaling analyze with state %s: %v", state, err)
			}

			// Test scaling scale with different states
			err = sc.scalingScale([]string{"test-instance", "L"})
			// Should handle invalid states gracefully
			if state != "running" && state != "stopped" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "expected 'running or stopped'")
			}
		})
	}
}

// BenchmarkScalingCommands benchmarks scaling command operations
func BenchmarkScalingCommands(b *testing.B) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	sc := NewScalingCommands(app)

	b.Run("RightsizingRecommendations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := sc.rightsizingRecommendations([]string{})
			if err != nil {
				b.Fatal("Rightsizing recommendations failed:", err)
			}
		}
	})

	b.Run("RightsizingSummary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := sc.rightsizingSummary([]string{})
			if err != nil {
				b.Fatal("Rightsizing summary failed:", err)
			}
		}
	})

	b.Run("ScalingAnalyze", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := sc.scalingAnalyze([]string{"test-instance"})
			if err != nil {
				b.Fatal("Scaling analyze failed:", err)
			}
		}
	})

	b.Run("HelperFunctions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = sc.parseInstanceSize("t3.medium")
			_ = sc.getInstanceTypeForSize("M")
			_ = sc.estimateCostForSize("L")
			_ = sc.getSizeSpecs("XL")
		}
	})
}
