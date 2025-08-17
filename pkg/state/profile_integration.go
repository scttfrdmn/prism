package state

import (
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/core"
)

// CoreProfileProvider integrates with the simplified core profile system
type CoreProfileProvider struct {
	profileManager *core.Manager
}

// NewCoreProfileProvider creates a profile provider using the simplified profile system
func NewCoreProfileProvider() (*CoreProfileProvider, error) {
	manager, err := core.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create profile manager: %w", err)
	}

	return &CoreProfileProvider{
		profileManager: manager,
	}, nil
}

// GetCurrentProfile returns the current profile's AWS profile name
func (cpp *CoreProfileProvider) GetCurrentProfile() (string, error) {
	profile, err := cpp.profileManager.GetCurrent()
	if err != nil {
		// Convert core errors to simpler messages
		switch err.(type) {
		case *core.NoCurrentProfileError:
			return "", fmt.Errorf("no current profile set")
		default:
			return "", err
		}
	}

	return profile.AWSProfile, nil
}

// GetProfileManager returns the underlying profile manager for advanced operations
func (cpp *CoreProfileProvider) GetProfileManager() *core.Manager {
	return cpp.profileManager
}

// Convenience function to create unified state manager with core profile integration
func NewUnifiedManagerWithCoreProfiles() (*UnifiedManager, error) {
	provider, err := NewCoreProfileProvider()
	if err != nil {
		return nil, err
	}

	return NewUnifiedManagerWithProfiles(provider)
}

// Legacy compatibility - this replaces ProfileAwareStateManager
func NewProfileAwareManager() (*UnifiedManager, error) {
	return NewUnifiedManagerWithCoreProfiles()
}
