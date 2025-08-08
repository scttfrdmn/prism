package profile

import (
	"time"
)

// ProfileType represents the type of a profile
type ProfileType string

const (
	// ProfileTypePersonal represents a personal AWS profile  
	ProfileTypePersonal ProfileType = "personal"
	
	// ProfileTypeInvitation represents an invitation-based profile
	ProfileTypeInvitation ProfileType = "invitation"
)

// Profile represents a CloudWorkstation profile configuration
type Profile struct {
	// Type is the profile type (personal or invitation)
	Type ProfileType `json:"type"`
	
	// Name is the display name for the profile
	Name string `json:"name"`
	
	// AWSProfile is the AWS CLI profile name to use
	AWSProfile string `json:"aws_profile"`
	
	// Region is the default AWS region for this profile
	Region string `json:"region"`
	
	// Default indicates if this is the default profile
	Default bool `json:"default"`
	
	// CreatedAt is when the profile was created
	CreatedAt time.Time `json:"created_at"`
	
	// UpdatedAt is when the profile was last updated
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	
	// ExpiresAt is when invitation profiles expire (optional)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	
	// InvitationID is the invitation ID for invitation profiles (optional)
	InvitationID string `json:"invitation_id,omitempty"`
	
	// OrganizationID is the organization ID for invitation profiles (optional)
	OrganizationID string `json:"organization_id,omitempty"`
	
	// LastUsed is when the profile was last used
	LastUsed *time.Time `json:"last_used,omitempty"`
	
	// InvitationToken for invitation profiles
	InvitationToken string `json:"invitation_token,omitempty"`
	
	// OwnerAccount for invitation profiles
	OwnerAccount string `json:"owner_account,omitempty"`
	
	// S3ConfigPath for invitation profiles
	S3ConfigPath string `json:"s3_config_path,omitempty"`
	
	// DeviceBound indicates if the profile is bound to a device
	DeviceBound bool `json:"device_bound,omitempty"`
	
	// Transferable indicates if the profile can be transferred
	Transferable bool `json:"transferable,omitempty"`
	
	// BindingRef is a reference to device binding information
	BindingRef string `json:"binding_ref,omitempty"`
	
	// SSH key configuration
	SSHKeyName    string `json:"ssh_key_name,omitempty"`     // AWS key pair name
	SSHKeyPath    string `json:"ssh_key_path,omitempty"`     // Local private key path
	UseDefaultKey bool   `json:"use_default_key,omitempty"`  // Use default SSH key (~/.ssh/id_rsa)
}

// Profiles represents the collection of all profiles
type Profiles struct {
	// Profiles is a map of profile ID to profile
	Profiles map[string]Profile `json:"profiles"`
	
	// CurrentProfile is the ID of the currently active profile
	CurrentProfile string `json:"current_profile"`
	
	// Version is the profiles file format version
	Version int `json:"version"`
}