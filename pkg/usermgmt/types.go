package usermgmt

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrDuplicateUsername  = errors.New("username already exists")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrGroupNotFound      = errors.New("group not found")
	ErrDuplicateGroup     = errors.New("group already exists")
)

// Provider represents a user management system provider
type Provider string

const (
	// ProviderAWSSO represents AWS SSO / IAM Identity Center
	ProviderAWSSO Provider = "aws-sso"
	
	// ProviderOkta represents Okta
	ProviderOkta Provider = "okta"
	
	// ProviderAzureAD represents Azure Active Directory
	ProviderAzureAD Provider = "azure-ad"
	
	// ProviderGoogleWorkspace represents Google Workspace
	ProviderGoogleWorkspace Provider = "google-workspace"
	
	// ProviderOneLogin represents OneLogin
	ProviderOneLogin Provider = "onelogin"
	
	// ProviderLocal represents local user management
	ProviderLocal Provider = "local"
	
	// ProviderOIDC represents generic OpenID Connect provider
	ProviderOIDC Provider = "oidc"
)

// UserRole represents a user role
type UserRole string

const (
	// UserRoleAdmin represents an administrator
	UserRoleAdmin UserRole = "admin"
	
	// UserRolePowerUser represents a power user
	UserRolePowerUser UserRole = "power-user"
	
	// UserRoleUser represents a regular user
	UserRoleUser UserRole = "user"
	
	// UserRoleReadOnly represents a read-only user
	UserRoleReadOnly UserRole = "read-only"
)

// User represents a user in the system
type User struct {
	// ID is a unique identifier for the user
	ID string `json:"id"`
	
	// Username is the username for the user
	Username string `json:"username"`
	
	// Email is the user's email address
	Email string `json:"email"`
	
	// DisplayName is the user's display name
	DisplayName string `json:"display_name"`
	
	// Roles are the roles assigned to the user
	Roles []UserRole `json:"roles"`
	
	// Provider is the user management provider for this user
	Provider Provider `json:"provider"`
	
	// ProviderID is the ID of the user in the provider's system
	ProviderID string `json:"provider_id"`
	
	// Attributes are additional attributes for the user
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	
	// Groups are the groups the user belongs to
	Groups []string `json:"groups,omitempty"`
	
	// CreatedAt is when the user was created
	CreatedAt time.Time `json:"created_at"`
	
	// UpdatedAt is when the user was last updated
	UpdatedAt time.Time `json:"updated_at"`
	
	// LastLogin is when the user last logged in
	LastLogin *time.Time `json:"last_login,omitempty"`
	
	// Enabled indicates if the user is enabled
	Enabled bool `json:"enabled"`
}

// Group represents a user group
type Group struct {
	// ID is a unique identifier for the group
	ID string `json:"id"`
	
	// Name is the name of the group
	Name string `json:"name"`
	
	// Description is a description of the group
	Description string `json:"description"`
	
	// Provider is the user management provider for this group
	Provider Provider `json:"provider"`
	
	// ProviderID is the ID of the group in the provider's system
	ProviderID string `json:"provider_id"`
	
	// Attributes are additional attributes for the group
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	
	// CreatedAt is when the group was created
	CreatedAt time.Time `json:"created_at"`
	
	// UpdatedAt is when the group was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// UserFilter represents filters for fetching users
type UserFilter struct {
	// Username filters by username
	Username string `json:"username,omitempty"`
	
	// Email filters by email
	Email string `json:"email,omitempty"`
	
	// Role filters by role
	Role UserRole `json:"role,omitempty"`
	
	// Group filters by group
	Group string `json:"group,omitempty"`
	
	// Provider filters by provider
	Provider Provider `json:"provider,omitempty"`
	
	// EnabledOnly filters to only enabled users
	EnabledOnly bool `json:"enabled_only,omitempty"`
	
	// DisabledOnly filters to only disabled users
	DisabledOnly bool `json:"disabled_only,omitempty"`
	
	// CreatedAfter filters to users created after the specified time
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	
	// CreatedBefore filters to users created before the specified time
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	
	// UpdatedAfter filters to users updated after the specified time
	UpdatedAfter *time.Time `json:"updated_after,omitempty"`
	
	// UpdatedBefore filters to users updated before the specified time
	UpdatedBefore *time.Time `json:"updated_before,omitempty"`
	
	// LastLoginAfter filters to users who logged in after the specified time
	LastLoginAfter *time.Time `json:"last_login_after,omitempty"`
	
	// LastLoginBefore filters to users who logged in before the specified time
	LastLoginBefore *time.Time `json:"last_login_before,omitempty"`
}

// GroupFilter represents filters for fetching groups
type GroupFilter struct {
	// Name filters by name
	Name string `json:"name,omitempty"`
	
	// Provider filters by provider
	Provider Provider `json:"provider,omitempty"`
	
	// CreatedAfter filters to groups created after the specified time
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	
	// CreatedBefore filters to groups created before the specified time
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}

// PaginationOptions represents options for paginating results
type PaginationOptions struct {
	// Page is the page number (1-based)
	Page int `json:"page"`
	
	// PageSize is the number of items per page
	PageSize int `json:"page_size"`
	
	// SortBy is the field to sort by
	SortBy string `json:"sort_by"`
	
	// SortOrder is the sort order ("asc" or "desc")
	SortOrder string `json:"sort_order"`
}

// UserProvisionOptions represents options for provisioning users
type UserProvisionOptions struct {
	// DefaultRole is the default role for new users
	DefaultRole UserRole `json:"default_role"`
	
	// AutoCreateGroups automatically creates groups if they don't exist
	AutoCreateGroups bool `json:"auto_create_groups"`
	
	// GroupRoleMapping maps group names to roles
	GroupRoleMapping map[string]UserRole `json:"group_role_mapping"`
	
	// AttributeMapping maps provider attributes to system attributes
	AttributeMapping map[string]string `json:"attribute_mapping"`
	
	// AutoProvision automatically provisions users on first login
	AutoProvision bool `json:"auto_provision"`
	
	// RequireGroup requires users to be in at least one group
	RequireGroup bool `json:"require_group"`
	
	// AllowedDomains is a list of allowed email domains
	AllowedDomains []string `json:"allowed_domains"`
}

// SyncOptions represents options for synchronizing users
type SyncOptions struct {
	// SyncGroups synchronizes group memberships
	SyncGroups bool `json:"sync_groups"`
	
	// SyncRoles synchronizes user roles based on group mappings
	SyncRoles bool `json:"sync_roles"`
	
	// SyncAttributes synchronizes user attributes
	SyncAttributes bool `json:"sync_attributes"`
	
	// DisableUnknownUsers disables users not found in the provider
	DisableUnknownUsers bool `json:"disable_unknown_users"`
	
	// DisableDeletedUsers disables users marked as deleted in the provider
	DisableDeletedUsers bool `json:"disable_deleted_users"`
	
	// CreateMissingUsers creates users that exist in the provider but not locally
	CreateMissingUsers bool `json:"create_missing_users"`
	
	// BatchSize is the batch size for processing users
	BatchSize int `json:"batch_size"`
}

// PaginatedUsers represents a paginated list of users
type PaginatedUsers struct {
	// Users is the list of users
	Users []*User `json:"users"`
	
	// Total is the total number of users
	Total int `json:"total"`
	
	// Page is the current page number
	Page int `json:"page"`
	
	// PageSize is the number of items per page
	PageSize int `json:"page_size"`
	
	// TotalPages is the total number of pages
	TotalPages int `json:"total_pages"`
}

// PaginatedGroups represents a paginated list of groups
type PaginatedGroups struct {
	// Groups is the list of groups
	Groups []*Group `json:"groups"`
	
	// Total is the total number of groups
	Total int `json:"total"`
	
	// Page is the current page number
	Page int `json:"page"`
	
	// PageSize is the number of items per page
	PageSize int `json:"page_size"`
	
	// TotalPages is the total number of pages
	TotalPages int `json:"total_pages"`
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	// Created is the number of users created
	Created int `json:"created"`
	
	// Updated is the number of users updated
	Updated int `json:"updated"`
	
	// Disabled is the number of users disabled
	Disabled int `json:"disabled"`
	
	// Failed is the number of users that failed to sync
	Failed int `json:"failed"`
	
	// FailedUsers is a list of users that failed to sync
	FailedUsers []string `json:"failed_users,omitempty"`
	
	// GroupsCreated is the number of groups created
	GroupsCreated int `json:"groups_created"`
	
	// GroupsUpdated is the number of groups updated
	GroupsUpdated int `json:"groups_updated"`
	
	// Started is when the sync started
	Started time.Time `json:"started"`
	
	// Completed is when the sync completed
	Completed time.Time `json:"completed"`
	
	// Duration is the duration of the sync in seconds
	Duration float64 `json:"duration"`
}