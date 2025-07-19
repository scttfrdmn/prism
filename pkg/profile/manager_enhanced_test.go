package profile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockCredentialProvider implements CredentialProvider for testing
type MockCredentialProvider struct {
	storedCredentials map[string]*Credentials
}

func NewMockCredentialProvider() *MockCredentialProvider {
	return &MockCredentialProvider{
		storedCredentials: make(map[string]*Credentials),
	}
}

func (m *MockCredentialProvider) StoreCredentials(profileID string, creds *Credentials) error {
	m.storedCredentials[profileID] = creds
	return nil
}

func (m *MockCredentialProvider) GetCredentials(profileID string) (*Credentials, error) {
	creds, exists := m.storedCredentials[profileID]
	if !exists {
		return nil, ErrCredentialsNotFound
	}
	return creds, nil
}

func (m *MockCredentialProvider) ClearCredentials(profileID string) error {
	delete(m.storedCredentials, profileID)
	return nil
}

func TestManagerEnhanced(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "profile-manager-enhanced-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock credential provider
	mockCredProvider := NewMockCredentialProvider()

	// Create a test manager
	manager := &ManagerEnhanced{
		configPath:         filepath.Join(tempDir, "profiles.json"),
		credentialProvider: mockCredProvider,
		profiles: &Profiles{
			Profiles:       make(map[string]Profile),
			CurrentProfile: "",
		},
	}

	// Test adding profiles
	personalProfile := Profile{
		Type:       ProfileTypePersonal,
		Name:       "Test Personal",
		AWSProfile: "test-aws-profile",
		Region:     "us-west-2",
		Default:    true,
		CreatedAt:  time.Now(),
	}

	// Add profile
	err = manager.AddProfile(personalProfile)
	if err != nil {
		t.Fatalf("Failed to add profile: %v", err)
	}

	// Check if profile exists
	exists := manager.ProfileExists("test-aws-profile")
	if !exists {
		t.Errorf("Expected profile to exist")
	}

	// Test getting profile
	profile, err := manager.GetProfile("test-aws-profile")
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}
	if profile.Name != "Test Personal" {
		t.Errorf("Expected profile name 'Test Personal', got '%s'", profile.Name)
	}

	// Test storing and retrieving credentials
	testCreds := &Credentials{
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		SessionToken:    "test-session-token",
		Expiration:      time.Now().Add(1 * time.Hour),
	}

	err = manager.StoreProfileCredentials("test-aws-profile", testCreds)
	if err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}

	// Get credentials
	creds, err := manager.GetProfileCredentials("test-aws-profile")
	if err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}
	if creds.AccessKeyID != "test-access-key" {
		t.Errorf("Expected access key 'test-access-key', got '%s'", creds.AccessKeyID)
	}

	// Test switching profile
	err = manager.SwitchProfile("test-aws-profile")
	if err != nil {
		t.Fatalf("Failed to switch profile: %v", err)
	}
	if manager.profiles.CurrentProfile != "test-aws-profile" {
		t.Errorf("Expected current profile to be 'test-aws-profile', got '%s'", manager.profiles.CurrentProfile)
	}

	// Test getting current profile
	currentProfile, err := manager.GetCurrentProfile()
	if err != nil {
		t.Fatalf("Failed to get current profile: %v", err)
	}
	if currentProfile.Name != "Test Personal" {
		t.Errorf("Expected current profile name 'Test Personal', got '%s'", currentProfile.Name)
	}

	// Test updating profile
	updatedProfile := Profile{
		Name:       "Updated Personal",
		AWSProfile: "test-aws-profile",
		Region:     "us-east-1",
	}
	err = manager.UpdateProfile("test-aws-profile", updatedProfile)
	if err != nil {
		t.Fatalf("Failed to update profile: %v", err)
	}

	// Check if profile was updated
	profile, _ = manager.GetProfile("test-aws-profile")
	if profile.Name != "Updated Personal" {
		t.Errorf("Expected profile name 'Updated Personal', got '%s'", profile.Name)
	}
	if profile.Region != "us-east-1" {
		t.Errorf("Expected profile region 'us-east-1', got '%s'", profile.Region)
	}

	// Test context functions
	ctx := context.Background()
	ctx, err = manager.WithProfile(ctx, "test-aws-profile")
	if err != nil {
		t.Fatalf("Failed to create context with profile: %v", err)
	}

	ctxProfile, ok := GetProfileFromContext(ctx)
	if !ok {
		t.Fatalf("Failed to get profile from context")
	}
	if ctxProfile.Name != "Updated Personal" {
		t.Errorf("Expected profile from context name 'Updated Personal', got '%s'", ctxProfile.Name)
	}

	// Test listing profiles
	profiles, err := manager.ListProfiles()
	if err != nil {
		t.Fatalf("Failed to list profiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(profiles))
	}

	// Test removing profile and credentials
	err = manager.RemoveProfile("test-aws-profile")
	if err == nil {
		t.Errorf("Expected error removing current profile")
	}

	// Add another profile to be able to switch
	anotherProfile := Profile{
		Type:      ProfileTypePersonal,
		Name:      "Another Profile",
		AWSProfile: "another-profile",
		Region:    "eu-west-1",
	}
	err = manager.AddProfile(anotherProfile)
	if err != nil {
		t.Fatalf("Failed to add another profile: %v", err)
	}

	// Switch to another profile
	err = manager.SwitchProfile("another-profile")
	if err != nil {
		t.Fatalf("Failed to switch profile: %v", err)
	}

	// Now remove the original profile
	err = manager.RemoveProfile("test-aws-profile")
	if err != nil {
		t.Fatalf("Failed to remove profile: %v", err)
	}

	// Check if profile was removed
	exists = manager.ProfileExists("test-aws-profile")
	if exists {
		t.Errorf("Expected profile to be removed")
	}

	// Check if credentials were removed
	_, err = manager.GetProfileCredentials("test-aws-profile")
	if err == nil {
		t.Errorf("Expected error getting credentials for removed profile")
	}

	// Test error cases
	_, err = manager.GetProfile("non-existent-profile")
	if err != ErrProfileNotFound {
		t.Errorf("Expected ErrProfileNotFound, got %v", err)
	}

	err = manager.UpdateProfile("non-existent-profile", Profile{})
	if err != ErrProfileNotFound {
		t.Errorf("Expected ErrProfileNotFound, got %v", err)
	}

	err = manager.StoreProfileCredentials("non-existent-profile", testCreds)
	if err != ErrProfileNotFound {
		t.Errorf("Expected ErrProfileNotFound, got %v", err)
	}

	_, err = manager.GetProfileCredentials("non-existent-profile")
	if err != ErrProfileNotFound {
		t.Errorf("Expected ErrProfileNotFound, got %v", err)
	}

	// Test validation
	err = manager.AddProfile(Profile{})
	if err == nil {
		t.Errorf("Expected error adding invalid profile")
	}

	err = manager.AddProfile(Profile{
		Name: "Invalid Type",
		Type: "invalid-type",
	})
	if err == nil {
		t.Errorf("Expected error adding profile with invalid type")
	}
}