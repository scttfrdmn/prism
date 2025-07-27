package cli

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/spf13/cobra"
)

// AddSecureInvitationCommands adds commands for secure invitation management
func AddSecureInvitationCommands(invitationsCmd *cobra.Command, config *Config) {
	// Create secure invitation command
	createSecureCmd := &cobra.Command{
		Use:   "create-secure [name]",
		Short: "Create a secure invitation with enhanced security features",
		Long: `Create a secure invitation with device binding and permissions control.
		
This command creates an invitation with enhanced security features including:
- Device binding to prevent casual sharing
- Permission controls for invitation delegation
- Maximum device limits for each recipient
- Transferability controls

Example:
  cws profiles invitations create-secure lab-access --type admin --can-invite=true --max-devices=3`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			// Get flags
			invType, _ := cmd.Flags().GetString("type")
			validDays, _ := cmd.Flags().GetInt("valid-days")
			s3ConfigPath, _ := cmd.Flags().GetString("s3-config")
			canInvite, _ := cmd.Flags().GetBool("can-invite")
			transferable, _ := cmd.Flags().GetBool("transferable")
			deviceBound, _ := cmd.Flags().GetBool("device-bound")
			maxDevices, _ := cmd.Flags().GetInt("max-devices")
			parentToken, _ := cmd.Flags().GetString("parent-token")

			// Parse invitation type
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

			// Create profile manager
			profileManager, err := createEnhancedProfileManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Create secure invitation manager
			invitationManager, err := profile.NewSecureInvitationManager(profileManager)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Create secure invitation
			invitation, err := invitationManager.CreateSecureInvitation(
				name,
				invitationType,
				validDays,
				s3ConfigPath,
				canInvite,
				transferable,
				deviceBound,
				maxDevices,
				parentToken,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating invitation: %v\n", err)
				os.Exit(1)
			}

			// Encode invitation for sharing
			encodedToken, err := invitation.EncodeToString()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding invitation: %v\n", err)
				os.Exit(1)
			}

			// Display invitation details
			fmt.Printf("Created secure invitation: %s\n\n", invitation.GetDescription())
			fmt.Printf("Invitation Token: %s\n\n", invitation.Token)
			fmt.Printf("Security Settings:\n")
			fmt.Printf("- Device Binding: %v\n", invitation.DeviceBound)
			fmt.Printf("- Max Devices: %d\n", invitation.MaxDevices)
			fmt.Printf("- Can Invite Others: %v\n", invitation.CanInvite)
			fmt.Printf("- Transferable: %v\n\n", invitation.Transferable)
			fmt.Printf("Full Encoded Token (for sharing):\n%s\n\n", encodedToken)
			fmt.Printf("Recipients can accept this invitation with:\n")
			fmt.Printf("cws profiles accept-invitation --token \"%s\" --name \"<profile_name>\"\n", invitation.Token)
		},
	}

	// Add flags for secure invitation creation
	createSecureCmd.Flags().String("type", "read_only", "Invitation type (read_only, read_write, admin)")
	createSecureCmd.Flags().Int("valid-days", 30, "Number of days the invitation is valid")
	createSecureCmd.Flags().String("s3-config", "", "Optional S3 path for shared configuration")
	createSecureCmd.Flags().Bool("can-invite", false, "Whether the recipient can create sub-invitations")
	createSecureCmd.Flags().Bool("transferable", false, "Whether the profile can be exported/shared")
	createSecureCmd.Flags().Bool("device-bound", true, "Whether to bind the profile to specific devices")
	createSecureCmd.Flags().Int("max-devices", 1, "Maximum number of devices allowed per user (1-5)")
	createSecureCmd.Flags().String("parent-token", "", "Parent invitation token (for delegation)")

	// Devices command
	devicesCmd := &cobra.Command{
		Use:   "devices [invitation-token]",
		Short: "List devices registered for an invitation",
		Long: `List all devices that have registered to use an invitation.
		
This command shows all devices that have been authorized to use a specific
invitation, including device information and last usage timestamp.

Example:
  cws profiles invitations devices inv-abc123def456`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			token := args[0]

			// Create profile manager
			profileManager, err := createEnhancedProfileManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Create secure invitation manager
			invitationManager, err := profile.NewSecureInvitationManager(profileManager)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Get invitation details
			invitation, err := invitationManager.GetInvitation(token)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Get devices for invitation
			devices, err := invitationManager.GetInvitationDevices(token)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error retrieving devices: %v\n", err)
				os.Exit(1)
			}

			// Display invitation details
			fmt.Printf("Devices for invitation: %s\n\n", invitation.GetDescription())

			if len(devices) == 0 {
				fmt.Println("No devices have been registered for this invitation.")
				return
			}

			// Display device information
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "DEVICE ID\tHOSTNAME\tUSERNAME\tREGISTERED")
			for _, device := range devices {
				deviceID := getStringValue(device, "device_id")
				hostname := getStringValue(device, "hostname")
				username := getStringValue(device, "username")
				timestamp := getStringValue(device, "timestamp")

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", deviceID, hostname, username, timestamp)
			}
			w.Flush()
		},
	}

	// Revoke device command
	revokeDeviceCmd := &cobra.Command{
		Use:   "revoke-device [invitation-token] [device-id]",
		Short: "Revoke device access to an invitation",
		Long: `Revoke a specific device's access to an invitation.
		
This command removes authorization for a specific device to use an invitation.
The device ID can be found using the 'devices' command.

Example:
  cws profiles invitations revoke-device inv-abc123def456 device-xyz789`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			token := args[0]
			deviceID := args[1]

			// Create profile manager
			profileManager, err := createEnhancedProfileManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Create secure invitation manager
			invitationManager, err := profile.NewSecureInvitationManager(profileManager)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Revoke device
			err = invitationManager.RevokeDevice(token, deviceID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error revoking device: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully revoked device %s from invitation %s\n", deviceID, token)
		},
	}

	// Revoke all devices command
	revokeAllCmd := &cobra.Command{
		Use:   "revoke-all [invitation-token]",
		Short: "Revoke all device access to an invitation",
		Long: `Revoke all devices' access to an invitation.
		
This command removes authorization for all devices to use an invitation.
This is useful when you suspect the invitation has been compromised.

Example:
  cws profiles invitations revoke-all inv-abc123def456`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			token := args[0]

			// Create profile manager
			profileManager, err := createEnhancedProfileManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Create secure invitation manager
			invitationManager, err := profile.NewSecureInvitationManager(profileManager)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Revoke all devices
			err = invitationManager.RevokeAllDevices(token)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error revoking all devices: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully revoked all devices from invitation %s\n", token)
		},
	}

	// Add commands to invitations command
	invitationsCmd.AddCommand(createSecureCmd)
	invitationsCmd.AddCommand(devicesCmd)
	invitationsCmd.AddCommand(revokeDeviceCmd)
	invitationsCmd.AddCommand(revokeAllCmd)
}

// Helper function to get string value from map
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'f', 0, 64)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return "-"
}