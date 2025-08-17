// Package core provides the simplified core profile management system.
//
// This package implements a clean, focused profile system that handles only
// the essential functionality: storing AWS profiles and regions, managing
// the current active profile, and persisting configuration.
//
// This replaces the over-engineered profile system with a simple, maintainable solution.
package core

import (
	"time"
)

// Profile represents a CloudWorkstation profile configuration.
// This is dramatically simplified from the original bloated Profile struct.
type Profile struct {
	// Name is the unique identifier for this profile
	Name string `json:"name"`

	// AWSProfile is the AWS CLI profile name to use
	AWSProfile string `json:"aws_profile"`

	// Region is the default AWS region for this profile
	Region string `json:"region"`

	// Default indicates if this is the default profile
	Default bool `json:"default"`

	// CreatedAt is when the profile was created
	CreatedAt time.Time `json:"created_at"`

	// LastUsed is when the profile was last used (optional)
	LastUsed *time.Time `json:"last_used,omitempty"`
}

// ProfileConfig represents the persisted configuration file format
type ProfileConfig struct {
	// Profiles maps profile names to profile configurations
	Profiles map[string]*Profile `json:"profiles"`

	// Current is the name of the currently active profile
	Current string `json:"current"`

	// Version is the config file format version
	Version int `json:"version"`

	// UpdatedAt is when the config was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// ValidationError represents profile validation errors
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "profile validation error in " + e.Field + ": " + e.Message
}

// ProfileNotFoundError represents a profile not found error
type ProfileNotFoundError struct {
	Name string
}

func (e *ProfileNotFoundError) Error() string {
	return "profile not found: " + e.Name
}

// NoProfilesError indicates no profiles are configured
type NoProfilesError struct{}

func (e *NoProfilesError) Error() string {
	return "no profiles configured"
}

// NoCurrentProfileError indicates no current profile is set
type NoCurrentProfileError struct{}

func (e *NoCurrentProfileError) Error() string {
	return "no current profile set"
}

// Constants for profile management
const (
	// DefaultConfigVersion is the current config file format version
	DefaultConfigVersion = 1

	// DefaultProfileName is the name used for the first profile
	DefaultProfileName = "default"

	// ConfigFileName is the name of the profile configuration file
	ConfigFileName = "profiles.json"
)
