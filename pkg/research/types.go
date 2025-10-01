// Package research provides the multi-user foundation for CloudWorkstation Phase 5A.
//
// This package implements the research user architecture with dual-user system
// supporting both system users (from templates) and research users (persistent
// across instances) for collaborative research environments.
package research

import (
	"time"
)

// ResearchUserConfig represents the configuration for a research user
// that persists across instances and EFS volumes
type ResearchUserConfig struct {
	// Basic user information
	Username string `json:"username" yaml:"username"`
	UID      int    `json:"uid" yaml:"uid"` // Consistent across instances
	GID      int    `json:"gid" yaml:"gid"` // Primary group ID
	FullName string `json:"full_name" yaml:"full_name"`
	Email    string `json:"email" yaml:"email"`

	// Home directory configuration
	HomeDirectory string `json:"home_directory" yaml:"home_directory"`   // Path on EFS volume
	EFSVolumeID   string `json:"efs_volume_id" yaml:"efs_volume_id"`     // EFS volume for home dir
	EFSMountPoint string `json:"efs_mount_point" yaml:"efs_mount_point"` // Where EFS is mounted
	Shell         string `json:"shell" yaml:"shell"`                     // Default shell
	CreateHomeDir bool   `json:"create_home_dir" yaml:"create_home_dir"` // Auto-create home directory

	// SSH key management
	SSHPublicKeys     []string `json:"ssh_public_keys" yaml:"ssh_public_keys"`         // Authorized public keys
	SSHKeyFingerprint string   `json:"ssh_key_fingerprint" yaml:"ssh_key_fingerprint"` // Primary key fingerprint

	// Groups and permissions
	SecondaryGroups []string `json:"secondary_groups" yaml:"secondary_groups"` // Additional groups
	SudoAccess      bool     `json:"sudo_access" yaml:"sudo_access"`           // Sudo permissions
	DockerAccess    bool     `json:"docker_access" yaml:"docker_access"`       // Docker group access

	// Research environment preferences
	DefaultEnvironment map[string]string `json:"default_environment" yaml:"default_environment"` // Environment variables
	DotfileRepo        string            `json:"dotfile_repo" yaml:"dotfile_repo"`               // Git repo for dotfiles

	// Metadata
	CreatedAt    time.Time  `json:"created_at" yaml:"created_at"`
	LastUsed     *time.Time `json:"last_used" yaml:"last_used"`
	ProfileOwner string     `json:"profile_owner" yaml:"profile_owner"` // Profile that owns this user
}

// DualUserSystem represents the combined system + research user configuration
type DualUserSystem struct {
	// System users (from template)
	SystemUsers []SystemUser `json:"system_users" yaml:"system_users"`

	// Research user (persistent across instances)
	ResearchUser *ResearchUserConfig `json:"research_user" yaml:"research_user"`

	// Integration settings
	PrimaryUser         string            `json:"primary_user" yaml:"primary_user"`                 // Which user gets primary access
	SharedDirectories   []string          `json:"shared_directories" yaml:"shared_directories"`     // Dirs shared between users
	EnvironmentHandling EnvironmentPolicy `json:"environment_handling" yaml:"environment_handling"` // How to merge environments
}

// SystemUser represents a user created by the template (e.g., ubuntu, researcher, rocky)
type SystemUser struct {
	Name            string            `json:"name" yaml:"name"`
	UID             int               `json:"uid" yaml:"uid"` // May vary by instance
	GID             int               `json:"gid" yaml:"gid"`
	Groups          []string          `json:"groups" yaml:"groups"`
	Shell           string            `json:"shell" yaml:"shell"`
	HomeDirectory   string            `json:"home_directory" yaml:"home_directory"`
	Environment     map[string]string `json:"environment" yaml:"environment"`
	Purpose         string            `json:"purpose" yaml:"purpose"`                   // e.g., "jupyter", "system", "application"
	TemplateCreated bool              `json:"template_created" yaml:"template_created"` // Created by template vs system
}

// EnvironmentPolicy defines how to handle environment merging between system and research users
type EnvironmentPolicy string

const (
	// EnvironmentPolicyResearchPrimary - Research user environment takes precedence
	EnvironmentPolicyResearchPrimary EnvironmentPolicy = "research_primary"

	// EnvironmentPolicySystemPrimary - System user environment takes precedence
	EnvironmentPolicySystemPrimary EnvironmentPolicy = "system_primary"

	// EnvironmentPolicyMerged - Merge environments (research user wins conflicts)
	EnvironmentPolicyMerged EnvironmentPolicy = "merged"

	// EnvironmentPolicyIsolated - Keep environments completely separate
	EnvironmentPolicyIsolated EnvironmentPolicy = "isolated"
)

// ResearchUserManager handles research user lifecycle management
type ResearchUserManager struct {
	// Configuration
	profileManager ProfileManager // Interface to profile system

	// UID/GID allocation
	baseUID        int            // Starting UID for research users (e.g., 5000)
	baseGID        int            // Starting GID for research users (e.g., 5000)
	uidAllocations map[string]int // Profile -> UID mapping

	// Storage
	configPath string // Where research user configs are stored
}

// ProfileManager interface for integration with existing profile system
type ProfileManager interface {
	GetCurrentProfile() (string, error)
	GetProfileConfig(profileID string) (interface{}, error)
	UpdateProfileConfig(profileID string, config interface{}) error
}

// UserProvisioningRequest represents a request to provision a research user on an instance
type UserProvisioningRequest struct {
	// Instance information
	InstanceID   string `json:"instance_id"`
	InstanceName string `json:"instance_name"`
	PublicIP     string `json:"public_ip"`

	// Template information
	TemplateName string       `json:"template_name"`
	SystemUsers  []SystemUser `json:"system_users"`

	// Research user to provision
	ResearchUser *ResearchUserConfig `json:"research_user"`

	// EFS integration
	EFSVolumeID   string `json:"efs_volume_id"`
	EFSMountPoint string `json:"efs_mount_point"`

	// SSH connection info
	SSHKeyPath string `json:"ssh_key_path"`
	SSHUser    string `json:"ssh_user"` // Initial user for connection (e.g., ubuntu)
}

// UserProvisioningResponse represents the result of user provisioning
type UserProvisioningResponse struct {
	Success          bool     `json:"success"`
	Message          string   `json:"message"`
	CreatedUsers     []string `json:"created_users"`
	ConfiguredEFS    bool     `json:"configured_efs"`
	SSHKeysInstalled bool     `json:"ssh_keys_installed"`
	ErrorDetails     string   `json:"error_details,omitempty"`
}

// ResearchUserStatus represents the current status of a research user on an instance
type ResearchUserStatus struct {
	Username          string     `json:"username"`
	InstanceID        string     `json:"instance_id"`
	InstanceName      string     `json:"instance_name"`
	HomeDirectoryPath string     `json:"home_directory_path"`
	EFSMounted        bool       `json:"efs_mounted"`
	SSHAccessible     bool       `json:"ssh_accessible"`
	LastLogin         *time.Time `json:"last_login"`
	ActiveProcesses   int        `json:"active_processes"`
	DiskUsage         int64      `json:"disk_usage"` // Bytes
}

// UID/GID Allocation Strategy
const (
	// Research user UID range: 5000-5999 (1000 users)
	ResearchUserBaseUID = 5000
	ResearchUserMaxUID  = 5999

	// Research user GID range: 5000-5999 (matches UID)
	ResearchUserBaseGID = 5000
	ResearchUserMaxGID  = 5999

	// Default group names
	ResearchUserGroup  = "research"       // Primary group for all research users
	ResearchAdminGroup = "research-admin" // Admin group for research user management
	EFSAccessGroup     = "efs-users"      // Group for EFS access
)

// Template Integration Types
// These extend the existing template system to support research users

// ResearchUserTemplate represents research user configuration in templates
type ResearchUserTemplate struct {
	// Auto-create research user if not exists
	AutoCreate bool `json:"auto_create" yaml:"auto_create"`

	// Research user defaults for this template
	DefaultShell       string            `json:"default_shell" yaml:"default_shell"`
	DefaultGroups      []string          `json:"default_groups" yaml:"default_groups"`
	DefaultEnvironment map[string]string `json:"default_environment" yaml:"default_environment"`

	// EFS integration
	RequireEFS          bool   `json:"require_efs" yaml:"require_efs"`
	EFSMountPoint       string `json:"efs_mount_point" yaml:"efs_mount_point"`
	EFSHomeSubdirectory string `json:"efs_home_subdirectory" yaml:"efs_home_subdirectory"`

	// SSH configuration
	InstallSSHKeys bool `json:"install_ssh_keys" yaml:"install_ssh_keys"`

	// Integration with system users
	UserIntegration DualUserIntegration `json:"user_integration" yaml:"user_integration"`
}

// DualUserIntegration defines how research users integrate with system users
type DualUserIntegration struct {
	Strategy          IntegrationStrategy `json:"strategy" yaml:"strategy"`
	PrimaryUser       string              `json:"primary_user" yaml:"primary_user"` // "research" or system user name
	SharedDirectories []string            `json:"shared_directories" yaml:"shared_directories"`
	ServiceOwnership  map[string]string   `json:"service_ownership" yaml:"service_ownership"` // service -> user mapping
}

// IntegrationStrategy defines how research and system users work together
type IntegrationStrategy string

const (
	// IntegrationStrategyPrimary - Research user is primary, system users for services only
	IntegrationStrategyPrimary IntegrationStrategy = "research_primary"

	// IntegrationStrategyCoexist - Both users coexist with shared access
	IntegrationStrategyCoexist IntegrationStrategy = "coexist"

	// IntegrationStrategySystemFirst - System user primary, research user secondary
	IntegrationStrategySystemFirst IntegrationStrategy = "system_first"
)

// SSH Key Management Types (Forward declaration - implemented in ssh_keys.go)

// KeyStore interface for storing and retrieving SSH keys
type KeyStore interface {
	StorePublicKey(profileID, username, keyID string, publicKey []byte) error
	GetPublicKeys(profileID, username string) (map[string][]byte, error)
	DeletePublicKey(profileID, username, keyID string) error
	ListKeyFingerprints(profileID, username string) ([]string, error)
}

// KeyGenerator interface for generating SSH key pairs
type KeyGenerator interface {
	GenerateKeyPair(keyType string, keySize int) (privateKey, publicKey []byte, err error)
	GetFingerprint(publicKey []byte) (string, error)
	ValidatePublicKey(publicKey []byte) error
}

// SSHKeyConfig represents SSH key configuration for a research user
type SSHKeyConfig struct {
	KeyID         string     `json:"key_id"`
	Fingerprint   string     `json:"fingerprint"`
	PublicKey     string     `json:"public_key"`
	Comment       string     `json:"comment"`
	CreatedAt     time.Time  `json:"created_at"`
	LastUsed      *time.Time `json:"last_used"`
	FromProfile   string     `json:"from_profile"`   // Which profile added this key
	AutoGenerated bool       `json:"auto_generated"` // Was this key auto-generated
}
