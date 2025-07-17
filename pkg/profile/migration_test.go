package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

func TestMigrateFromLegacyState(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Set up a test legacy state
	legacyState := types.State{
		Instances: map[string]types.Instance{
			"test-instance": {
				ID:                 "i-0123456789abcdef",
				Name:               "test-instance",
				Template:           "r-research",
				PublicIP:           "10.0.0.1",
				State:              "running",
				LaunchTime:         time.Now().Add(-24 * time.Hour),
				EstimatedDailyCost: 2.40,
			},
		},
		Volumes: map[string]types.EFSVolume{
			"test-volume": {
				Name:         "test-volume",
				FileSystemId: "fs-01234567",
				State:        "available",
				CreationTime: time.Now().Add(-48 * time.Hour),
			},
		},
		EBSVolumes: map[string]types.EBSVolume{
			"test-ebs": {
				Name:         "test-ebs",
				VolumeID:     "vol-01234567",
				SizeGB:       100,
				State:        "available",
				CreationTime: time.Now().Add(-48 * time.Hour),
			},
		},
		Config: types.Config{
			DefaultRegion: "us-west-2",
		},
	}
	
	// Create test directories
	cwsDir := filepath.Join(tempDir, ".cloudworkstation")
	statesDir := filepath.Join(cwsDir, "states")
	err = os.MkdirAll(statesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create states directory: %v", err)
	}
	
	// Create legacy state file
	legacyStatePath := filepath.Join(cwsDir, "state.json")
	legacyStateData, err := json.MarshalIndent(legacyState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal legacy state: %v", err)
	}
	err = os.WriteFile(legacyStatePath, legacyStateData, 0644)
	if err != nil {
		t.Fatalf("Failed to write legacy state file: %v", err)
	}
	
	// Create a test profile manager with a mock home directory
	origHomeDir := homeDir
	homeDir = func() (string, error) {
		return tempDir, nil
	}
	defer func() { homeDir = origHomeDir }()
	
	// Override currentTime function to return a fixed time
	fixedTime := time.Date(2024, 7, 17, 12, 0, 0, 0, time.UTC)
	origCurrentTime := currentTime
	currentTime = func() time.Time {
		return fixedTime
	}
	defer func() { currentTime = origCurrentTime }()
	
	// Create the manager
	m, err := NewManagerEnhanced()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	// Run migration
	result, err := m.MigrateFromLegacyState("Test Migration")
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}
	
	// Check migration result
	if !result.Success {
		t.Errorf("Migration reported failure")
	}
	if result.ProfileName != "Test Migration" {
		t.Errorf("Expected profile name 'Test Migration', got '%s'", result.ProfileName)
	}
	if result.InstanceCount != 1 {
		t.Errorf("Expected 1 instance, got %d", result.InstanceCount)
	}
	if result.VolumeCount != 1 {
		t.Errorf("Expected 1 volume, got %d", result.VolumeCount)
	}
	if result.StorageCount != 1 {
		t.Errorf("Expected 1 storage volume, got %d", result.StorageCount)
	}
	
	// Check that backup file exists
	backupPath := filepath.Join(cwsDir, "state.json.backup")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file not found at %s", backupPath)
	}
	
	// Check that the state was written for the new profile
	migratedStateFile := filepath.Join(statesDir, "default.json")
	if _, err := os.Stat(migratedStateFile); os.IsNotExist(err) {
		t.Errorf("Migrated state file not found at %s", migratedStateFile)
	}
}

// Helper function override for testing
var homeDir = os.UserHomeDir