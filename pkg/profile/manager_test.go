package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/core"
)

func TestProfileManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "profile-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test manager
	manager, err := core.NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test adding profiles
	personalProfile := Profile{
		Type:       ProfileTypePersonal,
		Name:       "Test Personal",
		AWSProfile: "default",
		Region:     "us-west-2",
		Default:    true,
		CreatedAt:  time.Now(),
	}

	// Add the personal profile
	err = manager.Add("personal", personalProfile)
	if err != nil {
		t.Fatalf("Failed to add personal profile: %v", err)
	}

	// Check if profile was added and set as current
	if manager.profiles.CurrentProfile != "personal" {
		t.Errorf("Expected current profile to be 'personal', got '%s'", manager.profiles.CurrentProfile)
	}

	// Add invitation profile
	invitationProfile := Profile{
		Type:            ProfileTypeInvitation,
		Name:            "Test Invitation",
		InvitationToken: "test-token",
		OwnerAccount:    "123456789012",
		Region:          "us-east-1",
		CreatedAt:       time.Now(),
	}

	err = manager.Add("invitation", invitationProfile)
	if err != nil {
		t.Fatalf("Failed to add invitation profile: %v", err)
	}

	// Check if both profiles exist
	if len(manager.profiles.Profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(manager.profiles.Profiles))
	}

	// Test getting a profile
	profile, exists := manager.Get("invitation")
	if !exists {
		t.Fatalf("Expected invitation profile to exist")
	}
	if profile.Type != ProfileTypeInvitation {
		t.Errorf("Expected profile type %s, got %s", ProfileTypeInvitation, profile.Type)
	}
	if profile.Name != "Test Invitation" {
		t.Errorf("Expected profile name 'Test Invitation', got '%s'", profile.Name)
	}

	// Test switching profiles
	err = manager.Use("invitation")
	if err != nil {
		t.Fatalf("Failed to switch profile: %v", err)
	}

	if manager.profiles.CurrentProfile != "invitation" {
		t.Errorf("Expected current profile to be 'invitation', got '%s'", manager.profiles.CurrentProfile)
	}

	// Test getting current profile
	currentProfile, exists := manager.Current()
	if !exists {
		t.Fatalf("Expected current profile to exist")
	}
	if currentProfile.Type != ProfileTypeInvitation {
		t.Errorf("Expected current profile type %s, got %s", ProfileTypeInvitation, currentProfile.Type)
	}

	// Test updating a profile
	updates := Profile{
		Type:            ProfileTypeInvitation,
		Name:            "Updated Invitation",
		InvitationToken: "new-token",
		OwnerAccount:    "123456789012",
		Region:          "us-east-2",
	}

	err = manager.Update("invitation", updates)
	if err != nil {
		t.Fatalf("Failed to update profile: %v", err)
	}

	// Check if profile was updated
	profile, _ = manager.Get("invitation")
	if profile.Name != "Updated Invitation" {
		t.Errorf("Expected profile name 'Updated Invitation', got '%s'", profile.Name)
	}
	if profile.Region != "us-east-2" {
		t.Errorf("Expected profile region 'us-east-2', got '%s'", profile.Region)
	}

	// Test removing a profile
	// First switch back to personal
	err = manager.Use("personal")
	if err != nil {
		t.Fatalf("Failed to switch profile: %v", err)
	}

	// Then remove invitation
	err = manager.Remove("invitation")
	if err != nil {
		t.Fatalf("Failed to remove profile: %v", err)
	}

	// Check if profile was removed
	_, exists = manager.Get("invitation")
	if exists {
		t.Errorf("Expected invitation profile to be removed")
	}

	// Test removing current profile (should fail)
	err = manager.Remove("personal")
	if err == nil {
		t.Errorf("Expected error when removing current profile")
	}

	// Test loading and saving
	// Save current state
	err = manager.save()
	if err != nil {
		t.Fatalf("Failed to save profiles: %v", err)
	}

	// Create a new manager that loads the saved state
	newManager := &core.Manager{
		configPath: filepath.Join(tempDir, "profiles.json"),
	}

	err = newManager.load()
	if err != nil {
		t.Fatalf("Failed to load profiles: %v", err)
	}

	// Check if state was loaded correctly
	if newManager.profiles.CurrentProfile != "personal" {
		t.Errorf("Expected loaded current profile to be 'personal', got '%s'", newManager.profiles.CurrentProfile)
	}

	if len(newManager.profiles.Profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(newManager.profiles.Profiles))
	}

	// Test AddInvitation helper
	err = newManager.AddInvitation(
		"new-invitation",
		"New Invitation",
		"invitation-token",
		"987654321098",
		"eu-west-1",
	)
	if err != nil {
		t.Fatalf("Failed to add invitation: %v", err)
	}

	// Check if invitation was added
	profile, exists = newManager.Get("new-invitation")
	if !exists {
		t.Fatalf("Expected new invitation profile to exist")
	}
	if profile.Type != ProfileTypeInvitation {
		t.Errorf("Expected profile type %s, got %s", ProfileTypeInvitation, profile.Type)
	}
	if profile.InvitationToken != "invitation-token" {
		t.Errorf("Expected token 'invitation-token', got '%s'", profile.InvitationToken)
	}

	// Test List method
	profiles := newManager.List()
	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles in list, got %d", len(profiles))
	}

	// Test adding duplicate profile
	err = newManager.Add("personal", personalProfile)
	if err == nil {
		t.Errorf("Expected error when adding duplicate profile")
	}
}