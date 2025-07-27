package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/state"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// MigrationResult provides information about the migration process
type MigrationResult struct {
	// Success indicates if the migration was successful
	Success bool
	
	// ProfileID is the profile ID created during migration
	ProfileID string
	
	// ProfileName is the display name of the created profile
	ProfileName string
	
	// InstanceCount is the number of instances migrated
	InstanceCount int
	
	// VolumeCount is the number of EFS volumes migrated
	VolumeCount int
	
	// StorageCount is the number of EBS volumes migrated
	StorageCount int
	
	// BackupPath is the path to the backup of the original state file
	BackupPath string
}

// MigrateFromLegacyState migrates data from the legacy state file to the profile-based structure
func (m *ManagerEnhanced) MigrateFromLegacyState(profileName string) (*MigrationResult, error) {
	// Check if legacy state exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	// Legacy state path
	legacyStatePath := filepath.Join(homeDir, ".cloudworkstation", "state.json")
	
	// Check if legacy state file exists
	if _, err := os.Stat(legacyStatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no legacy state file found to migrate")
	}
	
	// Read legacy state
	data, err := os.ReadFile(legacyStatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read legacy state file: %w", err)
	}
	
	// Parse legacy state
	var legacyState types.State
	if err := json.Unmarshal(data, &legacyState); err != nil {
		return nil, fmt.Errorf("failed to parse legacy state: %w", err)
	}
	
	// Use provided profile name or default
	if profileName == "" {
		profileName = "Migrated Data"
	}
	
	// Create a profile for the migrated data
	migratedProfile := Profile{
		Type:       ProfileTypePersonal, // Default to personal
		Name:       profileName,
		AWSProfile: "default",           // Default to "default" AWS profile
		Region:     legacyState.Config.DefaultRegion,
		CreatedAt:  time.Now(),
	}
	
	// Add the profile
	if err := m.AddProfile(migratedProfile); err != nil {
		return nil, fmt.Errorf("failed to create profile for migrated data: %w", err)
	}
	
	// Get the created profile ID
	profile, err := m.GetProfile(migratedProfile.AWSProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to get created profile: %w", err)
	}
	
	// Create state manager
	stateManager, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}
	
	// Save state for this profile
	if err := stateManager.SaveState(&legacyState); err != nil {
		return nil, fmt.Errorf("failed to save migrated state: %w", err)
	}
	
	// Create backup of the original state file
	backupPath := legacyStatePath + ".backup"
	if err := os.Rename(legacyStatePath, backupPath); err != nil {
		return nil, fmt.Errorf("failed to create backup of legacy state: %w", err)
	}
	
	// Create result
	result := &MigrationResult{
		Success:      true,
		ProfileID:    profile.AWSProfile,
		ProfileName:  profile.Name,
		InstanceCount: len(legacyState.Instances),
		VolumeCount:  len(legacyState.Volumes),
		StorageCount: len(legacyState.EBSVolumes),
		BackupPath:   backupPath,
	}
	
	return result, nil
}

// CreateProfileFromConfig creates a new profile based on AWS config and credentials
func CreateProfileFromConfig(profileManager *ManagerEnhanced, awsProfile string, region string, name string) (*Profile, error) {
	// Use provided name or default based on AWS profile
	if name == "" {
		name = fmt.Sprintf("AWS Profile: %s", awsProfile)
	}
	
	// Create profile
	profile := Profile{
		Type:       ProfileTypePersonal,
		Name:       name,
		AWSProfile: awsProfile,
		Region:     region,
		CreatedAt:  currentTime(),
	}
	
	// Add profile
	if err := profileManager.AddProfile(profile); err != nil {
		return nil, fmt.Errorf("failed to create profile from config: %w", err)
	}
	
	// Get the created profile
	createdProfile, err := profileManager.GetProfile(profile.AWSProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to get created profile: %w", err)
	}
	
	return createdProfile, nil
}

// currentTime returns the current time, but is a separate function
// to allow for testing with a mock time
var currentTime = func() time.Time {
	return time.Now()
}