package core

import (
	"fmt"
	"time"
)

// CompatibilityManager provides backward compatibility with the legacy profile system
type CompatibilityManager struct {
	coreManager *Manager
}

// NewCompatibilityManager creates a compatibility manager wrapping the core manager
func NewCompatibilityManager() (*CompatibilityManager, error) {
	core, err := NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create core manager: %w", err)
	}

	return &CompatibilityManager{
		coreManager: core,
	}, nil
}

// LegacyProfile represents the legacy profile format for compatibility
// This avoids import cycles while providing conversion capabilities
type LegacyProfile struct {
	Type            string     `json:"type"`
	Name            string     `json:"name"`
	AWSProfile      string     `json:"aws_profile"`
	Region          string     `json:"region"`
	Default         bool       `json:"default"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at,omitempty"`
	LastUsed        *time.Time `json:"last_used,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	InvitationID    string     `json:"invitation_id,omitempty"`
	OrganizationID  string     `json:"organization_id,omitempty"`
	InvitationToken string     `json:"invitation_token,omitempty"`
	OwnerAccount    string     `json:"owner_account,omitempty"`
	S3ConfigPath    string     `json:"s3_config_path,omitempty"`
	DeviceBound     bool       `json:"device_bound,omitempty"`
	Transferable    bool       `json:"transferable,omitempty"`
	BindingRef      string     `json:"binding_ref,omitempty"`
}

// LegacyProfiles represents the legacy profiles collection format
type LegacyProfiles struct {
	Profiles       map[string]LegacyProfile `json:"profiles"`
	CurrentProfile string                   `json:"current_profile"`
	Version        int                      `json:"version"`
}

// ConvertToLegacyProfile converts a core Profile to legacy format
func (cm *CompatibilityManager) ConvertToLegacyProfile(profile *Profile) LegacyProfile {
	legacyProfile := LegacyProfile{
		Type:       "personal", // All simplified profiles are personal
		Name:       profile.Name,
		AWSProfile: profile.AWSProfile,
		Region:     profile.Region,
		Default:    profile.Default,
		CreatedAt:  profile.CreatedAt,
		LastUsed:   profile.LastUsed,

		// Legacy fields set to defaults
		UpdatedAt:       time.Now(),
		ExpiresAt:       nil, // No expiration for simplified profiles
		InvitationID:    "",
		OrganizationID:  "",
		InvitationToken: "",
		OwnerAccount:    "",
		S3ConfigPath:    "",
		DeviceBound:     false,
		Transferable:    false,
		BindingRef:      "",
	}

	return legacyProfile
}

// ConvertFromLegacyProfile converts a legacy Profile to core format
func (cm *CompatibilityManager) ConvertFromLegacyProfile(legacyProfile LegacyProfile) *Profile {
	profile := &Profile{
		Name:       legacyProfile.Name,
		AWSProfile: legacyProfile.AWSProfile,
		Region:     legacyProfile.Region,
		Default:    legacyProfile.Default,
		CreatedAt:  legacyProfile.CreatedAt,
		LastUsed:   legacyProfile.LastUsed,
	}

	return profile
}

// Legacy API Compatibility Methods
// These methods maintain the interface that existing code expects

// ListProfiles returns profiles in legacy format for backward compatibility
func (cm *CompatibilityManager) ListProfiles() ([]LegacyProfile, error) {
	coreProfiles := cm.coreManager.List()
	legacyProfiles := make([]LegacyProfile, len(coreProfiles))

	for i, profile := range coreProfiles {
		legacyProfiles[i] = cm.ConvertToLegacyProfile(profile)
	}

	return legacyProfiles, nil
}

// GetCurrentProfile returns current profile in legacy format
func (cm *CompatibilityManager) GetCurrentProfile() (*LegacyProfile, error) {
	coreProfile, err := cm.coreManager.GetCurrent()
	if err != nil {
		// Convert core errors to legacy errors
		switch err.(type) {
		case *NoCurrentProfileError:
			return nil, fmt.Errorf("no current profile set")
		default:
			return nil, err
		}
	}

	legacyProfile := cm.ConvertToLegacyProfile(coreProfile)
	return &legacyProfile, nil
}

// GetProfile returns a profile by name in legacy format
func (cm *CompatibilityManager) GetProfile(name string) (*LegacyProfile, error) {
	coreProfile, err := cm.coreManager.Get(name)
	if err != nil {
		// Convert core errors to legacy errors
		switch err.(type) {
		case *ProfileNotFoundError:
			return nil, fmt.Errorf("profile not found: %s", name)
		default:
			return nil, err
		}
	}

	legacyProfile := cm.ConvertToLegacyProfile(coreProfile)
	return &legacyProfile, nil
}

// SetCurrentProfile sets the current profile (legacy API)
func (cm *CompatibilityManager) SetCurrentProfile(name string) error {
	return cm.coreManager.SetCurrent(name)
}

// CreateProfile creates a new profile (legacy API)
func (cm *CompatibilityManager) CreateProfile(legacyProfile LegacyProfile) error {
	coreProfile := cm.ConvertFromLegacyProfile(legacyProfile)
	return cm.coreManager.Set(legacyProfile.Name, coreProfile)
}

// DeleteProfile deletes a profile (legacy API)
func (cm *CompatibilityManager) DeleteProfile(name string) error {
	return cm.coreManager.Delete(name)
}

// Legacy state management compatibility

// ConvertToLegacyProfiles converts core config to legacy Profiles format
func (cm *CompatibilityManager) ConvertToLegacyProfiles() (*LegacyProfiles, error) {
	coreProfiles := cm.coreManager.List()
	currentName := cm.coreManager.GetCurrentName()

	legacyProfileMap := make(map[string]LegacyProfile)
	for _, coreProfile := range coreProfiles {
		legacyProfile := cm.ConvertToLegacyProfile(coreProfile)
		legacyProfileMap[coreProfile.Name] = legacyProfile
	}

	return &LegacyProfiles{
		Profiles:       legacyProfileMap,
		CurrentProfile: currentName,
		Version:        1, // Legacy version
	}, nil
}

// MigrateFromLegacy migrates from the complex legacy system to simplified core
func (cm *CompatibilityManager) MigrateFromLegacy(legacyManager interface{}) error {
	// This would implement migration from ManagerEnhanced to core Manager
	// For now, we'll implement this when we have the actual legacy manager interface
	return fmt.Errorf("legacy migration not yet implemented")
}

// GetCoreManager returns the underlying core manager for direct access
func (cm *CompatibilityManager) GetCoreManager() *Manager {
	return cm.coreManager
}

// Utility functions for existing code

// CreateDefaultProfileIfNeeded creates a default profile if none exist
func (cm *CompatibilityManager) CreateDefaultProfileIfNeeded(awsProfile, region string) error {
	return cm.coreManager.CreateDefault(awsProfile, region)
}

// ValidateProfile validates a profile (legacy API)
func (cm *CompatibilityManager) ValidateProfile(legacyProfile LegacyProfile) error {
	coreProfile := cm.ConvertFromLegacyProfile(legacyProfile)
	return cm.coreManager.validateProfile(coreProfile)
}
