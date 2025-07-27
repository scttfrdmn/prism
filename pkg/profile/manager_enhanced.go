package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
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
	
	// Create credential provider
	credProvider, err := NewCredentialProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create credential provider: %w", err)
	}
	
	manager := &ManagerEnhanced{
		configPath:         configPath,
		credentialProvider: credProvider,
	}
	
	// Load or create profiles
	if err := manager.load(); err != nil {
		// If file doesn't exist, create default profile
		if os.IsNotExist(err) {
			manager.profiles = &Profiles{
				Profiles:       make(map[string]Profile),
				CurrentProfile: "personal",
			}
			
			// Create default personal profile
			defaultProfile := Profile{
				Type:       ProfileTypePersonal,
				Name:       "My Account",
				AWSProfile: "default",
				Region:     "",  // Use AWS SDK default
				Default:    true,
				CreatedAt:  time.Now(),
			}
			
			manager.profiles.Profiles["personal"] = defaultProfile
			
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
	if profile.Type == ProfileTypeInvitation {
		// TODO: Check expiration from credentials
	}
	
	return &profile, nil
}

// GetProfile returns a specific profile by ID
func (m *ManagerEnhanced) GetProfile(id string) (*Profile, error) {
	profile, exists := m.profiles.Profiles[id]
	if !exists {
		return nil, ErrProfileNotFound
	}
	
	return &profile, nil
}

// ListProfiles returns all available profiles
func (m *ManagerEnhanced) ListProfiles() ([]Profile, error) {
	result := make([]Profile, 0, len(m.profiles.Profiles))
	for _, profile := range m.profiles.Profiles {
		result = append(result, profile)
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
	if profile.Type == ProfileTypeInvitation {
		// TODO: Check expiration from credentials
	}
	
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
	
	// Generate a unique ID based on name if not provided
	id := profile.AWSProfile
	if id == "" {
		id = createProfileID(profile.Name)
	}
	
	// Check for duplicate
	if _, exists := m.profiles.Profiles[id]; exists {
		return fmt.Errorf("profile '%s' already exists", id)
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
	
	// Clear credentials
	_ = m.credentialProvider.ClearCredentials(id) // Best effort, don't fail if this fails
	
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
	
	return m.credentialProvider.StoreCredentials(profileID, creds)
}

// GetProfileCredentials retrieves credentials for a profile
func (m *ManagerEnhanced) GetProfileCredentials(profileID string) (*Credentials, error) {
	if _, exists := m.profiles.Profiles[profileID]; !exists {
		return nil, ErrProfileNotFound
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
	// This would have more logic in a real implementation to create
	// a valid identifier from the name (lowercase, replace spaces, etc.)
	return "profile-" + fmt.Sprint(time.Now().UnixNano())
}