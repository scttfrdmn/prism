// Package models provides comprehensive test coverage for TUI storage management
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
)

// mockStorageAPIClient implements apiClient interface for storage testing
type mockStorageAPIClient struct {
	volumes           map[string]api.VolumeResponse
	storage           map[string]api.StorageResponse
	instances         []api.InstanceResponse
	shouldError       bool
	errorMessage      string
	callLog           []string
	responseDelay     time.Duration
	mountShouldError  bool
	mountErrorMessage string
}

// Storage operations
func (m *mockStorageAPIClient) ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error) {
	m.callLog = append(m.callLog, "ListVolumes")
	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListVolumesResponse{Volumes: m.volumes}, nil
}

func (m *mockStorageAPIClient) ListStorage(ctx context.Context) (*api.ListStorageResponse, error) {
	m.callLog = append(m.callLog, "ListStorage")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListStorageResponse{Storage: m.storage}, nil
}

func (m *mockStorageAPIClient) MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MountVolume:%s:%s:%s", volumeName, instanceName, mountPoint))
	if m.mountShouldError {
		return fmt.Errorf("%s", m.mountErrorMessage)
	}
	return nil
}

func (m *mockStorageAPIClient) UnmountVolume(ctx context.Context, volumeName, instanceName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("UnmountVolume:%s:%s", volumeName, instanceName))
	if m.mountShouldError {
		return fmt.Errorf("%s", m.mountErrorMessage)
	}
	return nil
}

// Instance operations for storage model dependencies
func (m *mockStorageAPIClient) ListInstances(ctx context.Context) (*api.ListInstancesResponse, error) {
	m.callLog = append(m.callLog, "ListInstances")
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}
	return &api.ListInstancesResponse{Instances: m.instances}, nil
}

// Stub implementations for other apiClient methods
func (m *mockStorageAPIClient) GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error) {
	m.callLog = append(m.callLog, "GetInstance:"+name)
	return &api.InstanceResponse{}, nil
}
func (m *mockStorageAPIClient) LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error) {
	m.callLog = append(m.callLog, "LaunchInstance:"+req.Name)
	return &api.LaunchInstanceResponse{}, nil
}
func (m *mockStorageAPIClient) StartInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, "StartInstance:"+name)
	return nil
}
func (m *mockStorageAPIClient) StopInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, "StopInstance:"+name)
	return nil
}
func (m *mockStorageAPIClient) DeleteInstance(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, "DeleteInstance:"+name)
	return nil
}
func (m *mockStorageAPIClient) ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error) {
	m.callLog = append(m.callLog, "ListTemplates")
	return &api.ListTemplatesResponse{}, nil
}
func (m *mockStorageAPIClient) GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error) {
	m.callLog = append(m.callLog, "GetTemplate:"+name)
	return &api.TemplateResponse{}, nil
}
func (m *mockStorageAPIClient) ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error) {
	m.callLog = append(m.callLog, "ListIdlePolicies")
	return &api.ListIdlePoliciesResponse{}, nil
}
func (m *mockStorageAPIClient) UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error {
	m.callLog = append(m.callLog, "UpdateIdlePolicy")
	return nil
}
func (m *mockStorageAPIClient) GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error) {
	m.callLog = append(m.callLog, "GetInstanceIdleStatus:"+name)
	return &api.IdleDetectionResponse{}, nil
}
func (m *mockStorageAPIClient) EnableIdleDetection(ctx context.Context, name, policy string) error {
	m.callLog = append(m.callLog, "EnableIdleDetection:"+name)
	return nil
}
func (m *mockStorageAPIClient) DisableIdleDetection(ctx context.Context, name string) error {
	m.callLog = append(m.callLog, "DisableIdleDetection:"+name)
	return nil
}
func (m *mockStorageAPIClient) GetStatus(ctx context.Context) (*api.SystemStatusResponse, error) {
	m.callLog = append(m.callLog, "GetStatus")
	return &api.SystemStatusResponse{}, nil
}

// TestStorageModelCreation tests basic storage model instantiation
func TestStorageModelCreation(t *testing.T) {
	mockClient := &mockStorageAPIClient{
		volumes: map[string]api.VolumeResponse{
			"shared-data": {
				Name:         "shared-data",
				FileSystemId: "fs-1234567890abcdef0",
				State:        "available",
				CreationTime: time.Now().Add(-24 * time.Hour),
				SizeBytes:    1024 * 1024 * 1024, // 1GB
			},
		},
		storage: map[string]api.StorageResponse{
			"project-storage": {
				Name:         "project-storage",
				VolumeID:     "vol-1234567890abcdef0",
				State:        "available",
				VolumeType:   "gp3",
				SizeGB:       100,
				CreationTime: time.Now().Add(-12 * time.Hour),
			},
		},
		instances: []api.InstanceResponse{
			{
				Name:     "test-instance",
				State:    "running",
				PublicIP: "54.123.45.67",
			},
		},
	}

	model := NewStorageModel(mockClient)

	// Validate model structure
	assert.NotNil(t, model.apiClient)
	assert.True(t, model.loading) // Model starts in loading state
	assert.Empty(t, model.error)
	assert.Equal(t, 0, model.selectedTab)                   // EFS volumes tab by default
	assert.Equal(t, 0, model.selectedItem)                  // First item selected
	assert.False(t, model.showMountDialog)                  // Mount dialog closed by default
	assert.Equal(t, "/mnt/shared-volume", model.mountPoint) // Default mount point
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
}

// TestStorageModelInit tests model initialization
func TestStorageModelInit(t *testing.T) {
	mockClient := &mockStorageAPIClient{
		volumes:   make(map[string]api.VolumeResponse),
		storage:   make(map[string]api.StorageResponse),
		instances: []api.InstanceResponse{},
	}

	model := NewStorageModel(mockClient)

	// Test Init command
	cmd := model.Init()
	assert.NotNil(t, cmd)
	// Init returns a tea.Batch command (spinner + fetchStorage)
}

// TestStorageModelUpdate tests model update logic with various messages
func TestStorageModelUpdate(t *testing.T) {
	mockStorageData := StorageDataMsg{
		Volumes: map[string]api.VolumeResponse{
			"efs-volume": {
				Name:            "efs-volume",
				FileSystemId:    "fs-abcd1234efgh5678",
				State:           "available",
				PerformanceMode: "generalPurpose",
				ThroughputMode:  "provisioned",
				SizeBytes:       2 * 1024 * 1024 * 1024, // 2GB
				EstimatedCostGB: 0.30,
			},
		},
		Storage: map[string]api.StorageResponse{
			"ebs-volume": {
				Name:       "ebs-volume",
				VolumeID:   "vol-9876543210fedcba",
				State:      "available",
				VolumeType: "gp3",
				SizeGB:     50,
			},
		},
		Instances: []api.InstanceResponse{
			{
				Name:      "storage-test-instance",
				State:     "running",
				PublicIP:  "192.168.1.100",
				PrivateIP: "10.0.1.50",
			},
		},
	}

	mockClient := &mockStorageAPIClient{
		volumes:   mockStorageData.Volumes,
		storage:   mockStorageData.Storage,
		instances: mockStorageData.Instances,
	}

	model := NewStorageModel(mockClient)

	// Test storage data message (successful load)
	t.Run("storage_data_message", func(t *testing.T) {
		newModel, cmd := model.Update(mockStorageData)

		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)

		assert.False(t, storageModel.loading)
		assert.Len(t, storageModel.volumes, 1)
		assert.Len(t, storageModel.storage, 1)
		assert.Len(t, storageModel.instances, 1)
		assert.Contains(t, storageModel.volumes, "efs-volume")
		assert.Contains(t, storageModel.storage, "ebs-volume")
		assert.Nil(t, cmd) // No follow-up command needed
	})

	// Test window size message
	t.Run("window_size_message", func(t *testing.T) {
		sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 50}
		newModel, cmd := model.Update(sizeMsg)

		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)
		assert.Equal(t, 120, storageModel.width)
		assert.Equal(t, 50, storageModel.height)
		assert.Nil(t, cmd)
	})

	// Test error message
	t.Run("error_message", func(t *testing.T) {
		errorMsg := fmt.Errorf("storage connection failed")
		newModel, cmd := model.Update(errorMsg)

		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)
		assert.False(t, storageModel.loading)
		assert.Equal(t, "storage connection failed", storageModel.error)
		assert.Nil(t, cmd)
	})

	// Test mount action message
	t.Run("mount_action_message", func(t *testing.T) {
		mountMsg := MountActionMsg{
			Success:  true,
			Message:  "Volume mounted successfully",
			Action:   "mount",
			Volume:   "test-volume",
			Instance: "test-instance",
		}

		newModel, _ := model.Update(mountMsg)

		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)
		assert.False(t, storageModel.showMountDialog) // Dialog should close on success
		// Mount action may trigger refresh command
	})

	// Test refresh message
	t.Run("refresh_message", func(t *testing.T) {
		refreshMsg := RefreshMsg{}
		newModel, cmd := model.Update(refreshMsg)

		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)
		assert.True(t, storageModel.loading)
		assert.Empty(t, storageModel.error)
		assert.NotNil(t, cmd) // Should trigger refresh
	})
}

// TestStorageModelKeyboardNavigation tests keyboard input handling
func TestStorageModelKeyboardNavigation(t *testing.T) {
	mockClient := &mockStorageAPIClient{
		volumes: map[string]api.VolumeResponse{
			"volume1": {Name: "volume1", State: "available"},
			"volume2": {Name: "volume2", State: "available"},
		},
		storage: map[string]api.StorageResponse{
			"storage1": {Name: "storage1", State: "available"},
			"storage2": {Name: "storage2", State: "available"},
		},
		instances: []api.InstanceResponse{
			{Name: "instance1", State: "running"},
		},
	}

	model := NewStorageModel(mockClient)
	model.loading = false // Set to loaded state
	model.volumes = mockClient.volumes
	model.storage = mockClient.storage
	model.instances = mockClient.instances

	// Test tab switching
	t.Run("tab_key_switching", func(t *testing.T) {
		// Start on volumes tab (0)
		assert.Equal(t, 0, model.selectedTab)

		// Press tab to switch to storage
		tabMsg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := model.Update(tabMsg)
		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)

		// Should switch to storage tab and reset item selection
		assert.Equal(t, 1, storageModel.selectedTab)
		assert.Equal(t, 0, storageModel.selectedItem)
	})

	// Test navigation keys
	t.Run("navigation_keys", func(t *testing.T) {
		// Test down arrow
		downMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := model.Update(downMsg)
		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)

		// Selected item should increase (within bounds)
		expectedSelection := 1
		if len(model.volumes) <= 1 {
			expectedSelection = 0 // Stay at 0 if only one item
		}
		assert.Equal(t, expectedSelection, storageModel.selectedItem)

		// Test up arrow
		upMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, _ = storageModel.Update(upMsg)
		storageModel, ok = newModel.(StorageModel)
		require.True(t, ok)

		// Should go back to first item
		assert.Equal(t, 0, storageModel.selectedItem)
	})

	// Test action keys
	t.Run("action_keys", func(t *testing.T) {
		// Test refresh key
		refreshMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		newModel, cmd := model.Update(refreshMsg)

		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)
		assert.True(t, storageModel.loading)
		assert.NotNil(t, cmd) // Should trigger refresh command

		// Test mount key (should show mount dialog if conditions are met)
		mountMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		_, _ = model.Update(mountMsg)
		// Note: Mount dialog behavior depends on selection and state
	})

	// Test quit keys
	t.Run("quit_keys", func(t *testing.T) {
		quitMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		newModel, cmd := model.Update(quitMsg)

		// Should return quit command
		assert.NotNil(t, cmd)
		assert.NotNil(t, newModel)
	})
}

// TestStorageModelView tests view rendering in different states
func TestStorageModelView(t *testing.T) {
	mockClient := &mockStorageAPIClient{
		volumes: map[string]api.VolumeResponse{
			"test-volume": {
				Name:         "test-volume",
				FileSystemId: "fs-test123",
				State:        "available",
				SizeBytes:    1024 * 1024 * 1024,
			},
		},
		storage: map[string]api.StorageResponse{
			"test-storage": {
				Name:       "test-storage",
				VolumeID:   "vol-test123",
				State:      "available",
				VolumeType: "gp3",
				SizeGB:     100,
			},
		},
	}

	model := NewStorageModel(mockClient)
	model.width = 100
	model.height = 50

	// Test loading state
	t.Run("loading_state", func(t *testing.T) {
		model.loading = true
		view := model.View()

		assert.NotEmpty(t, view)
		// Should show loading spinner
		assert.Greater(t, len(view), 50)
	})

	// Test error state
	t.Run("error_state", func(t *testing.T) {
		model.loading = false
		model.error = "Failed to load storage information"
		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Error")
	})

	// Test empty state
	t.Run("empty_state", func(t *testing.T) {
		model.loading = false
		model.error = ""
		model.volumes = make(map[string]api.VolumeResponse)
		model.storage = make(map[string]api.StorageResponse)
		view := model.View()

		assert.NotEmpty(t, view)
		// Should show empty message
	})

	// Test normal state with storage items
	t.Run("normal_state", func(t *testing.T) {
		model.loading = false
		model.error = ""
		model.volumes = mockClient.volumes
		model.storage = mockClient.storage
		view := model.View()

		assert.NotEmpty(t, view)
		assert.Greater(t, len(view), 100)   // Should have substantial content
		assert.Contains(t, view, "Storage") // Should show storage title
	})

	// Test mount dialog state
	t.Run("mount_dialog_state", func(t *testing.T) {
		model.loading = false
		model.error = ""
		model.volumes = mockClient.volumes
		model.showMountDialog = true
		model.mountVolumeName = "test-volume"
		model.mountInstanceName = "test-instance"
		view := model.View()

		assert.NotEmpty(t, view)
		// Should show mount dialog elements
		assert.Greater(t, len(view), 100)
	})
}

// TestStorageDataProcessing tests storage data handling and processing
func TestStorageDataProcessing(t *testing.T) {
	storageData := StorageDataMsg{
		Volumes: map[string]api.VolumeResponse{
			"large-volume": {
				Name:            "large-volume",
				FileSystemId:    "fs-large123456",
				State:           "available",
				PerformanceMode: "maxIO",
				ThroughputMode:  "provisioned",
				SizeBytes:       10 * 1024 * 1024 * 1024, // 10GB
				EstimatedCostGB: 0.30,
			},
		},
		Storage: map[string]api.StorageResponse{
			"high-performance-storage": {
				Name:       "high-performance-storage",
				VolumeID:   "vol-highperf123",
				State:      "in-use",
				VolumeType: "io2",
				SizeGB:     500,
			},
		},
		Instances: []api.InstanceResponse{
			{
				Name:               "ml-workstation",
				State:              "running",
				AttachedVolumes:    []string{"large-volume"},
				AttachedEBSVolumes: []string{"high-performance-storage"},
			},
		},
	}

	// Test data validation and processing
	t.Run("data_validation", func(t *testing.T) {
		assert.Len(t, storageData.Volumes, 1)
		assert.Len(t, storageData.Storage, 1)
		assert.Len(t, storageData.Instances, 1)

		volume := storageData.Volumes["large-volume"]
		assert.Equal(t, "large-volume", volume.Name)
		assert.Equal(t, "fs-large123456", volume.FileSystemId)
		assert.Equal(t, "available", volume.State)
		assert.Equal(t, int64(10*1024*1024*1024), volume.SizeBytes)

		storage := storageData.Storage["high-performance-storage"]
		assert.Equal(t, "high-performance-storage", storage.Name)
		assert.Equal(t, "vol-highperf123", storage.VolumeID)
		assert.Equal(t, "in-use", storage.State)
		assert.Equal(t, "io2", storage.VolumeType)
		assert.Equal(t, int32(500), storage.SizeGB)

		instance := storageData.Instances[0]
		assert.Equal(t, "ml-workstation", instance.Name)
		assert.Contains(t, instance.AttachedVolumes, "large-volume")
		assert.Contains(t, instance.AttachedEBSVolumes, "high-performance-storage")
	})

	// Test storage calculations
	t.Run("storage_calculations", func(t *testing.T) {
		volume := storageData.Volumes["large-volume"]

		// Calculate storage size in different units
		sizeGB := float64(volume.SizeBytes) / (1024 * 1024 * 1024)
		assert.Equal(t, float64(10), sizeGB)

		// Calculate estimated monthly cost
		monthlyCost := sizeGB * volume.EstimatedCostGB * 30 // 30 days
		assert.Equal(t, float64(90), monthlyCost)           // 10GB * $0.30 * 30 days

		storage := storageData.Storage["high-performance-storage"]
		assert.True(t, storage.SizeGB > 0)
		assert.Equal(t, "io2", storage.VolumeType) // High performance type
	})
}

// TestStorageMountOperations tests mount and unmount functionality
func TestStorageMountOperations(t *testing.T) {
	mockClient := &mockStorageAPIClient{
		volumes: map[string]api.VolumeResponse{
			"mount-test-volume": {
				Name:         "mount-test-volume",
				FileSystemId: "fs-mount123",
				State:        "available",
			},
		},
		instances: []api.InstanceResponse{
			{
				Name:  "mount-test-instance",
				State: "running",
			},
		},
	}

	model := NewStorageModel(mockClient)
	model.volumes = mockClient.volumes
	model.instances = mockClient.instances

	// Test successful mount action
	t.Run("successful_mount", func(t *testing.T) {
		mountMsg := MountActionMsg{
			Success:  true,
			Message:  "Volume mounted successfully at /mnt/shared-volume",
			Action:   "mount",
			Volume:   "mount-test-volume",
			Instance: "mount-test-instance",
		}

		newModel, cmd := model.Update(mountMsg)
		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)

		assert.False(t, storageModel.showMountDialog)
		assert.NotNil(t, cmd) // Should refresh data after successful mount
		// Status bar should reflect success (implementation detail)
	})

	// Test failed mount action
	t.Run("failed_mount", func(t *testing.T) {
		mountMsg := MountActionMsg{
			Success:  false,
			Message:  "Mount failed: instance not accessible",
			Action:   "mount",
			Volume:   "mount-test-volume",
			Instance: "mount-test-instance",
		}

		newModel, cmd := model.Update(mountMsg)
		_, ok := newModel.(StorageModel)
		require.True(t, ok)

		// Dialog should remain open on failure to allow retry
		assert.Nil(t, cmd)
		// Error should be reflected in status or error field
	})

	// Test unmount action
	t.Run("successful_unmount", func(t *testing.T) {
		unmountMsg := MountActionMsg{
			Success:  true,
			Message:  "Volume unmounted successfully",
			Action:   "unmount",
			Volume:   "mount-test-volume",
			Instance: "mount-test-instance",
		}

		newModel, cmd := model.Update(unmountMsg)
		_, ok := newModel.(StorageModel)
		require.True(t, ok)

		assert.NotNil(t, cmd) // Should refresh data after successful unmount
		// Should handle unmount success appropriately
	})
}

// TestStorageModelPerformance tests performance with large datasets
func TestStorageModelPerformance(t *testing.T) {
	// Generate large storage datasets
	largeVolumes := make(map[string]api.VolumeResponse)
	largeStorage := make(map[string]api.StorageResponse)
	largeInstances := make([]api.InstanceResponse, 20)

	for i := 0; i < 25; i++ {
		volumeName := fmt.Sprintf("volume-%d", i)
		largeVolumes[volumeName] = api.VolumeResponse{
			Name:         volumeName,
			FileSystemId: fmt.Sprintf("fs-%d", i),
			State:        "available",
			SizeBytes:    int64(i * 1024 * 1024 * 1024), // Variable sizes
		}

		storageName := fmt.Sprintf("storage-%d", i)
		largeStorage[storageName] = api.StorageResponse{
			Name:       storageName,
			VolumeID:   fmt.Sprintf("vol-%d", i),
			State:      "available",
			VolumeType: "gp3",
			SizeGB:     int32(100 + i*10), // Variable sizes
		}
	}

	for i := 0; i < 20; i++ {
		largeInstances[i] = api.InstanceResponse{
			Name:  fmt.Sprintf("instance-%d", i),
			State: "running",
		}
	}

	mockClient := &mockStorageAPIClient{
		volumes:   largeVolumes,
		storage:   largeStorage,
		instances: largeInstances,
	}

	model := NewStorageModel(mockClient)
	model.volumes = largeVolumes
	model.storage = largeStorage
	model.instances = largeInstances

	// Test view rendering performance
	t.Run("large_dataset_rendering", func(t *testing.T) {
		start := time.Now()

		// Render view multiple times
		for i := 0; i < 10; i++ {
			view := model.View()
			assert.NotEmpty(t, view)
		}

		duration := time.Since(start)
		assert.Less(t, duration, time.Second, "Storage view rendering should be performant")
	})

	// Test navigation performance
	t.Run("navigation_performance", func(t *testing.T) {
		start := time.Now()

		// Simulate rapid navigation
		currentModel := model
		for i := 0; i < len(largeVolumes); i++ {
			downMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			newModel, _ := currentModel.Update(downMsg)
			if storageModel, ok := newModel.(StorageModel); ok {
				currentModel = storageModel
			}
		}

		duration := time.Since(start)
		assert.Less(t, duration, 100*time.Millisecond, "Navigation should be fast")
	})

	// Test data processing performance
	t.Run("data_processing_performance", func(t *testing.T) {
		largeDataMsg := StorageDataMsg{
			Volumes:   largeVolumes,
			Storage:   largeStorage,
			Instances: largeInstances,
		}

		start := time.Now()

		// Process large dataset multiple times
		for i := 0; i < 5; i++ {
			newModel, _ := model.Update(largeDataMsg)
			assert.NotNil(t, newModel)
		}

		duration := time.Since(start)
		assert.Less(t, duration, 500*time.Millisecond, "Data processing should be fast")
	})
}

// TestStorageModelIntegration tests complete storage workflow
func TestStorageModelIntegration(t *testing.T) {
	mockStorageData := StorageDataMsg{
		Volumes: map[string]api.VolumeResponse{
			"integration-volume": {
				Name:         "integration-volume",
				FileSystemId: "fs-integration123",
				State:        "available",
				SizeBytes:    5 * 1024 * 1024 * 1024, // 5GB
			},
		},
		Storage: map[string]api.StorageResponse{
			"integration-storage": {
				Name:       "integration-storage",
				VolumeID:   "vol-integration123",
				State:      "available",
				VolumeType: "gp3",
				SizeGB:     200,
			},
		},
		Instances: []api.InstanceResponse{
			{
				Name:  "integration-instance",
				State: "running",
			},
		},
	}

	mockClient := &mockStorageAPIClient{
		volumes:   mockStorageData.Volumes,
		storage:   mockStorageData.Storage,
		instances: mockStorageData.Instances,
	}

	model := NewStorageModel(mockClient)

	// Test complete workflow
	t.Run("complete_storage_workflow", func(t *testing.T) {
		// 1. Initialize model
		cmd := model.Init()
		assert.NotNil(t, cmd)

		// 2. Load storage data
		newModel, cmd := model.Update(mockStorageData)
		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)

		assert.False(t, storageModel.loading)
		assert.Len(t, storageModel.volumes, 1)
		assert.Len(t, storageModel.storage, 1)
		assert.Len(t, storageModel.instances, 1)
		assert.Nil(t, cmd)

		// 3. Set window size
		sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
		sizedModel, cmd := storageModel.Update(sizeMsg)
		storageModel, ok = sizedModel.(StorageModel)
		require.True(t, ok)

		assert.Equal(t, 120, storageModel.width)
		assert.Equal(t, 40, storageModel.height)
		assert.Nil(t, cmd)

		// 4. Navigate between tabs
		tabMsg := tea.KeyMsg{Type: tea.KeyTab}
		tabbedModel, cmd := storageModel.Update(tabMsg)
		storageModel, ok = tabbedModel.(StorageModel)
		require.True(t, ok)

		assert.Equal(t, 1, storageModel.selectedTab) // Should switch to storage tab
		assert.Nil(t, cmd)

		// 5. Navigate items
		downMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		navigatedModel, cmd := storageModel.Update(downMsg)
		storageModel, ok = navigatedModel.(StorageModel)
		require.True(t, ok)
		assert.Nil(t, cmd)

		// 6. Render final view
		view := storageModel.View()
		assert.NotEmpty(t, view)
		assert.Greater(t, len(view), 200) // Should have substantial content
	})

	// Test error handling workflow
	t.Run("error_handling_workflow", func(t *testing.T) {
		// Test with error client
		errorClient := &mockStorageAPIClient{
			shouldError:  true,
			errorMessage: "Storage API connection failed",
		}

		errorModel := NewStorageModel(errorClient)

		// Simulate error during storage loading
		errorMsg := fmt.Errorf("Storage API connection failed")
		newModel, cmd := errorModel.Update(errorMsg)

		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)

		assert.False(t, storageModel.loading)
		assert.Equal(t, "Storage API connection failed", storageModel.error)
		assert.Nil(t, cmd)

		// Verify error is displayed in view
		view := storageModel.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Error")
	})

	// Test mount operation workflow
	t.Run("mount_operation_workflow", func(t *testing.T) {
		// Start with loaded model
		model.volumes = mockStorageData.Volumes
		model.instances = mockStorageData.Instances
		model.loading = false

		// Simulate mount operation
		mountMsg := MountActionMsg{
			Success:  true,
			Message:  "Volume mounted successfully",
			Action:   "mount",
			Volume:   "integration-volume",
			Instance: "integration-instance",
		}

		newModel, _ := model.Update(mountMsg)
		storageModel, ok := newModel.(StorageModel)
		require.True(t, ok)

		assert.False(t, storageModel.showMountDialog)
		// Mount action may trigger refresh command

		// Verify state after successful mount
		view := storageModel.View()
		assert.NotEmpty(t, view)
	})
}
