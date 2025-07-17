package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/spf13/cobra"
	"github.com/fatih/color"
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
	
	// List profiles
	profilesCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Long:  `List all configured profiles with their details.`,
		Run: func(cmd *cobra.Command, args []string) {
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
			fmt.Fprintln(w, "PROFILE ID\tNAME\tTYPE\tAWS PROFILE\tREGION\tLAST USED")
			
			for _, p := range profiles {
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
				lastUsed := "Never"
				if !p.LastUsed.IsZero() {
					duration := time.Since(p.LastUsed)
					if duration < time.Hour {
						lastUsed = fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
					} else if duration < 24*time.Hour {
						lastUsed = fmt.Sprintf("%d hours ago", int(duration.Hours()))
					} else if duration < 48*time.Hour {
						lastUsed = "yesterday"
					} else {
						lastUsed = fmt.Sprintf("%d days ago", int(duration.Hours()/24))
					}
				}
				
				fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\n",
					marker, p.AWSProfile,
					p.Name,
					profileType,
					valueOrEmpty(p.AWSProfile),
					valueOrEmpty(p.Region),
					lastUsed,
				)
			}
			w.Flush()
		},
	})
	
	// Current profile
	profilesCmd.AddCommand(&cobra.Command{
		Use:   "current",
		Short: "Show current profile",
		Long:  `Display information about the currently active profile.`,
		Run: func(cmd *cobra.Command, args []string) {
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
		},
	})
	
	// Switch profile
	profilesCmd.AddCommand(&cobra.Command{
		Use:   "switch [profile-id]",
		Short: "Switch to a different profile",
		Long:  `Activate a different profile for subsequent commands.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			profileID := args[0]
			
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
				saveConfig(config)
			}
		},
	})
	
	// Add profile (personal)
	addCmd := &cobra.Command{
		Use:   "add [type] [name] [options]",
		Short: "Add a new profile",
		Long:  `Add a new profile for working with AWS accounts.`,
	}
	profilesCmd.AddCommand(addCmd)
	
	// Add personal profile
	addPersonalCmd := &cobra.Command{
		Use:   "personal [name] --aws-profile [aws-profile] --region [region]",
		Short: "Add a personal profile",
		Long:  `Add a new personal profile connected to your AWS account.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
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
				LastUsed:   time.Now(),
			}
			
			// Add profile
			err = profileManager.AddProfile(prof)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding profile: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Added personal profile '%s'\n", name)
		},
	}
	addPersonalCmd.Flags().String("aws-profile", "default", "AWS profile name in ~/.aws/credentials")
	addPersonalCmd.Flags().String("region", "", "AWS region for this profile")
	addCmd.AddCommand(addPersonalCmd)
	
	// Add invitation profile
	addInvitationCmd := &cobra.Command{
		Use:   "invitation [name] --token [token] --owner [account] --region [region]",
		Short: "Add an invitation profile",
		Long:  `Add a new profile from an invitation to access another account.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
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
				LastUsed:        time.Now(),
			}
			
			// Add profile
			err = profileManager.AddProfile(prof)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding profile: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Added invitation profile '%s'\n", name)
		},
	}
	addInvitationCmd.Flags().String("token", "", "Invitation token")
	addInvitationCmd.Flags().String("owner", "", "Account owner")
	addInvitationCmd.Flags().String("region", "", "AWS region for this profile")
	addInvitationCmd.Flags().String("s3-config-path", "", "S3 path to configuration")
	addInvitationCmd.MarkFlagRequired("token")
	addInvitationCmd.MarkFlagRequired("owner")
	addCmd.AddCommand(addInvitationCmd)
	
	// Remove profile
	profilesCmd.AddCommand(&cobra.Command{
		Use:   "remove [profile-id]",
		Short: "Remove a profile",
		Long:  `Remove a profile from your configuration.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			profileID := args[0]
			
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
		},
	})
	
	// Validate profile
	profilesCmd.AddCommand(&cobra.Command{
		Use:   "validate [profile-id]",
		Short: "Validate a profile",
		Long:  `Check if a profile is valid and working correctly.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			profileID := args[0]
			
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
			
			// Create client
			client := api.NewClient(config.Daemon.URL)
			client.SetAWSProfile(config.AWS.Profile)
			client.SetAWSRegion(config.AWS.Region)
			
			// Create client with profile - use regular WithProfile method
			profileClient, err := client.WithProfile(profileManager, profileID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating client with profile: %v\n", err)
				os.Exit(1)
			}
			
			// Test API access
			err = profileClient.Ping()
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
		},
	})
	
	// Accept invitation (shortcut for add invitation)
	profilesCmd.AddCommand(&cobra.Command{
		Use:   "accept-invitation --token [token] --name [name] --owner [account] --region [region]",
		Short: "Accept an invitation",
		Long:  `Add a new profile from an invitation token.`,
		Run: func(cmd *cobra.Command, args []string) {
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
				LastUsed:        time.Now(),
			}
			
			// Add profile
			err = profileManager.AddProfile(prof)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error accepting invitation: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Accepted invitation and created profile '%s'\n", name)
		},
	})
	
	// Rename profile
	profilesCmd.AddCommand(&cobra.Command{
		Use:   "rename [profile-id] [new-name]",
		Short: "Rename a profile",
		Long:  `Change the display name of a profile.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			profileID := args[0]
			newName := args[1]
			
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
		},
	})
	
	// Invitation management commands
	invitationsCmd := &cobra.Command{
		Use:   "invitations",
		Short: "Manage shared access invitations",
		Long:  `Create and manage invitations for sharing access to your CloudWorkstation resources.`,
	}
	profilesCmd.AddCommand(invitationsCmd)
	
	// Create invitation
	invitationsCmd.AddCommand(&cobra.Command{
		Use:   "create [name] --type [read_only|read_write|admin] --valid-days [days]",
		Short: "Create a new invitation",
		Long:  `Generate a new invitation that can be shared with others to grant access to your CloudWorkstation resources.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
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
			var invitationType profile.InvitationType
			switch invType {
			case "read_only":
				invitationType = profile.InvitationTypeReadOnly
			case "read_write":
				invitationType = profile.InvitationTypeReadWrite
			case "admin":
				invitationType = profile.InvitationTypeAdmin
			default:
				fmt.Fprintf(os.Stderr, "Error: Invalid invitation type '%s'. Must be one of: read_only, read_write, admin\n", invType)
				os.Exit(1)
			}
			
			// Create the invitation
			invitation, err := invitationManager.CreateInvitation(name, invitationType, validDays, s3ConfigPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating invitation: %v\n", err)
				os.Exit(1)
			}
			
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
		},
	})
	// Add flags for the create command
	createCmd := invitationsCmd.Commands()[0]
	createCmd.Flags().String("type", "read_only", "Type of access (read_only, read_write, or admin)")
	createCmd.Flags().Int("valid-days", 30, "Number of days the invitation is valid")
	createCmd.Flags().String("s3-config", "", "Optional S3 path to configuration")
	
	// List invitations
	invitationsCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List active invitations",
		Long:  `Show all active invitations you've created.`,
		Run: func(cmd *cobra.Command, args []string) {
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
		},
	})
	
	// Revoke invitation
	invitationsCmd.AddCommand(&cobra.Command{
		Use:   "revoke [token]",
		Short: "Revoke an invitation",
		Long:  `Revoke an active invitation so it can no longer be used.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			token := args[0]
			
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
		},
	})
	
	// Add export and import commands
	AddExportCommands(profilesCmd, config)
	
	// Add batch invitation commands
	AddBatchInvitationCommands(invitationsCmd, config)
	
	// Update the accept-invitation command to use the new invitation system
	acceptCmd := profilesCmd.Commands()[len(profilesCmd.Commands())-4] // The accept-invitation command
	acceptCmd.Flags().String("encoded", "", "Encoded invitation string")
	acceptCmd.MarkFlagRequired("encoded")
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

// createAPIClient creates an API client from config
func createAPIClient(config *Config) api.CloudWorkstationAPI {
	client := api.NewClient(config.Daemon.URL)
	
	// Configure client with AWS credentials
	client.SetAWSProfile(config.AWS.Profile)
	client.SetAWSRegion(config.AWS.Region)
	
	return api.NewContextClientWithURL(config.Daemon.URL)
}