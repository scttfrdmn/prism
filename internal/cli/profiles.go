package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/spf13/cobra"
)

// AddProfileCommands adds profile-related commands to the CLI
func AddProfileCommands(rootCmd *cobra.Command, config *Config) {
	// This function is extended by AddExportCommands which adds export/import functionality
	// Profiles root command
	profilesCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage CloudWorkstation profiles",
		Long:  `Manage profiles for working with different AWS accounts and shared resources.`,
		Run: func(cmd *cobra.Command, args []string) {
			runProfilesMainCommand(config)
		},
	}
	rootCmd.AddCommand(profilesCmd)

	// Add individual commands
	profilesCmd.AddCommand(createListCommand(config))
	profilesCmd.AddCommand(createCurrentCommand(config))
	profilesCmd.AddCommand(createSwitchCommand(config))
	profilesCmd.AddCommand(createSetupCommand(config)) // Interactive wizard

	// Add profile management commands
	addCmd := &cobra.Command{
		Use:   "add [type] [name] [options]",
		Short: "Add a new profile",
		Long:  `Add a new profile for working with AWS accounts.`,
	}
	addCmd.AddCommand(createAddPersonalCommand(config))
	addCmd.AddCommand(createAddInvitationCommand(config))
	profilesCmd.AddCommand(addCmd)

	profilesCmd.AddCommand(createRemoveCommand(config))
	profilesCmd.AddCommand(createDeleteCommand(config)) // Alias for remove
	profilesCmd.AddCommand(createUpdateCommand(config))
	profilesCmd.AddCommand(createValidateCommand(config))
	profilesCmd.AddCommand(createAcceptInvitationCommand(config))
	profilesCmd.AddCommand(createRenameCommand(config))

	// Add invitation management
	createInvitationCommands(profilesCmd, config)

	// Add export and import commands
	AddExportCommands(profilesCmd, config)

	// Update the accept-invitation command to use the new invitation system
	updateAcceptInvitationCommand(profilesCmd, config)
}

// Command creation functions

// createListCommand creates the profiles list command
func createListCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Long:  `List all configured profiles with their details.`,
		Run: func(cmd *cobra.Command, args []string) {
			runListCommand(config)
		},
	}
}

// runListCommand handles the list command logic
func runListCommand(config *Config) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// List profiles with their IDs
	profilesWithIDs, err := profileManager.ListProfilesWithIDs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "list profiles"))
		os.Exit(1)
	}

	// Get current profile ID
	currentProfileID, err := profileManager.GetCurrentProfileID()
	if err != nil {
		currentProfileID = "" // No current profile
	}

	// Format and print profiles
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROFILE ID\tNAME\tTYPE\tAWS PROFILE\tREGION\tLAST USED")

	for _, pw := range profilesWithIDs {
		formatAndPrintProfileWithID(w, pw.ID, pw.Profile, currentProfileID)
	}
	_ = w.Flush()
}

// formatAndPrintProfileWithID formats and prints a single profile with its ID
func formatAndPrintProfileWithID(w *tabwriter.Writer, id string, p profile.Profile, currentProfileID string) {
	// Add marker for current profile
	marker := ""
	if id == currentProfileID {
		marker = "* "
	}

	// Format profile type
	profileType := "Personal"
	if p.Type == profile.ProfileTypeInvitation {
		profileType = "Invitation"
	}

	// Format last used time
	lastUsed := formatLastUsedTime(p.LastUsed)

	_, _ = fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\n",
		marker, id,
		p.Name,
		profileType,
		valueOrEmpty(p.AWSProfile),
		valueOrEmpty(p.Region),
		lastUsed,
	)
}

// formatLastUsedTime formats the last used time in a human-readable format
func formatLastUsedTime(lastUsed *time.Time) string {
	if lastUsed == nil || lastUsed.IsZero() {
		return "Never"
	}

	duration := time.Since(*lastUsed)
	if duration < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	} else if duration < 48*time.Hour {
		return "yesterday"
	} else {
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}
}

// createCurrentCommand creates the current profile command
func createCurrentCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current profile",
		Long:  `Display information about the currently active profile.`,
		Run: func(cmd *cobra.Command, args []string) {
			runCurrentCommand(config)
		},
	}
}

// runCurrentCommand handles the current command logic
func runCurrentCommand(config *Config) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Get current profile
	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "get current profile"))
		os.Exit(1)
	}

	printCurrentProfileInfo(currentProfile)
}

// printCurrentProfileInfo prints current profile information
func printCurrentProfileInfo(currentProfile *profile.Profile) {
	// Format profile type
	profileType := "Personal"
	if currentProfile.Type == profile.ProfileTypeInvitation {
		profileType = "Invitation"
	}

	fmt.Printf("Current profile: %s (%s)\n", currentProfile.AWSProfile, profileType)
	fmt.Printf("Name: %s\n", currentProfile.Name)

	if currentProfile.Type == profile.ProfileTypePersonal {
		fmt.Printf("AWS Profile: %s\n", currentProfile.AWSProfile)
		fmt.Printf("Region: %s\n", valueOrEmpty(currentProfile.Region))
	} else {
		fmt.Printf("Region: %s\n", valueOrEmpty(currentProfile.Region))
		fmt.Printf("Owner Account: %s\n", currentProfile.OwnerAccount)
	}
}

// createSwitchCommand creates the switch profile command
func createSwitchCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "switch [profile-id]",
		Short: "Switch to a different profile",
		Long:  `Activate a different profile for subsequent commands.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runSwitchCommand(config, args[0])
		},
	}
}

// runSwitchCommand handles the switch command logic
func runSwitchCommand(config *Config, profileID string) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Switch profile
	err = profileManager.SwitchProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "switch profile"))
		os.Exit(1)
	}

	// Get the profile
	prof, err := profileManager.GetProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "get profile for switch"))
		os.Exit(1)
	}

	fmt.Printf("Switched to profile '%s'\n", prof.Name)

	// Apply profile settings
	if prof.Region != "" {
		config.AWS.Region = prof.Region
		_ = saveConfig(config)
	}
}

// createAddPersonalCommand creates the add personal profile command
func createAddPersonalCommand(config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "personal [name] --aws-profile [aws-profile] --region [region]",
		Short: "Add a personal profile",
		Long:  `Add a new personal profile connected to your AWS account.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runAddPersonalCommand(config, cmd, args[0])
		},
	}
	cmd.Flags().String("aws-profile", "default", "AWS profile name in ~/.aws/credentials")
	cmd.Flags().String("region", "", "AWS region for this profile")
	return cmd
}

// runAddPersonalCommand handles the add personal command logic
func runAddPersonalCommand(config *Config, cmd *cobra.Command, name string) {
	awsProfile, _ := cmd.Flags().GetString("aws-profile")
	region, _ := cmd.Flags().GetString("region")

	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Create new profile
	prof := profile.Profile{
		Type:       profile.ProfileTypePersonal,
		Name:       name,
		AWSProfile: awsProfile,
		Region:     region,
		LastUsed:   func() *time.Time { t := time.Now(); return &t }(),
	}

	// Add profile
	err = profileManager.AddProfile(prof)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "add personal profile"))
		os.Exit(1)
	}

	fmt.Printf("Added personal profile '%s'\n", name)
}

// createAddInvitationCommand creates the add invitation profile command
func createAddInvitationCommand(config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invitation [name] --token [token] --owner [account] --region [region]",
		Short: "Add an invitation profile",
		Long:  `Add a new profile from an invitation to access another account.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runAddInvitationCommand(config, cmd, args[0])
		},
	}
	cmd.Flags().String("token", "", "Invitation token")
	cmd.Flags().String("owner", "", "Account owner")
	cmd.Flags().String("region", "", "AWS region for this profile")
	cmd.Flags().String("s3-config-path", "", "S3 path to configuration")
	_ = cmd.MarkFlagRequired("token")
	_ = cmd.MarkFlagRequired("owner")
	return cmd
}

// runAddInvitationCommand handles the add invitation command logic
func runAddInvitationCommand(config *Config, cmd *cobra.Command, name string) {
	token, _ := cmd.Flags().GetString("token")
	owner, _ := cmd.Flags().GetString("owner")
	region, _ := cmd.Flags().GetString("region")
	s3ConfigPath, _ := cmd.Flags().GetString("s3-config-path")

	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Create new profile
	prof := profile.Profile{
		Type:            profile.ProfileTypeInvitation,
		Name:            name,
		Region:          region,
		InvitationToken: token,
		OwnerAccount:    owner,
		S3ConfigPath:    s3ConfigPath,
		LastUsed:        func() *time.Time { t := time.Now(); return &t }(),
	}

	// Add profile
	err = profileManager.AddProfile(prof)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "add invitation profile"))
		os.Exit(1)
	}

	fmt.Printf("Added invitation profile '%s'\n", name)
}

// createRemoveCommand creates the remove profile command
func createRemoveCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "remove [profile-id]",
		Short: "Remove a profile",
		Long:  `Remove a profile from your configuration.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runRemoveCommand(config, args[0])
		},
	}
}

// createDeleteCommand creates the delete profile command (alias for remove)
func createDeleteCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "delete [profile-id]",
		Short: "Delete a profile (alias for remove)",
		Long:  `Delete a profile from your configuration. This is an alias for the remove command.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runRemoveCommand(config, args[0])
		},
	}
}

// runRemoveCommand handles the remove command logic
func runRemoveCommand(config *Config, profileID string) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Check if this is the current profile
	currentProfile, _ := profileManager.GetCurrentProfile()
	if currentProfile != nil && currentProfile.AWSProfile == profileID {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(fmt.Errorf("cannot remove the current profile"), "remove current profile"))
		os.Exit(1)
	}

	// Remove profile
	err = profileManager.RemoveProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "remove profile"))
		os.Exit(1)
	}

	fmt.Printf("Removed profile '%s'\n", profileID)
}

// createUpdateCommand creates the update profile command
func createUpdateCommand(config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [profile-id] [options]",
		Short: "Update an existing profile",
		Long:  `Update an existing profile's AWS profile, region, or name.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runUpdateCommand(config, cmd, args[0])
		},
	}
	cmd.Flags().String("aws-profile", "", "New AWS profile name in ~/.aws/credentials")
	cmd.Flags().String("region", "", "New AWS region for this profile")
	cmd.Flags().String("name", "", "New display name for the profile")
	return cmd
}

// runUpdateCommand handles the update command logic
func runUpdateCommand(config *Config, cmd *cobra.Command, profileID string) {
	awsProfile, _ := cmd.Flags().GetString("aws-profile")
	region, _ := cmd.Flags().GetString("region")
	name, _ := cmd.Flags().GetString("name")

	// Check if at least one flag is provided
	if awsProfile == "" && region == "" && name == "" {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(fmt.Errorf("at least one of --aws-profile, --region, or --name must be specified"), "update profile"))
		os.Exit(1)
	}

	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Get the current profile
	prof, err := profileManager.GetProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "get profile for update"))
		os.Exit(1)
	}

	// Create updated profile with new values
	updates := *prof

	// Track what's being updated for user feedback
	var changes []string

	if awsProfile != "" {
		updates.AWSProfile = awsProfile
		changes = append(changes, fmt.Sprintf("AWS profile: %s → %s", prof.AWSProfile, awsProfile))
	}
	if region != "" {
		updates.Region = region
		changes = append(changes, fmt.Sprintf("region: %s → %s", valueOrEmpty(prof.Region), region))
	}
	if name != "" {
		updates.Name = name
		changes = append(changes, fmt.Sprintf("name: %s → %s", prof.Name, name))
	}

	// Update the profile
	err = profileManager.UpdateProfile(profileID, updates)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "update profile"))
		os.Exit(1)
	}

	// Print success message with changes
	fmt.Printf("Updated profile '%s':\n", profileID)
	for _, change := range changes {
		fmt.Printf("  • %s\n", change)
	}
}

// createValidateCommand creates the validate profile command
func createValidateCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "validate [profile-id]",
		Short: "Validate a profile",
		Long:  `Check if a profile is valid and working correctly.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runValidateCommand(config, args[0])
		},
	}
}

// runValidateCommand handles the validate command logic
func runValidateCommand(config *Config, profileID string) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Get the profile
	prof, err := profileManager.GetProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "get profile for validation"))
		os.Exit(1)
	}

	// Create client with configuration
	client := client.NewClientWithOptions(config.Daemon.URL, client.Options{
		AWSProfile: config.AWS.Profile,
		AWSRegion:  config.AWS.Region,
	})

	// Test API access with current client
	ctx := context.Background()
	err = client.Ping(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "validate profile connection"))
		os.Exit(1)
	}

	// Success
	fmt.Printf("Profile '%s' is valid\n", prof.Name)

	// Check credentials
	if prof.Type == profile.ProfileTypeInvitation {
		// For invitation profiles, check token validity
		fmt.Println("Invitation token is valid")
	} else {
		// For personal profiles, check AWS credentials
		fmt.Println("AWS credentials are valid")
	}
}

// Helper functions

// valueOrEmpty returns a string or "-" if empty
func valueOrEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// createProfileManager creates a profile manager from config
func createProfileManager(config *Config) (*profile.ManagerEnhanced, error) {
	return profile.NewManagerEnhanced()
}

// createInvitationManager creates an invitation manager
func createInvitationManager(profileManager *profile.ManagerEnhanced) (*profile.InvitationManager, error) {
	return profile.NewInvitationManager(profileManager)
}

// createAcceptInvitationCommand creates the accept invitation command
func createAcceptInvitationCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "accept-invitation --token [token] --name [name] --owner [account] --region [region]",
		Short: "Accept an invitation",
		Long:  `Add a new profile from an invitation token.`,
		Run: func(cmd *cobra.Command, args []string) {
			runAcceptInvitationCommand(config, cmd)
		},
	}
}

// runAcceptInvitationCommand handles the accept invitation command logic
func runAcceptInvitationCommand(config *Config, cmd *cobra.Command) {
	token, _ := cmd.Flags().GetString("token")
	name, _ := cmd.Flags().GetString("name")
	owner, _ := cmd.Flags().GetString("owner")
	region, _ := cmd.Flags().GetString("region")
	s3ConfigPath, _ := cmd.Flags().GetString("s3-config-path")

	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Create new profile
	prof := profile.Profile{
		Type:            profile.ProfileTypeInvitation,
		Name:            name,
		Region:          region,
		InvitationToken: token,
		OwnerAccount:    owner,
		S3ConfigPath:    s3ConfigPath,
		LastUsed:        func() *time.Time { t := time.Now(); return &t }(),
	}

	// Add profile
	err = profileManager.AddProfile(prof)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "accept invitation"))
		os.Exit(1)
	}

	fmt.Printf("Accepted invitation and created profile '%s'\n", name)
}

// createSetupCommand creates the interactive profile setup command
func createSetupCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive profile setup wizard",
		Long: `Launch an interactive wizard to easily create and configure profiles.

This wizard guides you through:
  • Choosing between personal and invitation profiles
  • Setting up AWS credentials and regions
  • Validating your configuration
  • Switching to your new profile

Perfect for first-time users or when adding new AWS accounts.`,
		Run: func(cmd *cobra.Command, args []string) {
			runSetupCommand(config)
		},
	}
}

// runSetupCommand handles the interactive setup wizard
func runSetupCommand(config *Config) {
	wizard, err := NewProfileWizard(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile wizard"))
		os.Exit(1)
	}

	if err := wizard.RunInteractiveSetup(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "run profile setup wizard"))
		os.Exit(1)
	}
}

// runProfilesMainCommand handles the main profiles command with smart defaults
func runProfilesMainCommand(config *Config) {
	// Check if user has profiles (other than default)
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	profiles, err := profileManager.ListProfilesWithIDs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "list profiles"))
		os.Exit(1)
	}

	// Check if user only has the default profile (common case)
	hasOnlyDefault := len(profiles) == 1 && profiles[0].Profile.Default && profiles[0].Profile.Name == "AWS Default"

	if hasOnlyDefault {
		fmt.Printf("🚀 %s\n", color.CyanString("CloudWorkstation Profile Management"))
		fmt.Println()
		fmt.Println("You're using the default AWS profile - perfect! Most users don't need to change this.")
		fmt.Printf("Current setup: AWS profile '%s'", profiles[0].Profile.AWSProfile)
		if profiles[0].Profile.Region != "" {
			fmt.Printf(" in region '%s'", profiles[0].Profile.Region)
		}
		fmt.Println()
		fmt.Println()
		fmt.Println("💡 Your CloudWorkstation is ready to use! Try:")
		fmt.Println("   cws launch python-ml my-project")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		if promptYesNo(reader, "Would you like to add additional profiles for other AWS accounts or regions?", false) {
			runSetupCommand(config)
		} else {
			fmt.Println("👍 Great! Your default profile is all set.")
		}
	} else {
		// Show profile list when user has multiple profiles
		runListCommand(config)
	}
}

// promptYesNo prompts user for yes/no input
func promptYesNo(reader *bufio.Reader, prompt string, defaultValue bool) bool {
	defaultChar := "y/N"
	if defaultValue {
		defaultChar = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultChar)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultValue
	}

	return input == "y" || input == "yes"
}

// createRenameCommand creates the rename profile command
func createRenameCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "rename [profile-id] [new-name]",
		Short: "Rename a profile",
		Long:  `Change the display name of a profile.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			runRenameCommand(config, args[0], args[1])
		},
	}
}

// runRenameCommand handles the rename command logic
func runRenameCommand(config *Config, profileID, newName string) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Get the profile
	prof, err := profileManager.GetProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "get profile for rename"))
		os.Exit(1)
	}

	// Update profile
	updates := *prof
	updates.Name = newName

	err = profileManager.UpdateProfile(profileID, updates)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "rename profile"))
		os.Exit(1)
	}

	fmt.Printf("Renamed profile to '%s'\n", newName)
}

// createInvitationCommands creates invitation management commands
func createInvitationCommands(profilesCmd *cobra.Command, config *Config) {
	invitationsCmd := &cobra.Command{
		Use:   "invitations",
		Short: "Manage shared access invitations",
		Long:  `Create and manage invitations for sharing access to your CloudWorkstation resources.`,
	}
	profilesCmd.AddCommand(invitationsCmd)

	// Add individual invitation commands
	invitationsCmd.AddCommand(createInvitationCreateCommand(config))
	invitationsCmd.AddCommand(createInvitationListCommand(config))
	invitationsCmd.AddCommand(createInvitationRevokeCommand(config))

	// Batch commands temporarily disabled during Phase 1 profile system simplification
	// AddBatchInvitationCommands(invitationsCmd, config)
	// AddBatchDeviceCommands(invitationsCmd, config)
	// AddBatchConfigCommands(invitationsCmd, config)
}

// createInvitationCreateCommand creates the invitation create command
func createInvitationCreateCommand(config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name] --type [read_only|read_write|admin] --valid-days [days]",
		Short: "Create a new invitation",
		Long:  `Generate a new invitation that can be shared with others to grant access to your CloudWorkstation resources.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runInvitationCreateCommand(config, cmd, args[0])
		},
	}
	cmd.Flags().String("type", "read_only", "Type of access (read_only, read_write, or admin)")
	cmd.Flags().Int("valid-days", 30, "Number of days the invitation is valid")
	cmd.Flags().String("s3-config", "", "Optional S3 path to configuration")

	// Basic policy restriction flags (open source feature)
	cmd.Flags().StringSlice("template-whitelist", []string{}, "Allowed templates (comma-separated)")
	cmd.Flags().StringSlice("template-blacklist", []string{}, "Forbidden templates (comma-separated)")
	cmd.Flags().StringSlice("max-instance-types", []string{}, "Maximum allowed instance types (comma-separated)")
	cmd.Flags().StringSlice("forbidden-regions", []string{}, "Forbidden AWS regions (comma-separated)")
	cmd.Flags().Float64("max-hourly-cost", 0, "Maximum hourly cost limit (0 = no limit)")
	cmd.Flags().Float64("max-daily-budget", 0, "Maximum daily budget limit (0 = no limit)")

	return cmd
}

// runInvitationCreateCommand handles the invitation create command logic
func runInvitationCreateCommand(config *Config, cmd *cobra.Command, name string) {
	// Parse command flags
	flags := parseInvitationFlags(cmd)

	// Initialize managers
	invitationManager, err := initializeInvitationManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "initialize invitation manager"))
		os.Exit(1)
	}

	// Create the invitation
	invitation, err := createInvitationWithType(invitationManager, name, flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create invitation"))
		os.Exit(1)
	}

	// Apply and display policy restrictions
	applyPolicyRestrictions(invitation, flags)

	// Print invitation details
	printInvitationDetails(invitation)
}

// invitationFlags holds parsed command line flags
type invitationFlags struct {
	invType           string
	validDays         int
	s3ConfigPath      string
	templateWhitelist []string
	templateBlacklist []string
	maxInstanceTypes  []string
	forbiddenRegions  []string
	maxHourlyCost     float64
	maxDailyBudget    float64
}

// parseInvitationFlags extracts all flags from the command
func parseInvitationFlags(cmd *cobra.Command) *invitationFlags {
	invType, _ := cmd.Flags().GetString("type")
	validDays, _ := cmd.Flags().GetInt("valid-days")
	s3ConfigPath, _ := cmd.Flags().GetString("s3-config")
	templateWhitelist, _ := cmd.Flags().GetStringSlice("template-whitelist")
	templateBlacklist, _ := cmd.Flags().GetStringSlice("template-blacklist")
	maxInstanceTypes, _ := cmd.Flags().GetStringSlice("max-instance-types")
	forbiddenRegions, _ := cmd.Flags().GetStringSlice("forbidden-regions")
	maxHourlyCost, _ := cmd.Flags().GetFloat64("max-hourly-cost")
	maxDailyBudget, _ := cmd.Flags().GetFloat64("max-daily-budget")

	return &invitationFlags{
		invType:           invType,
		validDays:         validDays,
		s3ConfigPath:      s3ConfigPath,
		templateWhitelist: templateWhitelist,
		templateBlacklist: templateBlacklist,
		maxInstanceTypes:  maxInstanceTypes,
		forbiddenRegions:  forbiddenRegions,
		maxHourlyCost:     maxHourlyCost,
		maxDailyBudget:    maxDailyBudget,
	}
}

// initializeInvitationManager creates and initializes the invitation manager
func initializeInvitationManager(config *Config) (*profile.InvitationManager, error) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		return nil, fmt.Errorf("create profile manager: %w", err)
	}

	// Create invitation manager
	invitationManager, err := createInvitationManager(profileManager)
	if err != nil {
		return nil, fmt.Errorf("create invitation manager: %w", err)
	}

	return invitationManager, nil
}

// createInvitationWithType creates an invitation with the specified type
func createInvitationWithType(invitationManager *profile.InvitationManager, name string, flags *invitationFlags) (*profile.InvitationToken, error) {
	// Validate invitation type
	invitationType, err := parseInvitationType(flags.invType)
	if err != nil {
		return nil, fmt.Errorf("parse invitation type: %w", err)
	}

	// Create the invitation
	invitation, err := invitationManager.CreateInvitation(name, invitationType, flags.validDays, flags.s3ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	return invitation, nil
}

// applyPolicyRestrictions applies and displays policy restrictions if any are specified
func applyPolicyRestrictions(invitation *profile.InvitationToken, flags *invitationFlags) {
	if !hasPolicyRestrictions(flags) {
		return
	}

	// Apply policy restrictions
	invitation.PolicyRestrictions = &profile.BasicPolicyRestrictions{
		TemplateWhitelist: flags.templateWhitelist,
		TemplateBlacklist: flags.templateBlacklist,
		MaxInstanceTypes:  flags.maxInstanceTypes,
		ForbiddenRegions:  flags.forbiddenRegions,
		MaxHourlyCost:     flags.maxHourlyCost,
		MaxDailyBudget:    flags.maxDailyBudget,
	}

	// Display applied restrictions
	displayPolicyRestrictions(flags)
}

// hasPolicyRestrictions checks if any policy restrictions are specified
func hasPolicyRestrictions(flags *invitationFlags) bool {
	return len(flags.templateWhitelist) > 0 || len(flags.templateBlacklist) > 0 ||
		len(flags.maxInstanceTypes) > 0 || len(flags.forbiddenRegions) > 0 ||
		flags.maxHourlyCost > 0 || flags.maxDailyBudget > 0
}

// displayPolicyRestrictions shows the applied policy restrictions
func displayPolicyRestrictions(flags *invitationFlags) {
	fmt.Println(color.YellowString("Policy restrictions applied:"))

	if len(flags.templateWhitelist) > 0 {
		fmt.Printf("  - Allowed templates: %v\n", flags.templateWhitelist)
	}
	if len(flags.templateBlacklist) > 0 {
		fmt.Printf("  - Forbidden templates: %v\n", flags.templateBlacklist)
	}
	if len(flags.maxInstanceTypes) > 0 {
		fmt.Printf("  - Max instance types: %v\n", flags.maxInstanceTypes)
	}
	if len(flags.forbiddenRegions) > 0 {
		fmt.Printf("  - Forbidden regions: %v\n", flags.forbiddenRegions)
	}
	if flags.maxHourlyCost > 0 {
		fmt.Printf("  - Max hourly cost: $%.2f\n", flags.maxHourlyCost)
	}
	if flags.maxDailyBudget > 0 {
		fmt.Printf("  - Max daily budget: $%.2f\n", flags.maxDailyBudget)
	}

	fmt.Println()
}

// parseInvitationType parses and validates invitation type
func parseInvitationType(invType string) (profile.InvitationType, error) {
	switch invType {
	case "read_only":
		return profile.InvitationTypeReadOnly, nil
	case "read_write":
		return profile.InvitationTypeReadWrite, nil
	case "admin":
		return profile.InvitationTypeAdmin, nil
	default:
		return "", fmt.Errorf("invalid invitation type '%s'. Must be one of: read_only, read_write, admin", invType)
	}
}

// printInvitationDetails prints invitation creation details
func printInvitationDetails(invitation *profile.InvitationToken) {
	// Encode the invitation for sharing
	encodedInvitation, err := invitation.EncodeToString()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "encode invitation"))
		os.Exit(1)
	}

	// Print the invitation details
	fmt.Println("\nInvitation Created Successfully")
	fmt.Printf("Name: %s\n", invitation.Name)
	fmt.Printf("Type: %s\n", invitation.Type)
	fmt.Printf("Expires: %s (in %s)\n", invitation.Expires.Format("Jan 2, 2006"),
		invitation.GetExpirationDuration().Round(time.Hour))

	// Print the shareable token
	fmt.Println("\nShare this invitation code with the recipient:")
	fmt.Printf("\n%s\n", color.GreenString(encodedInvitation))

	// Print acceptance instructions
	fmt.Println("\nThey can accept it with:")
	fmt.Printf("cws profiles accept-invitation --encoded '%s' --name 'Collaboration'\n", encodedInvitation)
}

// createInvitationListCommand creates the invitation list command
func createInvitationListCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active invitations",
		Long:  `Show all active invitations you've created.`,
		Run: func(cmd *cobra.Command, args []string) {
			runInvitationListCommand(config)
		},
	}
}

// runInvitationListCommand handles the invitation list command logic
func runInvitationListCommand(config *Config) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Create invitation manager
	invitationManager, err := createInvitationManager(profileManager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create invitation manager"))
		os.Exit(1)
	}

	// Get invitations
	invitations := invitationManager.ListInvitations()
	if len(invitations) == 0 {
		fmt.Println("No active invitations found.")
		return
	}

	// Display invitations
	printInvitationsList(invitations)
}

// printInvitationsList prints the list of invitations
func printInvitationsList(invitations []profile.InvitationToken) {
	fmt.Printf("Found %d active invitation(s):\n\n", len(invitations))

	for i, invitation := range invitations {
		fmt.Printf("[%d] %s\n", i+1, invitation.Name)
		fmt.Printf("  - Token: %s\n", invitation.Token)
		fmt.Printf("  - Type: %s\n", invitation.Type)
		fmt.Printf("  - Created: %s\n", invitation.Created.Format("Jan 2, 2006"))
		fmt.Printf("  - Expires: %s (in %s)\n", invitation.Expires.Format("Jan 2, 2006"),
			invitation.GetExpirationDuration().Round(time.Hour))
		if i < len(invitations)-1 {
			fmt.Println()
		}
	}
}

// createInvitationRevokeCommand creates the invitation revoke command
func createInvitationRevokeCommand(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke [token]",
		Short: "Revoke an invitation",
		Long:  `Revoke an active invitation so it can no longer be used.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runInvitationRevokeCommand(config, args[0])
		},
	}
}

// runInvitationRevokeCommand handles the invitation revoke command logic
func runInvitationRevokeCommand(config *Config, token string) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create profile manager"))
		os.Exit(1)
	}

	// Create invitation manager
	invitationManager, err := createInvitationManager(profileManager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create invitation manager"))
		os.Exit(1)
	}

	// Get invitation details first
	invitation, err := invitationManager.GetInvitation(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "get invitation details"))
		os.Exit(1)
	}

	// Revoke the invitation
	err = invitationManager.RevokeInvitation(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "revoke invitation"))
		os.Exit(1)
	}

	fmt.Printf("Successfully revoked invitation '%s' (%s)\n", invitation.Name, token)
}

// updateAcceptInvitationCommand updates the accept-invitation command to use new system
func updateAcceptInvitationCommand(profilesCmd *cobra.Command, config *Config) {
	// Find the accept-invitation command (it should be the 4th from the end)
	commands := profilesCmd.Commands()
	var acceptCmd *cobra.Command
	for _, cmd := range commands {
		if cmd.Use == "accept-invitation --token [token] --name [name] --owner [account] --region [region]" {
			acceptCmd = cmd
			break
		}
	}

	if acceptCmd == nil {
		return // Command not found, skip update
	}

	acceptCmd.Flags().String("encoded", "", "Encoded invitation string")
	_ = acceptCmd.MarkFlagRequired("encoded")
	acceptCmd.Run = func(cmd *cobra.Command, args []string) {
		encoded, _ := cmd.Flags().GetString("encoded")
		name, _ := cmd.Flags().GetString("name")

		// Create profile manager
		profileManager, err := createProfileManager(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create invitation manager
		invitationManager, err := createInvitationManager(profileManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Add the invitation to profiles
		if err := invitationManager.AddToProfile(encoded, name); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "accept encoded invitation"))
			os.Exit(1)
		}

		fmt.Printf("Accepted invitation and created profile '%s'\n", name)
	}
}
