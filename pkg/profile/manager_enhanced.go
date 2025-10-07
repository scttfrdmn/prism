package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/security"
)

// ContextKey is used to store profile information in context
type contextKey string

// ProfileContextKey is the key for storing profile information in context
const ProfileContextKey contextKey = "profile"

// ErrProfileNotFound indicates that the requested profile doesn't exist
var ErrProfileNotFound = errors.New("profile not found")

// ErrProfileExpired indicates that an invitation profile has expired
var ErrProfileExpired = errors.New("profile invitation has expired")

// ManagerEnhanced extends the Manager with additional functionality for v0.4.2
type ManagerEnhanced struct {
	configPath         string
	profiles           *Profiles
	credentialProvider CredentialProvider
}

// NewManagerEnhanced creates a new enhanced profile manager
func NewManagerEnhanced() (*ManagerEnhanced, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create CloudWorkstation directory if it doesn't exist
	cwsDir := filepath.Join(homeDir, ".cloudworkstation")
	if err := os.MkdirAll(cwsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(cwsDir, "profiles.json")

	// Skip credential provider to avoid keychain prompts for basic profile usage
	// Users with AWS profiles in ~/.aws/credentials don't need secure storage
	var credProvider CredentialProvider = nil

	manager := &ManagerEnhanced{
		configPath:         configPath,
		credentialProvider: credProvider,
	}

	// Load or create profiles
	if err := manager.load(); err != nil {
		// If file doesn't exist, create a useful default profile that uses AWS default profile
		if os.IsNotExist(err) {
			manager.profiles = &Profiles{
				Profiles:       make(map[string]Profile),
				CurrentProfile: "default", // Use default profile
			}

			// Create a default profile that maps to user's default AWS configuration
			// This means most users need zero configuration - it just works
			defaultProfile := Profile{
				Type:       ProfileTypePersonal,
				Name:       "AWS Default",
				AWSProfile: "default", // Maps to default AWS profile in ~/.aws/credentials
				Region:     "",        // Use AWS SDK default (from config or environment)
				Default:    true,
				CreatedAt:  time.Now(),
			}

			manager.profiles.Profiles["default"] = defaultProfile

			if err := manager.save(); err != nil {
				return nil, fmt.Errorf("failed to create initial profiles: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load profiles: %w", err)
		}
	}

	return manager, nil
}

// load reads profiles from disk
func (m *ManagerEnhanced) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var profiles Profiles
	if err := json.Unmarshal(data, &profiles); err != nil {
		return fmt.Errorf("invalid profile data: %w", err)
	}

	m.profiles = &profiles
	return nil
}

// save writes profiles to disk
func (m *ManagerEnhanced) save() error {
	data, err := json.MarshalIndent(m.profiles, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// GetCurrentProfile returns the currently active profile
func (m *ManagerEnhanced) GetCurrentProfile() (*Profile, error) {
	profile, exists := m.profiles.Profiles[m.profiles.CurrentProfile]
	if !exists {
		return nil, ErrProfileNotFound
	}

	// Check for expired invitation
	if profile.Type == ProfileTypeInvitation && profile.ExpiresAt != nil {
		if time.Now().After(*profile.ExpiresAt) {
			return nil, fmt.Errorf("invitation profile has expired (expired at: %s)", profile.ExpiresAt.Format(time.RFC3339))
		}
	}

	return &profile, nil
}

// GetCurrentProfileID returns the ID of the currently active profile
func (m *ManagerEnhanced) GetCurrentProfileID() (string, error) {
	if m.profiles.CurrentProfile == "" {
		return "", ErrProfileNotFound
	}
	if _, exists := m.profiles.Profiles[m.profiles.CurrentProfile]; !exists {
		return "", ErrProfileNotFound
	}
	return m.profiles.CurrentProfile, nil
}

// GetProfile returns a specific profile by ID
func (m *ManagerEnhanced) GetProfile(id string) (*Profile, error) {
	profile, exists := m.profiles.Profiles[id]
	if !exists {
		return nil, ErrProfileNotFound
	}

	return &profile, nil
}

// ProfileWithID represents a profile with its ID
type ProfileWithID struct {
	ID      string  `json:"id"`
	Profile Profile `json:"profile"`
}

// ListProfiles returns all available profiles
func (m *ManagerEnhanced) ListProfiles() ([]Profile, error) {
	result := make([]Profile, 0, len(m.profiles.Profiles))
	for _, profile := range m.profiles.Profiles {
		result = append(result, profile)
	}
	return result, nil
}

// ListProfilesWithIDs returns all available profiles with their IDs
func (m *ManagerEnhanced) ListProfilesWithIDs() ([]ProfileWithID, error) {
	result := make([]ProfileWithID, 0, len(m.profiles.Profiles))
	for id, profile := range m.profiles.Profiles {
		result = append(result, ProfileWithID{
			ID:      id,
			Profile: profile,
		})
	}
	return result, nil
}

// SwitchProfile activates a profile by ID
func (m *ManagerEnhanced) SwitchProfile(id string) error {
	profile, exists := m.profiles.Profiles[id]
	if !exists {
		return ErrProfileNotFound
	}

	// Check for expired invitation
	if profile.Type == ProfileTypeInvitation && profile.ExpiresAt != nil {
		if time.Now().After(*profile.ExpiresAt) {
			return fmt.Errorf("invitation profile has expired (expired at: %s)", profile.ExpiresAt.Format(time.RFC3339))
		}
	}

	// SECURITY: Enforce device binding validation ONLY for profiles that explicitly have device binding enabled
	// Skip keychain access entirely for basic profiles (those without DeviceBound=true and BindingRef)
	if profile.DeviceBound && profile.BindingRef != "" {
		valid, err := security.ValidateDeviceBinding(profile.BindingRef)
		if err != nil {
			return fmt.Errorf("device binding validation failed: %w", err)
		}
		if !valid {
			return fmt.Errorf("profile '%s' is not authorized for use on this device - device binding violation detected", profile.Name)
		}
	}
	// For basic profiles (DeviceBound=false or BindingRef=""), skip security validation entirely

	// Update last used timestamp
	now := time.Now()
	profile.LastUsed = &now
	m.profiles.Profiles[id] = profile

	// Set as current
	m.profiles.CurrentProfile = id

	return m.save()
}

// AddProfile creates a new profile
func (m *ManagerEnhanced) AddProfile(profile Profile) error {
	// Validate profile
	if profile.Name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if profile.Type != ProfileTypePersonal && profile.Type != ProfileTypeInvitation {
		return fmt.Errorf("invalid profile type: %s", profile.Type)
	}

	// Generate a unique ID based on profile name (not AWS profile)
	id := createProfileID(profile.Name)

	// Check for duplicate profile name
	if _, exists := m.profiles.Profiles[id]; exists {
		return fmt.Errorf("CloudWorkstation profile named '%s' already exists. Choose a different name or use 'cws profiles list' to see existing profiles", profile.Name)
	}

	// Initialize created time
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}

	// Add profile
	m.profiles.Profiles[id] = profile

	// If this is the first profile, make it default
	if len(m.profiles.Profiles) == 1 {
		m.profiles.CurrentProfile = id
	}

	return m.save()
}

// UpdateProfile modifies an existing profile
func (m *ManagerEnhanced) UpdateProfile(id string, updates Profile) error {
	profile, exists := m.profiles.Profiles[id]
	if !exists {
		return ErrProfileNotFound
	}

	// Apply updates, maintaining created time and type
	createdAt := profile.CreatedAt
	profileType := profile.Type
	profile = updates
	profile.CreatedAt = createdAt
	profile.Type = profileType

	m.profiles.Profiles[id] = profile
	return m.save()
}

// RemoveProfile deletes a profile and its associated credentials
func (m *ManagerEnhanced) RemoveProfile(id string) error {
	if _, exists := m.profiles.Profiles[id]; !exists {
		return ErrProfileNotFound
	}

	// Don't allow removing the current profile
	if id == m.profiles.CurrentProfile {
		return fmt.Errorf("cannot remove the active profile, switch to another profile first")
	}

	// Clear credentials if credential provider is available
	if m.credentialProvider != nil {
		_ = m.credentialProvider.ClearCredentials(id) // Best effort, don't fail if this fails
	}

	// Remove profile
	delete(m.profiles.Profiles, id)
	return m.save()
}

// ProfileExists checks if a profile exists
func (m *ManagerEnhanced) ProfileExists(id string) bool {
	_, exists := m.profiles.Profiles[id]
	return exists
}

// StoreProfileCredentials stores credentials for a profile
func (m *ManagerEnhanced) StoreProfileCredentials(profileID string, creds *Credentials) error {
	if _, exists := m.profiles.Profiles[profileID]; !exists {
		return ErrProfileNotFound
	}

	if m.credentialProvider == nil {
		return fmt.Errorf("credential storage not available - use AWS CLI credentials instead")
	}

	return m.credentialProvider.StoreCredentials(profileID, creds)
}

// GetProfileCredentials retrieves credentials for a profile
func (m *ManagerEnhanced) GetProfileCredentials(profileID string) (*Credentials, error) {
	if _, exists := m.profiles.Profiles[profileID]; !exists {
		return nil, ErrProfileNotFound
	}

	if m.credentialProvider == nil {
		return nil, fmt.Errorf("credential storage not available - use AWS CLI credentials instead")
	}

	return m.credentialProvider.GetCredentials(profileID)
}

// WithProfile returns a new context with profile information
func (m *ManagerEnhanced) WithProfile(ctx context.Context, profileID string) (context.Context, error) {
	profile, err := m.GetProfile(profileID)
	if err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, ProfileContextKey, profile), nil
}

// GetProfileFromContext extracts the profile from a context
func GetProfileFromContext(ctx context.Context) (*Profile, bool) {
	profile, ok := ctx.Value(ProfileContextKey).(*Profile)
	return profile, ok
}

// Helper function to create a valid profile ID from a name
func createProfileID(name string) string {
	// Use the profile name directly as the ID for simplicity
	// This allows retrieval by name as expected by the tests
	return name
}
