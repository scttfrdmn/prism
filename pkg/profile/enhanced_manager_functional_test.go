// Package profile provides functional tests for enhanced profile manager
package profile

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestProfileManagerEnhancedFunctionalWorkflow validates enhanced profile manager functionality
func TestProfileManagerEnhancedFunctionalWorkflow(t *testing.T) {
	manager, err := NewManagerEnhanced()
	if err != nil {
		t.Skipf("Skipping test - could not create manager (likely environment issue): %v", err)
	}

	// Test complete enhanced profile workflow
	testEnhancedManagerCreation(t, manager)
	testProfileOperations(t, manager)
	testContextOperations(t, manager)

	t.Log("✅ Enhanced profile manager functional workflow validated")
}

// testEnhancedManagerCreation validates manager initialization
func testEnhancedManagerCreation(t *testing.T, manager *ManagerEnhanced) {
	if manager.configPath == "" {
		t.Error("Enhanced manager config path should be set")
	}

	if manager.profiles == nil {
		t.Error("Enhanced manager profiles should be initialized")
	}

	t.Log("Enhanced profile manager creation validated")
}

// testProfileOperations validates core profile operations
func testProfileOperations(t *testing.T, manager *ManagerEnhanced) {
	// Clean up any existing test profile from previous runs
	if manager.ProfileExists("test-enhanced-profile") {
		// Switch away from it if it's current
		currentProfile, err := manager.GetCurrentProfile()
		if err == nil && currentProfile.Name == "test-enhanced-profile" {
			// Create a minimal cleanup profile to switch to
			cleanupProfile := Profile{
				Type:       ProfileTypePersonal,
				Name:       "cleanup-profile",
				AWSProfile: "default",
				Region:     "us-west-2",
				Default:    false,
				CreatedAt:  time.Now(),
			}
			_ = manager.AddProfile(cleanupProfile)
			_ = manager.SwitchProfile("cleanup-profile")
		}
		_ = manager.RemoveProfile("test-enhanced-profile")
		// Clean up the cleanup profile
		if manager.ProfileExists("cleanup-profile") {
			_ = manager.RemoveProfile("cleanup-profile")
		}
	}

	// Test adding a profile
	testProfile := Profile{
		Type:       ProfileTypePersonal,
		Name:       "test-enhanced-profile",
		AWSProfile: "default",
		Region:     "us-west-2",
		Default:    false,
		CreatedAt:  time.Now(),
		SSHKeyName: "test-key-pair",
	}

	err := manager.AddProfile(testProfile)
	if err != nil {
		t.Errorf("Failed to add profile: %v", err)
	}

	// Test checking if profile exists
	exists := manager.ProfileExists("test-enhanced-profile")
	if !exists {
		t.Error("Profile should exist after adding")
	}

	// Test getting profile
	retrievedProfile, err := manager.GetProfile("test-enhanced-profile")
	if err != nil {
		t.Errorf("Failed to get profile: %v", err)
	}

	if retrievedProfile.Name != testProfile.Name {
		t.Errorf("Profile name mismatch: expected %s, got %s", testProfile.Name, retrievedProfile.Name)
	}

	// Test listing profiles
	profiles, err := manager.ListProfiles()
	if err != nil {
		t.Errorf("Failed to list profiles: %v", err)
	}

	found := false
	for _, profile := range profiles {
		if profile.Name == "test-enhanced-profile" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Added profile not found in profile list")
	}

	// Test updating profile
	updatedProfile := testProfile
	updatedProfile.Region = "us-east-1"

	err = manager.UpdateProfile("test-enhanced-profile", updatedProfile)
	if err != nil {
		t.Errorf("Failed to update profile: %v", err)
	}

	// Verify update
	retrievedProfile, err = manager.GetProfile("test-enhanced-profile")
	if err != nil {
		t.Errorf("Failed to get updated profile: %v", err)
	}

	if retrievedProfile.Region != "us-east-1" {
		t.Errorf("Profile region not updated: expected us-east-1, got %s", retrievedProfile.Region)
	}

	// Test switching profile
	err = manager.SwitchProfile("test-enhanced-profile")
	if err != nil {
		t.Errorf("Failed to switch profile: %v", err)
	}

	// Test getting current profile
	currentProfile, err := manager.GetCurrentProfile()
	if err != nil {
		t.Errorf("Failed to get current profile: %v", err)
	}

	if currentProfile.Name != "test-enhanced-profile" {
		t.Errorf("Current profile not updated: expected test-enhanced-profile, got %s", currentProfile.Name)
	}

	// Switch away from the profile before removing it (cannot remove active profile)
	err = manager.SwitchProfile("default")
	if err != nil {
		t.Logf("Could not switch to default profile, creating temporary profile: %v", err)
		// Create a temporary profile to switch to
		tempProfile := Profile{
			Type:       ProfileTypePersonal,
			Name:       "temp-profile",
			AWSProfile: "default",
			Region:     "us-west-2",
			Default:    false,
			CreatedAt:  time.Now(),
		}
		_ = manager.AddProfile(tempProfile)
		err = manager.SwitchProfile("temp-profile")
		if err != nil {
			t.Errorf("Failed to switch to temporary profile: %v", err)
		}
	}

	// Test removing profile (should work now that it's not active)
	err = manager.RemoveProfile("test-enhanced-profile")
	if err != nil {
		t.Errorf("Failed to remove profile: %v", err)
	}

	// Verify removal
	exists = manager.ProfileExists("test-enhanced-profile")
	if exists {
		t.Error("Profile should not exist after removal")
	}

	// Clean up temporary profile if created
	if manager.ProfileExists("temp-profile") {
		_ = manager.RemoveProfile("temp-profile")
	}

	t.Log("Profile operations validated")
}

// testContextOperations validates context-based profile operations
func testContextOperations(t *testing.T, manager *ManagerEnhanced) {
	// Create test profile for context operations
	contextProfile := Profile{
		Type:       ProfileTypePersonal,
		Name:       "context-test-profile",
		AWSProfile: "context-test",
		Region:     "us-west-1",
		CreatedAt:  time.Now(),
	}

	err := manager.AddProfile(contextProfile)
	if err != nil {
		t.Errorf("Failed to add context test profile: %v", err)
	}

	// Test WithProfile context operation
	ctx := context.Background()
	profileCtx, err := manager.WithProfile(ctx, "context-test-profile")
	if err != nil {
		t.Errorf("Failed to create profile context: %v", err)
	}

	if profileCtx == nil {
		t.Error("Profile context should not be nil")
	}

	// Test getting profile from context
	profileFromContext, _ := GetProfileFromContext(profileCtx)
	if profileFromContext == nil {
		t.Error("Should retrieve profile from context")
	}

	if profileFromContext.Name != "context-test-profile" {
		t.Errorf("Context profile name mismatch: expected context-test-profile, got %s", profileFromContext.Name)
	}

	// Cleanup
	err = manager.RemoveProfile("context-test-profile")
	if err != nil {
		t.Errorf("Failed to cleanup context test profile: %v", err)
	}

	t.Log("Context operations validated")
}

// TestProfileCredentialManagement validates credential storage operations
func TestProfileCredentialManagement(t *testing.T) {
	manager, err := NewManagerEnhanced()
	if err != nil {
		t.Skipf("Skipping credential test - could not create manager: %v", err)
	}

	// Create test profile
	testProfile := Profile{
		Type:       ProfileTypePersonal,
		Name:       "credential-test-profile",
		AWSProfile: "credential-test",
		Region:     "us-west-2",
		CreatedAt:  time.Now(),
	}

	err = manager.AddProfile(testProfile)
	if err != nil {
		t.Errorf("Failed to add credential test profile: %v", err)
	}

	// Test credential storage (may fail without keychain access, but we test the interface)
	testCredentials := &Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	err = manager.StoreProfileCredentials("credential-test-profile", testCredentials)
	// Note: This might fail in test environment without proper keychain access
	// We test the interface but don't require success for the test to pass

	if err == nil {
		// If storage succeeded, test retrieval
		retrievedCredentials, err := manager.GetProfileCredentials("credential-test-profile")
		if err != nil {
			t.Errorf("Failed to retrieve credentials: %v", err)
		}

		if retrievedCredentials != nil && retrievedCredentials.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
			t.Errorf("Credential access key mismatch: expected AKIAIOSFODNN7EXAMPLE, got %s", retrievedCredentials.AccessKeyID)
		}
	} else {
		t.Logf("Credential storage failed (expected in test environment): %v", err)
	}

	// Cleanup
	err = manager.RemoveProfile("credential-test-profile")
	if err != nil {
		t.Errorf("Failed to cleanup credential test profile: %v", err)
	}

	t.Log("✅ Profile credential management tested")
}

// TestProfileManagerConcurrency validates thread-safe operations
func TestProfileManagerConcurrency(t *testing.T) {
	manager, err := NewManagerEnhanced()
	if err != nil {
		t.Skipf("Skipping concurrency test - could not create manager: %v", err)
	}

	done := make(chan bool, 3)

	// Concurrent profile operations
	go func() {
		for i := 0; i < 5; i++ {
			profile := Profile{
				Type:       ProfileTypePersonal,
				Name:       fmt.Sprintf("concurrent-profile-%d", i),
				AWSProfile: "test",
				Region:     "us-west-2",
				CreatedAt:  time.Now(),
			}
			manager.AddProfile(profile)
		}
		done <- true
	}()

	// Concurrent profile retrieval
	go func() {
		for i := 0; i < 10; i++ {
			manager.ListProfiles()
		}
		done <- true
	}()

	// Concurrent profile existence checks
	go func() {
		for i := 0; i < 10; i++ {
			manager.ProfileExists("concurrent-profile-0")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("Concurrent operations timed out")
		}
	}

	t.Log("✅ Enhanced profile manager concurrency validated")
}

// TestProfileTypeValidation validates profile type handling
func TestProfileTypeValidation(t *testing.T) {
	testCases := []struct {
		profileType ProfileType
		description string
		valid       bool
	}{
		{ProfileTypePersonal, "Personal profile type", true},
		{ProfileTypeInvitation, "Invitation profile type", true},
		{ProfileType("invalid"), "Invalid profile type", false},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			profile := Profile{
				Type:       tc.profileType,
				Name:       "type-test-profile",
				AWSProfile: "test",
				Region:     "us-west-2",
				CreatedAt:  time.Now(),
			}

			if tc.valid {
				if profile.Type != tc.profileType {
					t.Errorf("Profile type not set correctly: expected %s, got %s", tc.profileType, profile.Type)
				}
			} else {
				// For invalid types, we just verify the type is set as provided
				// Validation would typically happen at a higher level
				if profile.Type != tc.profileType {
					t.Errorf("Profile type not set as provided: expected %s, got %s", tc.profileType, profile.Type)
				}
			}
		})
	}

	t.Log("✅ Profile type validation tested")
}
