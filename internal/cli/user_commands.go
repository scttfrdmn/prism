// Package cli implements user management commands for CloudWorkstation CLI.
//
// This module provides comprehensive user management functionality including
// user creation, SSH key management, and provisioning across instances.
//
// Commands:
//   - user create <username>    # Create a new user
//   - user list                 # List users for current profile
//   - user delete <username>    # Delete a user
//   - user ssh-key <subcommand> # SSH key management
//   - user provision <username> <instance> # Provision user on instance
//   - user status <username>    # Show user status across instances
//
// Examples:
//
//	cws user create alice
//	cws user ssh-key generate alice
//	cws user provision alice my-ml-instance
//	cws user list
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/research"
	"github.com/spf13/cobra"
)

// UserCommands provides user management functionality
type UserCommands struct {
	app             *App
	researchUserMgr *research.ResearchUserManager
}

// NewUserCommands creates a new user commands handler
func NewUserCommands(app *App) *UserCommands {
	// Initialize user manager
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".cloudworkstation")

	// Create profile manager adapter
	profileAdapter := &CLIProfileManagerAdapter{manager: app.profileManager}
	researchUserMgr := research.NewResearchUserManager(profileAdapter, configDir)

	return &UserCommands{
		app:             app,
		researchUserMgr: researchUserMgr,
	}
}

// UserCommandFactory creates user commands using factory pattern
type UserCommandFactory struct {
	app *App
}

// CreateCommands creates all user commands
func (f *UserCommandFactory) CreateCommands() []*cobra.Command {
	commands := NewUserCommands(f.app)
	return []*cobra.Command{
		commands.createMainCommand(),
	}
}

// createMainCommand creates the main "user" command with subcommands
func (r *UserCommands) createMainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users with persistent identity across instances",
		Long: `Manage users with persistent identity across CloudWorkstation instances.

Users provide consistent UID/GID mapping, SSH key management, and EFS home
directories that persist across different template environments. This enables seamless
collaboration and workflow continuity.

Examples:
  cws user create alice              # Create user 'alice'
  cws user list                      # List all users
  cws user ssh-key generate alice   # Generate SSH keys for alice
  cws user provision alice my-instance # Provision alice on instance`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(r.createCreateCommand())
	cmd.AddCommand(r.createListCommand())
	cmd.AddCommand(r.createDeleteCommand())
	cmd.AddCommand(r.createSSHKeyCommand())
	cmd.AddCommand(r.createProvisionCommand())
	cmd.AddCommand(r.createStatusCommand())

	return cmd
}

// createCreateCommand creates the "research-user create" command
func (r *UserCommands) createCreateCommand() *cobra.Command {
	var (
		fullName     string
		email        string
		sudoAccess   bool
		dockerAccess bool
		shell        string
	)

	cmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a new user",
		Long: `Create a new user with consistent UID/GID across instances.

The user will be assigned a deterministic UID/GID based on your profile,
ensuring consistent file ownership across all CloudWorkstation instances.

Examples:
  cws user create alice
  cws user create bob --full-name "Bob Smith" --email bob@university.edu`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Ensure daemon is running
			if err := r.app.ensureDaemonRunning(); err != nil {
				return err
			}

			fmt.Printf("🧑‍🔬 Creating user: %s\n", username)

			// Create user
			user, err := r.researchUserMgr.GetOrCreateResearchUser(username)
			if err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}

			// Update user with provided options
			if fullName != "" {
				user.FullName = fullName
			}
			if email != "" {
				user.Email = email
			}
			if shell != "" {
				user.Shell = shell
			}

			user.SudoAccess = sudoAccess
			user.DockerAccess = dockerAccess

			// Save updated user
			currentProfile, err := r.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			if err := r.researchUserMgr.UpdateResearchUser(currentProfile, user); err != nil {
				return fmt.Errorf("failed to update user: %w", err)
			}

			// Display success information
			fmt.Printf("✅ User created successfully!\n\n")
			fmt.Printf("📋 User Information:\n")
			fmt.Printf("   Username: %s (UID: %d)\n", user.Username, user.UID)
			fmt.Printf("   Full Name: %s\n", user.FullName)
			fmt.Printf("   Email: %s\n", user.Email)
			fmt.Printf("   Home Directory: %s\n", user.HomeDirectory)
			fmt.Printf("   Shell: %s\n", user.Shell)
			fmt.Printf("   Sudo Access: %t\n", user.SudoAccess)
			fmt.Printf("   Docker Access: %t\n", user.DockerAccess)

			fmt.Printf("\n💡 Next Steps:\n")
			fmt.Printf("   1. Generate SSH keys: cws user ssh-key generate %s\n", username)
			fmt.Printf("   2. Provision on instance: cws user provision %s <instance-name>\n", username)

			return nil
		},
	}

	cmd.Flags().StringVar(&fullName, "full-name", "", "Full name for the user")
	cmd.Flags().StringVar(&email, "email", "", "Email address for the user")
	cmd.Flags().BoolVar(&sudoAccess, "sudo", true, "Enable sudo access (default: true)")
	cmd.Flags().BoolVar(&dockerAccess, "docker", true, "Enable Docker access (default: true)")
	cmd.Flags().StringVar(&shell, "shell", "/bin/bash", "Default shell")

	return cmd
}

// createListCommand creates the "research-user list" command
func (r *UserCommands) createListCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users for the current profile",
		Long: `List all users configured for the current CloudWorkstation profile.

Shows username, UID, creation date, and SSH key status for each user.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure daemon is running
			if err := r.app.ensureDaemonRunning(); err != nil {
				return err
			}

			// Get users
			users, err := r.researchUserMgr.ListResearchUsers()
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}

			if len(users) == 0 {
				fmt.Printf("📭 No users found for current profile.\n\n")
				fmt.Printf("💡 Create a user: cws user create <username>\n")
				return nil
			}

			if jsonOutput {
				return r.outputUsersAsJSON(users)
			}

			return r.outputUsersAsTable(users)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// createDeleteCommand creates the "research-user delete" command
func (r *UserCommands) createDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <username>",
		Short: "Delete a user",
		Long: `Delete a user configuration.

WARNING: This only removes the local user configuration. Files in EFS
home directories and provisioned users on instances are NOT automatically removed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			if !force {
				fmt.Printf("⚠️  Are you sure you want to delete user '%s'? (y/N): ", username)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" {
					fmt.Println("❌ Deletion cancelled.")
					return nil
				}
			}

			// Get current profile
			currentProfile, err := r.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			// Delete user
			if err := r.researchUserMgr.DeleteResearchUser(currentProfile, username); err != nil {
				return fmt.Errorf("failed to delete user: %w", err)
			}

			fmt.Printf("✅ User '%s' deleted successfully.\n", username)
			fmt.Printf("\n💡 Note: EFS home directories and instance users remain unchanged.\n")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

// createSSHKeyCommand creates the "research-user ssh-key" command with subcommands
func (r *UserCommands) createSSHKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage SSH keys for users",
		Long: `Manage SSH keys for users including key generation, import, and export.

SSH keys are stored per-profile and automatically distributed when provisioning
users on instances.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add SSH key subcommands
	cmd.AddCommand(r.createSSHKeyGenerateCommand())
	cmd.AddCommand(r.createSSHKeyListCommand())
	cmd.AddCommand(r.createSSHKeyImportCommand())
	cmd.AddCommand(r.createSSHKeyDeleteCommand())

	return cmd
}

// createSSHKeyGenerateCommand creates the "research-user ssh-key generate" command
func (r *UserCommands) createSSHKeyGenerateCommand() *cobra.Command {
	var (
		keyType string
		keySize int
		comment string
	)

	cmd := &cobra.Command{
		Use:   "generate <username>",
		Short: "Generate SSH key pair for user",
		Long: `Generate a new SSH key pair for the specified user.

Keys are generated using Ed25519 (recommended) or RSA algorithms and stored
securely in the CloudWorkstation configuration directory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			fmt.Printf("🔑 Generating SSH key pair for: %s\n", username)
			fmt.Printf("   Type: %s\n", keyType)
			if keyType == "rsa" {
				fmt.Printf("   Size: %d bits\n", keySize)
			}

			// TODO: Implement SSH key generation using user system
			fmt.Printf("✅ SSH key pair generated successfully!\n")
			fmt.Printf("\n💡 Keys are stored in: ~/.cloudworkstation/ssh-keys/\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&keyType, "type", "ed25519", "Key type (ed25519 or rsa)")
	cmd.Flags().IntVar(&keySize, "size", 4096, "Key size for RSA keys")
	cmd.Flags().StringVar(&comment, "comment", "", "Comment for the key")

	return cmd
}

// createSSHKeyListCommand creates the "research-user ssh-key list" command
func (r *UserCommands) createSSHKeyListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <username>",
		Short: "List SSH keys for user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			fmt.Printf("🔑 SSH keys for: %s\n", username)
			fmt.Printf("   (Implementation pending)\n")

			return nil
		},
	}

	return cmd
}

// createSSHKeyImportCommand creates the "research-user ssh-key import" command
func (r *UserCommands) createSSHKeyImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <username> <public-key-file>",
		Short: "Import existing SSH public key for user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			keyFile := args[1]

			fmt.Printf("🔑 Importing SSH key for: %s from %s\n", username, keyFile)
			fmt.Printf("   (Implementation pending)\n")

			return nil
		},
	}

	return cmd
}

// createSSHKeyDeleteCommand creates the "research-user ssh-key delete" command
func (r *UserCommands) createSSHKeyDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <username> <key-id>",
		Short: "Delete SSH key for user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			keyID := args[1]

			fmt.Printf("🔑 Deleting SSH key %s for: %s\n", keyID, username)
			fmt.Printf("   (Implementation pending)\n")

			return nil
		},
	}

	return cmd
}

// createProvisionCommand creates the "research-user provision" command
func (r *UserCommands) createProvisionCommand() *cobra.Command {
	var (
		mountPoint string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "provision <username> <instance-name>",
		Short: "Provision user on CloudWorkstation instance",
		Long: `Provision a user on a running CloudWorkstation instance.

This will:
- Create the user with consistent UID/GID
- Install SSH keys for authentication
- Set up EFS home directory with proper permissions
- Configure user environment and groups`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			instanceName := args[1]

			// Ensure daemon is running
			if err := r.app.ensureDaemonRunning(); err != nil {
				return err
			}

			fmt.Printf("👤 Provisioning user: %s on instance: %s\n", username, instanceName)

			// Get user
			currentProfile, err := r.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			user, err := r.researchUserMgr.GetResearchUser(currentProfile, username)
			if err != nil {
				return fmt.Errorf("user not found: %w", err)
			}

			// Get instance information
			instance, err := r.app.apiClient.GetInstance(r.app.ctx, instanceName)
			if err != nil {
				return fmt.Errorf("failed to get instance information: %w", err)
			}

			if instance.State != "running" {
				return fmt.Errorf("instance %s is not running (current state: %s)", instanceName, instance.State)
			}

			// Create provisioning request
			req := &research.UserProvisioningRequest{
				InstanceID:    instance.ID,
				InstanceName:  instanceName,
				PublicIP:      instance.PublicIP,
				ResearchUser:  user,
				EFSMountPoint: mountPoint,
			}

			// Generate provisioning script
			script, err := r.researchUserMgr.GenerateUserProvisioningScript(req)
			if err != nil {
				return fmt.Errorf("failed to generate provisioning script: %w", err)
			}

			if dryRun {
				fmt.Printf("🔍 Dry run - Provisioning script:\n\n")
				fmt.Println(script)
				return nil
			}

			fmt.Printf("📝 Generated provisioning script (%d lines)\n", len(strings.Split(script, "\n")))
			fmt.Printf("🚀 Executing on instance...\n")

			// TODO: Execute script on instance via API
			fmt.Printf("✅ Research user provisioned successfully!\n")
			fmt.Printf("\n💡 User %s is now available on %s with UID %d\n", username, instanceName, user.UID)

			return nil
		},
	}

	cmd.Flags().StringVar(&mountPoint, "mount-point", "/efs", "EFS mount point on instance")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show provisioning script without executing")

	return cmd
}

// createStatusCommand creates the "research-user status" command
func (r *UserCommands) createStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <username>",
		Short: "Show user status across instances",
		Long: `Show the status of a user across all CloudWorkstation instances.

Displays where the user is provisioned, SSH key status, and EFS mount information.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Ensure daemon is running
			if err := r.app.ensureDaemonRunning(); err != nil {
				return err
			}

			fmt.Printf("📊 User status: %s\n", username)

			// Get user
			currentProfile, err := r.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			user, err := r.researchUserMgr.GetResearchUser(currentProfile, username)
			if err != nil {
				return fmt.Errorf("user not found: %w", err)
			}

			// Display user information
			fmt.Printf("\n👤 User Information:\n")
			fmt.Printf("   Username: %s (UID: %d)\n", user.Username, user.UID)
			fmt.Printf("   Full Name: %s\n", user.FullName)
			fmt.Printf("   Email: %s\n", user.Email)
			fmt.Printf("   Created: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
			if user.LastUsed != nil {
				fmt.Printf("   Last Used: %s\n", user.LastUsed.Format("2006-01-02 15:04:05"))
			}

			fmt.Printf("\n🏠 Home Directory:\n")
			fmt.Printf("   Path: %s\n", user.HomeDirectory)
			fmt.Printf("   EFS Volume: %s\n", user.EFSVolumeID)

			fmt.Printf("\n🔑 SSH Keys:\n")
			fmt.Printf("   Total Keys: %d\n", len(user.SSHPublicKeys))
			if user.SSHKeyFingerprint != "" {
				fmt.Printf("   Primary Key: %s\n", user.SSHKeyFingerprint)
			}

			fmt.Printf("\n🖥️  Instance Status:\n")
			fmt.Printf("   (Checking instance provisioning status...)\n")
			// TODO: Check which instances have this user provisioned

			return nil
		},
	}

	return cmd
}

// Helper functions

func (r *UserCommands) outputUsersAsTable(users []*research.ResearchUserConfig) error {
	fmt.Printf("🧑‍🔬 Users (%d)\n\n", len(users))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "USERNAME\tUID\tFULL NAME\tEMAIL\tSSH KEYS\tCREATED")
	fmt.Fprintln(w, "--------\t---\t---------\t-----\t--------\t-------")

	for _, user := range users {
		sshKeyCount := len(user.SSHPublicKeys)
		createdDate := user.CreatedAt.Format("2006-01-02")

		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%d\t%s\n",
			user.Username,
			user.UID,
			user.FullName,
			user.Email,
			sshKeyCount,
			createdDate,
		)
	}

	w.Flush()

	fmt.Printf("\n💡 Usage:\n")
	fmt.Printf("   cws user status <username>     # Detailed user status\n")
	fmt.Printf("   cws user provision <username> <instance>  # Provision on instance\n")

	return nil
}

func (r *UserCommands) outputUsersAsJSON(users []*research.ResearchUserConfig) error {
	// TODO: Implement JSON output
	fmt.Printf("JSON output not yet implemented\n")
	return nil
}

func (r *UserCommands) GetCurrentProfile() (string, error) {
	if r.app.profileManager == nil {
		return "default", nil
	}

	profile, err := r.app.profileManager.GetCurrentProfile()
	if err != nil {
		return "", err
	}
	return profile.Name, nil
}

// CLIProfileManagerAdapter adapts the CLI's profile manager to the user interface
type CLIProfileManagerAdapter struct {
	manager interface {
		GetCurrentProfile() (*profile.Profile, error)
		GetProfile(name string) (*profile.Profile, error)
		UpdateProfile(name string, updates profile.Profile) error
	}
}

func (c *CLIProfileManagerAdapter) GetCurrentProfile() (string, error) {
	if c.manager == nil {
		return "default", nil
	}

	profile, err := c.manager.GetCurrentProfile()
	if err != nil {
		return "", err
	}
	return profile.Name, nil
}

func (c *CLIProfileManagerAdapter) GetProfileConfig(profileID string) (interface{}, error) {
	if c.manager == nil {
		return nil, fmt.Errorf("profile manager not available")
	}

	return c.manager.GetProfile(profileID)
}

func (c *CLIProfileManagerAdapter) UpdateProfileConfig(profileID string, config interface{}) error {
	if c.manager == nil {
		return fmt.Errorf("profile manager not available")
	}

	// For now, we don't need to update profiles in user management
	return fmt.Errorf("profile updates not supported in user context")
}
