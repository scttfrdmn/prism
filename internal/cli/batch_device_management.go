package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// AddBatchDeviceCommands adds secure invitation and device-management subcommands.
// All logic delegates to pkg/profile.SecureInvitationManager, which already exists.
func AddBatchDeviceCommands(invitationsCmd *cobra.Command, config *Config) {
	// create-secure command
	createSecureCmd := &cobra.Command{
		Use:   "create-secure <name>",
		Short: "Create a device-bound secure invitation",
		Long: `Generate a secure invitation that is bound to specific devices.

Device-bound invitations can only be used on registered devices.  When a
recipient accepts such an invitation the device is registered automatically.
Use 'devices', 'revoke-device', and 'revoke-all' to manage registered devices.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			invType, _ := cmd.Flags().GetString("type")
			validDays, _ := cmd.Flags().GetInt("valid-days")
			deviceBound, _ := cmd.Flags().GetBool("device-bound")
			maxDevices, _ := cmd.Flags().GetInt("max-devices")
			canInvite, _ := cmd.Flags().GetBool("can-invite")
			transferable, _ := cmd.Flags().GetBool("transferable")

			secureManager, err := initializeSecureManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "initialize invitation manager"))
				os.Exit(1)
			}

			// Parse invitation type
			parsedType, err := parseInvitationType(invType)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			invitation, err := secureManager.CreateSecureInvitation(
				name,
				parsedType,
				validDays,
				"", // s3ConfigPath — not exposed via CLI flag
				canInvite,
				transferable,
				deviceBound,
				maxDevices,
				"", // parentToken — not exposed via CLI flag
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create secure invitation"))
				os.Exit(1)
			}

			// Encode for sharing
			encoded, err := invitation.EncodeToString()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "encode invitation"))
				os.Exit(1)
			}

			fmt.Printf("✅ Secure invitation created: %s\n\n", name)
			fmt.Printf("Token:        %s\n", invitation.Token)
			fmt.Printf("Type:         %s\n", invitation.Type)
			fmt.Printf("Expires:      %s\n", invitation.Expires.Format("2006-01-02"))
			fmt.Printf("Device bound: %v\n", invitation.DeviceBound)
			fmt.Printf("Max devices:  %d\n", invitation.MaxDevices)
			fmt.Printf("Can invite:   %v\n", invitation.CanInvite)
			fmt.Printf("Transferable: %v\n", invitation.Transferable)
			fmt.Printf("\nEncoded invitation (share this):\n%s\n", encoded)
		},
	}
	createSecureCmd.Flags().String("type", "read_only", "Invitation type: read_only, read_write, or admin")
	createSecureCmd.Flags().Int("valid-days", 30, "Number of days the invitation is valid")
	createSecureCmd.Flags().Bool("device-bound", true, "Restrict invitation to registered devices")
	createSecureCmd.Flags().Int("max-devices", 1, "Maximum number of devices that can use this invitation")
	createSecureCmd.Flags().Bool("can-invite", false, "Allow the recipient to create sub-invitations")
	createSecureCmd.Flags().Bool("transferable", false, "Allow the invitation to be transferred to another user")
	invitationsCmd.AddCommand(createSecureCmd)

	// devices command
	devicesCmd := &cobra.Command{
		Use:   "devices <token>",
		Short: "List devices registered for an invitation",
		Long:  `Show all devices that have registered against a specific invitation token.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			token := args[0]

			secureManager, err := initializeSecureManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "initialize invitation manager"))
				os.Exit(1)
			}

			devices, err := secureManager.GetInvitationDevices(token)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "get invitation devices"))
				os.Exit(1)
			}

			if len(devices) == 0 {
				fmt.Println("No devices registered for this invitation.")
				return
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "DEVICE ID\tHOSTNAME\tUSERNAME\tCREATED\tLAST VALIDATED")
			fmt.Fprintln(w, "---------\t--------\t--------\t-------\t--------------")
			for _, d := range devices {
				deviceID, _ := d["device_id"].(string)
				hostname, _ := d["hostname"].(string)
				username, _ := d["username"].(string)
				created, _ := d["created"].(string)
				lastValidated, _ := d["last_validated"].(string)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					deviceID, hostname, username, created, lastValidated)
			}
			_ = w.Flush() // stdout table; Run has no error return and a failed terminal write is not actionable
		},
	}
	invitationsCmd.AddCommand(devicesCmd)

	// revoke-device command
	revokeDeviceCmd := &cobra.Command{
		Use:   "revoke-device <token> <device-id>",
		Short: "Revoke a specific device from an invitation",
		Long:  `Remove a single device's registration so it can no longer use the invitation.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			token := args[0]
			deviceID := args[1]

			secureManager, err := initializeSecureManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "initialize invitation manager"))
				os.Exit(1)
			}

			if err := secureManager.RevokeDevice(token, deviceID); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "revoke device"))
				os.Exit(1)
			}

			fmt.Printf("✅ Device revoked: %s\n", deviceID)
		},
	}
	invitationsCmd.AddCommand(revokeDeviceCmd)

	// revoke-all command
	revokeAllCmd := &cobra.Command{
		Use:   "revoke-all <token>",
		Short: "Revoke all devices from an invitation",
		Long:  `Remove all device registrations, effectively revoking the invitation from every device.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			token := args[0]
			confirm, _ := cmd.Flags().GetBool("confirm")

			if !confirm {
				fmt.Fprintf(os.Stderr, "This will revoke all devices for invitation token '%s'.\n", token)
				fmt.Fprintf(os.Stderr, "To confirm, pass the --confirm flag.\n")
				os.Exit(1)
			}

			secureManager, err := initializeSecureManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "initialize invitation manager"))
				os.Exit(1)
			}

			if err := secureManager.RevokeAllDevices(token); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "revoke all devices"))
				os.Exit(1)
			}

			fmt.Printf("✅ All devices revoked for invitation token: %s\n", token)
		},
	}
	revokeAllCmd.Flags().Bool("confirm", false, "Confirm that all devices should be revoked")
	invitationsCmd.AddCommand(revokeAllCmd)
}
