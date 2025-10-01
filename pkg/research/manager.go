package research

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// NewResearchUserManager creates a new research user manager
func NewResearchUserManager(profileMgr ProfileManager, configDir string) *ResearchUserManager {
	return &ResearchUserManager{
		profileManager: profileMgr,
		baseUID:        ResearchUserBaseUID,
		baseGID:        ResearchUserBaseGID,
		uidAllocations: make(map[string]int),
		configPath:     configDir,
	}
}

// GetOrCreateResearchUser gets or creates a research user for the current profile
func (rum *ResearchUserManager) GetOrCreateResearchUser(username string) (*ResearchUserConfig, error) {
	// Get current profile
	currentProfile, err := rum.profileManager.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	// Check if research user already exists
	existing, err := rum.GetResearchUser(currentProfile, username)
	if err == nil {
		return existing, nil
	}

	// Create new research user
	return rum.CreateResearchUser(currentProfile, username)
}

// CreateResearchUser creates a new research user for the specified profile
func (rum *ResearchUserManager) CreateResearchUser(profileID, username string) (*ResearchUserConfig, error) {
	// Validate username
	if err := rum.validateUsername(username); err != nil {
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	// Allocate UID/GID
	uid, gid, err := rum.allocateUIGID(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate UID/GID: %w", err)
	}

	// Generate email if not provided
	email := fmt.Sprintf("%s@cloudworkstation.local", username)

	// Create research user config
	researchUser := &ResearchUserConfig{
		Username:        username,
		UID:             uid,
		GID:             gid,
		FullName:        strings.Title(username),
		Email:           email,
		HomeDirectory:   fmt.Sprintf("/efs/home/%s", username),
		EFSMountPoint:   "/efs",
		Shell:           "/bin/bash",
		CreateHomeDir:   true,
		SecondaryGroups: []string{ResearchUserGroup, EFSAccessGroup},
		SudoAccess:      true, // Research users get sudo by default
		DockerAccess:    true, // Docker access for research workflows
		DefaultEnvironment: map[string]string{
			"RESEARCH_USER": "true",
			"RESEARCH_HOME": fmt.Sprintf("/efs/home/%s", username),
		},
		CreatedAt:    time.Now(),
		ProfileOwner: profileID,
	}

	// Save to disk
	if err := rum.saveResearchUser(profileID, researchUser); err != nil {
		return nil, fmt.Errorf("failed to save research user: %w", err)
	}

	return researchUser, nil
}

// GetResearchUser retrieves a research user for the specified profile
func (rum *ResearchUserManager) GetResearchUser(profileID, username string) (*ResearchUserConfig, error) {
	configFile := rum.getResearchUserConfigPath(profileID, username)

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("research user %s not found for profile %s", username, profileID)
		}
		return nil, fmt.Errorf("failed to read research user config: %w", err)
	}

	var researchUser ResearchUserConfig
	if err := json.Unmarshal(data, &researchUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal research user config: %w", err)
	}

	return &researchUser, nil
}

// ListResearchUsers lists all research users for the current profile
func (rum *ResearchUserManager) ListResearchUsers() ([]*ResearchUserConfig, error) {
	currentProfile, err := rum.profileManager.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	return rum.ListResearchUsersForProfile(currentProfile)
}

// ListResearchUsersForProfile lists all research users for a specific profile
func (rum *ResearchUserManager) ListResearchUsersForProfile(profileID string) ([]*ResearchUserConfig, error) {
	profileDir := filepath.Join(rum.configPath, "research-users", profileID)

	entries, err := os.ReadDir(profileDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ResearchUserConfig{}, nil // No users exist yet
		}
		return nil, fmt.Errorf("failed to read profile directory: %w", err)
	}

	var users []*ResearchUserConfig
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			username := strings.TrimSuffix(entry.Name(), ".json")
			user, err := rum.GetResearchUser(profileID, username)
			if err != nil {
				continue // Skip invalid configs
			}
			users = append(users, user)
		}
	}

	// Sort by username
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})

	return users, nil
}

// UpdateResearchUser updates an existing research user
func (rum *ResearchUserManager) UpdateResearchUser(profileID string, user *ResearchUserConfig) error {
	// Validate the user exists
	existing, err := rum.GetResearchUser(profileID, user.Username)
	if err != nil {
		return fmt.Errorf("research user does not exist: %w", err)
	}

	// Preserve creation metadata
	user.CreatedAt = existing.CreatedAt
	user.UID = existing.UID // UIDs should not change
	user.GID = existing.GID // GIDs should not change
	user.ProfileOwner = existing.ProfileOwner

	// Save updated config
	return rum.saveResearchUser(profileID, user)
}

// DeleteResearchUser removes a research user configuration
func (rum *ResearchUserManager) DeleteResearchUser(profileID, username string) error {
	configFile := rum.getResearchUserConfigPath(profileID, username)

	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete research user config: %w", err)
	}

	// Remove from UID allocation cache
	delete(rum.uidAllocations, fmt.Sprintf("%s:%s", profileID, username))

	return nil
}

// GenerateUserProvisioningScript generates a shell script to provision the research user on an instance
func (rum *ResearchUserManager) GenerateUserProvisioningScript(req *UserProvisioningRequest) (string, error) {
	if req.ResearchUser == nil {
		return "", fmt.Errorf("research user configuration required")
	}

	user := req.ResearchUser
	script := []string{
		"#!/bin/bash",
		"set -e",
		"",
		"# CloudWorkstation Research User Provisioning Script",
		fmt.Sprintf("# Instance: %s (%s)", req.InstanceName, req.InstanceID),
		fmt.Sprintf("# Research User: %s (UID: %d)", user.Username, user.UID),
		fmt.Sprintf("# Generated: %s", time.Now().Format(time.RFC3339)),
		"",
		"echo 'Starting research user provisioning...'",
		"",
	}

	// Create research groups
	script = append(script, rum.generateGroupCreationCommands()...)

	// Create research user
	script = append(script, rum.generateUserCreationCommands(user)...)

	// Configure EFS if specified
	if req.EFSVolumeID != "" && req.EFSMountPoint != "" {
		script = append(script, rum.generateEFSSetupCommands(user, req.EFSVolumeID, req.EFSMountPoint)...)
	}

	// Install SSH keys
	if len(user.SSHPublicKeys) > 0 {
		script = append(script, rum.generateSSHKeyInstallCommands(user)...)
	}

	// Configure environment
	script = append(script, rum.generateEnvironmentSetupCommands(user)...)

	// Final configuration
	script = append(script,
		"",
		"# Set permissions and finalize setup",
		fmt.Sprintf("chown -R %s:%s %s || true", user.Username, ResearchUserGroup, user.HomeDirectory),
		fmt.Sprintf("chmod 750 %s || true", user.HomeDirectory),
		"",
		"echo 'Research user provisioning complete!'",
		fmt.Sprintf("echo 'User: %s (UID: %d)'", user.Username, user.UID),
		fmt.Sprintf("echo 'Home: %s'", user.HomeDirectory),
		"echo 'SSH access configured with provided keys'",
		"",
	)

	return strings.Join(script, "\n"), nil
}

// Private helper methods

func (rum *ResearchUserManager) validateUsername(username string) error {
	if len(username) < 2 || len(username) > 32 {
		return fmt.Errorf("username must be between 2 and 32 characters")
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("username contains invalid character: %c", r)
		}
	}

	// Username cannot start with a number or dash
	if username[0] >= '0' && username[0] <= '9' || username[0] == '-' {
		return fmt.Errorf("username cannot start with a number or dash")
	}

	return nil
}

func (rum *ResearchUserManager) allocateUIGID(profileID string) (uid, gid int, err error) {
	// Generate deterministic UID/GID based on profile ID
	hash := md5.Sum([]byte(profileID))

	// Use first 4 bytes of hash to generate offset within allowed range
	offset := int(hash[0])<<24 | int(hash[1])<<16 | int(hash[2])<<8 | int(hash[3])
	if offset < 0 {
		offset = -offset
	}

	// Map to allowed UID range
	uid = rum.baseUID + (offset % (ResearchUserMaxUID - rum.baseUID))

	// Ensure we don't conflict with existing allocations
	for {
		key := fmt.Sprintf("%s:%d", profileID, uid)
		if _, exists := rum.uidAllocations[key]; !exists {
			rum.uidAllocations[key] = uid
			break
		}
		uid++
		if uid > ResearchUserMaxUID {
			return 0, 0, fmt.Errorf("no available UIDs in range %d-%d", rum.baseUID, ResearchUserMaxUID)
		}
	}

	// GID matches UID for simplicity
	gid = uid

	return uid, gid, nil
}

func (rum *ResearchUserManager) saveResearchUser(profileID string, user *ResearchUserConfig) error {
	configFile := rum.getResearchUserConfigPath(profileID, user.Username)
	configDir := filepath.Dir(configFile)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal research user config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write research user config: %w", err)
	}

	return nil
}

func (rum *ResearchUserManager) getResearchUserConfigPath(profileID, username string) string {
	return filepath.Join(rum.configPath, "research-users", profileID, username+".json")
}

func (rum *ResearchUserManager) generateGroupCreationCommands() []string {
	return []string{
		"# Create research user groups",
		fmt.Sprintf("groupadd -g %d %s || true", ResearchUserBaseGID, ResearchUserGroup),
		fmt.Sprintf("groupadd -g %d %s || true", ResearchUserBaseGID+1, ResearchAdminGroup),
		fmt.Sprintf("groupadd -g %d %s || true", ResearchUserBaseGID+2, EFSAccessGroup),
		"",
	}
}

func (rum *ResearchUserManager) generateUserCreationCommands(user *ResearchUserConfig) []string {
	// Build secondary groups list
	groups := append([]string{ResearchUserGroup}, user.SecondaryGroups...)
	if user.SudoAccess {
		groups = append(groups, "sudo")
	}
	if user.DockerAccess {
		groups = append(groups, "docker")
	}

	groupsStr := strings.Join(groups, ",")

	return []string{
		"# Create research user",
		fmt.Sprintf("useradd -m -u %d -g %d -G %s -s %s -c '%s' %s || true",
			user.UID, user.GID, groupsStr, user.Shell, user.FullName, user.Username),
		"",
		"# Create home directory structure if needed",
		fmt.Sprintf("mkdir -p %s", user.HomeDirectory),
		fmt.Sprintf("mkdir -p %s/.ssh", user.HomeDirectory),
		fmt.Sprintf("mkdir -p %s/research", user.HomeDirectory),
		fmt.Sprintf("mkdir -p %s/projects", user.HomeDirectory),
		"",
	}
}

func (rum *ResearchUserManager) generateEFSSetupCommands(user *ResearchUserConfig, volumeID, mountPoint string) []string {
	return []string{
		"# Configure EFS for research user",
		fmt.Sprintf("mkdir -p %s", mountPoint),
		fmt.Sprintf("mkdir -p %s/home", mountPoint),
		fmt.Sprintf("mkdir -p %s/home/%s", mountPoint, user.Username),
		"",
		"# Set EFS permissions",
		fmt.Sprintf("chown root:%s %s", EFSAccessGroup, mountPoint),
		"chmod 755 " + mountPoint,
		fmt.Sprintf("chown %s:%s %s/home/%s", user.Username, ResearchUserGroup, mountPoint, user.Username),
		fmt.Sprintf("chmod 750 %s/home/%s", mountPoint, user.Username),
		"",
	}
}

func (rum *ResearchUserManager) generateSSHKeyInstallCommands(user *ResearchUserConfig) []string {
	commands := []string{
		"# Install SSH keys for research user",
		fmt.Sprintf("mkdir -p %s/.ssh", user.HomeDirectory),
		fmt.Sprintf("chmod 700 %s/.ssh", user.HomeDirectory),
		"",
	}

	// Add each public key
	for i, pubkey := range user.SSHPublicKeys {
		commands = append(commands, fmt.Sprintf("# SSH Key %d", i+1))
		commands = append(commands, fmt.Sprintf("echo '%s' >> %s/.ssh/authorized_keys", pubkey, user.HomeDirectory))
	}

	commands = append(commands,
		"",
		fmt.Sprintf("chmod 600 %s/.ssh/authorized_keys", user.HomeDirectory),
		fmt.Sprintf("chown -R %s:%s %s/.ssh", user.Username, ResearchUserGroup, user.HomeDirectory),
		"",
	)

	return commands
}

func (rum *ResearchUserManager) generateEnvironmentSetupCommands(user *ResearchUserConfig) []string {
	commands := []string{
		"# Configure research user environment",
	}

	// Add environment variables to .bashrc
	if len(user.DefaultEnvironment) > 0 {
		commands = append(commands, fmt.Sprintf("cat >> %s/.bashrc << 'ENV_EOF'", user.HomeDirectory))
		commands = append(commands, "")
		commands = append(commands, "# CloudWorkstation Research User Environment")
		for key, value := range user.DefaultEnvironment {
			commands = append(commands, fmt.Sprintf("export %s='%s'", key, value))
		}
		commands = append(commands, "ENV_EOF")
		commands = append(commands, "")
	}

	return commands
}

// ProfileManagerAdapter adapts the existing profile.Manager to the ProfileManager interface
type ProfileManagerAdapter struct {
	manager interface {
		GetCurrentProfile() (*profile.Profile, error)
		GetProfile(name string) (*profile.Profile, error)
		UpdateProfile(profile *profile.Profile) error
	}
}

func NewProfileManagerAdapter(manager interface {
	GetCurrentProfile() (*profile.Profile, error)
	GetProfile(name string) (*profile.Profile, error)
	UpdateProfile(profile *profile.Profile) error
}) *ProfileManagerAdapter {
	return &ProfileManagerAdapter{manager: manager}
}

func (pma *ProfileManagerAdapter) GetCurrentProfile() (string, error) {
	profile, err := pma.manager.GetCurrentProfile()
	if err != nil {
		return "", err
	}
	return profile.Name, nil
}

func (pma *ProfileManagerAdapter) GetProfileConfig(profileID string) (interface{}, error) {
	return pma.manager.GetProfile(profileID)
}

func (pma *ProfileManagerAdapter) UpdateProfileConfig(profileID string, config interface{}) error {
	if profileConfig, ok := config.(*profile.Profile); ok {
		return pma.manager.UpdateProfile(profileConfig)
	}
	return fmt.Errorf("invalid profile config type")
}
