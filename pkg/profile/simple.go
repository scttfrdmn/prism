// Package profile provides a simplified profile management system.
//
// This file serves as the main entry point for the new simplified profile system,
// providing a clean API that gradually replaces the complex legacy system.
//
// Migration Strategy:
// 1. New code should use functions from this file
// 2. Legacy code will be gradually migrated
// 3. Complex features (invitations, batch processing) moved to separate packages
package profile

import (
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/core"
)

// Simple profile management API - these are the functions new code should use

// GetDefaultManager returns a simplified profile manager
// This replaces the complex ManagerEnhanced with clean, focused functionality
func GetDefaultManager() (*core.Manager, error) {
	return core.NewManager()
}

// GetCompatibilityManager returns a manager with legacy API compatibility
// Use this for gradual migration of existing code
func GetCompatibilityManager() (*core.CompatibilityManager, error) {
	return core.NewCompatibilityManager()
}

// Convenience functions for common operations

// ListProfiles returns all configured profiles using the simplified system
func ListProfiles() ([]*core.Profile, error) {
	manager, err := GetDefaultManager()
	if err != nil {
		return nil, err
	}

	return manager.List(), nil
}

// GetCurrentProfile returns the currently active profile
func GetCurrentProfile() (*core.Profile, error) {
	manager, err := GetDefaultManager()
	if err != nil {
		return nil, err
	}

	return manager.GetCurrent()
}

// SetCurrentProfile sets the active profile
func SetCurrentProfile(name string) error {
	manager, err := GetDefaultManager()
	if err != nil {
		return err
	}

	return manager.SetCurrent(name)
}

// CreateProfile creates a new profile with the simplified system
func CreateProfile(name, awsProfile, region string, makeDefault bool) error {
	manager, err := GetDefaultManager()
	if err != nil {
		return err
	}

	profile := &core.Profile{
		Name:       name,
		AWSProfile: awsProfile,
		Region:     region,
		Default:    makeDefault,
	}

	return manager.Set(name, profile)
}

// DeleteProfile removes a profile
func DeleteProfile(name string) error {
	manager, err := GetDefaultManager()
	if err != nil {
		return err
	}

	return manager.Delete(name)
}

// EnsureDefaultProfile creates a default profile if none exist
func EnsureDefaultProfile(awsProfile, region string) error {
	manager, err := GetDefaultManager()
	if err != nil {
		return err
	}

	return manager.CreateDefault(awsProfile, region)
}

// Migration utilities

// MigrateFromLegacy provides a migration path from the complex legacy system
func MigrateFromLegacy() error {
	// This function will help migrate from the complex ManagerEnhanced
	// to the simplified core.Manager

	// For now, return not implemented - we'll implement this as needed
	return fmt.Errorf("legacy migration not yet implemented - use GetCompatibilityManager() for gradual migration")
}

// GetProfileStats returns statistics about the profile system
func GetProfileStats() (map[string]interface{}, error) {
	manager, err := GetDefaultManager()
	if err != nil {
		return nil, err
	}

	return manager.Stats(), nil
}

// ValidateProfileConfig validates a profile configuration
func ValidateProfileConfig(name, awsProfile, region string) error {
	manager, err := GetDefaultManager()
	if err != nil {
		return err
	}

	profile := &core.Profile{
		Name:       name,
		AWSProfile: awsProfile,
		Region:     region,
	}

	// Attempt to set the profile (this will validate it)
	// We use a temp name to avoid conflicts
	tempName := "__validation_temp__"
	err = manager.Set(tempName, profile)
	if err != nil {
		return err
	}

	// Clean up the temp profile
	_ = manager.Delete(tempName)
	return nil
}

// Advanced utilities

// ExportProfiles exports all profiles for backup/migration
func ExportProfiles() (*core.ProfileConfig, error) {
	manager, err := GetDefaultManager()
	if err != nil {
		return nil, err
	}

	profiles := manager.List()
	currentName := manager.GetCurrentName()

	config := &core.ProfileConfig{
		Profiles: make(map[string]*core.Profile),
		Current:  currentName,
		Version:  1,
	}

	for _, profile := range profiles {
		config.Profiles[profile.Name] = profile
	}

	return config, nil
}

// ImportProfiles imports profiles from a configuration
func ImportProfiles(config *core.ProfileConfig) error {
	manager, err := GetDefaultManager()
	if err != nil {
		return err
	}

	// Import all profiles
	for name, profile := range config.Profiles {
		if err := manager.Set(name, profile); err != nil {
			return fmt.Errorf("failed to import profile %s: %w", name, err)
		}
	}

	// Set current profile if specified
	if config.Current != "" {
		if err := manager.SetCurrent(config.Current); err != nil {
			return fmt.Errorf("failed to set current profile %s: %w", config.Current, err)
		}
	}

	return nil
}
