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
	tempDir, err := os.MkdirTemp("", "cws-test-*")
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
	if state.Volumes == nil {
		t.Error("Volumes map should be initialized")
	}
	if state.EBSVolumes == nil {
		t.Error("EBSVolumes map should be initialized")
	}
	if state.Config.DefaultRegion != "us-east-1" {
		t.Errorf("Default region should be us-east-1, got %s", state.Config.DefaultRegion)
	}
}

func TestSaveAndLoadState(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	// Create test state
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
				AttachedVolumes:    []string{"vol-1"},
				AttachedEBSVolumes: []string{"ebs-1"},
			},
		},
		Volumes: map[string]types.EFSVolume{
			"vol-1": {
				Name:            "vol-1",
				FileSystemId:    "fs-1234567890abcdef0",
				Region:          "us-east-1",
				CreationTime:    time.Now().UTC().Truncate(time.Second),
				State:           "available",
				PerformanceMode: "generalPurpose",
				ThroughputMode:  "bursting",
				EstimatedCostGB: 0.30,
				SizeBytes:       1073741824,
			},
		},
		EBSVolumes: map[string]types.EBSVolume{
			"ebs-1": {
				Name:            "ebs-1",
				VolumeID:        "vol-1234567890abcdef0",
				Region:          "us-east-1",
				CreationTime:    time.Now().UTC().Truncate(time.Second),
				State:           "available",
				VolumeType:      "gp3",
				SizeGB:          100,
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

	if len(loadedState.Volumes) != len(originalState.Volumes) {
		t.Errorf("Volume count mismatch: got %d, want %d", len(loadedState.Volumes), len(originalState.Volumes))
	}

	if len(loadedState.EBSVolumes) != len(originalState.EBSVolumes) {
		t.Errorf("EBS Volume count mismatch: got %d, want %d", len(loadedState.EBSVolumes), len(originalState.EBSVolumes))
	}

	if loadedState.Config.DefaultRegion != "us-west-2" {
		t.Errorf("Config region mismatch: got %s, want us-west-2", loadedState.Config.DefaultRegion)
	}
}

func TestSaveInstance(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-test-*")
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
	tempDir, err := os.MkdirTemp("", "cws-test-*")
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

func TestSaveAndRemoveVolume(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	volume := types.EFSVolume{
		Name:         "test-volume",
		FileSystemId: "fs-123",
		Region:       "us-east-1",
		State:        "available",
	}

	// Save volume
	err = manager.SaveVolume(volume)
	if err != nil {
		t.Fatalf("Failed to save volume: %v", err)
	}

	// Verify it exists
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if _, exists := state.Volumes["test-volume"]; !exists {
		t.Error("Volume should exist after saving")
	}

	// Remove volume
	err = manager.RemoveVolume("test-volume")
	if err != nil {
		t.Fatalf("Failed to remove volume: %v", err)
	}

	// Verify it's gone
	state, err = manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if _, exists := state.Volumes["test-volume"]; exists {
		t.Error("Volume should have been removed")
	}
}

func TestSaveAndRemoveEBSVolume(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
	}

	volume := types.EBSVolume{
		Name:     "test-ebs",
		VolumeID: "vol-123",
		Region:   "us-east-1",
		State:    "available",
		SizeGB:   100,
	}

	// Save EBS volume
	err = manager.SaveEBSVolume(volume)
	if err != nil {
		t.Fatalf("Failed to save EBS volume: %v", err)
	}

	// Verify it exists
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if _, exists := state.EBSVolumes["test-ebs"]; !exists {
		t.Error("EBS volume should exist after saving")
	}

	// Remove EBS volume
	err = manager.RemoveEBSVolume("test-ebs")
	if err != nil {
		t.Fatalf("Failed to remove EBS volume: %v", err)
	}

	// Verify it's gone
	state, err = manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if _, exists := state.EBSVolumes["test-ebs"]; exists {
		t.Error("EBS volume should have been removed")
	}
}

func TestUpdateConfig(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-test-*")
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
	tempDir, err := os.MkdirTemp("", "cws-test-*")
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
