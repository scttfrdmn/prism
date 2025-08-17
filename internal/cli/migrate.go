package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// AddMigrateCommand adds the migrate command to the CLI
func AddMigrateCommand(rootCmd *cobra.Command, config *Config) {
	// Migrate command
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from legacy state to profile-based state",
		Long:  `Migrate data from the legacy state file to the new profile-based state structure.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Parse flags
			profileName, _ := cmd.Flags().GetString("profile-name")

			// Create profile manager
			profileManager, err := createProfileManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Run migration
			fmt.Println("Starting migration of legacy state data...")
			fmt.Println("This will move your existing CloudWorkstation data into the new multi-profile system.")

			// Perform migration
			result, err := profileManager.MigrateFromLegacyState(profileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
				os.Exit(1)
			}

			// Print results
			fmt.Println("\nMigration successful!")
			fmt.Printf("Created profile: %s (ID: %s)\n", result.ProfileName, result.ProfileID)
			fmt.Printf("Migrated %d instances, %d EFS volumes, and %d EBS volumes\n",
				result.InstanceCount, result.VolumeCount, result.StorageCount)
			fmt.Printf("Original state file backed up to: %s\n", result.BackupPath)

			// Switch to the new profile
			if err := profileManager.SwitchProfile(result.ProfileID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to switch to the new profile: %v\n", err)
			} else {
				fmt.Printf("\nAutomatically switched to the new profile '%s'\n", result.ProfileName)
			}

			// Print next steps
			fmt.Println("\nNext steps:")
			fmt.Println("1. Run 'cws profiles list' to verify your migrated profile")
			fmt.Println("2. Run 'cws instances list' to check your migrated instances")
			fmt.Println("3. Run 'cws volumes list' to check your migrated storage volumes")
			fmt.Println("\nIf you need to revert the migration, you can restore the backup file:")
			fmt.Printf("   mv %s ~/.cloudworkstation/state.json\n", result.BackupPath)
		},
	}

	// Add flags
	migrateCmd.Flags().String("profile-name", fmt.Sprintf("Migrated Data (%s)",
		time.Now().Format("2006-01-02")), "Name for the migrated profile")

	// Add to root command
	rootCmd.AddCommand(migrateCmd)
}
