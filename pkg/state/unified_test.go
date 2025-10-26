package state

import (
	"os"
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
)

func TestUnifiedManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	// Test basic unified manager without profiles
	manager, err := NewUnifiedManager()
	if err != nil {
		t.Fatalf("NewUnifiedManager() failed: %v", err)
	}

	// Test loading empty state
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	if state.Instances == nil {
		t.Error("LoadState() returned nil Instances map")
	}

	if len(state.Instances) != 0 {
		t.Errorf("LoadState() returned non-empty Instances map: %d items", len(state.Instances))
	}

	// Test saving an instance
	instance := types.Instance{
		ID:       "test-instance-id",
		Name:     "test-instance",
		Template: "test-template",
		State:    "running",
	}

	err = manager.SaveInstance(instance)
	if err != nil {
		t.Fatalf("SaveInstance() failed: %v", err)
	}

	// Verify instance was saved
	state, err = manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState() after SaveInstance() failed: %v", err)
	}

	if len(state.Instances) != 1 {
		t.Errorf("Expected 1 instance after SaveInstance(), got %d", len(state.Instances))
	}

	savedInstance, exists := state.Instances["test-instance"]
	if !exists {
		t.Error("SaveInstance() did not save instance correctly")
	}

	if savedInstance.ID != instance.ID {
		t.Errorf("SaveInstance() saved incorrect instance ID: got %s, want %s", savedInstance.ID, instance.ID)
	}
}

func TestUnifiedManagerWithProfiles(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	// Create static profile provider for testing
	profileProvider := NewStaticProfileProvider("test-profile")

	// Test unified manager with profiles
	manager, err := NewUnifiedManagerWithProfiles(profileProvider)
	if err != nil {
		t.Fatalf("NewUnifiedManagerWithProfiles() failed: %v", err)
	}

	// Test getting current profile
	profile, err := manager.GetCurrentProfile()
	if err != nil {
		t.Fatalf("GetCurrentProfile() failed: %v", err)
	}

	if profile != "test-profile" {
		t.Errorf("GetCurrentProfile() returned wrong profile: got %s, want test-profile", profile)
	}

	// Test state operations with profile context
	state, err := manager.LoadStateForProfile()
	if err != nil {
		t.Fatalf("LoadStateForProfile() failed: %v", err)
	}

	if state.Instances == nil {
		t.Error("LoadStateForProfile() returned nil Instances map")
	}

	// Test saving state with profile context
	state.Config.DefaultRegion = "us-west-2"
	err = manager.SaveStateForProfile(state)
	if err != nil {
		t.Fatalf("SaveStateForProfile() failed: %v", err)
	}

	// Verify state was saved
	reloadedState, err := manager.LoadStateForProfile()
	if err != nil {
		t.Fatalf("LoadStateForProfile() after save failed: %v", err)
	}

	if reloadedState.Config.DefaultRegion != "us-west-2" {
		t.Errorf("SaveStateForProfile() did not save config correctly: got %s, want us-west-2", reloadedState.Config.DefaultRegion)
	}
}

func TestStaticProfileProvider(t *testing.T) {
	// Test with profile set
	provider := NewStaticProfileProvider("test-aws-profile")

	profile, err := provider.GetCurrentProfile()
	if err != nil {
		t.Fatalf("GetCurrentProfile() failed: %v", err)
	}

	if profile != "test-aws-profile" {
		t.Errorf("GetCurrentProfile() returned wrong profile: got %s, want test-aws-profile", profile)
	}

	// Test with empty profile
	emptyProvider := NewStaticProfileProvider("")

	_, err = emptyProvider.GetCurrentProfile()
	if err == nil {
		t.Error("GetCurrentProfile() should fail with empty profile")
	}
}

func TestGetDefaultManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	// Test convenience function
	manager, err := GetDefaultManager()
	if err != nil {
		t.Fatalf("GetDefaultManager() failed: %v", err)
	}

	if manager == nil {
		t.Error("GetDefaultManager() returned nil manager")
	}

	// Test that it works like a regular manager
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState() on default manager failed: %v", err)
	}

	if state.Instances == nil {
		t.Error("Default manager LoadState() returned nil Instances map")
	}
}

func TestProfileIntegration(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	// Test the legacy compatibility function
	manager, err := NewProfileAwareManager()
	if err != nil {
		t.Fatalf("NewProfileAwareManager() failed: %v", err)
	}

	if manager == nil {
		t.Error("NewProfileAwareManager() returned nil manager")
	}

	// Test that basic state operations work
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState() on profile-aware manager failed: %v", err)
	}

	if state == nil {
		t.Error("Profile-aware manager LoadState() returned nil state")
	}
}
