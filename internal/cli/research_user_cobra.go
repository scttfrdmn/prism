// Package cli provides the command-line interface for CloudWorkstation
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// ResearchUserCobraCommands handles research user-related commands with proper Cobra structure
type ResearchUserCobraCommands struct {
	app *App
}

// NewResearchUserCobraCommands creates a new research user commands handler
func NewResearchUserCobraCommands(app *App) *ResearchUserCobraCommands {
	return &ResearchUserCobraCommands{app: app}
}

// CreateResearchUserCommand creates the main research-user command with subcommands
func (ruc *ResearchUserCobraCommands) CreateResearchUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "research-user",
		Short: "Manage research users for collaborative environments",
		Long: `Manage research users that persist across instances and provide collaborative research environments.

Research users are designed for Phase 5A multi-user foundation with:
- Persistent identity across all instances
- Consistent UID/GID allocation for seamless EFS sharing
- SSH key management with Ed25519 and RSA support
- Integration with existing CloudWorkstation profile system
- Dual-user system supporting both system and research users

Research users complement template-created system users and enable:
- Collaborative research environments with proper permissions
- Persistent home directories on EFS volumes
- Consistent development environments across instance types
- Professional multi-user research computing workflows`,
		Example: `  # List all research users in the current profile
  cws research-user list

  # Create a new research user
  cws research-user create alice

  # Get details about a research user
  cws research-user info alice

  # Generate SSH keys for a research user
  cws research-user keys generate alice

  # List SSH keys for a research user
  cws research-user keys list alice

  # Update research user settings
  cws research-user update alice --full-name "Alice Smith" --email "alice@university.edu"`,
	}

	// Add subcommands
	cmd.AddCommand(
		ruc.createListCommand(),
		ruc.createCreateCommand(),
		ruc.createInfoCommand(),
		ruc.createUpdateCommand(),
		ruc.createDeleteCommand(),
		ruc.createKeysCommand(),
	)

	return cmd
}

// createListCommand lists all research users
func (ruc *ResearchUserCobraCommands) createListCommand() *cobra.Command {
	var showAll bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all research users",
		Long:  "Display all research users in the current profile with their basic information",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Make API request to list research users
			resp, err := ruc.app.apiClient.MakeRequest("GET", "/api/v1/research-users", nil)
			if err != nil {
				return fmt.Errorf("failed to list research users: %w", err)
			}

			var users []ResearchUserSummary
			if err := json.Unmarshal(resp, &users); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if len(users) == 0 {
				fmt.Println("No research users found")
				fmt.Println("\n💡 Tip: Use 'cws research-user create <username>' to create your first research user")
				return nil
			}

			if outputFormat == "json" {
				return json.NewEncoder(os.Stdout).Encode(users)
			}

			// Create table writer
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "USERNAME\tUID\tFULL NAME\tSSH KEYS\tCREATED\tLAST USED")
			fmt.Fprintln(w, "────────\t───\t─────────\t────────\t───────\t─────────")

			for _, user := range users {
				lastUsed := "never"
				if user.LastUsed != nil {
					lastUsed = user.LastUsed.Format("2006-01-02")
				}

				fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%s\t%s\n",
					user.Username,
					user.UID,
					user.FullName,
					len(user.SSHPublicKeys),
					user.CreatedAt.Format("2006-01-02"),
					lastUsed,
				)
			}

			w.Flush()

			fmt.Println("\n🔍 Research User Architecture:")
			fmt.Println("   • Each user has consistent UID/GID across all instances")
			fmt.Println("   • Home directories persist on EFS volumes for collaboration")
			fmt.Println("   • SSH keys enable secure, passwordless access")
			fmt.Printf("\n💡 Use 'cws research-user info <username>' for detailed information\n")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show research users from all profiles")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

// createCreateCommand creates a new research user
func (ruc *ResearchUserCobraCommands) createCreateCommand() *cobra.Command {
	var fullName string
	var email string
	var shell string
	var generateSSHKey bool
	var keyType string

	cmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a new research user",
		Long:  "Create a new research user with persistent identity and consistent UID/GID allocation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Validate username
			if err := validateUsername(username); err != nil {
				return fmt.Errorf("invalid username: %w", err)
			}

			fmt.Printf("🔄 Creating research user '%s'...\n", username)

			// Create request
			request := map[string]interface{}{
				"username": username,
			}

			// Make API request to create research user
			resp, err := ruc.app.apiClient.MakeRequest("POST", "/api/v1/research-users", request)
			if err != nil {
				return fmt.Errorf("failed to create research user: %w", err)
			}

			var user ResearchUserSummary
			if err := json.Unmarshal(resp, &user); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("✅ Successfully created research user!\n\n")
			fmt.Printf("📊 User Details:\n")
			fmt.Printf("   Username: %s\n", user.Username)
			fmt.Printf("   UID: %d\n", user.UID)
			fmt.Printf("   GID: %d\n", user.GID)
			fmt.Printf("   Full Name: %s\n", user.FullName)
			fmt.Printf("   Home Directory: %s\n", user.HomeDirectory)
			fmt.Printf("   Shell: %s\n", user.Shell)
			fmt.Printf("   SSH Keys: %d configured\n", len(user.SSHPublicKeys))

			if len(user.SSHPublicKeys) > 0 {
				fmt.Printf("\n🔑 SSH key automatically generated for secure access\n")
			}

			fmt.Printf("\n🎯 Research User Benefits:\n")
			fmt.Printf("   • Consistent identity across all instances (UID %d)\n", user.UID)
			fmt.Printf("   • Persistent home directory for research continuity\n")
			fmt.Printf("   • Seamless EFS collaboration with proper permissions\n")
			fmt.Printf("   • Professional research computing environment\n")

			fmt.Printf("\n💡 Next steps:\n")
			fmt.Printf("   • Generate SSH keys: cws research-user keys generate %s\n", username)
			fmt.Printf("   • View full details: cws research-user info %s\n", username)
			fmt.Printf("   • Launch instance with research user integration\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&fullName, "full-name", "", "Full name for the research user")
	cmd.Flags().StringVar(&email, "email", "", "Email address for the research user")
	cmd.Flags().StringVar(&shell, "shell", "/bin/bash", "Default shell for the research user")
	cmd.Flags().BoolVar(&generateSSHKey, "generate-ssh-key", true, "Automatically generate SSH key pair")
	cmd.Flags().StringVar(&keyType, "key-type", "ed25519", "SSH key type (ed25519, rsa)")

	return cmd
}

// createInfoCommand shows detailed information about a research user
func (ruc *ResearchUserCobraCommands) createInfoCommand() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "info <username>",
		Short: "Show detailed information about a research user",
		Long:  "Display comprehensive details about a research user including configuration and status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Make API request to get research user info
			resp, err := ruc.app.apiClient.MakeRequest("GET", fmt.Sprintf("/api/v1/research-users/%s", username), nil)
			if err != nil {
				return fmt.Errorf("failed to get research user info: %w", err)
			}

			var user ResearchUserSummary
			if err := json.Unmarshal(resp, &user); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if outputFormat == "json" {
				return json.NewEncoder(os.Stdout).Encode(user)
			}

			fmt.Printf("👤 Research User: %s\n", user.Username)
			fmt.Printf("═══════════════════════════════════════════════\n\n")

			fmt.Printf("📋 Basic Information:\n")
			fmt.Printf("   Username: %s\n", user.Username)
			fmt.Printf("   UID: %d (consistent across instances)\n", user.UID)
			fmt.Printf("   GID: %d (primary group)\n", user.GID)
			fmt.Printf("   Full Name: %s\n", user.FullName)
			fmt.Printf("   Email: %s\n", user.Email)
			fmt.Printf("   Shell: %s\n", user.Shell)

			fmt.Printf("\n🏠 Home Directory:\n")
			fmt.Printf("   Path: %s\n", user.HomeDirectory)
			if user.EFSVolumeID != "" {
				fmt.Printf("   EFS Volume: %s\n", user.EFSVolumeID)
				fmt.Printf("   EFS Mount Point: %s\n", user.EFSMountPoint)
			}

			fmt.Printf("\n🔑 SSH Access:\n")
			fmt.Printf("   SSH Keys: %d configured\n", len(user.SSHPublicKeys))
			if len(user.SSHPublicKeys) > 0 {
				fmt.Printf("   Key Fingerprint: %s\n", user.SSHKeyFingerprint)
			}

			fmt.Printf("\n👥 Groups & Permissions:\n")
			fmt.Printf("   Secondary Groups: %v\n", user.SecondaryGroups)
			fmt.Printf("   Sudo Access: %v\n", user.SudoAccess)
			fmt.Printf("   Docker Access: %v\n", user.DockerAccess)

			fmt.Printf("\n⏰ Activity:\n")
			fmt.Printf("   Created: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
			if user.LastUsed != nil {
				fmt.Printf("   Last Used: %s\n", user.LastUsed.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("   Last Used: never\n")
			}
			fmt.Printf("   Profile Owner: %s\n", user.ProfileOwner)

			if len(user.DefaultEnvironment) > 0 {
				fmt.Printf("\n🔧 Default Environment Variables:\n")
				for k, v := range user.DefaultEnvironment {
					fmt.Printf("   %s=%s\n", k, v)
				}
			}

			if user.DotfileRepo != "" {
				fmt.Printf("\n📄 Dotfiles Repository: %s\n", user.DotfileRepo)
			}

			fmt.Printf("\n🎯 Research User Architecture:\n")
			fmt.Printf("   • Persistent identity with UID %d across ALL instances\n", user.UID)
			fmt.Printf("   • Home directory persists on EFS for collaboration\n")
			fmt.Printf("   • Complements template system users (dual-user architecture)\n")
			fmt.Printf("   • Enables professional multi-user research computing\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

// createUpdateCommand updates research user settings
func (ruc *ResearchUserCobraCommands) createUpdateCommand() *cobra.Command {
	var fullName string
	var email string
	var shell string
	var addGroups []string
	var removeGroups []string
	var sudoAccess *bool
	var dockerAccess *bool

	cmd := &cobra.Command{
		Use:   "update <username>",
		Short: "Update research user settings",
		Long:  "Update configuration and settings for an existing research user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Build update request
			updates := make(map[string]interface{})
			if fullName != "" {
				updates["full_name"] = fullName
			}
			if email != "" {
				updates["email"] = email
			}
			if shell != "" {
				updates["shell"] = shell
			}
			if len(addGroups) > 0 {
				updates["add_groups"] = addGroups
			}
			if len(removeGroups) > 0 {
				updates["remove_groups"] = removeGroups
			}
			if sudoAccess != nil {
				updates["sudo_access"] = *sudoAccess
			}
			if dockerAccess != nil {
				updates["docker_access"] = *dockerAccess
			}

			if len(updates) == 0 {
				return fmt.Errorf("no updates specified. Use --help to see available options")
			}

			fmt.Printf("🔄 Updating research user '%s'...\n", username)

			// Make API request to update research user (currently returns method not implemented)
			_, err := ruc.app.apiClient.MakeRequest("PATCH", fmt.Sprintf("/api/v1/research-users/%s", username), updates)
			if err != nil {
				if strings.Contains(err.Error(), "not implemented") {
					fmt.Printf("⚠️  User update API not yet implemented in daemon\n")
					fmt.Printf("📝 Planned updates:\n")
					for key, value := range updates {
						fmt.Printf("   %s: %v\n", key, value)
					}
					fmt.Printf("\n💡 This feature will be available in a future CloudWorkstation release\n")
					return nil
				}
				return fmt.Errorf("failed to update research user: %w", err)
			}

			fmt.Printf("✅ Successfully updated research user!\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&fullName, "full-name", "", "Update full name")
	cmd.Flags().StringVar(&email, "email", "", "Update email address")
	cmd.Flags().StringVar(&shell, "shell", "", "Update default shell")
	cmd.Flags().StringSliceVar(&addGroups, "add-groups", nil, "Add secondary groups")
	cmd.Flags().StringSliceVar(&removeGroups, "remove-groups", nil, "Remove secondary groups")

	// Use custom flag parsing for boolean pointers
	cmd.Flags().BoolVar(new(bool), "sudo", false, "Enable sudo access")
	cmd.Flags().BoolVar(new(bool), "no-sudo", false, "Disable sudo access")
	cmd.Flags().BoolVar(new(bool), "docker", false, "Enable docker access")
	cmd.Flags().BoolVar(new(bool), "no-docker", false, "Disable docker access")

	// Custom flag processing
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flag("sudo").Changed && cmd.Flag("no-sudo").Changed {
			return fmt.Errorf("cannot use both --sudo and --no-sudo")
		}
		if cmd.Flag("docker").Changed && cmd.Flag("no-docker").Changed {
			return fmt.Errorf("cannot use both --docker and --no-docker")
		}

		if cmd.Flag("sudo").Changed {
			val := true
			sudoAccess = &val
		} else if cmd.Flag("no-sudo").Changed {
			val := false
			sudoAccess = &val
		}

		if cmd.Flag("docker").Changed {
			val := true
			dockerAccess = &val
		} else if cmd.Flag("no-docker").Changed {
			val := false
			dockerAccess = &val
		}

		return nil
	}

	return cmd
}

// createDeleteCommand deletes a research user
func (ruc *ResearchUserCobraCommands) createDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <username>",
		Short: "Delete a research user",
		Long:  "Delete a research user and remove associated configuration (WARNING: This action cannot be undone)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			if !force {
				fmt.Printf("⚠️  WARNING: This will permanently delete research user '%s'\n", username)
				fmt.Printf("   • User configuration will be removed\n")
				fmt.Printf("   • SSH keys will be deleted\n")
				fmt.Printf("   • Home directory data will remain on EFS\n")
				fmt.Printf("   • This action cannot be undone\n\n")
				fmt.Printf("Use --force to confirm deletion\n")
				return nil
			}

			fmt.Printf("🔄 Deleting research user '%s'...\n", username)

			// Make API request to delete research user
			_, err := ruc.app.apiClient.MakeRequest("DELETE", fmt.Sprintf("/api/v1/research-users/%s", username), nil)
			if err != nil {
				if strings.Contains(err.Error(), "not implemented") {
					fmt.Printf("⚠️  User deletion API not yet implemented in daemon\n")
					fmt.Printf("💡 This feature will be available in a future CloudWorkstation release\n")
					return nil
				}
				return fmt.Errorf("failed to delete research user: %w", err)
			}

			fmt.Printf("✅ Successfully deleted research user '%s'\n", username)

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force deletion without confirmation")

	return cmd
}

// createKeysCommand creates the SSH key management command
func (ruc *ResearchUserCobraCommands) createKeysCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage SSH keys for research users",
		Long:  "Manage SSH key pairs for research users to enable secure, passwordless access",
	}

	cmd.AddCommand(
		ruc.createKeysListCommand(),
		ruc.createKeysGenerateCommand(),
		ruc.createKeysAddCommand(),
		ruc.createKeysRemoveCommand(),
	)

	return cmd
}

// createKeysListCommand lists SSH keys for a research user
func (ruc *ResearchUserCobraCommands) createKeysListCommand() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list <username>",
		Short: "List SSH keys for a research user",
		Long:  "Display all SSH keys configured for the specified research user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Make API request to list SSH keys
			resp, err := ruc.app.apiClient.MakeRequest("GET", fmt.Sprintf("/api/v1/research-users/%s/ssh-key", username), nil)
			if err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}

			var keyResponse struct {
				Username string       `json:"username"`
				Keys     []SSHKeyInfo `json:"keys"`
			}
			if err := json.Unmarshal(resp, &keyResponse); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if outputFormat == "json" {
				return json.NewEncoder(os.Stdout).Encode(keyResponse)
			}

			if len(keyResponse.Keys) == 0 {
				fmt.Printf("No SSH keys found for research user '%s'\n", username)
				fmt.Printf("\n💡 Generate a key pair: cws research-user keys generate %s\n", username)
				return nil
			}

			fmt.Printf("🔑 SSH Keys for Research User: %s\n", username)
			fmt.Printf("═══════════════════════════════════════════════\n\n")

			// Create table writer
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TYPE\tFINGERPRINT\tCOMMENT\tCREATED")
			fmt.Fprintln(w, "────\t───────────\t───────\t───────")

			for _, key := range keyResponse.Keys {
				comment := key.Comment
				if comment == "" {
					comment = "<no comment>"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					key.KeyType,
					key.Fingerprint[:20]+"...", // Truncate for display
					comment,
					key.CreatedAt.Format("2006-01-02"),
				)
			}

			w.Flush()

			fmt.Printf("\n🔐 SSH Key Security:\n")
			fmt.Printf("   • All keys use modern cryptography (Ed25519 preferred)\n")
			fmt.Printf("   • Keys enable passwordless, secure access to instances\n")
			fmt.Printf("   • Consistent across all instances for seamless research workflows\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

// createKeysGenerateCommand generates a new SSH key pair
func (ruc *ResearchUserCobraCommands) createKeysGenerateCommand() *cobra.Command {
	var keyType string
	var comment string

	cmd := &cobra.Command{
		Use:   "generate <username>",
		Short: "Generate a new SSH key pair for a research user",
		Long:  "Generate and store a new SSH key pair for secure access to research instances",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			fmt.Printf("🔄 Generating %s SSH key pair for research user '%s'...\n", keyType, username)

			// Create request
			request := map[string]interface{}{
				"key_type": keyType,
			}

			// Make API request to generate SSH key
			resp, err := ruc.app.apiClient.MakeRequest("POST", fmt.Sprintf("/api/v1/research-users/%s/ssh-key", username), request)
			if err != nil {
				return fmt.Errorf("failed to generate SSH key: %w", err)
			}

			var keyResponse struct {
				Username    string `json:"username"`
				KeyType     string `json:"key_type"`
				PublicKey   string `json:"public_key"`
				Fingerprint string `json:"fingerprint"`
			}
			if err := json.Unmarshal(resp, &keyResponse); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("✅ Successfully generated SSH key pair!\n\n")
			fmt.Printf("🔑 Key Details:\n")
			fmt.Printf("   Type: %s\n", keyResponse.KeyType)
			fmt.Printf("   Fingerprint: %s\n", keyResponse.Fingerprint)
			fmt.Printf("   Username: %s\n", keyResponse.Username)

			fmt.Printf("\n🚀 Public Key:\n")
			fmt.Printf("   %s\n", keyResponse.PublicKey)

			fmt.Printf("\n💡 SSH Key Benefits:\n")
			fmt.Printf("   • Passwordless, secure access to all research instances\n")
			fmt.Printf("   • Modern %s cryptography for maximum security\n", strings.ToUpper(keyResponse.KeyType))
			fmt.Printf("   • Automatically configured across all instances\n")
			fmt.Printf("   • Enables professional research computing workflows\n")

			fmt.Printf("\n🔗 Next steps:\n")
			fmt.Printf("   • SSH keys are automatically installed on new instances\n")
			fmt.Printf("   • Connect with: ssh %s@<instance-ip>\n", username)
			fmt.Printf("   • View all keys: cws research-user keys list %s\n", username)

			return nil
		},
	}

	cmd.Flags().StringVar(&keyType, "key-type", "ed25519", "SSH key type (ed25519, rsa)")
	cmd.Flags().StringVar(&comment, "comment", "", "Comment for the SSH key")

	return cmd
}

// createKeysAddCommand adds an existing SSH public key
func (ruc *ResearchUserCobraCommands) createKeysAddCommand() *cobra.Command {
	var keyFile string
	var keyData string

	cmd := &cobra.Command{
		Use:   "add <username> [key-file-or-data]",
		Short: "Add an existing SSH public key for a research user",
		Long:  "Add an existing SSH public key from a file or direct input",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			var publicKey string

			if len(args) == 2 {
				// Key data provided as argument
				publicKey = args[1]
			} else if keyFile != "" {
				// Read from file
				data, err := os.ReadFile(keyFile)
				if err != nil {
					return fmt.Errorf("failed to read key file: %w", err)
				}
				publicKey = strings.TrimSpace(string(data))
			} else if keyData != "" {
				// Key data provided via flag
				publicKey = keyData
			} else {
				return fmt.Errorf("must provide key data via argument, --key-file, or --key-data")
			}

			fmt.Printf("🔄 Adding SSH public key for research user '%s'...\n", username)
			fmt.Printf("⚠️  SSH key addition API not yet implemented in daemon\n")
			fmt.Printf("📝 Key to be added:\n%s\n", publicKey)
			fmt.Printf("\n💡 This feature will be available in a future CloudWorkstation release\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&keyFile, "key-file", "", "Path to SSH public key file")
	cmd.Flags().StringVar(&keyData, "key-data", "", "SSH public key data directly")

	return cmd
}

// createKeysRemoveCommand removes an SSH key
func (ruc *ResearchUserCobraCommands) createKeysRemoveCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <username> <key-id-or-fingerprint>",
		Short: "Remove an SSH key for a research user",
		Long:  "Remove a specific SSH key by ID or fingerprint",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			keyIdentifier := args[1]

			if !force {
				fmt.Printf("⚠️  This will remove SSH key '%s' for user '%s'\n", keyIdentifier, username)
				fmt.Printf("Use --force to confirm removal\n")
				return nil
			}

			fmt.Printf("🔄 Removing SSH key for research user '%s'...\n", username)
			fmt.Printf("⚠️  SSH key removal API not yet implemented in daemon\n")
			fmt.Printf("📝 Key to be removed: %s\n", keyIdentifier)
			fmt.Printf("\n💡 This feature will be available in a future CloudWorkstation release\n")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force removal without confirmation")

	return cmd
}

// Helper types for API responses
type ResearchUserSummary struct {
	Username           string            `json:"username"`
	UID                int               `json:"uid"`
	GID                int               `json:"gid"`
	FullName           string            `json:"full_name"`
	Email              string            `json:"email"`
	HomeDirectory      string            `json:"home_directory"`
	EFSVolumeID        string            `json:"efs_volume_id"`
	EFSMountPoint      string            `json:"efs_mount_point"`
	Shell              string            `json:"shell"`
	SSHPublicKeys      []string          `json:"ssh_public_keys"`
	SSHKeyFingerprint  string            `json:"ssh_key_fingerprint"`
	SecondaryGroups    []string          `json:"secondary_groups"`
	SudoAccess         bool              `json:"sudo_access"`
	DockerAccess       bool              `json:"docker_access"`
	DefaultEnvironment map[string]string `json:"default_environment"`
	DotfileRepo        string            `json:"dotfile_repo"`
	CreatedAt          time.Time         `json:"created_at"`
	LastUsed           *time.Time        `json:"last_used"`
	ProfileOwner       string            `json:"profile_owner"`
}

type SSHKeyInfo struct {
	KeyID         string     `json:"key_id"`
	KeyType       string     `json:"key_type"`
	Fingerprint   string     `json:"fingerprint"`
	PublicKey     string     `json:"public_key"`
	Comment       string     `json:"comment"`
	CreatedAt     time.Time  `json:"created_at"`
	LastUsed      *time.Time `json:"last_used"`
	AutoGenerated bool       `json:"auto_generated"`
}

// validateUsername validates a username according to Unix standards
func validateUsername(username string) error {
	if len(username) == 0 {
		return fmt.Errorf("username cannot be empty")
	}
	if len(username) > 32 {
		return fmt.Errorf("username too long (max 32 characters)")
	}

	// Check first character (must be letter or underscore)
	first := username[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return fmt.Errorf("username must start with a letter or underscore")
	}

	// Check remaining characters
	for _, c := range username[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return fmt.Errorf("username contains invalid character '%c'", c)
		}
	}

	// Check for reserved names
	reserved := []string{"root", "daemon", "bin", "sys", "sync", "games", "man", "lp", "mail", "news", "uucp", "proxy", "www-data", "backup", "list", "irc", "gnats", "nobody"}
	for _, r := range reserved {
		if strings.EqualFold(username, r) {
			return fmt.Errorf("username '%s' is reserved", username)
		}
	}

	return nil
}
