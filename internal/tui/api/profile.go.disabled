package api

import (
	"context"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// Profile-related types and methods

// ProfileResponse represents a profile in the API
type ProfileResponse struct {
	Type            string `json:"type"`
	Name            string `json:"name"`
	AWSProfile      string `json:"aws_profile"`
	Region          string `json:"region"`
	Default         bool   `json:"default,omitempty"`
	InvitationToken string `json:"invitation_token,omitempty"`
	OwnerAccount    string `json:"owner_account,omitempty"`
	S3ConfigPath    string `json:"s3_config_path,omitempty"`
	
	// Security attributes
	CanInvite       bool   `json:"can_invite,omitempty"`
	Transferable    bool   `json:"transferable,omitempty"`
	DeviceBound     bool   `json:"device_bound,omitempty"`
	BindingRef      string `json:"binding_ref,omitempty"`
	
	// Metadata
	LastUsed        string `json:"last_used,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// ListProfilesResponse represents the response for listing profiles
type ListProfilesResponse struct {
	Profiles []ProfileResponse `json:"profiles"`
}

// ToProfileResponse converts a profile to a response
func ToProfileResponse(p profile.Profile) ProfileResponse {
	lastUsed := ""
	if !p.LastUsed.IsZero() {
		lastUsed = p.LastUsed.Format("2006-01-02T15:04:05Z")
	}
	
	return ProfileResponse{
		Type:            string(p.Type),
		Name:            p.Name,
		AWSProfile:      p.AWSProfile,
		Region:          p.Region,
		Default:         p.Default,
		InvitationToken: p.InvitationToken,
		OwnerAccount:    p.OwnerAccount,
		S3ConfigPath:    p.S3ConfigPath,
		CanInvite:       p.CanInvite,
		Transferable:    p.Transferable,
		DeviceBound:     p.DeviceBound,
		BindingRef:      p.BindingRef,
		LastUsed:        lastUsed,
		CreatedAt:       p.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ToListProfilesResponse converts profiles to a response
func ToListProfilesResponse(profiles []profile.Profile) *ListProfilesResponse {
	resp := &ListProfilesResponse{
		Profiles: make([]ProfileResponse, 0, len(profiles)),
	}
	
	for _, p := range profiles {
		resp.Profiles = append(resp.Profiles, ToProfileResponse(p))
	}
	
	return resp
}

// Client interface extensions for profile management

// ProfileManager returns the enhanced profile manager
func (c *TUIClient) ProfileManager() *profile.ManagerEnhanced {
	// Check if we have a profile-aware client
	if paClient, ok := c.client.(*pkgapi.ProfileAwareClient); ok {
		return paClient.ProfileManager()
	}
	return nil
}

// ListProfiles returns all available profiles
func (c *TUIClient) ListProfiles(ctx context.Context) (*ListProfilesResponse, error) {
	pm := c.ProfileManager()
	if pm == nil {
		return &ListProfilesResponse{}, nil
	}
	
	profiles, err := pm.ListProfiles()
	if err != nil {
		return nil, err
	}
	
	return ToListProfilesResponse(profiles), nil
}

// GetCurrentProfile returns the current active profile
func (c *TUIClient) GetCurrentProfile(ctx context.Context) (*ProfileResponse, error) {
	pm := c.ProfileManager()
	if pm == nil {
		return nil, nil
	}
	
	profile, err := pm.GetCurrentProfile()
	if err != nil {
		return nil, err
	}
	
	resp := ToProfileResponse(profile)
	return &resp, nil
}

// SwitchProfile switches to the specified profile
func (c *TUIClient) SwitchProfile(ctx context.Context, profileID string) error {
	pm := c.ProfileManager()
	if pm == nil {
		return nil
	}
	
	return pm.SwitchProfile(profileID)
}

// Client is the interface for TUI API operations
type Client interface {
	// Profile methods
	ProfileManager() *profile.ManagerEnhanced
	ListProfiles(ctx context.Context) (*ListProfilesResponse, error)
	GetCurrentProfile(ctx context.Context) (*ProfileResponse, error)
	SwitchProfile(ctx context.Context, profileID string) error
	
	// Other methods would be added here
	// ListInstances, LaunchInstance, etc.
}