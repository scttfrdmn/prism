package api

import (
	"context"
	"fmt"
	"net/http"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// profileContextKey is the key used to store profile information in context
type profileContextKey struct{}

// ProfileContextKey is used to access profile information in context
var ProfileContextKey = profileContextKey{}

// ClientOptions represents configuration options for the API client
type ClientOptions struct {
	// AWS configuration
	AWSProfile      string
	AWSRegion       string
	
	// Invitation details
	InvitationToken string
	OwnerAccount    string
	S3ConfigPath    string
	
	// Profile information
	ProfileID       string
}

// SetAWSProfile sets the AWS profile to use for requests
func (c *Client) SetAWSProfile(profile string) {
	c.awsProfile = profile
}

// SetAWSRegion sets the AWS region to use for requests
func (c *Client) SetAWSRegion(region string) {
	c.awsRegion = region
}

// SetInvitationToken sets the invitation token and related information
func (c *Client) SetInvitationToken(token, ownerAccount, s3ConfigPath string) {
	c.invitationToken = token
	c.ownerAccount = ownerAccount
	c.s3ConfigPath = s3ConfigPath
}

// SetProfileID sets the profile ID for the client
func (c *Client) SetProfileID(profileID string) {
	c.profileID = profileID
}

// SetOptions updates the client's configuration options
func (c *Client) SetOptions(options ClientOptions) {
	// Set AWS profile
	if options.AWSProfile != "" {
		c.awsProfile = options.AWSProfile
	}
	
	// Set AWS region
	if options.AWSRegion != "" {
		c.awsRegion = options.AWSRegion
	}
	
	// Set invitation details
	if options.InvitationToken != "" {
		c.invitationToken = options.InvitationToken
		c.ownerAccount = options.OwnerAccount
		c.s3ConfigPath = options.S3ConfigPath
	}
	
	// Set profile ID
	if options.ProfileID != "" {
		c.profileID = options.ProfileID
	}
}

// addRequestHeaders adds common headers and auth headers to requests
func (c *Client) addRequestHeaders(req *http.Request) {
	// Add profile header if configured
	if c.awsProfile != "" {
		req.Header.Set("X-AWS-Profile", c.awsProfile)
	}
	
	// Add region header if configured
	if c.awsRegion != "" {
		req.Header.Set("X-AWS-Region", c.awsRegion)
	}
	
	// Add invitation headers if configured
	if c.invitationToken != "" {
		req.Header.Set("X-Invitation-Token", c.invitationToken)
		req.Header.Set("X-Owner-Account", c.ownerAccount)
		if c.s3ConfigPath != "" {
			req.Header.Set("X-S3-Config-Path", c.s3ConfigPath)
		}
	}
	
	// Add profile ID header if configured
	if c.profileID != "" {
		req.Header.Set("X-Profile-ID", c.profileID)
	}
}

// WithProfile returns a new client using the specified profile
func (c *Client) WithProfile(profileManager *profile.ManagerEnhanced, profileID string) (*Client, error) {
	// Get the profile
	prof, err := profileManager.GetProfile(profileID)
	if err != nil {
		return nil, err
	}
	
	// Create a new client with the same base URL
	client := NewClient(c.baseURL)
	
	// Configure the client based on profile type
	if prof.Type == profile.ProfileTypePersonal {
		client.SetAWSProfile(prof.AWSProfile)
		client.SetAWSRegion(prof.Region)
	} else if prof.Type == profile.ProfileTypeInvitation {
		client.SetInvitationToken(prof.InvitationToken, prof.OwnerAccount, prof.S3ConfigPath)
		client.SetAWSRegion(prof.Region)
	}
	
	// Set the profile ID
	client.SetProfileID(profileID)
	
	return client, nil
}

// WithProfileContext returns a new context with profile information
func WithProfileContext(ctx context.Context, prof *profile.Profile) context.Context {
	return context.WithValue(ctx, ProfileContextKey, prof)
}

// GetProfileFromContext retrieves profile information from context
func GetProfileFromContext(ctx context.Context) (*profile.Profile, bool) {
	prof, ok := ctx.Value(ProfileContextKey).(*profile.Profile)
	return prof, ok
}

// ContextClient profile management extensions

// WithProfile returns a new context client using the specified profile
func (c *ContextClient) WithProfile(profileManager *profile.ManagerEnhanced, profileID string) (*ContextClient, error) {
	// Create a client with the profile
	client, err := c.client.WithProfile(profileManager, profileID)
	if err != nil {
		return nil, err
	}
	
	// Create a new context client wrapping the base client
	return &ContextClient{
		client: client,
	}, nil
}

// SetProfile sets the current profile for the client
func (c *ContextClient) SetProfile(prof *profile.Profile) {
	// Configure the client based on profile type
	if prof.Type == profile.ProfileTypePersonal {
		c.client.SetAWSProfile(prof.AWSProfile)
		c.client.SetAWSRegion(prof.Region)
	} else if prof.Type == profile.ProfileTypeInvitation {
		c.client.SetInvitationToken(prof.InvitationToken, prof.OwnerAccount, prof.S3ConfigPath)
		c.client.SetAWSRegion(prof.Region)
	}
	
	// Set the profile ID
	c.client.SetProfileID(prof.AWSProfile)
}