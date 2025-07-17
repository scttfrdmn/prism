package security

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// DeviceBinding represents the binding between a profile and a device
type DeviceBinding struct {
	DeviceID        string    `json:"device_id"`
	ProfileID       string    `json:"profile_id"`
	InvitationToken string    `json:"invitation_token,omitempty"`
	Created         time.Time `json:"created"`
	LastValidated   time.Time `json:"last_validated"`
	DeviceName      string    `json:"device_name"`
	MacAddresses    []string  `json:"mac_addresses,omitempty"`
	UserName        string    `json:"user_name"`
}

// CreateDeviceBinding generates a new device binding
func CreateDeviceBinding(profileID, invitationToken string) (*DeviceBinding, error) {
	// Generate device ID
	deviceID, err := generateDeviceID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device ID: %w", err)
	}
	
	// Get device name
	deviceName, _ := os.Hostname()
	
	// Get MAC addresses (best effort)
	macAddresses := getMacAddresses()
	
	// Get username
	userName := getUserName()
	
	binding := &DeviceBinding{
		DeviceID:        deviceID,
		ProfileID:       profileID,
		InvitationToken: invitationToken,
		Created:         time.Now(),
		LastValidated:   time.Now(),
		DeviceName:      deviceName,
		MacAddresses:    macAddresses,
		UserName:        userName,
	}
	
	return binding, nil
}

// StoreDeviceBinding stores a device binding in the keychain
func StoreDeviceBinding(binding *DeviceBinding, profileName string) (string, error) {
	// Generate keychain key
	bindingRef := fmt.Sprintf("com.cloudworkstation.profile.%s", profileName)
	
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

// ValidateDeviceBinding validates if a device binding is valid
func ValidateDeviceBinding(bindingRef string) (bool, error) {
	// Retrieve binding
	binding, err := RetrieveDeviceBinding(bindingRef)
	if err != nil {
		return false, err
	}
	
	// Check device identity (additional validation)
	// This is a lightweight check to detect if the binding has been copied
	// to another device, but it's not meant to be foolproof
	
	// Check hostname
	currentHostname, _ := os.Hostname()
	if binding.DeviceName != "" && binding.DeviceName != currentHostname {
		// Hostname has changed - this is suspicious but not conclusive
		// We'll just log it for now but still allow access
		fmt.Fprintf(os.Stderr, "Warning: Device hostname mismatch for profile %s\n", binding.ProfileID)
	}
	
	// Check user
	currentUser := getUserName()
	if binding.UserName != "" && binding.UserName != currentUser {
		// Username has changed - this is suspicious but not conclusive
		// We'll just log it for now but still allow access
		fmt.Fprintf(os.Stderr, "Warning: Username mismatch for profile %s\n", binding.ProfileID)
	}
	
	// Update last validated timestamp
	if err := UpdateLastValidated(bindingRef); err != nil {
		// Non-fatal error, just log it
		fmt.Fprintf(os.Stderr, "Warning: Failed to update validation timestamp: %v\n", err)
	}
	
	return true, nil
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

// getMacAddresses gets the MAC addresses of network interfaces
func getMacAddresses() []string {
	var addresses []string
	
	interfaces, err := net.Interfaces()
	if err != nil {
		return addresses
	}
	
	for _, iface := range interfaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		
		// Skip interfaces without a hardware address
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		
		// Add the MAC address
		addresses = append(addresses, iface.HardwareAddr.String())
	}
	
	return addresses
}

// getUserName gets the current username
func getUserName() string {
	user, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	
	// Extract username from home directory path
	parts := strings.Split(user, string(os.PathSeparator))
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	return ""
}