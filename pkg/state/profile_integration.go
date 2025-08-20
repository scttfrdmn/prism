package state

import (
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// EnhancedProfileProvider integrates with the ManagerEnhanced profile system
type EnhancedProfileProvider struct {
	profileManager *profile.ManagerEnhanced
}

// NewEnhancedProfileProvider creates a profile provider using the ManagerEnhanced system
func NewEnhancedProfileProvider() (*EnhancedProfileProvider, error) {
	manager, err := profile.NewManagerEnhanced()
	if err != nil {
		return nil, fmt.Errorf("failed to create profile manager: %w", err)
	}

	return &EnhancedProfileProvider{
		profileManager: manager,
	}, nil
}

// GetCurrentProfile returns the current profile's AWS profile name
func (epp *EnhancedProfileProvider) GetCurrentProfile() (string, error) {
	profile, err := epp.profileManager.GetCurrentProfile()
	if err != nil {
		return "", fmt.Errorf("no current profile set: %w", err)
	}

	return profile.AWSProfile, nil
}

// GetProfileManager returns the underlying profile manager for advanced operations
func (epp *EnhancedProfileProvider) GetProfileManager() *profile.ManagerEnhanced {
	return epp.profileManager
}

// Convenience function to create unified state manager with enhanced profile integration
func NewUnifiedManagerWithEnhancedProfiles() (*UnifiedManager, error) {
	provider, err := NewEnhancedProfileProvider()
	if err != nil {
		return nil, err
	}

	return NewUnifiedManagerWithProfiles(provider)
}

// Legacy compatibility - this replaces ProfileAwareStateManager
func NewProfileAwareManager() (*UnifiedManager, error) {
	return NewUnifiedManagerWithEnhancedProfiles()
}
