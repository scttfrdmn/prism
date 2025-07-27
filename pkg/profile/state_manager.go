package profile

import (
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/state"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// ProfileAwareStateManager is a state manager that works with profiles
type ProfileAwareStateManager struct {
	profileManager *ManagerEnhanced
	baseStateManager *state.Manager
}

// NewProfileAwareStateManager creates a new profile-aware state manager
func NewProfileAwareStateManager(profileManager *ManagerEnhanced) (*ProfileAwareStateManager, error) {
	baseManager, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create base state manager: %w", err)
	}

	return &ProfileAwareStateManager{
		profileManager: profileManager,
		baseStateManager: baseManager,
	}, nil
}

// LoadState loads state for the current profile
func (psm *ProfileAwareStateManager) LoadState() (*types.State, error) {
	// Get current profile
	currentProfile, err := psm.profileManager.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}
	
	return psm.LoadStateForProfile(currentProfile.AWSProfile)
}

// LoadStateForProfile loads state for a specific profile
func (psm *ProfileAwareStateManager) LoadStateForProfile(profileID string) (*types.State, error) {
	// For now, use the base state manager
	// In the future, this could load profile-specific state files
	return psm.baseStateManager.LoadState()
}

// SaveState saves state for the current profile
func (psm *ProfileAwareStateManager) SaveState(state *types.State) error {
	// Get current profile
	currentProfile, err := psm.profileManager.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}
	
	return psm.SaveStateForProfile(currentProfile.AWSProfile, state)
}

// SaveStateForProfile saves state for a specific profile
func (psm *ProfileAwareStateManager) SaveStateForProfile(profileID string, state *types.State) error {
	// For now, use the base state manager
	// In the future, this could save to profile-specific state files
	return psm.baseStateManager.SaveState(state)
}

// SaveInstance saves an instance for the current profile
func (psm *ProfileAwareStateManager) SaveInstance(instance types.Instance) error {
	return psm.baseStateManager.SaveInstance(instance)
}

// RemoveInstance removes an instance for the current profile
func (psm *ProfileAwareStateManager) RemoveInstance(name string) error {
	return psm.baseStateManager.RemoveInstance(name)
}

// SaveVolume saves an EFS volume for the current profile
func (psm *ProfileAwareStateManager) SaveVolume(volume types.EFSVolume) error {
	return psm.baseStateManager.SaveVolume(volume)
}

// RemoveVolume removes an EFS volume for the current profile
func (psm *ProfileAwareStateManager) RemoveVolume(name string) error {
	return psm.baseStateManager.RemoveVolume(name)
}

// SaveEBSVolume saves an EBS volume for the current profile
func (psm *ProfileAwareStateManager) SaveEBSVolume(volume types.EBSVolume) error {
	return psm.baseStateManager.SaveEBSVolume(volume)
}

// RemoveEBSVolume removes an EBS volume for the current profile
func (psm *ProfileAwareStateManager) RemoveEBSVolume(name string) error {
	return psm.baseStateManager.RemoveEBSVolume(name)
}

// UpdateConfig updates the configuration for the current profile
func (psm *ProfileAwareStateManager) UpdateConfig(config types.Config) error {
	return psm.baseStateManager.UpdateConfig(config)
}