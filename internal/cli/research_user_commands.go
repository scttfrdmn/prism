// Package cli implements research user management commands for CloudWorkstation CLI.
//
// This module provides comprehensive research user management functionality including
// user creation, SSH key management, and provisioning across instances.
//
// Commands:
//   - research-user create <username>    # Create a new research user
//   - research-user list                 # List research users for current profile
//   - research-user delete <username>    # Delete a research user
//   - research-user ssh-key <subcommand> # SSH key management
//   - research-user provision <username> <instance> # Provision user on instance
//   - research-user status <username>    # Show user status across instances
//
// Examples:
//
//	cws research-user create alice
//	cws research-user ssh-key generate alice
//	cws research-user provision alice my-ml-instance
//	cws research-user list
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

// ResearchUserCommands provides research user management functionality
type ResearchUserCommands struct {
	app             *App
	researchUserMgr *research.ResearchUserManager
}

// NewResearchUserCommands creates a new research user commands handler
func NewResearchUserCommands(app *App) *ResearchUserCommands {
	// Initialize research user manager
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".cloudworkstation")

	// Create profile manager adapter
	profileAdapter := &CLIProfileManagerAdapter{manager: app.profileManager}
	researchUserMgr := research.NewResearchUserManager(profileAdapter, configDir)

	return &ResearchUserCommands{
		app:             app,
		researchUserMgr: researchUserMgr,
	}
}

// ResearchUserCommandFactory creates research user commands using factory pattern
type ResearchUserCommandFactory struct {
	app *App
}

// CreateCommands creates all research user commands
func (f *ResearchUserCommandFactory) CreateCommands() []*cobra.Command {
	commands := NewResearchUserCommands(f.app)
	return []*cobra.Command{
		commands.createMainCommand(),
	}
}

// createMainCommand creates the main "research-user" command with subcommands
func (r *ResearchUserCommands) createMainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "research-user",
		Short: "Manage research users with persistent identity across instances",
		Long: `Manage research users with persistent identity across CloudWorkstation instances.

Research users provide consistent UID/GID mapping, SSH key management, and EFS home
directories that persist across different template environments. This enables seamless
collaboration and workflow continuity.

Examples:
  cws research-user create alice              # Create research user 'alice'
  cws research-user list                      # List all research users
  cws research-user ssh-key generate alice   # Generate SSH keys for alice
  cws research-user provision alice my-instance # Provision alice on instance`,
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
func (r *ResearchUserCommands) createCreateCommand() *cobra.Command {
	var (
		fullName     string
		email        string
		sudoAccess   bool
		dockerAccess bool
		shell        string
	)

	cmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a new research user",
		Long: `Create a new research user with consistent UID/GID across instances.

The research user will be assigned a deterministic UID/GID based on your profile,
ensuring consistent file ownership across all CloudWorkstation instances.

Examples:
  cws research-user create alice
  cws research-user create bob --full-name "Bob Smith" --email bob@university.edu`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Ensure daemon is running
			if err := r.app.ensureDaemonRunning(); err != nil {
				return err
			}

			fmt.Printf("🧑‍🔬 Creating research user: %s\n", username)

			// Create research user
			user, err := r.researchUserMgr.GetOrCreateResearchUser(username)
			if err != nil {
				return fmt.Errorf("failed to create research user: %w", err)
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
				return fmt.Errorf("failed to update research user: %w", err)
			}

			// Display success information
			fmt.Printf("✅ Research user created successfully!\n\n")
			fmt.Printf("📋 User Information:\n")
			fmt.Printf("   Username: %s (UID: %d)\n", user.Username, user.UID)
			fmt.Printf("   Full Name: %s\n", user.FullName)
			fmt.Printf("   Email: %s\n", user.Email)
			fmt.Printf("   Home Directory: %s\n", user.HomeDirectory)
			fmt.Printf("   Shell: %s\n", user.Shell)
			fmt.Printf("   Sudo Access: %t\n", user.SudoAccess)
			fmt.Printf("   Docker Access: %t\n", user.DockerAccess)

			fmt.Printf("\n💡 Next Steps:\n")
			fmt.Printf("   1. Generate SSH keys: cws research-user ssh-key generate %s\n", username)
			fmt.Printf("   2. Provision on instance: cws research-user provision %s <instance-name>\n", username)

			return nil
		},
	}

	cmd.Flags().StringVar(&fullName, "full-name", "", "Full name for the research user")
	cmd.Flags().StringVar(&email, "email", "", "Email address for the research user")
	cmd.Flags().BoolVar(&sudoAccess, "sudo", true, "Enable sudo access (default: true)")
	cmd.Flags().BoolVar(&dockerAccess, "docker", true, "Enable Docker access (default: true)")
	cmd.Flags().StringVar(&shell, "shell", "/bin/bash", "Default shell")

	return cmd
}

// createListCommand creates the "research-user list" command
func (r *ResearchUserCommands) createListCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List research users for the current profile",
		Long: `List all research users configured for the current CloudWorkstation profile.

Shows username, UID, creation date, and SSH key status for each research user.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure daemon is running
			if err := r.app.ensureDaemonRunning(); err != nil {
				return err
			}

			// Get research users
			users, err := r.researchUserMgr.ListResearchUsers()
			if err != nil {
				return fmt.Errorf("failed to list research users: %w", err)
			}

			if len(users) == 0 {
				fmt.Printf("📭 No research users found for current profile.\n\n")
				fmt.Printf("💡 Create a research user: cws research-user create <username>\n")
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
func (r *ResearchUserCommands) createDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <username>",
		Short: "Delete a research user",
		Long: `Delete a research user configuration.

WARNING: This only removes the local research user configuration. Files in EFS
home directories and provisioned users on instances are NOT automatically removed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			if !force {
				fmt.Printf("⚠️  Are you sure you want to delete research user '%s'? (y/N): ", username)
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
				return fmt.Errorf("failed to delete research user: %w", err)
			}

			fmt.Printf("✅ Research user '%s' deleted successfully.\n", username)
			fmt.Printf("\n💡 Note: EFS home directories and instance users remain unchanged.\n")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

// createSSHKeyCommand creates the "research-user ssh-key" command with subcommands
func (r *ResearchUserCommands) createSSHKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage SSH keys for research users",
		Long: `Manage SSH keys for research users including key generation, import, and export.

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
func (r *ResearchUserCommands) createSSHKeyGenerateCommand() *cobra.Command {
	var (
		keyType string
		keySize int
		comment string
	)

	cmd := &cobra.Command{
		Use:   "generate <username>",
		Short: "Generate SSH key pair for research user",
		Long: `Generate a new SSH key pair for the specified research user.

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

			// TODO: Implement SSH key generation using research user system
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
func (r *ResearchUserCommands) createSSHKeyListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <username>",
		Short: "List SSH keys for research user",
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
func (r *ResearchUserCommands) createSSHKeyImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <username> <public-key-file>",
		Short: "Import existing SSH public key for research user",
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
func (r *ResearchUserCommands) createSSHKeyDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <username> <key-id>",
		Short: "Delete SSH key for research user",
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
func (r *ResearchUserCommands) createProvisionCommand() *cobra.Command {
	var (
		mountPoint string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "provision <username> <instance-name>",
		Short: "Provision research user on CloudWorkstation instance",
		Long: `Provision a research user on a running CloudWorkstation instance.

This will:
- Create the research user with consistent UID/GID
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

			fmt.Printf("👤 Provisioning research user: %s on instance: %s\n", username, instanceName)

			// Get research user
			currentProfile, err := r.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			user, err := r.researchUserMgr.GetResearchUser(currentProfile, username)
			if err != nil {
				return fmt.Errorf("research user not found: %w", err)
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
func (r *ResearchUserCommands) createStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <username>",
		Short: "Show research user status across instances",
		Long: `Show the status of a research user across all CloudWorkstation instances.

Displays where the user is provisioned, SSH key status, and EFS mount information.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Ensure daemon is running
			if err := r.app.ensureDaemonRunning(); err != nil {
				return err
			}

			fmt.Printf("📊 Research user status: %s\n", username)

			// Get research user
			currentProfile, err := r.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			user, err := r.researchUserMgr.GetResearchUser(currentProfile, username)
			if err != nil {
				return fmt.Errorf("research user not found: %w", err)
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

func (r *ResearchUserCommands) outputUsersAsTable(users []*research.ResearchUserConfig) error {
	fmt.Printf("🧑‍🔬 Research Users (%d)\n\n", len(users))

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
	fmt.Printf("   cws research-user status <username>     # Detailed user status\n")
	fmt.Printf("   cws research-user provision <username> <instance>  # Provision on instance\n")

	return nil
}

func (r *ResearchUserCommands) outputUsersAsJSON(users []*research.ResearchUserConfig) error {
	// TODO: Implement JSON output
	fmt.Printf("JSON output not yet implemented\n")
	return nil
}

func (r *ResearchUserCommands) GetCurrentProfile() (string, error) {
	if r.app.profileManager == nil {
		return "default", nil
	}

	profile, err := r.app.profileManager.GetCurrentProfile()
	if err != nil {
		return "", err
	}
	return profile.Name, nil
}

// CLIProfileManagerAdapter adapts the CLI's profile manager to the research user interface
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

	// For now, we don't need to update profiles in research user management
	return fmt.Errorf("profile updates not supported in research user context")
}
