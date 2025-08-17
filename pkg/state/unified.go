// Package state provides unified state management for CloudWorkstation.
//
// This file provides the unified state management that eliminates the unnecessary
// ProfileAwareStateManager wrapper while maintaining all functionality.
package state

import (
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// UnifiedManager provides state management with optional profile integration.
// This replaces both the base Manager and the ProfileAwareStateManager wrapper.
type UnifiedManager struct {
	*Manager // Embed the base manager for all core functionality

	// Profile integration (optional)
	profileProvider ProfileProvider
}

// ProfileProvider interface allows plugging in different profile systems
type ProfileProvider interface {
	GetCurrentProfile() (string, error) // Returns current profile ID
}

// NewUnifiedManager creates a unified state manager
func NewUnifiedManager() (*UnifiedManager, error) {
	baseManager, err := NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create base manager: %w", err)
	}

	return &UnifiedManager{
		Manager:         baseManager,
		profileProvider: nil, // No profile integration by default
	}, nil
}

// NewUnifiedManagerWithProfiles creates a unified manager with profile integration
func NewUnifiedManagerWithProfiles(provider ProfileProvider) (*UnifiedManager, error) {
	baseManager, err := NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create base manager: %w", err)
	}

	return &UnifiedManager{
		Manager:         baseManager,
		profileProvider: provider,
	}, nil
}

// GetCurrentProfile returns the current profile if a provider is set
func (um *UnifiedManager) GetCurrentProfile() (string, error) {
	if um.profileProvider == nil {
		return "", fmt.Errorf("no profile provider configured")
	}

	return um.profileProvider.GetCurrentProfile()
}

// LoadStateForProfile loads state with profile context (if available)
func (um *UnifiedManager) LoadStateForProfile() (*types.State, error) {
	// If no profile provider, just use base functionality
	if um.profileProvider == nil {
		return um.Manager.LoadState()
	}

	// With profile provider, we could extend this for profile-specific state
	// For now, use global state (matches current behavior)
	return um.Manager.LoadState()
}

// SaveStateForProfile saves state with profile context (if available)
func (um *UnifiedManager) SaveStateForProfile(state *types.State) error {
	// If no profile provider, just use base functionality
	if um.profileProvider == nil {
		return um.Manager.SaveState(state)
	}

	// With profile provider, we could extend this for profile-specific state
	// For now, use global state (matches current behavior)
	return um.Manager.SaveState(state)
}

// Convenience methods that work with or without profiles

// LoadState loads state (profile-aware if provider is set)
func (um *UnifiedManager) LoadState() (*types.State, error) {
	return um.LoadStateForProfile()
}

// SaveState saves state (profile-aware if provider is set)
func (um *UnifiedManager) SaveState(state *types.State) error {
	return um.SaveStateForProfile(state)
}

// All other methods are inherited from the embedded Manager
// This includes: SaveInstance, RemoveInstance, SaveVolume, RemoveVolume,
// SaveEBSVolume, RemoveEBSVolume, UpdateConfig, SaveAPIKey, GetAPIKey, ClearAPIKey

// StaticProfileProvider is a simple profile provider for testing
type StaticProfileProvider struct {
	profile string
}

// NewStaticProfileProvider creates a profile provider that returns a fixed profile
func NewStaticProfileProvider(profile string) *StaticProfileProvider {
	return &StaticProfileProvider{profile: profile}
}

// GetCurrentProfile returns the static profile
func (spp *StaticProfileProvider) GetCurrentProfile() (string, error) {
	if spp.profile == "" {
		return "", fmt.Errorf("no profile configured")
	}
	return spp.profile, nil
}

// Legacy Compatibility Functions
// These maintain compatibility with existing code

// GetDefaultManager returns a unified manager (replaces state.NewManager)
func GetDefaultManager() (*UnifiedManager, error) {
	return NewUnifiedManager()
}

// GetManagerWithProfiles returns a unified manager with profile support
func GetManagerWithProfiles(provider ProfileProvider) (*UnifiedManager, error) {
	return NewUnifiedManagerWithProfiles(provider)
}
