// Package cli tests for storage command module
package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestNewStorageCommands tests storage commands creation
func TestNewStorageCommands(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)

	sc := NewStorageCommands(app)

	assert.NotNil(t, sc)
	assert.Equal(t, app, sc.app)
}

// TestStorageCommands_Volume tests the volume command routing
func TestStorageCommands_Volume(t *testing.T) {
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
			name:        "Invalid action",
			args:        []string{"invalid-action"},
			expectError: true,
			errorMsg:    "invalid volume action",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Valid list action",
			args:        []string{"list"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Valid create action",
			args:        []string{"create", "test-volume"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"list"},
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
			sc := NewStorageCommands(app)

			err := sc.Volume(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVolumeCreateCommand tests EFS volume creation
func TestVolumeCreateCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
		checkReq    func(*testing.T, types.VolumeCreateRequest)
	}{
		{
			name:        "Basic volume creation",
			args:        []string{"create", "test-volume"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
			checkReq: func(t *testing.T, req types.VolumeCreateRequest) {
				assert.Equal(t, "test-volume", req.Name)
				assert.Empty(t, req.PerformanceMode)
				assert.Empty(t, req.ThroughputMode)
				assert.Empty(t, req.Region)
			},
		},
		{
			name:        "Volume creation with performance mode",
			args:        []string{"create", "test-volume", "--performance", "generalPurpose"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
			checkReq: func(t *testing.T, req types.VolumeCreateRequest) {
				assert.Equal(t, "test-volume", req.Name)
				assert.Equal(t, "generalPurpose", req.PerformanceMode)
			},
		},
		{
			name:        "Volume creation with all options",
			args:        []string{"create", "test-volume", "--performance", "maxIO", "--throughput", "provisioned", "--region", "us-west-2"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
			checkReq: func(t *testing.T, req types.VolumeCreateRequest) {
				assert.Equal(t, "test-volume", req.Name)
				assert.Equal(t, "maxIO", req.PerformanceMode)
				assert.Equal(t, "provisioned", req.ThroughputMode)
				assert.Equal(t, "us-west-2", req.Region)
			},
		},
		{
			name:        "No volume name",
			args:        []string{"create"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Invalid option",
			args:        []string{"create", "test-volume", "--invalid"},
			expectError: true,
			errorMsg:    "invalid volume option",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"create", "test-volume"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "create failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Volume(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Len(t, mockClient.CreateVolumeCalls, 1)
				if tt.checkReq != nil {
					tt.checkReq(t, mockClient.CreateVolumeCalls[0])
				}
			}
		})
	}
}

// TestVolumeListCommand tests EFS volume listing
func TestVolumeListCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "List volumes with data",
			args:        []string{"list"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "List empty volumes",
			args:        []string{"list"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.Volumes = []types.EFSVolume{}
			},
		},
		{
			name:        "API error",
			args:        []string{"list"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "list failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Volume(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVolumeInfoCommand tests EFS volume info
func TestVolumeInfoCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid volume info",
			args:        []string{"info", "test-volume"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No volume name",
			args:        []string{"info"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"info", "test-volume"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "volume not found"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Volume(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVolumeDeleteCommand tests EFS volume deletion
func TestVolumeDeleteCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid volume delete",
			args:        []string{"delete", "test-volume"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No volume name",
			args:        []string{"delete"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"delete", "test-volume"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "delete failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Volume(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVolumeMountCommand tests EFS volume mounting
func TestVolumeMountCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid mount with default mount point",
			args:        []string{"mount", "test-volume", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Valid mount with custom mount point",
			args:        []string{"mount", "test-volume", "test-instance", "/custom/path"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Missing arguments",
			args:        []string{"mount", "test-volume"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"mount", "test-volume", "test-instance"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "mount failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Volume(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVolumeUnmountCommand tests EFS volume unmounting
func TestVolumeUnmountCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid unmount",
			args:        []string{"unmount", "test-volume", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Missing arguments",
			args:        []string{"unmount", "test-volume"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"unmount", "test-volume", "test-instance"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "unmount failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Volume(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorageCommands_Storage tests the storage command routing
func TestStorageCommands_Storage(t *testing.T) {
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
			name:        "Invalid action",
			args:        []string{"invalid-action"},
			expectError: true,
			errorMsg:    "invalid storage action",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Valid list action",
			args:        []string{"list"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Valid create action",
			args:        []string{"create", "test-storage", "100GB"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Daemon not running",
			args:        []string{"list"},
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
			sc := NewStorageCommands(app)

			err := sc.Storage(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorageCreateCommand tests EBS volume creation
func TestStorageCreateCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
		checkReq    func(*testing.T, types.StorageCreateRequest)
	}{
		{
			name:        "Basic storage creation",
			args:        []string{"create", "test-storage", "100GB"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
			checkReq: func(t *testing.T, req types.StorageCreateRequest) {
				assert.Equal(t, "test-storage", req.Name)
				assert.Equal(t, "100GB", req.Size)
				assert.Equal(t, DefaultVolumeType, req.VolumeType)
				assert.Empty(t, req.Region)
			},
		},
		{
			name:        "Storage creation with type",
			args:        []string{"create", "test-storage", "100GB", "io2"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
			checkReq: func(t *testing.T, req types.StorageCreateRequest) {
				assert.Equal(t, "test-storage", req.Name)
				assert.Equal(t, "100GB", req.Size)
				assert.Equal(t, "io2", req.VolumeType)
			},
		},
		{
			name:        "Storage creation with region",
			args:        []string{"create", "test-storage", "100GB", "gp3", "--region", "us-west-2"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
			checkReq: func(t *testing.T, req types.StorageCreateRequest) {
				assert.Equal(t, "test-storage", req.Name)
				assert.Equal(t, "100GB", req.Size)
				assert.Equal(t, "gp3", req.VolumeType)
				assert.Equal(t, "us-west-2", req.Region)
			},
		},
		{
			name:        "Missing arguments",
			args:        []string{"create", "test-storage"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Invalid option",
			args:        []string{"create", "test-storage", "100GB", "--invalid"},
			expectError: true,
			errorMsg:    "invalid storage option",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"create", "test-storage", "100GB"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "create failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Storage(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Len(t, mockClient.CreateStorageCalls, 1)
				if tt.checkReq != nil {
					tt.checkReq(t, mockClient.CreateStorageCalls[0])
				}
			}
		})
	}
}

// TestStorageListCommand tests EBS volume listing
func TestStorageListCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "List storage with data",
			args:        []string{"list"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "List empty storage",
			args:        []string{"list"},
			expectError: false,
			setupMock: func(mock *MockAPIClient) {
				mock.StorageVolumes = []types.EBSVolume{}
			},
		},
		{
			name:        "API error",
			args:        []string{"list"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "list failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Storage(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorageInfoCommand tests EBS volume info
func TestStorageInfoCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid storage info",
			args:        []string{"info", "test-storage"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "No storage name",
			args:        []string{"info"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"info", "test-storage"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "storage not found"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Storage(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorageAttachCommand tests EBS volume attachment
func TestStorageAttachCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid attach",
			args:        []string{"attach", "test-storage", "test-instance"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Missing arguments",
			args:        []string{"attach", "test-storage"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"attach", "test-storage", "test-instance"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "attach failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Storage(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorageDetachCommand tests EBS volume detachment
func TestStorageDetachCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid detach",
			args:        []string{"detach", "test-storage"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Missing arguments",
			args:        []string{"detach"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"detach", "test-storage"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "detach failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Storage(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorageDeleteCommand tests EBS volume deletion
func TestStorageDeleteCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		setupMock   func(*MockAPIClient)
	}{
		{
			name:        "Valid delete",
			args:        []string{"delete", "test-storage"},
			expectError: false,
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "Missing arguments",
			args:        []string{"delete"},
			expectError: true,
			errorMsg:    "usage:",
			setupMock:   func(mock *MockAPIClient) {},
		},
		{
			name:        "API error",
			args:        []string{"delete", "test-storage"},
			expectError: true,
			errorMsg:    "daemon",
			setupMock: func(mock *MockAPIClient) {
				mock.ShouldReturnError = true
				mock.ErrorMessage = "delete failed"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			tt.setupMock(mockClient)

			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Storage(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorageCommandArgumentParsing tests argument parsing across storage commands
func TestStorageCommandArgumentParsing(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	sc := NewStorageCommands(app)

	// Test volume commands
	volumeCommands := []string{"create", "list", "info", "delete", "mount", "unmount"}
	for _, cmd := range volumeCommands {
		t.Run("volume_"+cmd, func(t *testing.T) {
			var args []string

			switch cmd {
			case "create":
				args = []string{"volume", cmd, "test-volume"}
			case "info", "delete":
				args = []string{"volume", cmd, "test-volume"}
			case "mount":
				args = []string{"volume", cmd, "test-volume", "test-instance"}
			case "unmount":
				args = []string{"volume", cmd, "test-volume", "test-instance"}
			case "list":
				args = []string{"volume", cmd}
			}

			err := sc.Volume(args[1:]) // Skip "volume" prefix

			// Some commands may error, but should not be usage errors for valid args
			if err != nil && strings.Contains(err.Error(), "usage:") && cmd != "create" {
				t.Errorf("Unexpected usage error for valid args in command %s: %v", cmd, err)
			}
		})
	}

	// Test storage commands
	storageCommands := []string{"create", "list", "info", "attach", "detach", "delete"}
	for _, cmd := range storageCommands {
		t.Run("storage_"+cmd, func(t *testing.T) {
			var args []string

			switch cmd {
			case "create":
				args = []string{"storage", cmd, "test-storage", "100GB"}
			case "info", "delete", "detach":
				args = []string{"storage", cmd, "test-storage"}
			case "attach":
				args = []string{"storage", cmd, "test-storage", "test-instance"}
			case "list":
				args = []string{"storage", cmd}
			}

			err := sc.Storage(args[1:]) // Skip "storage" prefix

			// Some commands may error, but should not be usage errors for valid args
			if err != nil && strings.Contains(err.Error(), "usage:") && cmd != "create" {
				t.Errorf("Unexpected usage error for valid args in command %s: %v", cmd, err)
			}
		})
	}
}

// TestStorageCommandsCallTracking tests that API calls are properly tracked
func TestStorageCommandsCallTracking(t *testing.T) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	sc := NewStorageCommands(app)

	// Test volume create call tracking
	mockClient.ResetCallTracking()
	err := sc.Volume([]string{"create", "test-volume"})
	assert.NoError(t, err)
	assert.Len(t, mockClient.CreateVolumeCalls, 1)
	assert.Equal(t, "test-volume", mockClient.CreateVolumeCalls[0].Name)

	// Test storage create call tracking
	mockClient.ResetCallTracking()
	err = sc.Storage([]string{"create", "test-storage", "100GB"})
	assert.NoError(t, err)
	assert.Len(t, mockClient.CreateStorageCalls, 1)
	assert.Equal(t, "test-storage", mockClient.CreateStorageCalls[0].Name)
	assert.Equal(t, "100GB", mockClient.CreateStorageCalls[0].Size)
}

// TestVolumeOptionParsing tests EFS volume option parsing
func TestVolumeOptionParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		checkReq func(*testing.T, types.VolumeCreateRequest)
	}{
		{
			name: "All options",
			args: []string{"create", "test-volume", "--performance", "maxIO", "--throughput", "provisioned", "--region", "us-east-1"},
			checkReq: func(t *testing.T, req types.VolumeCreateRequest) {
				assert.Equal(t, "maxIO", req.PerformanceMode)
				assert.Equal(t, "provisioned", req.ThroughputMode)
				assert.Equal(t, "us-east-1", req.Region)
			},
		},
		{
			name: "Performance only",
			args: []string{"create", "test-volume", "--performance", "generalPurpose"},
			checkReq: func(t *testing.T, req types.VolumeCreateRequest) {
				assert.Equal(t, "generalPurpose", req.PerformanceMode)
				assert.Empty(t, req.ThroughputMode)
				assert.Empty(t, req.Region)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Volume(tt.args)
			assert.NoError(t, err)

			require.Len(t, mockClient.CreateVolumeCalls, 1)
			tt.checkReq(t, mockClient.CreateVolumeCalls[0])
		})
	}
}

// TestStorageOptionParsing tests EBS volume option parsing
func TestStorageOptionParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		checkReq func(*testing.T, types.StorageCreateRequest)
	}{
		{
			name: "With region",
			args: []string{"create", "test-storage", "100GB", "gp3", "--region", "us-west-2"},
			checkReq: func(t *testing.T, req types.StorageCreateRequest) {
				assert.Equal(t, "test-storage", req.Name)
				assert.Equal(t, "100GB", req.Size)
				assert.Equal(t, "gp3", req.VolumeType)
				assert.Equal(t, "us-west-2", req.Region)
			},
		},
		{
			name: "Default type",
			args: []string{"create", "test-storage", "100GB"},
			checkReq: func(t *testing.T, req types.StorageCreateRequest) {
				assert.Equal(t, DefaultVolumeType, req.VolumeType)
				assert.Empty(t, req.Region)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockAPIClient()
			app := NewAppWithClient("1.0.0", mockClient)
			sc := NewStorageCommands(app)

			err := sc.Storage(tt.args)
			assert.NoError(t, err)

			require.Len(t, mockClient.CreateStorageCalls, 1)
			tt.checkReq(t, mockClient.CreateStorageCalls[0])
		})
	}
}

// BenchmarkStorageCommands benchmarks storage command operations
func BenchmarkStorageCommands(b *testing.B) {
	mockClient := NewMockAPIClient()
	app := NewAppWithClient("1.0.0", mockClient)
	sc := NewStorageCommands(app)

	b.Run("VolumeList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := sc.Volume([]string{"list"})
			if err != nil {
				b.Fatal("Volume list failed:", err)
			}
		}
	})

	b.Run("StorageList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := sc.Storage([]string{"list"})
			if err != nil {
				b.Fatal("Storage list failed:", err)
			}
		}
	})

	b.Run("VolumeCreate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mockClient.ResetCallTracking()
			err := sc.Volume([]string{"create", "test-volume"})
			if err != nil {
				b.Fatal("Volume create failed:", err)
			}
		}
	})
}
