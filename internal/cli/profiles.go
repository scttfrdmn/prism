package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
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
	}
	rootCmd.AddCommand(profilesCmd)

	// Add individual commands
	profilesCmd.AddCommand(createListCommand(config))
	profilesCmd.AddCommand(createCurrentCommand(config))
	profilesCmd.AddCommand(createSwitchCommand(config))

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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// List profiles
	profiles, err := profileManager.ListProfiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing profiles: %v\n", err)
		os.Exit(1)
	}

	// Get current profile
	currentProfile, err := profileManager.GetCurrentProfile()
	currentProfileID := ""
	if err == nil {
		currentProfileID = currentProfile.AWSProfile
	}

	// Format and print profiles
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROFILE ID\tNAME\tTYPE\tAWS PROFILE\tREGION\tLAST USED")

	for _, p := range profiles {
		formatAndPrintProfile(w, p, currentProfileID)
	}
	_ = w.Flush()
}

// formatAndPrintProfile formats and prints a single profile
func formatAndPrintProfile(w *tabwriter.Writer, p profile.Profile, currentProfileID string) {
	// Add marker for current profile
	marker := ""
	if p.AWSProfile == currentProfileID {
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
		marker, p.AWSProfile,
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get current profile
	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current profile: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Switch profile
	err = profileManager.SwitchProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error switching profile: %v\n", err)
		os.Exit(1)
	}

	// Get the profile
	prof, err := profileManager.GetProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting profile: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error adding profile: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error adding profile: %v\n", err)
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

// runRemoveCommand handles the remove command logic
func runRemoveCommand(config *Config, profileID string) {
	// Create profile manager
	profileManager, err := createProfileManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if this is the current profile
	currentProfile, _ := profileManager.GetCurrentProfile()
	if currentProfile != nil && currentProfile.AWSProfile == profileID {
		fmt.Fprintf(os.Stderr, "Error: Cannot remove the current profile. Switch to another profile first.\n")
		os.Exit(1)
	}

	// Remove profile
	err = profileManager.RemoveProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error removing profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed profile '%s'\n", profileID)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get the profile
	prof, err := profileManager.GetProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting profile: %v\n", err)
		os.Exit(1)
	}

	// Create client with configuration
	client := api.NewClientWithOptions(config.Daemon.URL, client.Options{
		AWSProfile: config.AWS.Profile,
		AWSRegion:  config.AWS.Region,
	})

	// Test API access with current client
	ctx := context.Background()
	err = client.Ping(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Profile validation failed: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error accepting invitation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Accepted invitation and created profile '%s'\n", name)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get the profile
	prof, err := profileManager.GetProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting profile: %v\n", err)
		os.Exit(1)
	}

	// Update profile
	updates := *prof
	updates.Name = newName

	err = profileManager.UpdateProfile(profileID, updates)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error renaming profile: %v\n", err)
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
	return cmd
}

// runInvitationCreateCommand handles the invitation create command logic
func runInvitationCreateCommand(config *Config, cmd *cobra.Command, name string) {
	invType, _ := cmd.Flags().GetString("type")
	validDays, _ := cmd.Flags().GetInt("valid-days")
	s3ConfigPath, _ := cmd.Flags().GetString("s3-config")

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

	// Validate invitation type
	invitationType, err := parseInvitationType(invType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create the invitation
	invitation, err := invitationManager.CreateInvitation(name, invitationType, validDays, s3ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating invitation: %v\n", err)
		os.Exit(1)
	}

	// Print invitation details
	printInvitationDetails(invitation)
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
		fmt.Fprintf(os.Stderr, "Error encoding invitation: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create invitation manager
	invitationManager, err := createInvitationManager(profileManager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create invitation manager
	invitationManager, err := createInvitationManager(profileManager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get invitation details first
	invitation, err := invitationManager.GetInvitation(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Revoke the invitation
	err = invitationManager.RevokeInvitation(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error revoking invitation: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "Error accepting invitation: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Accepted invitation and created profile '%s'\n", name)
	}
}

