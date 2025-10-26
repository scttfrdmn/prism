package profile

import (
	"fmt"
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

// Profile represents a Prism profile configuration
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
	SSHKeyName    string `json:"ssh_key_name,omitempty"`    // AWS key pair name
	SSHKeyPath    string `json:"ssh_key_path,omitempty"`    // Local private key path
	UseDefaultKey bool   `json:"use_default_key,omitempty"` // Use default SSH key (~/.ssh/id_rsa)

	// Basic policy restrictions inherited from invitation (open source feature)
	PolicyRestrictions *BasicPolicyRestrictions `json:"policy_restrictions,omitempty"`
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

// Policy validation methods for BasicPolicyRestrictions

// IsTemplateAllowed checks if a template is allowed by policy restrictions
func (p *BasicPolicyRestrictions) IsTemplateAllowed(templateName string) bool {
	if p == nil {
		return true // No restrictions
	}

	// Check blacklist first
	for _, blacklisted := range p.TemplateBlacklist {
		if blacklisted == templateName {
			return false
		}
	}

	// If whitelist is specified, template must be in it
	if len(p.TemplateWhitelist) > 0 {
		for _, whitelisted := range p.TemplateWhitelist {
			if whitelisted == templateName {
				return true
			}
		}
		return false // Not in whitelist
	}

	return true // No restrictions or passed blacklist
}

// IsInstanceTypeAllowed checks if an instance type is allowed by policy restrictions
func (p *BasicPolicyRestrictions) IsInstanceTypeAllowed(instanceType string) bool {
	if p == nil {
		return true // No restrictions
	}

	// If max instance types specified, check against list
	if len(p.MaxInstanceTypes) > 0 {
		for _, allowedType := range p.MaxInstanceTypes {
			if allowedType == instanceType {
				return true
			}
		}
		return false // Not in allowed list
	}

	return true // No restrictions
}

// IsRegionAllowed checks if a region is allowed by policy restrictions
func (p *BasicPolicyRestrictions) IsRegionAllowed(region string) bool {
	if p == nil {
		return true // No restrictions
	}

	// Check forbidden regions
	for _, forbiddenRegion := range p.ForbiddenRegions {
		if forbiddenRegion == region {
			return false
		}
	}

	return true // Not in forbidden list
}

// IsCostAllowed checks if hourly cost is within policy limits
func (p *BasicPolicyRestrictions) IsCostAllowed(hourlyCost float64) bool {
	if p == nil {
		return true // No restrictions
	}

	// Check hourly cost limit
	if p.MaxHourlyCost > 0 && hourlyCost > p.MaxHourlyCost {
		return false
	}

	return true
}

// GetPolicyViolations returns a list of policy violations for launch parameters
func (p *BasicPolicyRestrictions) GetPolicyViolations(templateName, instanceType, region string, hourlyCost float64) []string {
	var violations []string

	if !p.IsTemplateAllowed(templateName) {
		if len(p.TemplateWhitelist) > 0 {
			violations = append(violations, fmt.Sprintf("Template '%s' not in allowed list: %v", templateName, p.TemplateWhitelist))
		} else {
			violations = append(violations, fmt.Sprintf("Template '%s' is blacklisted", templateName))
		}
	}

	if !p.IsInstanceTypeAllowed(instanceType) {
		violations = append(violations, fmt.Sprintf("Instance type '%s' not allowed. Maximum allowed: %v", instanceType, p.MaxInstanceTypes))
	}

	if !p.IsRegionAllowed(region) {
		violations = append(violations, fmt.Sprintf("Region '%s' is forbidden", region))
	}

	if !p.IsCostAllowed(hourlyCost) {
		violations = append(violations, fmt.Sprintf("Hourly cost $%.2f exceeds maximum allowed $%.2f", hourlyCost, p.MaxHourlyCost))
	}

	return violations
}
