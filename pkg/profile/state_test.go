package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// ProfileState is a sample structure to test state operations
type ProfileState struct {
	Instances map[string]string `json:"instances"`
	Volumes   map[string]string `json:"volumes"`
	Version   string            `json:"version"`
}

func TestStateOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "profile-state-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create profile state directory
	profileID := "test-profile"
	stateDir := filepath.Join(tempDir, profileID)
	err = os.MkdirAll(stateDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create profile state directory: %v", err)
	}

	// Test saving state
	testState := ProfileState{
		Instances: map[string]string{
			"instance1": "i-12345",
			"instance2": "i-67890",
		},
		Volumes: map[string]string{
			"volume1": "vol-12345",
			"volume2": "vol-67890",
		},
		Version: "1.0",
	}

	err = SaveProfileState(tempDir, profileID, "state.json", testState)
	if err != nil {
		t.Fatalf("Failed to save profile state: %v", err)
	}

	// Check if file was created
	statePath := filepath.Join(stateDir, "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatalf("State file was not created")
	}

	// Test loading state
	var loadedState ProfileState
	err = LoadProfileState(tempDir, profileID, "state.json", &loadedState)
	if err != nil {
		t.Fatalf("Failed to load profile state: %v", err)
	}

	// Verify loaded state
	if loadedState.Version != testState.Version {
		t.Errorf("Expected version %s, got %s", testState.Version, loadedState.Version)
	}

	if len(loadedState.Instances) != len(testState.Instances) {
		t.Errorf("Expected %d instances, got %d", len(testState.Instances), len(loadedState.Instances))
	}

	if loadedState.Instances["instance1"] != testState.Instances["instance1"] {
		t.Errorf("Expected instance1 value %s, got %s", 
			testState.Instances["instance1"], loadedState.Instances["instance1"])
	}

	if loadedState.Volumes["volume2"] != testState.Volumes["volume2"] {
		t.Errorf("Expected volume2 value %s, got %s", 
			testState.Volumes["volume2"], loadedState.Volumes["volume2"])
	}

	// Test loading non-existent state
	var emptyState ProfileState
	err = LoadProfileState(tempDir, "non-existent", "state.json", &emptyState)
	if err == nil {
		t.Errorf("Expected error loading non-existent state")
	}

	// Test clearing state
	err = ClearProfileState(tempDir, profileID)
	if err != nil {
		t.Fatalf("Failed to clear profile state: %v", err)
	}

	// Check if directory was removed
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Errorf("Profile state directory was not removed")
	}

	// Test clearing non-existent state
	err = ClearProfileState(tempDir, "non-existent")
	if err != nil {
		t.Errorf("Expected clearing non-existent state to succeed, got error: %v", err)
	}
}