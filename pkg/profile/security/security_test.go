package security_test

import (
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/profile/security"
)

// TestDeviceBinding tests the device binding functionality
func TestDeviceBinding(t *testing.T) {
	// Create a device binding
	binding, err := security.CreateDeviceBinding("test-profile", "inv-test123")
	if err != nil {
		t.Fatalf("Failed to create device binding: %v", err)
	}

	// Check binding properties
	if binding.DeviceID == "" {
		t.Errorf("Device ID should not be empty")
	}

	if binding.ProfileID != "test-profile" {
		t.Errorf("Expected profile ID to be 'test-profile', got '%s'", binding.ProfileID)
	}

	if binding.InvitationToken != "inv-test123" {
		t.Errorf("Expected invitation token to be 'inv-test123', got '%s'", binding.InvitationToken)
	}

	// Check creation time
	now := time.Now()
	if binding.Created.After(now) || binding.Created.Before(now.Add(-time.Minute)) {
		t.Errorf("Binding creation time is out of expected range")
	}

	// DeviceName should be non-empty (will be the hostname)
	if binding.DeviceName == "" {
		t.Errorf("Device name should not be empty")
	}
}

// TestBindingStorage tests storing and retrieving device bindings
func TestBindingStorage(t *testing.T) {
	// Create a test binding
	binding, err := security.CreateDeviceBinding("test-profile", "inv-test123")
	if err != nil {
		t.Fatalf("Failed to create device binding: %v", err)
	}

	// Store binding
	bindingRef, err := security.StoreDeviceBinding(binding, "test-profile-name")
	if err != nil {
		t.Fatalf("Failed to store device binding: %v", err)
	}

	// Check binding reference
	if bindingRef == "" {
		t.Errorf("Binding reference should not be empty")
	}

	// Retrieve binding
	retrievedBinding, err := security.RetrieveDeviceBinding(bindingRef)
	if err != nil {
		t.Fatalf("Failed to retrieve device binding: %v", err)
	}

	// Check retrieved binding
	if retrievedBinding.DeviceID != binding.DeviceID {
		t.Errorf("Expected device ID '%s', got '%s'", binding.DeviceID, retrievedBinding.DeviceID)
	}

	if retrievedBinding.ProfileID != binding.ProfileID {
		t.Errorf("Expected profile ID '%s', got '%s'", binding.ProfileID, retrievedBinding.ProfileID)
	}

	if retrievedBinding.InvitationToken != binding.InvitationToken {
		t.Errorf("Expected invitation token '%s', got '%s'", binding.InvitationToken, retrievedBinding.InvitationToken)
	}
}

// TestDeviceBindingValidation tests the validation of device bindings
func TestDeviceBindingValidation(t *testing.T) {
	// Create a test binding
	binding, err := security.CreateDeviceBinding("test-profile", "inv-test123")
	if err != nil {
		t.Fatalf("Failed to create device binding: %v", err)
	}

	// Store binding
	bindingRef, err := security.StoreDeviceBinding(binding, "test-profile-name")
	if err != nil {
		t.Fatalf("Failed to store device binding: %v", err)
	}

	// Validate binding
	valid, err := security.ValidateDeviceBinding(bindingRef)
	if err != nil {
		t.Fatalf("Failed to validate device binding: %v", err)
	}

	if !valid {
		t.Errorf("Device binding should be valid")
	}

	// Validate non-existent binding
	valid, err = security.ValidateDeviceBinding("non-existent-binding")
	if err == nil {
		t.Errorf("Expected error for non-existent binding")
	}

	if valid {
		t.Errorf("Non-existent binding should not be valid")
	}
}

// TestKeychain tests the keychain abstraction
func TestKeychain(t *testing.T) {
	// Get keychain provider
	keychain, err := security.NewKeychainProvider()
	if err != nil {
		t.Fatalf("Failed to create keychain provider: %v", err)
	}

	// Test data
	testKey := "test-key-" + time.Now().Format(time.RFC3339Nano)
	testData := []byte("test-data")

	// Store data
	err = keychain.Store(testKey, testData)
	if err != nil {
		t.Fatalf("Failed to store data in keychain: %v", err)
	}

	// Check existence
	if !keychain.Exists(testKey) {
		t.Errorf("Key should exist in keychain")
	}

	// Retrieve data
	retrievedData, err := keychain.Retrieve(testKey)
	if err != nil {
		t.Fatalf("Failed to retrieve data from keychain: %v", err)
	}

	// Check retrieved data
	if string(retrievedData) != string(testData) {
		t.Errorf("Expected data '%s', got '%s'", string(testData), string(retrievedData))
	}

	// Delete data
	err = keychain.Delete(testKey)
	if err != nil {
		t.Fatalf("Failed to delete data from keychain: %v", err)
	}

	// Check deletion
	if keychain.Exists(testKey) {
		t.Errorf("Key should not exist in keychain after deletion")
	}
}

// TestRegistry tests the registry client with local mode
func TestRegistry(t *testing.T) {
	registry := setupRegistryClient(t)

	invitationToken := "inv-test123"
	deviceID := "device-test456"
	deviceID2 := "device-test789"

	testDeviceRegistrationAndValidation(t, registry, invitationToken, deviceID)
	testDeviceListingAndRetrieval(t, registry, invitationToken, deviceID)
	testDeviceRevocation(t, registry, invitationToken, deviceID)
	testInvitationRevocation(t, registry, invitationToken, deviceID2)
}

func setupRegistryClient(t *testing.T) security.RegistryClient {
	config := security.S3RegistryConfig{
		BucketName: "test-bucket",
		Region:     "us-west-2",
		Enabled:    false, // Use local mode
	}

	registry, err := security.NewRegistryClient(config)
	if err != nil {
		t.Fatalf("Failed to create registry client: %v", err)
	}

	return *registry
}

func testDeviceRegistrationAndValidation(t *testing.T, registry security.RegistryClient, invitationToken, deviceID string) {
	// Register a device
	err := registry.RegisterDevice(invitationToken, deviceID)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}

	// Validate the device
	valid, err := registry.ValidateDevice(invitationToken, deviceID)
	if err != nil {
		t.Fatalf("Failed to validate device: %v", err)
	}

	if !valid {
		t.Errorf("Device should be valid")
	}
}

func testDeviceListingAndRetrieval(t *testing.T, registry security.RegistryClient, invitationToken, deviceID string) {
	devices, err := registry.GetInvitationDevices(invitationToken)
	if err != nil {
		t.Fatalf("Failed to get invitation devices: %v", err)
	}

	if len(devices) == 0 {
		t.Errorf("Expected at least one device")
	}

	if !findDeviceInList(devices, deviceID) {
		t.Errorf("Registered device not found in devices list")
	}
}

func findDeviceInList(devices []map[string]interface{}, deviceID string) bool {
	for _, device := range devices {
		if id, ok := device["device_id"]; ok && id == deviceID {
			return true
		}
	}
	return false
}

func testDeviceRevocation(t *testing.T, registry security.RegistryClient, invitationToken, deviceID string) {
	// Revoke device
	err := registry.RevokeDevice(invitationToken, deviceID)
	if err != nil {
		t.Fatalf("Failed to revoke device: %v", err)
	}

	// Validate revoked device
	valid, err := registry.ValidateDevice(invitationToken, deviceID)
	if err != nil {
		t.Fatalf("Failed to validate revoked device: %v", err)
	}

	if valid {
		t.Errorf("Revoked device should not be valid")
	}
}

func testInvitationRevocation(t *testing.T, registry security.RegistryClient, invitationToken, deviceID2 string) {
	// Register another device
	err := registry.RegisterDevice(invitationToken, deviceID2)
	if err != nil {
		t.Fatalf("Failed to register second device: %v", err)
	}

	// Revoke all devices
	err = registry.RevokeInvitation(invitationToken)
	if err != nil {
		t.Fatalf("Failed to revoke invitation: %v", err)
	}

	// Validate after revocation
	valid, err := registry.ValidateDevice(invitationToken, deviceID2)
	if err != nil {
		t.Fatalf("Failed to validate after revocation: %v", err)
	}

	if valid {
		t.Errorf("Device should not be valid after revoking invitation")
	}
}
