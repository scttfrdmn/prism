package research

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// ResearchUserService provides the main service layer for research user management
// This integrates all research user components into a single, easy-to-use interface
type ResearchUserService struct {
	// Core components
	userManager *ResearchUserManager
	provisioner *ResearchUserProvisioner
	uidMapper   *ProfileUIDMapper
	sshManager  *ResearchUserSSHManager
	keyManager  *SSHKeyManager

	// Configuration
	configDir  string
	profileMgr ProfileManager
}

// ResearchUserServiceConfig holds configuration for the research user service
type ResearchUserServiceConfig struct {
	ConfigDir  string         // Base configuration directory
	ProfileMgr ProfileManager // Profile manager interface
}

// NewResearchUserService creates a new research user service with all components integrated
func NewResearchUserService(config *ResearchUserServiceConfig) *ResearchUserService {
	// Create core components
	userManager := NewResearchUserManager(config.ProfileMgr, config.ConfigDir)
	uidMapper := NewProfileUIDMapper(config.ProfileMgr)
	keyManager := NewSSHKeyManager(config.ConfigDir)
	sshManager := NewResearchUserSSHManager(keyManager, userManager)
	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)

	return &ResearchUserService{
		userManager: userManager,
		provisioner: provisioner,
		uidMapper:   uidMapper,
		sshManager:  sshManager,
		keyManager:  keyManager,
		configDir:   config.ConfigDir,
		profileMgr:  config.ProfileMgr,
	}
}

// CreateResearchUser creates a new research user with full setup
func (rus *ResearchUserService) CreateResearchUser(username string, options *CreateResearchUserOptions) (*ResearchUserConfig, error) {
	if options == nil {
		options = &CreateResearchUserOptions{}
	}

	// Get current profile
	profileID, err := rus.profileMgr.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	// Create research user
	researchUser, err := rus.userManager.CreateResearchUser(profileID, username)
	if err != nil {
		return nil, fmt.Errorf("failed to create research user: %w", err)
	}

	// Set up SSH keys if requested
	if options.GenerateSSHKey {
		if err := rus.sshManager.SetupSSHKeysForUser(profileID, username); err != nil {
			return nil, fmt.Errorf("failed to setup SSH keys: %w", err)
		}

		// Update research user with SSH key information
		publicKeys, err := rus.sshManager.GetSSHKeysForProvisioning(profileID, username)
		if err == nil {
			researchUser.SSHPublicKeys = publicKeys
			if err := rus.userManager.UpdateResearchUser(profileID, researchUser); err != nil {
				// Log warning but don't fail
				fmt.Printf("Warning: Failed to update research user with SSH keys: %v\n", err)
			}
		}
	}

	// Import existing SSH keys if provided
	if options.ImportSSHKey != "" {
		_, err := rus.keyManager.ImportSSHPublicKey(profileID, username, options.ImportSSHKey, options.SSHKeyComment)
		if err != nil {
			return nil, fmt.Errorf("failed to import SSH key: %w", err)
		}

		// Update research user with imported key
		researchUser.SSHPublicKeys = append(researchUser.SSHPublicKeys, options.ImportSSHKey)
		if err := rus.userManager.UpdateResearchUser(profileID, researchUser); err != nil {
			fmt.Printf("Warning: Failed to update research user with imported SSH key: %v\n", err)
		}
	}

	return researchUser, nil
}

// CreateResearchUserOptions provides options for creating research users
type CreateResearchUserOptions struct {
	GenerateSSHKey bool   // Automatically generate SSH key pair
	ImportSSHKey   string // Import existing public key
	SSHKeyComment  string // Comment for imported SSH key
}

// GetResearchUser retrieves a research user for the current profile
func (rus *ResearchUserService) GetResearchUser(username string) (*ResearchUserConfig, error) {
	profileID, err := rus.profileMgr.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	return rus.userManager.GetResearchUser(profileID, username)
}

// ListResearchUsers lists all research users for the current profile
func (rus *ResearchUserService) ListResearchUsers() ([]*ResearchUserConfig, error) {
	return rus.userManager.ListResearchUsers()
}

// ProvisionUserOnInstance provisions a research user on a specific instance
func (rus *ResearchUserService) ProvisionUserOnInstance(ctx context.Context, req *ProvisionInstanceRequest) (*UserProvisioningResponse, error) {
	// Get or create research user
	researchUser, err := rus.userManager.GetOrCreateResearchUser(req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to get research user: %w", err)
	}

	// Create provisioning request
	provisionReq := &UserProvisioningRequest{
		InstanceID:    req.InstanceID,
		InstanceName:  req.InstanceName,
		PublicIP:      req.PublicIP,
		TemplateName:  req.TemplateName,
		SystemUsers:   req.SystemUsers,
		ResearchUser:  researchUser,
		EFSVolumeID:   req.EFSVolumeID,
		EFSMountPoint: req.EFSMountPoint,
		SSHKeyPath:    req.SSHKeyPath,
		SSHUser:       req.SSHUser,
	}

	// Provision user
	return rus.provisioner.ProvisionResearchUser(ctx, provisionReq)
}

// ProvisionInstanceRequest represents a request to provision a research user on an instance
type ProvisionInstanceRequest struct {
	// Instance information
	InstanceID   string
	InstanceName string
	PublicIP     string

	// Template information
	TemplateName string
	SystemUsers  []SystemUser

	// Research user
	Username string

	// EFS integration
	EFSVolumeID   string
	EFSMountPoint string

	// SSH connection
	SSHKeyPath string
	SSHUser    string
}

// GetResearchUserStatus gets the status of a research user on an instance
func (rus *ResearchUserService) GetResearchUserStatus(ctx context.Context, instanceIP, username, sshKeyPath string) (*ResearchUserStatus, error) {
	return rus.provisioner.GetResearchUserStatus(ctx, instanceIP, username, sshKeyPath)
}

// ManageSSHKeys provides SSH key management functionality
func (rus *ResearchUserService) ManageSSHKeys() *ResearchUserSSHKeyManager {
	return &ResearchUserSSHKeyManager{
		service: rus,
	}
}

// GetUIDGIDForUser gets the consistent UID/GID for a research user
func (rus *ResearchUserService) GetUIDGIDForUser(username string) (*UIDGIDAllocation, error) {
	return rus.uidMapper.GetCurrentProfileUIDGID(username)
}

// GetOrCreateResearchUser provides access to the underlying user manager functionality
func (rus *ResearchUserService) GetOrCreateResearchUser(username string) (*ResearchUserConfig, error) {
	return rus.userManager.GetOrCreateResearchUser(username)
}

// DeleteResearchUser provides access to the underlying user manager functionality
func (rus *ResearchUserService) DeleteResearchUser(profileID, username string) error {
	return rus.userManager.DeleteResearchUser(profileID, username)
}

// UpdateResearchUser provides access to the underlying user manager functionality
func (rus *ResearchUserService) UpdateResearchUser(profileID string, user *ResearchUserConfig) error {
	return rus.userManager.UpdateResearchUser(profileID, user)
}

// GenerateUserProvisioningScript provides access to provisioning script generation
func (rus *ResearchUserService) GenerateUserProvisioningScript(req *UserProvisioningRequest) (string, error) {
	return rus.userManager.GenerateUserProvisioningScript(req)
}

// ResearchUserSSHKeyManager provides SSH key management operations
type ResearchUserSSHKeyManager struct {
	service *ResearchUserService
}

// GenerateKeyPair generates a new SSH key pair for a research user
func (ruskm *ResearchUserSSHKeyManager) GenerateKeyPair(username, keyType string) (*SSHKeyConfig, []byte, error) {
	profileID, err := ruskm.service.profileMgr.GetCurrentProfile()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	return ruskm.service.keyManager.GenerateSSHKeyPair(profileID, username, keyType)
}

// ImportPublicKey imports an existing SSH public key
func (ruskm *ResearchUserSSHKeyManager) ImportPublicKey(username, publicKey, comment string) (*SSHKeyConfig, error) {
	profileID, err := ruskm.service.profileMgr.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	return ruskm.service.keyManager.ImportSSHPublicKey(profileID, username, publicKey, comment)
}

// ListKeys lists all SSH keys for a research user
func (ruskm *ResearchUserSSHKeyManager) ListKeys(username string) (map[string]*SSHKeyConfig, error) {
	profileID, err := ruskm.service.profileMgr.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	return ruskm.service.keyManager.GetPublicKeysForUser(profileID, username)
}

// DeleteKey removes an SSH key
func (ruskm *ResearchUserSSHKeyManager) DeleteKey(username, keyID string) error {
	profileID, err := ruskm.service.profileMgr.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	return ruskm.service.keyManager.DeletePublicKey(profileID, username, keyID)
}

// Template Integration Functions

// ExtendTemplateWithResearchUser extends a template configuration to include research user support
func (rus *ResearchUserService) ExtendTemplateWithResearchUser(templateName string, researchUserTemplate *ResearchUserTemplate) error {
	// This would integrate with the template system to add research user configuration
	// Implementation depends on the template system architecture

	// For now, this is a placeholder that demonstrates the concept
	fmt.Printf("Extending template %s with research user configuration\n", templateName)

	if researchUserTemplate.AutoCreate {
		fmt.Printf("  - Auto-create research user: enabled\n")
	}

	if researchUserTemplate.RequireEFS {
		fmt.Printf("  - EFS integration: required at %s\n", researchUserTemplate.EFSMountPoint)
	}

	fmt.Printf("  - Integration strategy: %s\n", researchUserTemplate.UserIntegration.Strategy)

	return nil
}

// GetRecommendedDualUserConfig returns recommended dual user configuration for a template
func (rus *ResearchUserService) GetRecommendedDualUserConfig(templateName string) (*DualUserSystem, error) {
	// Analyze template and recommend dual user configuration
	// This is a simplified example - actual implementation would parse template content

	switch templateName {
	case "Python Machine Learning (Simplified)", "simple-python-ml":
		return &DualUserSystem{
			SystemUsers: []SystemUser{
				{
					Name:            "ubuntu",
					Purpose:         "system",
					TemplateCreated: false,
				},
				{
					Name:            "researcher",
					Purpose:         "jupyter",
					TemplateCreated: true,
				},
			},
			PrimaryUser:         "research",
			SharedDirectories:   []string{"/home/shared", "/opt/notebooks"},
			EnvironmentHandling: EnvironmentPolicyMerged,
		}, nil

	case "R Research Environment (Simplified)", "simple-r-research":
		return &DualUserSystem{
			SystemUsers: []SystemUser{
				{
					Name:            "ubuntu",
					Purpose:         "system",
					TemplateCreated: false,
				},
				{
					Name:            "rstudio",
					Purpose:         "rstudio",
					TemplateCreated: true,
				},
			},
			PrimaryUser:         "research",
			SharedDirectories:   []string{"/home/shared", "/opt/R"},
			EnvironmentHandling: EnvironmentPolicyMerged,
		}, nil

	default:
		// Generic configuration
		return &DualUserSystem{
			SystemUsers: []SystemUser{
				{
					Name:            "ubuntu",
					Purpose:         "system",
					TemplateCreated: false,
				},
			},
			PrimaryUser:         "research",
			SharedDirectories:   []string{"/home/shared"},
			EnvironmentHandling: EnvironmentPolicyResearchPrimary,
		}, nil
	}
}

// Migration and Compatibility Functions

// MigrateExistingUser migrates an existing system user to research user
func (rus *ResearchUserService) MigrateExistingUser(instanceIP, existingUser, newResearchUser, sshKeyPath string) error {
	// This would handle migration of existing users to research users
	// Implementation would involve:
	// 1. Backing up existing user data
	// 2. Creating research user with same files
	// 3. Updating permissions
	// 4. Migrating SSH keys

	fmt.Printf("Migrating user %s to research user %s on instance %s\n",
		existingUser, newResearchUser, instanceIP)

	// Placeholder implementation
	return fmt.Errorf("user migration not yet implemented")
}

// ValidateInstanceCompatibility checks if an instance is compatible with research users
func (rus *ResearchUserService) ValidateInstanceCompatibility(instanceInfo *InstanceCompatibilityInfo) (*CompatibilityReport, error) {
	report := &CompatibilityReport{
		Compatible:      true,
		Issues:          []string{},
		Recommendations: []string{},
	}

	// Check UID/GID range availability
	if instanceInfo.MinAvailableUID > ResearchUserMaxUID || instanceInfo.MaxAvailableUID < ResearchUserBaseUID {
		report.Compatible = false
		report.Issues = append(report.Issues,
			fmt.Sprintf("UID range %d-%d conflicts with research user range %d-%d",
				instanceInfo.MinAvailableUID, instanceInfo.MaxAvailableUID,
				ResearchUserBaseUID, ResearchUserMaxUID))
	}

	// Check EFS support
	if !instanceInfo.SupportsEFS {
		report.Recommendations = append(report.Recommendations,
			"Consider enabling EFS support for persistent home directories")
	}

	// Check SSH access
	if !instanceInfo.HasSSHAccess {
		report.Compatible = false
		report.Issues = append(report.Issues, "SSH access required for research user provisioning")
	}

	return report, nil
}

// InstanceCompatibilityInfo holds information about an instance for compatibility checking
type InstanceCompatibilityInfo struct {
	MinAvailableUID int
	MaxAvailableUID int
	SupportsEFS     bool
	HasSSHAccess    bool
	OSType          string
	KernelVersion   string
}

// CompatibilityReport provides the results of compatibility checking
type CompatibilityReport struct {
	Compatible      bool
	Issues          []string
	Recommendations []string
}

// Utility Functions

// GetResearchUserHomeDirectory returns the EFS home directory path for a research user
func (rus *ResearchUserService) GetResearchUserHomeDirectory(username string) string {
	return fmt.Sprintf("/efs/home/%s", username)
}

// GenerateResearchUserScript generates a complete provisioning script for an instance
func (rus *ResearchUserService) GenerateResearchUserScript(templateName, username string, options *ScriptGenerationOptions) (string, error) {
	if options == nil {
		options = &ScriptGenerationOptions{}
	}

	// Get research user
	researchUser, err := rus.GetResearchUser(username)
	if err != nil {
		// Create user if it doesn't exist
		researchUser, err = rus.CreateResearchUser(username, &CreateResearchUserOptions{
			GenerateSSHKey: options.GenerateSSHKeys,
		})
		if err != nil {
			return "", fmt.Errorf("failed to get/create research user: %w", err)
		}
	}

	// Get dual user configuration
	dualUserConfig, err := rus.GetRecommendedDualUserConfig(templateName)
	if err != nil {
		return "", fmt.Errorf("failed to get dual user configuration: %w", err)
	}

	// Create provisioning request
	req := &UserProvisioningRequest{
		InstanceID:    options.InstanceID,
		InstanceName:  options.InstanceName,
		TemplateName:  templateName,
		SystemUsers:   dualUserConfig.SystemUsers,
		ResearchUser:  researchUser,
		EFSVolumeID:   options.EFSVolumeID,
		EFSMountPoint: options.EFSMountPoint,
	}

	return rus.userManager.GenerateUserProvisioningScript(req)
}

// ScriptGenerationOptions provides options for script generation
type ScriptGenerationOptions struct {
	InstanceID      string
	InstanceName    string
	EFSVolumeID     string
	EFSMountPoint   string
	GenerateSSHKeys bool
}

// CreateDefaultResearchUserService creates a research user service with default configuration
func CreateDefaultResearchUserService(profileMgr interface {
	GetCurrentProfile() (*profile.Profile, error)
	GetProfile(name string) (*profile.Profile, error)
	UpdateProfile(profile *profile.Profile) error
}) *ResearchUserService {
	// Get config directory
	configDir := filepath.Join(os.Getenv("HOME"), ".cloudworkstation")

	// Create profile manager adapter
	profileAdapter := NewProfileManagerAdapter(profileMgr)

	// Create service config
	config := &ResearchUserServiceConfig{
		ConfigDir:  configDir,
		ProfileMgr: profileAdapter,
	}

	return NewResearchUserService(config)
}
