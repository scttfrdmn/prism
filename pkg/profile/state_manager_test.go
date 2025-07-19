package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

func TestStateManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "state-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test state manager
	manager := &StateManager{
		baseDir: tempDir,
	}

	// Test getting state for non-existent profile (should create empty state)
	state, err := manager.GetState("test-profile")
	if err != nil {
		t.Fatalf("Failed to get state for non-existent profile: %v", err)
	}
	if state == nil {
		t.Fatalf("Expected non-nil state for non-existent profile")
	}
	if len(state.Instances) != 0 {
		t.Errorf("Expected empty instances map, got %d entries", len(state.Instances))
	}

	// Test saving state
	testState := &types.State{
		Instances: map[string]types.Instance{
			"instance1": {
				ID:   "i-12345",
				Name: "test-instance",
			},
		},
		Volumes: map[string]types.EFSVolume{
			"volume1": {
				ID:   "fs-12345",
				Name: "test-volume",
			},
		},
		EBSVolumes: map[string]types.EBSVolume{
			"ebs1": {
				ID:   "vol-12345",
				Name: "test-ebs",
			},
		},
		Config: types.Config{
			APIKey: "test-api-key",
		},
	}

	err = manager.SaveState("test-profile", testState)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Check if file was created
	statePath := filepath.Join(tempDir, "test-profile.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatalf("State file was not created")
	}

	// Test loading saved state
	loadedState, err := manager.GetState("test-profile")
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify loaded state
	if len(loadedState.Instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(loadedState.Instances))
	}
	if loadedState.Instances["instance1"].ID != "i-12345" {
		t.Errorf("Expected instance ID 'i-12345', got '%s'", loadedState.Instances["instance1"].ID)
	}
	if len(loadedState.Volumes) != 1 {
		t.Errorf("Expected 1 volume, got %d", len(loadedState.Volumes))
	}
	if loadedState.Config.APIKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", loadedState.Config.APIKey)
	}

	// Test listing states
	states, err := manager.ListStates()
	if err != nil {
		t.Fatalf("Failed to list states: %v", err)
	}
	if len(states) != 1 {
		t.Errorf("Expected 1 state, got %d", len(states))
	}
	if states[0] != "test-profile" {
		t.Errorf("Expected state 'test-profile', got '%s'", states[0])
	}

	// Test deleting state
	err = manager.DeleteState("test-profile")
	if err != nil {
		t.Fatalf("Failed to delete state: %v", err)
	}

	// Check if file was deleted
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Errorf("State file was not deleted")
	}

	// Test deleting non-existent state
	err = manager.DeleteState("non-existent")
	if err != nil {
		t.Errorf("Expected deleting non-existent state to succeed, got error: %v", err)
	}
}

func TestProfileAwareStateManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "profile-aware-state-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock profiles
	mockProfiles := &Profiles{
		Profiles: map[string]Profile{
			"profile1": {
				Type:       ProfileTypePersonal,
				Name:       "Test Profile",
				AWSProfile: "test-aws-profile",
			},
			"profile2": {
				Type:       ProfileTypePersonal,
				Name:       "Another Profile",
				AWSProfile: "another-aws-profile",
			},
		},
		CurrentProfile: "profile1",
	}

	// Create mock profile manager
	mockProfileManager := &ManagerEnhanced{
		profiles: mockProfiles,
	}

	// Create state manager with temp directory
	stateManager := &StateManager{
		baseDir: tempDir,
	}

	// Create profile-aware state manager
	profileAwareManager := &ProfileAwareStateManager{
		stateManager:   stateManager,
		profileManager: mockProfileManager,
	}

	// Test getting current state
	state, err := profileAwareManager.GetCurrentState()
	if err != nil {
		t.Fatalf("Failed to get current state: %v", err)
	}
	if state == nil {
		t.Fatalf("Expected non-nil state")
	}

	// Test saving current state
	testState := &types.State{
		Instances: map[string]types.Instance{
			"instance1": {
				ID:   "i-12345",
				Name: "test-instance",
			},
		},
	}

	err = profileAwareManager.SaveCurrentState(testState)
	if err != nil {
		t.Fatalf("Failed to save current state: %v", err)
	}

	// Check if file was created with AWS profile name
	statePath := filepath.Join(tempDir, "test-aws-profile.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatalf("State file was not created")
	}

	// Test getting profile state
	profileState, err := profileAwareManager.GetProfileState("profile2")
	if err != nil {
		t.Fatalf("Failed to get profile state: %v", err)
	}
	if profileState == nil {
		t.Fatalf("Expected non-nil profile state")
	}

	// Test saving profile state
	anotherState := &types.State{
		Instances: map[string]types.Instance{
			"instance2": {
				ID:   "i-67890",
				Name: "another-instance",
			},
		},
	}

	err = profileAwareManager.SaveProfileState("profile2", anotherState)
	if err != nil {
		t.Fatalf("Failed to save profile state: %v", err)
	}

	// Check if file was created with AWS profile name
	anotherStatePath := filepath.Join(tempDir, "another-aws-profile.json")
	if _, err := os.Stat(anotherStatePath); os.IsNotExist(err) {
		t.Fatalf("Profile state file was not created")
	}

	// Test deleting profile state
	err = profileAwareManager.DeleteProfileState("profile2")
	if err != nil {
		t.Fatalf("Failed to delete profile state: %v", err)
	}

	// Check if file was deleted
	if _, err := os.Stat(anotherStatePath); !os.IsNotExist(err) {
		t.Errorf("Profile state file was not deleted")
	}
}