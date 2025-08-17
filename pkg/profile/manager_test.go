package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/core"
)

func TestProfileManager(t *testing.T) {
	// Backup existing profile config and clean up on test completion
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".cloudworkstation", "profiles.json")

	// Backup existing config if it exists
	var backupData []byte
	var hadExistingConfig bool
	if data, err := os.ReadFile(configPath); err == nil {
		backupData = data
		hadExistingConfig = true
		_ = os.Remove(configPath) // Remove for clean test
	}

	defer func() {
		if hadExistingConfig {
			_ = os.MkdirAll(filepath.Dir(configPath), 0755)
			_ = os.WriteFile(configPath, backupData, 0644) // Restore
		} else {
			_ = os.Remove(configPath) // Clean up test data
		}
	}()

	// Create a clean test manager
	manager, err := core.NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test adding profiles
	personalProfile := &core.Profile{
		Name:       "Test Personal",
		AWSProfile: "default",
		Region:     "us-west-2",
		CreatedAt:  time.Now(),
	}

	// Add the personal profile
	err = manager.Set("personal", personalProfile)
	if err != nil {
		t.Fatalf("Failed to set personal profile: %v", err)
	}

	// Check if profile was added
	_, err = manager.Get("personal")
	if err != nil {
		t.Errorf("Failed to get personal profile: %v", err)
	}

	// Add invitation profile
	invitationProfile := &core.Profile{
		Name:       "Test Invitation",
		AWSProfile: "invitation-profile",
		Region:     "us-east-1",
		CreatedAt:  time.Now(),
	}

	err = manager.Set("invitation", invitationProfile)
	if err != nil {
		t.Fatalf("Failed to set invitation profile: %v", err)
	}

	// Check if both profiles exist
	profiles := manager.List()
	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}

	// Test getting a profile
	profile, err := manager.Get("invitation")
	if err != nil {
		t.Fatalf("Failed to get invitation profile: %v", err)
	}
	if profile.Name != "invitation" {
		t.Errorf("Expected profile name 'invitation', got '%s'", profile.Name)
	}

	// Test switching profiles
	err = manager.SetCurrent("invitation")
	if err != nil {
		t.Fatalf("Failed to switch profile: %v", err)
	}

	if manager.GetCurrentName() != "invitation" {
		t.Errorf("Expected current profile to be 'invitation', got '%s'", manager.GetCurrentName())
	}

	// Test getting current profile
	currentProfile, err := manager.GetCurrent()
	if err != nil {
		t.Fatalf("Failed to get current profile: %v", err)
	}
	if currentProfile.Name != "invitation" {
		t.Errorf("Expected current profile name 'invitation', got '%s'", currentProfile.Name)
	}

	// Test updating a profile
	updates := &core.Profile{
		Name:       "Updated Invitation",
		AWSProfile: "updated-profile",
		Region:     "us-east-2",
		CreatedAt:  time.Now(),
	}

	err = manager.Set("invitation", updates)
	if err != nil {
		t.Fatalf("Failed to update profile: %v", err)
	}

	// Check if profile was updated
	profile, err = manager.Get("invitation")
	if err != nil {
		t.Fatalf("Failed to get updated profile: %v", err)
	}
	if profile.Name != "invitation" {
		t.Errorf("Expected profile name 'invitation', got '%s'", profile.Name)
	}
	if profile.Region != "us-east-2" {
		t.Errorf("Expected region 'us-east-2', got '%s'", profile.Region)
	}

	// Test removing a profile
	// First switch back to personal
	err = manager.SetCurrent("personal")
	if err != nil {
		t.Fatalf("Failed to switch profile: %v", err)
	}

	// Then remove invitation
	err = manager.Delete("invitation")
	if err != nil {
		t.Fatalf("Failed to delete profile: %v", err)
	}

	// Check if profile was removed
	_, err = manager.Get("invitation")
	if err == nil {
		t.Errorf("Expected invitation profile to be removed, but it still exists")
	}

	// Test that we now have only 1 profile remaining
	finalProfiles := manager.List()
	if len(finalProfiles) != 1 {
		t.Errorf("Expected 1 profile remaining after deletion, got %d", len(finalProfiles))
	}

	// Test that the remaining profile is personal
	if manager.GetCurrentName() != "personal" {
		t.Errorf("Expected current profile to be 'personal', got '%s'", manager.GetCurrentName())
	}
}
