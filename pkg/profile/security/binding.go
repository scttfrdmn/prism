package security

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// DeviceBinding represents the binding between a profile and a device
type DeviceBinding struct {
	DeviceID          string             `json:"device_id"`
	ProfileID         string             `json:"profile_id"`
	InvitationToken   string             `json:"invitation_token,omitempty"`
	Created           time.Time          `json:"created"`
	LastValidated     time.Time          `json:"last_validated"`
	DeviceFingerprint *DeviceFingerprint `json:"device_fingerprint"`
	
	// Legacy fields (deprecated but kept for compatibility)
	DeviceName    string   `json:"device_name,omitempty"`
	MacAddresses  []string `json:"mac_addresses,omitempty"`
	UserName      string   `json:"user_name,omitempty"`
}

// CreateDeviceBinding generates a new device binding with robust fingerprinting
func CreateDeviceBinding(profileID, invitationToken string) (*DeviceBinding, error) {
	// Generate device ID
	deviceID, err := generateDeviceID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device ID: %w", err)
	}
	
	// Generate comprehensive device fingerprint
	fingerprint, err := GenerateDeviceFingerprint()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device fingerprint: %w", err)
	}
	
	binding := &DeviceBinding{
		DeviceID:          deviceID,
		ProfileID:         profileID,
		InvitationToken:   invitationToken,
		Created:           time.Now(),
		LastValidated:     time.Now(),
		DeviceFingerprint: fingerprint,
		
		// Legacy fields for backward compatibility
		DeviceName:   fingerprint.Hostname,
		MacAddresses: fingerprint.MACAddresses,
		UserName:     fingerprint.Username,
	}
	
	return binding, nil
}

// StoreDeviceBinding stores a device binding in the keychain
func StoreDeviceBinding(binding *DeviceBinding, profileName string) (string, error) {
	// IMPROVED UX: Use consistent service name to avoid multiple keychain prompts
	bindingRef := fmt.Sprintf("CloudWorkstation.profile.%s", profileName)
	
	// Convert to JSON
	data, err := json.Marshal(binding)
	if err != nil {
		return "", fmt.Errorf("failed to marshal binding: %w", err)
	}
	
	// Get keychain provider
	keychain, err := NewKeychainProvider()
	if err != nil {
		return "", fmt.Errorf("failed to create keychain provider: %w", err)
	}
	
	// Store in keychain
	if err := keychain.Store(bindingRef, data); err != nil {
		return "", fmt.Errorf("failed to store binding: %w", err)
	}
	
	return bindingRef, nil
}

// RetrieveDeviceBinding retrieves a device binding from the keychain
func RetrieveDeviceBinding(bindingRef string) (*DeviceBinding, error) {
	// Get keychain provider
	keychain, err := NewKeychainProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create keychain provider: %w", err)
	}
	
	// Check if binding exists
	if !keychain.Exists(bindingRef) {
		return nil, ErrKeychainNotFound
	}
	
	// Retrieve binding data
	data, err := keychain.Retrieve(bindingRef)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve binding: %w", err)
	}
	
	// Parse binding
	var binding DeviceBinding
	if err := json.Unmarshal(data, &binding); err != nil {
		return nil, fmt.Errorf("invalid binding data: %w", err)
	}
	
	return &binding, nil
}

// UpdateLastValidated updates the last validated timestamp of a binding
func UpdateLastValidated(bindingRef string) error {
	// Retrieve current binding
	binding, err := RetrieveDeviceBinding(bindingRef)
	if err != nil {
		return err
	}
	
	// Update timestamp
	binding.LastValidated = time.Now()
	
	// Convert to JSON
	data, err := json.Marshal(binding)
	if err != nil {
		return fmt.Errorf("failed to marshal binding: %w", err)
	}
	
	// Get keychain provider
	keychain, err := NewKeychainProvider()
	if err != nil {
		return fmt.Errorf("failed to create keychain provider: %w", err)
	}
	
	// Store updated binding
	return keychain.Store(bindingRef, data)
}

// ValidateDeviceBinding performs strict device binding validation
func ValidateDeviceBinding(bindingRef string) (bool, error) {
	// Retrieve stored binding
	binding, err := RetrieveDeviceBinding(bindingRef)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve device binding: %w", err)
	}
	
	// Generate current device fingerprint
	currentFingerprint, err := GenerateDeviceFingerprint()
	if err != nil {
		return false, fmt.Errorf("failed to generate current device fingerprint: %w", err)
	}
	
	// Strict validation: fingerprints must match
	if binding.DeviceFingerprint == nil {
		// Legacy binding without fingerprint - migrate or reject
		return false, fmt.Errorf("binding lacks device fingerprint - migration required")
	}
	
	if !binding.DeviceFingerprint.Matches(currentFingerprint) {
		// Device binding violation - BLOCK ACCESS
		riskLevel := binding.DeviceFingerprint.GetRiskLevel(currentFingerprint)
		
		return false, &DeviceBindingViolation{
			ProfileID:         binding.ProfileID,
			ExpectedDevice:    binding.DeviceFingerprint.String(),
			CurrentDevice:     currentFingerprint.String(),
			RiskLevel:         riskLevel,
			ViolationType:     determineViolationType(binding.DeviceFingerprint, currentFingerprint),
		}
	}
	
	// Validation successful - update timestamp
	if err := UpdateLastValidated(bindingRef); err != nil {
		// Non-fatal error, but log it
		fmt.Fprintf(os.Stderr, "Warning: Failed to update validation timestamp: %v\n", err)
	}
	
	return true, nil
}

// DeviceBindingViolation represents a device binding security violation
type DeviceBindingViolation struct {
	ProfileID      string
	ExpectedDevice string
	CurrentDevice  string
	RiskLevel      RiskLevel
	ViolationType  ViolationType
}

func (e *DeviceBindingViolation) Error() string {
	return fmt.Sprintf("device binding violation for profile %s: expected %s, got %s (risk: %s, type: %s)",
		e.ProfileID, e.ExpectedDevice, e.CurrentDevice, e.RiskLevel, e.ViolationType)
}

// ViolationType categorizes the type of binding violation
type ViolationType string

const (
	ViolationTypeHostname     ViolationType = "hostname_mismatch"
	ViolationTypeUser         ViolationType = "user_mismatch"
	ViolationTypeMAC          ViolationType = "mac_address_mismatch"
	ViolationTypeSystemID     ViolationType = "system_id_mismatch"
	ViolationTypeProfileCopy  ViolationType = "profile_copy_detected"
	ViolationTypeUnknown      ViolationType = "unknown"
)

func (v ViolationType) String() string {
	return string(v)
}

// determineViolationType analyzes fingerprint differences to categorize violation
func determineViolationType(expected, current *DeviceFingerprint) ViolationType {
	if expected.Hostname != current.Hostname {
		return ViolationTypeHostname
	}
	
	if expected.Username != current.Username || expected.UserID != current.UserID {
		return ViolationTypeUser
	}
	
	if !expected.HasMatchingMAC(current) {
		return ViolationTypeMAC
	}
	
	if (expected.SystemUUID != "" && current.SystemUUID != "" && expected.SystemUUID != current.SystemUUID) ||
	   (expected.MachineID != "" && current.MachineID != "" && expected.MachineID != current.MachineID) {
		return ViolationTypeSystemID
	}
	
	// Multiple differences suggest profile copying
	differences := 0
	if expected.Hostname != current.Hostname { differences++ }
	if expected.Username != current.Username { differences++ }
	if !expected.HasMatchingMAC(current) { differences++ }
	
	if differences >= 2 {
		return ViolationTypeProfileCopy
	}
	
	return ViolationTypeUnknown
}

// Helper functions

// generateDeviceID generates a unique device identifier
func generateDeviceID() (string, error) {
	// Generate random bytes
	idBytes := make([]byte, 16)
	_, err := rand.Read(idBytes)
	if err != nil {
		return "", err
	}
	
	// Encode as base64
	id := base64.RawURLEncoding.EncodeToString(idBytes)
	
	// Format with prefix
	return fmt.Sprintf("device-%s", id), nil
}

// Legacy helper functions removed - functionality moved to fingerprint.go