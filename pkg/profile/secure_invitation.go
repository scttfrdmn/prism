// Package profile provides functionality for managing CloudWorkstation profiles
package profile

import (
	"fmt"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/security"
)

// SecureInvitationManager extends the standard invitation manager with security features
type SecureInvitationManager struct {
	*InvitationManager
	registry *security.RegistryClient
}

// NewSecureInvitationManager creates a new secure invitation manager
func NewSecureInvitationManager(profileManager *ManagerEnhanced) (*SecureInvitationManager, error) {
	// Create standard invitation manager
	baseManager, err := NewInvitationManager(profileManager)
	if err != nil {
		return nil, err
	}

	// Create registry client with default config
	registryConfig := security.S3RegistryConfig{
		BucketName: "cloudworkstation-invitations",
		Region:     "us-west-2", // Default region for registry
		Enabled:    true,        // Enable by default, will fall back to local if unavailable
	}

	registry, err := security.NewRegistryClient(registryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}

	return &SecureInvitationManager{
		InvitationManager: baseManager,
		registry:          registry,
	}, nil
}

// CreateSecureInvitation generates a new invitation with security features
func (m *SecureInvitationManager) CreateSecureInvitation(
	name string,
	invType InvitationType,
	validDays int,
	s3ConfigPath string,
	canInvite bool,
	transferable bool,
	deviceBound bool,
	maxDevices int,
	parentToken string,
) (*InvitationToken, error) {
	// Get current profile
	currentProfile, err := m.profileManager.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	// Validate permissions based on parent token
	if parentToken != "" {
		parentInv, err := m.GetInvitation(parentToken)
		if err != nil {
			return nil, fmt.Errorf("parent invitation not found: %w", err)
		}

		// Enforce permission inheritance
		if !parentInv.CanInvite {
			return nil, fmt.Errorf("parent invitation does not allow creating sub-invitations")
		}

		// Cannot grant permissions you don't have
		if !parentInv.Transferable && transferable {
			transferable = false
		}
		if parentInv.DeviceBound {
			deviceBound = true
		}
		if parentInv.MaxDevices < maxDevices {
			maxDevices = parentInv.MaxDevices
		}
		if invType == InvitationTypeAdmin && parentInv.Type != InvitationTypeAdmin {
			return nil, fmt.Errorf("cannot create admin invitation from non-admin parent")
		}
	}

	// Generate the invitation token
	invitation, err := GenerateSecureInvitationToken(
		currentProfile.AWSProfile,
		currentProfile.AWSProfile, // Using profile name as account ID for now
		name,
		invType,
		validDays,
		s3ConfigPath,
		canInvite,
		transferable,
		deviceBound,
		maxDevices,
		parentToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation: %w", err)
	}

	// Save the invitation
	if err := m.saveInvitation(invitation); err != nil {
		return nil, fmt.Errorf("failed to save invitation: %w", err)
	}

	return invitation, nil
}

// SecureAddToProfile accepts an invitation and creates a profile with security binding
func (m *SecureInvitationManager) SecureAddToProfile(encoded string, profileName string) error {
	// Decode the invitation
	invitation, err := DecodeFromString(encoded)
	if err != nil {
		return fmt.Errorf("invalid invitation: %w", err)
	}

	// Verify it's still valid
	if !invitation.IsValid() {
		return fmt.Errorf("invitation has expired")
	}

	// Create profile from invitation
	profile := Profile{
		Type:            ProfileTypeInvitation,
		Name:            profileName,
		AWSProfile:      profileName,
		InvitationToken: invitation.Token,
		OwnerAccount:    invitation.OwnerAccount,
		S3ConfigPath:    invitation.S3ConfigPath,
		Region:          "", // Use default region
		CreatedAt:       time.Now(),
		// Security properties
		DeviceBound:  invitation.DeviceBound,
		Transferable: invitation.Transferable,
	}

	// Create device binding if required
	if invitation.DeviceBound {
		binding, err := security.CreateDeviceBinding(profile.AWSProfile, invitation.Token)
		if err != nil {
			return fmt.Errorf("failed to create device binding: %w", err)
		}

		// Store binding in keychain
		bindingRef, err := security.StoreDeviceBinding(binding, profileName)
		if err != nil {
			return fmt.Errorf("failed to store device binding: %w", err)
		}

		// Set binding reference in profile
		profile.BindingRef = bindingRef

		// Register device with registry
		err = m.registry.RegisterDevice(invitation.Token, binding.DeviceID)
		if err != nil {
			// Non-fatal error, log but continue
			fmt.Printf("Warning: Failed to register device with registry: %v\n", err)
		}
	}

	// Add the profile
	if err := m.profileManager.AddProfile(profile); err != nil {
		return fmt.Errorf("failed to create profile from invitation: %w", err)
	}

	return nil
}

// ValidateSecureProfile validates that a secure profile can be used on this device
func (m *SecureInvitationManager) ValidateSecureProfile(profile *Profile) error {
	// Skip validation for non-device-bound profiles
	if !profile.DeviceBound || profile.BindingRef == "" {
		return nil
	}

	// Validate device binding
	valid, err := security.ValidateDeviceBinding(profile.BindingRef)
	if err != nil {
		return fmt.Errorf("failed to validate device binding: %w", err)
	}

	if !valid {
		return fmt.Errorf("profile is not authorized for use on this device")
	}

	// Check with registry if possible (non-blocking)
	binding, err := security.RetrieveDeviceBinding(profile.BindingRef)
	if err == nil && binding != nil {
		go func() {
			// Asynchronously validate with registry
			valid, _ := m.registry.ValidateDevice(binding.InvitationToken, binding.DeviceID)
			if !valid {
				fmt.Printf("Warning: Device validation failed for profile %s\n", profile.Name)
			}
		}()
	}

	return nil
}

// GetInvitationDevices gets the list of devices registered for an invitation
func (m *SecureInvitationManager) GetInvitationDevices(invitationToken string) ([]map[string]interface{}, error) {
	return m.registry.GetInvitationDevices(invitationToken)
}

// RevokeDevice revokes a specific device from using an invitation
func (m *SecureInvitationManager) RevokeDevice(invitationToken, deviceID string) error {
	return m.registry.RevokeDevice(invitationToken, deviceID)
}

// RevokeAllDevices revokes all devices from using an invitation
func (m *SecureInvitationManager) RevokeAllDevices(invitationToken string) error {
	return m.registry.RevokeInvitation(invitationToken)
}
