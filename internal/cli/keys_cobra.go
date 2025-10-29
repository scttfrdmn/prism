package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/prism/pkg/profile"
	"github.com/spf13/cobra"
)

// KeysCobraCommands handles SSH key management commands
type KeysCobraCommands struct {
	app *App
}

// NewKeysCobraCommands creates a new keys command handler
func NewKeysCobraCommands(app *App) *KeysCobraCommands {
	return &KeysCobraCommands{app: app}
}

// CreateKeysCommand creates the keys command with subcommands
func (k *KeysCobraCommands) CreateKeysCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage SSH keys",
		Long: `Manage Prism SSH keys for instance access.

Prism automatically manages SSH keys for secure instance access.
Use these commands to view, export, or manage your SSH keys.`,
	}

	cmd.AddCommand(k.createListCommand())
	cmd.AddCommand(k.createShowCommand())
	cmd.AddCommand(k.createExportCommand())
	cmd.AddCommand(k.createImportCommand())
	cmd.AddCommand(k.createPublicCommand())
	cmd.AddCommand(k.createDeleteCommand())

	return cmd
}

// createListCommand creates the 'keys list' command
func (k *KeysCobraCommands) createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all SSH keys",
		Long:  `List all Prism SSH keys with their associated instances.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return k.handleListKeys()
		},
	}
}

// createShowCommand creates the 'keys show' command
func (k *KeysCobraCommands) createShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <profile>",
		Short: "Show detailed key information",
		Long:  `Show detailed information about a specific SSH key including associated instances.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return k.handleShowKey(args[0])
		},
	}
}

// createExportCommand creates the 'keys export' command
func (k *KeysCobraCommands) createExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <profile>",
		Short: "Export SSH private key",
		Long: `Export the SSH private key for backup or team sharing.

⚠️  SECURITY WARNING: The exported key provides full access to instances.
Keep it secure and never commit it to version control.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			return k.handleExportKey(args[0], output)
		},
	}

	cmd.Flags().StringP("output", "o", "", "Output file path (required)")
	cmd.MarkFlagRequired("output")

	return cmd
}

// createImportCommand creates the 'keys import' command
func (k *KeysCobraCommands) createImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <profile>",
		Short: "Import an existing SSH key",
		Long: `Import an existing SSH private key for use with Prism.

This is useful for:
• Team sharing - import a shared team key
• Migration - import keys from other systems
• Backup restoration - import previously exported keys

The key must be in PEM format (RSA private key).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyFile, _ := cmd.Flags().GetString("key-file")
			region, _ := cmd.Flags().GetString("region")
			return k.handleImportKey(args[0], region, keyFile)
		},
	}

	cmd.Flags().StringP("key-file", "f", "", "Path to SSH private key file (required)")
	cmd.Flags().StringP("region", "r", "us-east-1", "AWS region for this key")
	cmd.MarkFlagRequired("key-file")

	return cmd
}

// createPublicCommand creates the 'keys public' command
func (k *KeysCobraCommands) createPublicCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "public <profile>",
		Short: "Show SSH public key",
		Long:  `Display the SSH public key for adding to other systems or services.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return k.handleShowPublicKey(args[0])
		},
	}
}

// createDeleteCommand creates the 'keys delete' command
func (k *KeysCobraCommands) createDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <profile>",
		Short: "Delete an SSH key",
		Long: `Delete an SSH key that is no longer needed.

⚠️  WARNING: This will prevent access to instances using this key.
Only delete keys that are not associated with any instances.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			return k.handleDeleteKey(args[0], force)
		},
	}

	cmd.Flags().Bool("force", false, "Force deletion even if instances are associated")

	return cmd
}

// handleListKeys lists all SSH keys
func (k *KeysCobraCommands) handleListKeys() error {
	keyManager, err := profile.NewSSHKeyManagerV2()
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	keys, err := keyManager.ListKeys()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println("No SSH keys found.")
		fmt.Println("\n💡 Keys are automatically created when launching instances")
		return nil
	}

	fmt.Printf("🔑 Prism SSH Keys\n\n")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROFILE\tAWS KEY NAME\tREGION\tINSTANCES\tCREATED")

	for _, key := range keys {
		instanceCount := len(key.Instances)
		createdStr := key.CreatedAt.Format("2006-01-02")

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			key.Profile,
			key.AWSKeyName,
			key.Region,
			instanceCount,
			createdStr,
		)
	}

	w.Flush()

	fmt.Printf("\n💡 Use 'prism keys show <profile>' for detailed information\n")
	fmt.Printf("💡 Use 'prism keys export <profile> -o <file>' to backup a key\n")

	return nil
}

// handleShowKey shows detailed information about a specific key
func (k *KeysCobraCommands) handleShowKey(profileName string) error {
	keyManager, err := profile.NewSSHKeyManagerV2()
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	metadata, err := keyManager.GetKeyMetadata(profileName)
	if err != nil {
		return fmt.Errorf("failed to get key metadata: %w", err)
	}

	fmt.Printf("🔑 SSH Key Details: %s\n\n", profileName)
	fmt.Printf("📋 Key Information:\n")
	fmt.Printf("   Profile:      %s\n", metadata.Profile)
	fmt.Printf("   AWS KeyName:  %s\n", metadata.AWSKeyName)
	fmt.Printf("   Region:       %s\n", metadata.Region)
	fmt.Printf("   Key Type:     %s\n", metadata.KeyType)
	fmt.Printf("   Created:      %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Local Path:   %s\n", metadata.LocalPath)

	fmt.Printf("\n📂 Associated Instances (%d):\n", len(metadata.Instances))
	if len(metadata.Instances) == 0 {
		fmt.Printf("   (none)\n")
	} else {
		for _, instanceName := range metadata.Instances {
			fmt.Printf("   • %s\n", instanceName)
		}
	}

	fmt.Printf("\n🔐 Public Key Fingerprint:\n")
	if metadata.PublicKey != "" {
		// Show first 80 chars of public key
		pubKey := metadata.PublicKey
		if len(pubKey) > 80 {
			pubKey = pubKey[:77] + "..."
		}
		fmt.Printf("   %s\n", pubKey)
	}

	fmt.Printf("\n💡 Commands:\n")
	fmt.Printf("   Export key:       prism keys export %s -o ~/backup/%s.pem\n", profileName, profileName)
	fmt.Printf("   Show public key:  prism keys public %s\n", profileName)

	return nil
}

// handleExportKey exports an SSH key to a file
func (k *KeysCobraCommands) handleExportKey(profileName, outputPath string) error {
	keyManager, err := profile.NewSSHKeyManagerV2()
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	// Expand ~ in output path
	if strings.HasPrefix(outputPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		outputPath = filepath.Join(homeDir, outputPath[2:])
	}

	// Check if output file already exists
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("⚠️  File already exists: %s\n", outputPath)
		fmt.Printf("Overwrite? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Export cancelled")
			return nil
		}
	}

	// Export the key
	if err := keyManager.ExportKey(profileName, outputPath); err != nil {
		return fmt.Errorf("failed to export key: %w", err)
	}

	fmt.Printf("✅ Key exported successfully\n\n")
	fmt.Printf("📁 Location: %s\n", outputPath)
	fmt.Printf("\n⚠️  SECURITY WARNING:\n")
	fmt.Printf("   • This key provides full access to your instances\n")
	fmt.Printf("   • Keep it secure and never commit to version control\n")
	fmt.Printf("   • Share only via secure channels (encrypted email, password manager)\n")
	fmt.Printf("\n💡 Usage:\n")
	fmt.Printf("   ssh -i %s ubuntu@<instance-ip>\n", outputPath)

	return nil
}

// handleImportKey imports an SSH key from a file
func (k *KeysCobraCommands) handleImportKey(profileName, region, keyFilePath string) error {
	keyManager, err := profile.NewSSHKeyManagerV2()
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	// Expand ~ in key file path
	if strings.HasPrefix(keyFilePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		keyFilePath = filepath.Join(homeDir, keyFilePath[2:])
	}

	// Check if key file exists
	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {
		return fmt.Errorf("key file not found: %s", keyFilePath)
	}

	// Import the key
	if err := keyManager.ImportKey(profileName, region, keyFilePath); err != nil {
		return fmt.Errorf("failed to import key: %w", err)
	}

	fmt.Printf("✅ Key imported successfully\n\n")
	fmt.Printf("📋 Details:\n")
	fmt.Printf("   Profile:     %s\n", profileName)
	fmt.Printf("   Region:      %s\n", region)
	fmt.Printf("   Source File: %s\n", keyFilePath)
	fmt.Printf("\n💡 The key is now ready to use with Prism\n")
	fmt.Printf("💡 View details: prism keys show %s\n", profileName)
	fmt.Printf("💡 Launch an instance: prism launch <template> <name>\n")

	return nil
}

// handleShowPublicKey displays the public key
func (k *KeysCobraCommands) handleShowPublicKey(profileName string) error {
	keyManager, err := profile.NewSSHKeyManagerV2()
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	publicKey, err := keyManager.GetPublicKeyContent(profileName)
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	fmt.Printf("🔑 Public SSH Key for '%s':\n\n", profileName)
	fmt.Println(publicKey)
	fmt.Printf("\n💡 You can add this public key to:\n")
	fmt.Printf("   • GitHub/GitLab for repository access\n")
	fmt.Printf("   • Other servers' ~/.ssh/authorized_keys\n")
	fmt.Printf("   • Team members' Prism instances\n")

	return nil
}

// handleDeleteKey deletes an SSH key
func (k *KeysCobraCommands) handleDeleteKey(profileName string, force bool) error {
	keyManager, err := profile.NewSSHKeyManagerV2()
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	// Get metadata to show what's being deleted
	metadata, err := keyManager.GetKeyMetadata(profileName)
	if err != nil {
		return fmt.Errorf("failed to get key metadata: %w", err)
	}

	// Show warning if instances are associated
	if len(metadata.Instances) > 0 && !force {
		fmt.Printf("⚠️  Key '%s' is associated with %d instance(s):\n", profileName, len(metadata.Instances))
		for _, instance := range metadata.Instances {
			fmt.Printf("   • %s\n", instance)
		}
		fmt.Printf("\n❌ Cannot delete key while instances are using it\n")
		fmt.Printf("💡 Options:\n")
		fmt.Printf("   1. Delete the instances first: prism delete <workspace-name>\n")
		fmt.Printf("   2. Force deletion (DANGEROUS): prism keys delete %s --force\n", profileName)
		return fmt.Errorf("key is in use")
	}

	// Confirmation prompt
	if !force {
		fmt.Printf("⚠️  Delete SSH key '%s'?\n", profileName)
		fmt.Printf("   AWS KeyName: %s\n", metadata.AWSKeyName)
		fmt.Printf("   Local Path:  %s\n", metadata.LocalPath)
		fmt.Printf("\nType the profile name to confirm: ")
		var confirmation string
		fmt.Scanln(&confirmation)

		if confirmation != profileName {
			fmt.Println("❌ Profile name doesn't match. Deletion cancelled.")
			return nil
		}
	}

	// Delete the key
	if err := keyManager.DeleteKey(profileName); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	fmt.Printf("✅ SSH key '%s' deleted successfully\n", profileName)

	if len(metadata.Instances) > 0 {
		fmt.Printf("\n⚠️  WARNING: %d instance(s) may no longer be accessible:\n", len(metadata.Instances))
		for _, instance := range metadata.Instances {
			fmt.Printf("   • %s\n", instance)
		}
	}

	return nil
}
