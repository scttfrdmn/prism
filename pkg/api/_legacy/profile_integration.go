package api

import (
	"context"
	"fmt"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// ProfileAwareAPI extends the CloudWorkstationAPI with profile management capabilities
type ProfileAwareAPI interface {
	CloudWorkstationAPI
	
	// WithProfile returns a new CloudWorkstationAPI client for a specific profile
	WithProfile(profileID string) (CloudWorkstationAPI, error)
	
	// Current profile operations
	CurrentProfile() string
	SwitchProfile(profileID string) error
	
	// Profile management
	ListProfiles() ([]profile.Profile, error)
	GetProfile(profileID string) (*profile.Profile, error)
	AddProfile(prof profile.Profile) error
	UpdateProfile(profileID string, updates profile.Profile) error
	RemoveProfile(profileID string) error
	StoreProfileCredentials(profileID string, creds *profile.Credentials) error
}

// ProfileAwareClient provides a client that integrates with the profile management system
type ProfileAwareClient struct {
	baseClient      *Client
	contextClient   *ContextClient
	profileManager  *profile.ManagerEnhanced
	stateManager    *profile.ProfileAwareStateManager
	currentProfile  string
}

// NewProfileAwareClient creates a new profile-aware client
func NewProfileAwareClient(baseURL string, profileManager *profile.ManagerEnhanced, stateManager *profile.ProfileAwareStateManager) (*ProfileAwareClient, error) {
	// Create base client
	baseClient := NewClient(baseURL)
	
	// Create context client
	contextClient := &ContextClient{
		client: baseClient,
	}
	
	// Create profile-aware client
	client := &ProfileAwareClient{
		baseClient:     baseClient,
		contextClient:  contextClient,
		profileManager: profileManager,
		stateManager:   stateManager,
	}
	
	// Set current profile
	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		// Use default profile ID if no current profile
		client.currentProfile = "personal"
	} else {
		client.currentProfile = currentProfile.AWSProfile
		
		// Configure client with current profile
		if currentProfile.Type == profile.ProfileTypePersonal {
			baseClient.SetAWSProfile(currentProfile.AWSProfile)
			baseClient.SetAWSRegion(currentProfile.Region)
		} else if currentProfile.Type == profile.ProfileTypeInvitation {
			baseClient.SetInvitationToken(currentProfile.InvitationToken, currentProfile.OwnerAccount, currentProfile.S3ConfigPath)
			baseClient.SetAWSRegion(currentProfile.Region)
		}
		
		// Set profile ID
		baseClient.SetProfileID(currentProfile.AWSProfile)
	}
	
	return client, nil
}

// Client returns the underlying CloudWorkstationAPI client
func (c *ProfileAwareClient) Client() CloudWorkstationAPI {
	return c.contextClient
}

// WithProfile returns a new CloudWorkstationAPI client for the specified profile
func (c *ProfileAwareClient) WithProfile(profileID string) (CloudWorkstationAPI, error) {
	// Check if profile exists
	_, err := c.profileManager.GetProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	
	// Create new context client with profile
	client, err := c.contextClient.WithProfile(c.profileManager, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to create client with profile: %w", err)
	}
	
	return client, nil
}

// CurrentProfile returns the current profile ID
func (c *ProfileAwareClient) CurrentProfile() string {
	return c.currentProfile
}

// SwitchProfile switches to a different profile
func (c *ProfileAwareClient) SwitchProfile(profileID string) error {
	// Switch profile in profile manager
	if err := c.profileManager.SwitchProfile(profileID); err != nil {
		return fmt.Errorf("failed to switch profile: %w", err)
	}
	
	// Get the profile
	prof, err := c.profileManager.GetProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}
	
	// Update current profile
	c.currentProfile = profileID
	
	// Configure base client with new profile
	if prof.Type == profile.ProfileTypePersonal {
		c.baseClient.SetAWSProfile(prof.AWSProfile)
		c.baseClient.SetAWSRegion(prof.Region)
	} else if prof.Type == profile.ProfileTypeInvitation {
		c.baseClient.SetInvitationToken(prof.InvitationToken, prof.OwnerAccount, prof.S3ConfigPath)
		c.baseClient.SetAWSRegion(prof.Region)
	}
	
	// Set profile ID
	c.baseClient.SetProfileID(profileID)
	
	return nil
}

// ListProfiles returns all available profiles
func (c *ProfileAwareClient) ListProfiles() ([]profile.Profile, error) {
	return c.profileManager.ListProfiles()
}

// GetProfile returns a specific profile
func (c *ProfileAwareClient) GetProfile(profileID string) (*profile.Profile, error) {
	return c.profileManager.GetProfile(profileID)
}

// AddProfile adds a new profile
func (c *ProfileAwareClient) AddProfile(prof profile.Profile) error {
	return c.profileManager.AddProfile(prof)
}

// UpdateProfile updates an existing profile
func (c *ProfileAwareClient) UpdateProfile(profileID string, updates profile.Profile) error {
	return c.profileManager.UpdateProfile(profileID, updates)
}

// RemoveProfile removes a profile
func (c *ProfileAwareClient) RemoveProfile(profileID string) error {
	return c.profileManager.RemoveProfile(profileID)
}

// StoreProfileCredentials stores credentials for a profile
func (c *ProfileAwareClient) StoreProfileCredentials(profileID string, creds *profile.Credentials) error {
	return c.profileManager.StoreProfileCredentials(profileID, creds)
}

// WithProfileContext returns a context with profile information
func (c *ProfileAwareClient) WithProfileContext(ctx context.Context) (context.Context, error) {
	// Get current profile
	prof, err := c.profileManager.GetCurrentProfile()
	if err != nil {
		return ctx, fmt.Errorf("failed to get current profile: %w", err)
	}
	
	// Add profile to context
	return WithProfileContext(ctx, prof), nil
}