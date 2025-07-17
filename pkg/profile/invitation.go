// Package profile provides functionality for managing CloudWorkstation profiles
package profile

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// InvitationType specifies the type of invitation being generated
type InvitationType string

const (
	// InvitationTypeReadOnly allows read-only access to resources
	InvitationTypeReadOnly InvitationType = "read_only"
	
	// InvitationTypeReadWrite allows read and write access to resources
	InvitationTypeReadWrite InvitationType = "read_write"
	
	// InvitationTypeAdmin allows full access to resources
	InvitationTypeAdmin InvitationType = "admin"
)

// InvitationToken represents a secure invitation with metadata
type InvitationToken struct {
	// Token is the unique identifier for this invitation
	Token string `json:"token"`
	
	// OwnerProfile is the profile ID of the creator
	OwnerProfile string `json:"owner_profile"`
	
	// OwnerAccount is the AWS account ID of the owner
	OwnerAccount string `json:"owner_account"`
	
	// S3ConfigPath is the optional path to shared config in S3
	S3ConfigPath string `json:"s3_config_path,omitempty"`
	
	// Type defines the permission level
	Type InvitationType `json:"type"`
	
	// Name is a human-readable name for this invitation
	Name string `json:"name"`
	
	// Created is when this invitation was generated
	Created time.Time `json:"created"`
	
	// Expires is when this invitation will no longer be valid
	Expires time.Time `json:"expires"`
}

// GenerateInvitationToken creates a new secure invitation token
func GenerateInvitationToken(ownerProfile, ownerAccount, name string, invType InvitationType, validDays int, s3ConfigPath string) (*InvitationToken, error) {
	if validDays <= 0 {
		validDays = 30 // Default to 30 days
	}
	
	// Generate random bytes for the token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}
	
	// Format the token with prefix for easy identification
	token := fmt.Sprintf("inv-%s", base64.RawURLEncoding.EncodeToString(tokenBytes))
	
	// Create the invitation
	invitation := &InvitationToken{
		Token:        token,
		OwnerProfile: ownerProfile,
		OwnerAccount: ownerAccount,
		S3ConfigPath: s3ConfigPath,
		Type:         invType,
		Name:         name,
		Created:      time.Now(),
		Expires:      time.Now().AddDate(0, 0, validDays),
	}
	
	return invitation, nil
}

// EncodeToString encodes the invitation token to a shareable string
func (i *InvitationToken) EncodeToString() (string, error) {
	// Marshal to JSON
	jsonData, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("failed to encode invitation: %w", err)
	}
	
	// Encode as base64 for easy sharing
	encoded := base64.RawURLEncoding.EncodeToString(jsonData)
	
	return encoded, nil
}

// DecodeFromString decodes an invitation token from a string
func DecodeFromString(encoded string) (*InvitationToken, error) {
	// Decode from base64
	jsonData, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid invitation format: %w", err)
	}
	
	// Unmarshal from JSON
	var invitation InvitationToken
	if err := json.Unmarshal(jsonData, &invitation); err != nil {
		return nil, fmt.Errorf("invalid invitation data: %w", err)
	}
	
	return &invitation, nil
}

// IsValid checks if the invitation token is still valid
func (i *InvitationToken) IsValid() bool {
	// Check if token has expired
	return time.Now().Before(i.Expires)
}

// GetExpirationDuration returns the duration until this invitation expires
func (i *InvitationToken) GetExpirationDuration() time.Duration {
	return time.Until(i.Expires)
}

// GetDescription returns a human-readable description of the invitation
func (i *InvitationToken) GetDescription() string {
	return fmt.Sprintf("Invitation '%s' to %s (%s access, expires in %s)",
		i.Name, i.OwnerAccount, i.Type, i.GetExpirationDuration().Round(time.Hour))
}