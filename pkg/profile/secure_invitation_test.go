package profile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

func TestSecureInvitationManager(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-secure-invitation-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config directory
	configDir := filepath.Join(tempDir, ".cloudworkstation")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create a profile manager for testing
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		t.Fatalf("Failed to create profile manager: %v", err)
	}

	// Create a secure invitation manager
	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		t.Fatalf("Failed to create secure invitation manager: %v", err)
	}

	// Test creating a secure invitation
	invitation, err := secureManager.CreateSecureInvitation(
		"Test Invitation",
		profile.InvitationTypeAdmin,
		30, // valid for 30 days
		"s3://test-bucket/config.json",
		true,  // can invite
		false, // not transferable
		true,  // device bound
		3,     // max 3 devices
		"",    // no parent token
	)

	if err != nil {
		t.Fatalf("Failed to create secure invitation: %v", err)
	}

	// Verify invitation properties
	if invitation.Name != "Test Invitation" {
		t.Errorf("Expected invitation name to be 'Test Invitation', got '%s'", invitation.Name)
	}

	if invitation.Type != profile.InvitationTypeAdmin {
		t.Errorf("Expected invitation type to be admin, got '%s'", invitation.Type)
	}

	if invitation.S3ConfigPath != "s3://test-bucket/config.json" {
		t.Errorf("Expected S3 config path to be 's3://test-bucket/config.json', got '%s'", invitation.S3ConfigPath)
	}

	if !invitation.CanInvite {
		t.Errorf("Expected invitation to allow invites")
	}

	if invitation.Transferable {
		t.Errorf("Expected invitation to not be transferable")
	}

	if !invitation.DeviceBound {
		t.Errorf("Expected invitation to be device bound")
	}

	if invitation.MaxDevices != 3 {
		t.Errorf("Expected max devices to be 3, got %d", invitation.MaxDevices)
	}

	// Encode invitation to string for sharing
	encodedToken, err := invitation.EncodeToString()
	if err != nil {
		t.Fatalf("Failed to encode invitation: %v", err)
	}

	// Test secure adding to profile
	err = secureManager.SecureAddToProfile(encodedToken, "Test Profile")
	if err != nil {
		t.Fatalf("Failed to add secure profile: %v", err)
	}

	// Get the created profile
	profile, err := profileManager.GetProfile("Test Profile")
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}

	// Check profile properties
	if profile.Name != "Test Profile" {
		t.Errorf("Expected profile name to be 'Test Profile', got '%s'", profile.Name)
	}

	if profile.Type != "invitation" {
		t.Errorf("Expected profile type to be 'invitation', got '%s'", profile.Type)
	}

	if !profile.DeviceBound {
		t.Errorf("Expected profile to be device bound")
	}

	if profile.InvitationToken != invitation.Token {
		t.Errorf("Expected profile token to match invitation token")
	}

	if profile.BindingRef == "" {
		t.Errorf("Expected binding reference to not be empty")
	}

	// Validate the secure profile
	err = secureManager.ValidateSecureProfile(profile)
	if err != nil {
		t.Errorf("Failed to validate secure profile: %v", err)
	}

	// Test permissions inheritance for sub-invitations
	subInvitation, err := secureManager.CreateSecureInvitation(
		"Sub Invitation",
		profile.InvitationTypeReadWrite, // lower permissions than parent
		15, // valid for 15 days (shorter than parent)
		"",  // no S3 config
		true, // try to allow invites (should work because parent can)
		true, // try to make transferable (should fail because parent is not)
		false, // try to make not device bound (should fail because parent is)
		5,     // try more devices than parent allows (should be limited)
		invitation.Token, // use parent token
	)

	if err != nil {
		t.Fatalf("Failed to create sub-invitation: %v", err)
	}

	// Verify sub-invitation enforces parent constraints
	if subInvitation.Transferable {
		t.Errorf("Sub-invitation should not be transferable since parent is not")
	}

	if !subInvitation.DeviceBound {
		t.Errorf("Sub-invitation should be device bound since parent is")
	}

	if subInvitation.MaxDevices > invitation.MaxDevices {
		t.Errorf("Sub-invitation should not allow more devices than parent")
	}

	// Test registry integration with devices
	// Note: This test uses local storage mode so it won't actually connect to S3
	devices, err := secureManager.GetInvitationDevices(invitation.Token)
	if err != nil {
		t.Fatalf("Failed to get invitation devices: %v", err)
	}

	// Should have at least one device (the one we registered with SecureAddToProfile)
	if len(devices) == 0 {
		t.Errorf("Expected at least one registered device")
	}

	// Test revoking devices
	if len(devices) > 0 {
		deviceID := ""
		for _, device := range devices {
			if id, ok := device["device_id"].(string); ok {
				deviceID = id
				break
			}
		}

		if deviceID != "" {
			err = secureManager.RevokeDevice(invitation.Token, deviceID)
			if err != nil {
				t.Errorf("Failed to revoke device: %v", err)
			}

			// Check that device is revoked
			devicesAfterRevoke, err := secureManager.GetInvitationDevices(invitation.Token)
			if err != nil {
				t.Fatalf("Failed to get devices after revocation: %v", err)
			}

			for _, device := range devicesAfterRevoke {
				if id, ok := device["device_id"].(string); ok && id == deviceID {
					t.Errorf("Device should be revoked but was found in devices list")
				}
			}
		}
	}
}

// TestInvitationPermissionEnforcement tests the enforcement of parent invitation permissions
func TestInvitationPermissionEnforcement(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-permission-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config directory
	configDir := filepath.Join(tempDir, ".cloudworkstation")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create a profile manager for testing
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		t.Fatalf("Failed to create profile manager: %v", err)
	}

	// Create a secure invitation manager
	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		t.Fatalf("Failed to create secure invitation manager: %v", err)
	}

	// Create a restrictive parent invitation (no delegation allowed)
	parentInvitation, err := secureManager.CreateSecureInvitation(
		"Restrictive Parent",
		profile.InvitationTypeReadOnly,
		30, // valid for 30 days
		"", // no S3 config
		false, // cannot invite others
		false, // not transferable
		true,  // device bound
		1,     // max 1 device
		"",    // no parent token
	)
	if err != nil {
		t.Fatalf("Failed to create parent invitation: %v", err)
	}

	// Attempt to create a sub-invitation with higher permissions (should fail)
	_, err = secureManager.CreateSecureInvitation(
		"Invalid Sub Invitation",
		profile.InvitationTypeAdmin, // try higher permissions
		30, // valid for 30 days
		"", // no S3 config
		true, // try to allow invites (should fail)
		true, // try to make transferable (should fail)
		false, // try to remove device binding (should fail)
		2,     // try more devices (should fail)
		parentInvitation.Token, // use restrictive parent token
	)

	// Should fail because parent doesn't allow delegation
	if err == nil {
		t.Errorf("Should not be able to create sub-invitation when parent doesn't allow it")
	}

	// Create a permissive parent invitation
	parentInvitation, err = secureManager.CreateSecureInvitation(
		"Permissive Parent",
		profile.InvitationTypeAdmin,
		30, // valid for 30 days
		"", // no S3 config
		true, // can invite others
		false, // not transferable
		true,  // device bound
		2,     // max 2 devices
		"",    // no parent token
	)
	if err != nil {
		t.Fatalf("Failed to create parent invitation: %v", err)
	}

	// Create a sub-invitation with attempt at higher permissions
	subInvitation, err := secureManager.CreateSecureInvitation(
		"Sub Invitation",
		profile.InvitationTypeAdmin, // same permissions (should work)
		30, // valid for 30 days
		"", // no S3 config
		true, // try to allow invites (should work)
		true, // try to make transferable (should fail because parent is not)
		false, // try to remove device binding (should fail because parent has it)
		3,     // try more devices (should be limited to parent's max)
		parentInvitation.Token, // use permissive parent token
	)

	if err != nil {
		t.Fatalf("Failed to create sub-invitation: %v", err)
	}

	// Verify permission inheritance
	if subInvitation.Transferable {
		t.Errorf("Sub-invitation should inherit non-transferable from parent")
	}

	if !subInvitation.DeviceBound {
		t.Errorf("Sub-invitation should inherit device binding from parent")
	}

	if subInvitation.MaxDevices > parentInvitation.MaxDevices {
		t.Errorf("Sub-invitation should be limited to parent's max devices (expected %d, got %d)",
			parentInvitation.MaxDevices, subInvitation.MaxDevices)
	}
}