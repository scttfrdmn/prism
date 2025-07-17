package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile/export"
	"github.com/spf13/cobra"
)

// AddExportCommands adds the export and import commands to the profiles command
func AddExportCommands(profilesCmd *cobra.Command, config *Config) {
	// Export command
	exportCmd := &cobra.Command{
		Use:   "export [output-file]",
		Short: "Export profiles to file",
		Long: `Export CloudWorkstation profiles to a file for backup or sharing.
		
By default, credentials are not included in exports for security reasons.
Use the --include-credentials flag to include credentials (use with caution).

Examples:
  cws profiles export my-profiles.zip                # Export all profiles
  cws profiles export my-profiles.json --format json # Export in JSON format
  cws profiles export --profiles work,personal       # Export specific profiles`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			outputPath := args[0]

			// Get flags
			includeCredentials, _ := cmd.Flags().GetBool("include-credentials")
			includeInvitations, _ := cmd.Flags().GetBool("include-invitations")
			profilesFlag, _ := cmd.Flags().GetString("profiles")
			formatFlag, _ := cmd.Flags().GetString("format")
			passwordFlag, _ := cmd.Flags().GetString("password")

			// Create profile manager
			profileManager, err := createProfileManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Get all profiles
			allProfiles, err := profileManager.ListProfiles()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing profiles: %v\n", err)
				os.Exit(1)
			}

			// Filter profiles if specified
			var selectedProfiles []profile.Profile
			if profilesFlag != "" {
				profileNames := strings.Split(profilesFlag, ",")
				for _, prof := range allProfiles {
					for _, name := range profileNames {
						if prof.Name == name || prof.AWSProfile == name {
							selectedProfiles = append(selectedProfiles, prof)
							break
						}
					}
				}
			} else {
				selectedProfiles = allProfiles
			}

			// Filter out invitation profiles if not included
			if !includeInvitations {
				filteredProfiles := make([]profile.Profile, 0, len(selectedProfiles))
				for _, prof := range selectedProfiles {
					if prof.Type != "invitation" {
						filteredProfiles = append(filteredProfiles, prof)
					}
				}
				selectedProfiles = filteredProfiles
			}

			// Check if we have any profiles to export
			if len(selectedProfiles) == 0 {
				fmt.Println("No profiles found to export.")
				return
			}

			// Set export options
			options := export.ExportOptions{
				IncludeCredentials: includeCredentials,
				IncludeInvitations: includeInvitations,
				Password:           passwordFlag,
				Format:             formatFlag,
			}

			// Make sure directory exists
			if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
				os.Exit(1)
			}

			// Export profiles
			if err := export.ExportProfiles(profileManager, selectedProfiles, outputPath, options); err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting profiles: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully exported %d profiles to %s\n", len(selectedProfiles), outputPath)
		},
	}

	// Add flags for the export command
	exportCmd.Flags().Bool("include-credentials", false, "Include credentials in the export (use with caution)")
	exportCmd.Flags().Bool("include-invitations", true, "Include invitation profiles in the export")
	exportCmd.Flags().String("profiles", "", "Comma-separated list of profile names to export")
	exportCmd.Flags().String("format", "zip", "Export format (zip, json)")
	exportCmd.Flags().String("password", "", "Password to encrypt the export (only for zip format)")

	// Import command
	importCmd := &cobra.Command{
		Use:   "import [input-file]",
		Short: "Import profiles from file",
		Long: `Import CloudWorkstation profiles from a previously exported file.
		
By default, imported profiles will be renamed if they conflict with existing ones.
Use --mode to control how conflicts are handled (skip, overwrite, rename).

Examples:
  cws profiles import my-profiles.zip              # Import all profiles
  cws profiles import my-profiles.zip --mode skip  # Skip existing profiles
  cws profiles import --profiles work,personal     # Import specific profiles`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			inputPath := args[0]

			// Get flags
			modeFlag, _ := cmd.Flags().GetString("mode")
			profilesFlag, _ := cmd.Flags().GetString("profiles")
			importCredentialsFlag, _ := cmd.Flags().GetBool("import-credentials")
			passwordFlag, _ := cmd.Flags().GetString("password")

			// Parse import mode
			var importMode export.ImportMode
			switch modeFlag {
			case "skip":
				importMode = export.ImportModeSkip
			case "overwrite":
				importMode = export.ImportModeOverwrite
			case "rename":
				importMode = export.ImportModeRename
			default:
				fmt.Fprintf(os.Stderr, "Error: Invalid import mode '%s'. Must be one of: skip, overwrite, rename\n", modeFlag)
				os.Exit(1)
			}

			// Parse profile filter
			var profileFilter []string
			if profilesFlag != "" {
				profileFilter = strings.Split(profilesFlag, ",")
			}

			// Create profile manager
			profileManager, err := createProfileManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Set import options
			options := export.ImportOptions{
				ImportMode:        importMode,
				ProfileFilter:     profileFilter,
				ImportCredentials: importCredentialsFlag,
				Password:          passwordFlag,
			}

			// Import profiles
			result, err := export.ImportProfiles(profileManager, inputPath, options)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error importing profiles: %v\n", err)
				os.Exit(1)
			}

			// Show import results
			if !result.Success {
				fmt.Fprintf(os.Stderr, "Import failed: %s\n", result.Error)
				os.Exit(1)
			}

			fmt.Printf("Successfully imported %d profiles\n", result.ProfilesImported)

			// Show failed profiles if any
			if len(result.FailedProfiles) > 0 {
				fmt.Println("\nThe following profiles could not be imported:")
				for name, reason := range result.FailedProfiles {
					fmt.Printf("  %s: %s\n", name, reason)
				}
			}
		},
	}

	// Add flags for the import command
	importCmd.Flags().String("mode", "rename", "How to handle existing profiles (skip, overwrite, rename)")
	importCmd.Flags().String("profiles", "", "Comma-separated list of profile names to import")
	importCmd.Flags().Bool("import-credentials", false, "Import credentials if available")
	importCmd.Flags().String("password", "", "Password for encrypted imports")

	// Add commands to profiles command
	profilesCmd.AddCommand(exportCmd)
	profilesCmd.AddCommand(importCmd)
}