package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	
	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)
	
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	if manager == nil {
		t.Fatal("NewManager() returned nil manager")
	}
	
	// Check that config directory was created
	cwsDir := filepath.Join(tempDir, ".cloudworkstation")
	if _, err := os.Stat(cwsDir); os.IsNotExist(err) {
		t.Errorf("Config directory was not created: %s", cwsDir)
	}
}

func TestProfileCRUD(t *testing.T) {
	manager := createTestManager(t)
	
	// Test creating a profile
	profile := &Profile{
		Name:       "test-profile",
		AWSProfile: "test-aws-profile",
		Region:     "us-east-1",
		Default:    false,
	}
	
	err := manager.Set("test-profile", profile)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}
	
	// Test retrieving the profile
	retrieved, err := manager.Get("test-profile")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	
	if retrieved.Name != profile.Name {
		t.Errorf("Retrieved profile name mismatch: got %s, want %s", retrieved.Name, profile.Name)
	}
	
	if retrieved.AWSProfile != profile.AWSProfile {
		t.Errorf("Retrieved AWS profile mismatch: got %s, want %s", retrieved.AWSProfile, profile.AWSProfile)
	}
	
	if retrieved.Region != profile.Region {
		t.Errorf("Retrieved region mismatch: got %s, want %s", retrieved.Region, profile.Region)
	}
	
	// Test listing profiles
	profiles := manager.List()
	if len(profiles) != 1 {
		t.Errorf("List() returned %d profiles, expected 1", len(profiles))
	}
	
	// Test deleting the profile
	err = manager.Delete("test-profile")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}
	
	// Verify profile is gone
	_, err = manager.Get("test-profile")
	if err == nil {
		t.Error("Get() should have failed after Delete()")
	}
	
	if _, ok := err.(*ProfileNotFoundError); !ok {
		t.Errorf("Get() returned wrong error type: %T", err)
	}
}

func TestCurrentProfile(t *testing.T) {
	manager := createTestManager(t)
	
	// Test with no current profile
	_, err := manager.GetCurrent()
	if err == nil {
		t.Error("GetCurrent() should fail when no current profile is set")
	}
	
	if _, ok := err.(*NoCurrentProfileError); !ok {
		t.Errorf("GetCurrent() returned wrong error type: %T", err)
	}
	
	// Create a profile
	profile := &Profile{
		Name:       "current-test",
		AWSProfile: "test-aws",
		Region:     "us-west-2",
		Default:    false,
	}
	
	err = manager.Set("current-test", profile)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}
	
	// Set as current
	err = manager.SetCurrent("current-test")
	if err != nil {
		t.Fatalf("SetCurrent() failed: %v", err)
	}
	
	// Test getting current profile
	current, err := manager.GetCurrent()
	if err != nil {
		t.Fatalf("GetCurrent() failed: %v", err)
	}
	
	if current.Name != "current-test" {
		t.Errorf("GetCurrent() returned wrong profile: got %s, want current-test", current.Name)
	}
	
	// Test getting current name
	currentName := manager.GetCurrentName()
	if currentName != "current-test" {
		t.Errorf("GetCurrentName() returned wrong name: got %s, want current-test", currentName)
	}
}

func TestDefaultProfile(t *testing.T) {
	manager := createTestManager(t)
	
	// Test creating default profile
	err := manager.CreateDefault("default-aws", "us-east-1")
	if err != nil {
		t.Fatalf("CreateDefault() failed: %v", err)
	}
	
	// Verify default profile was created
	profile, err := manager.Get(DefaultProfileName)
	if err != nil {
		t.Fatalf("Get() failed for default profile: %v", err)
	}
	
	if !profile.Default {
		t.Error("Default profile should have Default=true")
	}
	
	if profile.AWSProfile != "default-aws" {
		t.Errorf("Default profile AWS profile mismatch: got %s, want default-aws", profile.AWSProfile)
	}
	
	// Verify it's set as current
	current, err := manager.GetCurrent()
	if err != nil {
		t.Fatalf("GetCurrent() failed: %v", err)
	}
	
	if current.Name != DefaultProfileName {
		t.Errorf("Default profile not set as current: got %s, want %s", current.Name, DefaultProfileName)
	}
	
	// Test that CreateDefault doesn't overwrite existing profiles
	err = manager.CreateDefault("should-not-create", "us-west-1")
	if err != nil {
		t.Fatalf("CreateDefault() failed on second call: %v", err)
	}
	
	// Verify original profile is unchanged
	profile, err = manager.Get(DefaultProfileName)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	
	if profile.AWSProfile != "default-aws" {
		t.Error("CreateDefault() overwrote existing profile")
	}
}

func TestProfileValidation(t *testing.T) {
	manager := createTestManager(t)
	
	tests := []struct {
		name          string
		profile       *Profile
		expectedError string
	}{
		{
			name: "empty name",
			profile: &Profile{
				Name:       "",
				AWSProfile: "test",
				Region:     "us-east-1",
			},
			expectedError: "profile name is required",
		},
		{
			name: "whitespace name",
			profile: &Profile{
				Name:       "  test  ",
				AWSProfile: "test",
				Region:     "us-east-1",
			},
			expectedError: "profile name cannot have leading/trailing whitespace",
		},
		{
			name: "empty aws profile",
			profile: &Profile{
				Name:       "test",
				AWSProfile: "",
				Region:     "us-east-1",
			},
			expectedError: "AWS profile is required",
		},
		{
			name: "empty region",
			profile: &Profile{
				Name:       "test",
				AWSProfile: "test",
				Region:     "",
			},
			expectedError: "region is required",
		},
		{
			name: "invalid region format",
			profile: &Profile{
				Name:       "test",
				AWSProfile: "test",
				Region:     "invalid",
			},
			expectedError: "invalid region format",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Set("test", tt.profile)
			if err == nil {
				t.Errorf("Set() should have failed for %s", tt.name)
				return
			}
			
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("Set() returned wrong error type: %T", err)
				return
			}
			
			if validationErr.Message != tt.expectedError {
				t.Errorf("Set() returned wrong error message: got %s, want %s", validationErr.Message, tt.expectedError)
			}
		})
	}
}

func TestDefaultProfileLogic(t *testing.T) {
	manager := createTestManager(t)
	
	// Create first profile as default
	profile1 := &Profile{
		Name:       "profile1",
		AWSProfile: "aws1",
		Region:     "us-east-1",
		Default:    true,
	}
	
	err := manager.Set("profile1", profile1)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}
	
	// Verify it's current
	currentName := manager.GetCurrentName()
	if currentName != "profile1" {
		t.Errorf("Default profile not set as current: got %s, want profile1", currentName)
	}
	
	// Create second profile as default
	profile2 := &Profile{
		Name:       "profile2",
		AWSProfile: "aws2",
		Region:     "us-west-2",
		Default:    true,
	}
	
	err = manager.Set("profile2", profile2)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}
	
	// Verify first profile is no longer default
	retrieved1, err := manager.Get("profile1")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	
	if retrieved1.Default {
		t.Error("First profile should no longer be default")
	}
	
	// Verify second profile is current
	currentName = manager.GetCurrentName()
	if currentName != "profile2" {
		t.Errorf("New default profile not set as current: got %s, want profile2", currentName)
	}
}

func TestPersistence(t *testing.T) {
	tempDir := t.TempDir()
	
	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)
	
	// Create manager and add profile
	manager1, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	profile := &Profile{
		Name:       "persist-test",
		AWSProfile: "persist-aws",
		Region:     "us-east-1",
		Default:    true,
	}
	
	err = manager1.Set("persist-test", profile)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}
	
	// Create new manager instance (should load from disk)
	manager2, err := NewManager()
	if err != nil {
		t.Fatalf("Second NewManager() failed: %v", err)
	}
	
	// Verify profile was loaded
	retrieved, err := manager2.Get("persist-test")
	if err != nil {
		t.Fatalf("Get() failed after reload: %v", err)
	}
	
	if retrieved.AWSProfile != "persist-aws" {
		t.Errorf("Profile not persisted correctly: got %s, want persist-aws", retrieved.AWSProfile)
	}
	
	// Verify current profile was loaded
	currentName := manager2.GetCurrentName()
	if currentName != "persist-test" {
		t.Errorf("Current profile not persisted: got %s, want persist-test", currentName)
	}
}

func createTestManager(t *testing.T) *Manager {
	tempDir := t.TempDir()
	
	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tempDir)
	
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("createTestManager() failed: %v", err)
	}
	
	return manager
}