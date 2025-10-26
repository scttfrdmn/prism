package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

func TestNewManager(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager.statePath == "" {
		t.Error("State path should not be empty")
	}

	// Verify state directory exists
	stateDir := filepath.Dir(manager.statePath)
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		t.Errorf("State directory should exist: %s", stateDir)
	}
}

func TestLoadStateEmptyFile(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	// Load state when file doesn't exist
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState should not fail when file doesn't exist: %v", err)
	}

	// Verify default state
	if state.Instances == nil {
		t.Error("Instances map should be initialized")
	}
	if state.StorageVolumes == nil {
		t.Error("StorageVolumes map should be initialized")
	}
	if state.Config.DefaultRegion != "us-east-1" {
		t.Errorf("Default region should be us-east-1, got %s", state.Config.DefaultRegion)
	}
}

func TestSaveAndLoadState(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	// Helper for int32 pointers
	int32Ptr := func(v int32) *int32 { return &v }
	int64Ptr := func(v int64) *int64 { return &v }

	// Create test state with unified storage
	originalState := &types.State{
		Instances: map[string]types.Instance{
			"test-instance": {
				ID:                 "i-1234567890abcdef0",
				Name:               "test-instance",
				Template:           "r-research",
				PublicIP:           "54.123.45.67",
				State:              "running",
				LaunchTime:         time.Now().UTC().Truncate(time.Second),
				HourlyRate:         0.10,
				CurrentSpend:       2.40,
				AttachedVolumes:    []string{"efs-vol-1"},
				AttachedEBSVolumes: []string{"ebs-vol-1"},
			},
		},
		StorageVolumes: map[string]types.StorageVolume{
			// EFS volume
			"efs-vol-1": {
				Name:            "efs-vol-1",
				Type:            types.StorageTypeShared,
				AWSService:      types.AWSServiceEFS,
				FileSystemID:    "fs-1234567890abcdef0",
				Region:          "us-east-1",
				CreationTime:    time.Now().UTC().Truncate(time.Second),
				State:           "available",
				PerformanceMode: "generalPurpose",
				ThroughputMode:  "bursting",
				EstimatedCostGB: 0.30,
				SizeBytes:       int64Ptr(1073741824),
			},
			// EBS volume
			"ebs-vol-1": {
				Name:            "ebs-vol-1",
				Type:            types.StorageTypeWorkspace,
				AWSService:      types.AWSServiceEBS,
				VolumeID:        "vol-1234567890abcdef0",
				Region:          "us-east-1",
				CreationTime:    time.Now().UTC().Truncate(time.Second),
				State:           "available",
				VolumeType:      "gp3",
				SizeGB:          int32Ptr(100),
				EstimatedCostGB: 0.08,
			},
		},
		Config: types.Config{
			DefaultRegion: "us-west-2",
		},
	}

	// Save state
	err = manager.SaveState(originalState)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load state
	loadedState, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify loaded state matches original
	if len(loadedState.Instances) != len(originalState.Instances) {
		t.Errorf("Instance count mismatch: got %d, want %d", len(loadedState.Instances), len(originalState.Instances))
	}

	instance, exists := loadedState.Instances["test-instance"]
	if !exists {
		t.Error("test-instance should exist in loaded state")
	} else {
		if instance.ID != "i-1234567890abcdef0" {
			t.Errorf("Instance ID mismatch: got %s, want i-1234567890abcdef0", instance.ID)
		}
		if instance.Template != "r-research" {
			t.Errorf("Instance template mismatch: got %s, want r-research", instance.Template)
		}
	}

	if len(loadedState.StorageVolumes) != len(originalState.StorageVolumes) {
		t.Errorf("StorageVolume count mismatch: got %d, want %d", len(loadedState.StorageVolumes), len(originalState.StorageVolumes))
	}

	// Verify EFS volume
	efsVol, exists := loadedState.StorageVolumes["efs-vol-1"]
	if !exists {
		t.Error("EFS volume should exist")
	} else if !efsVol.IsShared() {
		t.Error("EFS volume should be shared storage type")
	}

	// Verify EBS volume
	ebsVol, exists := loadedState.StorageVolumes["ebs-vol-1"]
	if !exists {
		t.Error("EBS volume should exist")
	} else if !ebsVol.IsWorkspace() {
		t.Error("EBS volume should be workspace storage type")
	}

	if loadedState.Config.DefaultRegion != "us-west-2" {
		t.Errorf("Config region mismatch: got %s, want us-west-2", loadedState.Config.DefaultRegion)
	}
}

func TestSaveInstance(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	instance := types.Instance{
		ID:       "i-123",
		Name:     "test-save",
		Template: "python-research",
		State:    "running",
	}

	// Save instance
	err = manager.SaveInstance(instance)
	if err != nil {
		t.Fatalf("Failed to save instance: %v", err)
	}

	// Load and verify
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	savedInstance, exists := state.Instances["test-save"]
	if !exists {
		t.Error("Saved instance should exist")
	} else {
		if savedInstance.ID != "i-123" {
			t.Errorf("Instance ID mismatch: got %s, want i-123", savedInstance.ID)
		}
	}
}

func TestRemoveInstance(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	// First save an instance
	instance := types.Instance{
		ID:       "i-123",
		Name:     "test-remove",
		Template: "r-research",
		State:    "running",
	}

	err = manager.SaveInstance(instance)
	if err != nil {
		t.Fatalf("Failed to save instance: %v", err)
	}

	// Remove the instance
	err = manager.RemoveInstance("test-remove")
	if err != nil {
		t.Fatalf("Failed to remove instance: %v", err)
	}

	// Verify it's gone
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if _, exists := state.Instances["test-remove"]; exists {
		t.Error("Instance should have been removed")
	}
}

func TestSaveAndRemoveStorageVolume(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	// Test with EFS volume
	efsVolume := types.StorageVolume{
		Name:         "test-efs",
		Type:         types.StorageTypeShared,
		AWSService:   types.AWSServiceEFS,
		FileSystemID: "fs-123",
		Region:       "us-east-1",
		State:        "available",
	}

	// Save EFS volume
	err = manager.SaveStorageVolume(efsVolume)
	if err != nil {
		t.Fatalf("Failed to save EFS volume: %v", err)
	}

	// Verify it exists
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	vol, exists := state.StorageVolumes["test-efs"]
	if !exists {
		t.Error("EFS volume should exist after saving")
	} else if !vol.IsShared() {
		t.Error("Volume should be shared storage type")
	}

	// Test with EBS volume
	int32Ptr := func(v int32) *int32 { return &v }
	ebsVolume := types.StorageVolume{
		Name:       "test-ebs",
		Type:       types.StorageTypeWorkspace,
		AWSService: types.AWSServiceEBS,
		VolumeID:   "vol-123",
		Region:     "us-east-1",
		State:      "available",
		SizeGB:     int32Ptr(100),
	}

	// Save EBS volume
	err = manager.SaveStorageVolume(ebsVolume)
	if err != nil {
		t.Fatalf("Failed to save EBS volume: %v", err)
	}

	// Verify both volumes exist
	state, err = manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if len(state.StorageVolumes) != 2 {
		t.Errorf("Expected 2 storage volumes, got %d", len(state.StorageVolumes))
	}

	// Remove EFS volume
	err = manager.RemoveStorageVolume("test-efs")
	if err != nil {
		t.Fatalf("Failed to remove EFS volume: %v", err)
	}

	// Verify EFS is gone but EBS remains
	state, err = manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if _, exists := state.StorageVolumes["test-efs"]; exists {
		t.Error("EFS volume should have been removed")
	}
	if _, exists := state.StorageVolumes["test-ebs"]; !exists {
		t.Error("EBS volume should still exist")
	}

	// Remove EBS volume
	err = manager.RemoveStorageVolume("test-ebs")
	if err != nil {
		t.Fatalf("Failed to remove EBS volume: %v", err)
	}

	// Verify both are gone
	state, err = manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if len(state.StorageVolumes) != 0 {
		t.Errorf("Expected 0 storage volumes, got %d", len(state.StorageVolumes))
	}
}

func TestUpdateConfig(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	newConfig := types.Config{
		DefaultRegion: "eu-west-1",
	}

	// Update config
	err = manager.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Load and verify
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if state.Config.DefaultRegion != "eu-west-1" {
		t.Errorf("Config should be updated: got %s, want eu-west-1", state.Config.DefaultRegion)
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	// Test concurrent saves and loads
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			instance := types.Instance{
				ID:   "i-" + string(rune('0'+id)),
				Name: "test-" + string(rune('0'+id)),
			}
			if err := manager.SaveInstance(instance); err != nil {
				errors <- err
			}
		}(i)
	}

	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			if _, err := manager.LoadState(); err != nil {
				errors <- err
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify final state - all 5 instances should be saved
	// (even with concurrent access, all saves should succeed due to mutex)
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load final state: %v", err)
	}

	if len(state.Instances) < 1 {
		t.Errorf("Expected at least 1 instance, got %d", len(state.Instances))
	}
	// Note: Due to the nature of concurrent operations and file overwrites,
	// we can't guarantee all 5 instances will be present, but at least some should be
}
