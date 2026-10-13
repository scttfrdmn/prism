// Package cli provides shared token CLI commands for Prism v0.5.13+
package cli

import (
	"github.com/spf13/cobra"
)

// createInvitationsSharedCommand creates the 'prism project invitations shared' command
func createInvitationsSharedCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shared",
		Short: "Manage shared invitation tokens",
		Long: `Manage shared invitation tokens for workshops and conferences.

Shared tokens allow multiple users to redeem the same token code up to a limit.
Ideal for:
  • Workshop registration (60 participants, one QR code)
  • Conference tutorials (first-come-first-served access)
  • Guest lectures (no pre-registration needed)

Examples:
  # Create shared token for workshop
  prism project invitations shared create neurips-workshop \
    --name "NeurIPS Workshop" \
    --redemption-limit 60 \
    --expires-in 7d

  # Show token info
  prism project invitations shared show WORKSHOP-NEURIPS-2025

  # List project tokens
  prism project invitations shared list neurips-workshop

  # Extend expiration
  prism project invitations shared extend WORKSHOP-NEURIPS-2025 --add-days 1

  # Revoke token
  prism project invitations shared revoke WORKSHOP-NEURIPS-2025`,
	}

	// Add subcommands
	cmd.AddCommand(createSharedTokenCreateCommand(app))
	cmd.AddCommand(createSharedTokenShowCommand(app))
	cmd.AddCommand(createSharedTokenListCommand(app))
	cmd.AddCommand(createSharedTokenQRCommand(app))
	cmd.AddCommand(createSharedTokenExtendCommand(app))
	cmd.AddCommand(createSharedTokenRevokeCommand(app))

	return cmd
}

// createSharedTokenCreateCommand creates the 'prism project invitations shared create' command
func createSharedTokenCreateCommand(app *App) *cobra.Command {
	var (
		name            string
		role            string
		message         string
		redemptionLimit int
		expiresIn       string
		expiresOn       string
	)

	cmd := &cobra.Command{
		Use:   "create <project>",
		Short: "Create a shared invitation token",
		Long: `Create a shared invitation token that multiple users can redeem.

A shared token provides a single code that can be redeemed by multiple users
up to a specified limit. Perfect for workshops, conferences, and guest lectures.

Examples:
  # Basic workshop token (60 participants, 7 day expiration)
  prism project invitations shared create neurips-workshop \
    --name "NeurIPS Workshop" \
    --redemption-limit 60 \
    --expires-in 7d

  # Custom role and message
  prism project invitations shared create ml-bootcamp \
    --name "ML Bootcamp Access" \
    --redemption-limit 100 \
    --role member \
    --message "Welcome to the ML Bootcamp!" \
    --expires-in 14d

  # Absolute expiration date
  prism project invitations shared create guest-lecture \
    --name "Guest Lecture Access" \
    --redemption-limit 30 \
    --expires-on 2025-12-31`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.projectInvitationsSharedCreate(args, name, role, message, redemptionLimit, expiresIn, expiresOn)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name for the token (required)")
	cmd.Flags().IntVar(&redemptionLimit, "redemption-limit", 0, "Maximum number of redemptions (required)")
	cmd.Flags().StringVar(&role, "role", "member", "Role assigned to redeemers (owner, admin, member, viewer)")
	cmd.Flags().StringVar(&message, "message", "", "Optional welcome message for redeemers")
	cmd.Flags().StringVar(&expiresIn, "expires-in", "7d", "Expiration duration (e.g., 7d, 14d, 48h)")
	cmd.Flags().StringVar(&expiresOn, "expires-on", "", "Absolute expiration date (e.g., 2025-12-31)")

	_ = cmd.MarkFlagRequired("name")             // only errors if the flag is undefined; it is defined above
	_ = cmd.MarkFlagRequired("redemption-limit") // only errors if the flag is undefined; it is defined above

	return cmd
}

// createSharedTokenShowCommand creates the 'prism project invitations shared show' command
func createSharedTokenShowCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <token>",
		Short: "Show shared token information",
		Long: `Display detailed information about a shared token.

Shows token status, redemption count, expiration, and audit trail.

Examples:
  prism project invitations shared show WORKSHOP-NEURIPS-2025`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.projectInvitationsSharedShow(args)
		},
	}

	return cmd
}

// createSharedTokenListCommand creates the 'prism project invitations shared list' command
func createSharedTokenListCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <project>",
		Short: "List all shared tokens for a project",
		Long: `List all shared tokens for a project with their status.

Shows token code, name, redemptions, status, and expiration.

Examples:
  prism project invitations shared list neurips-workshop`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.projectInvitationsSharedList(args)
		},
	}

	return cmd
}

// createSharedTokenQRCommand creates the 'prism project invitations shared qr' command
func createSharedTokenQRCommand(app *App) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "qr <token>",
		Short: "Generate QR code for shared token",
		Long: `Generate a QR code for a shared token redemption URL.

The QR code can be printed and displayed at registration desks,
conference booths, or workshop materials for easy mobile scanning.

Examples:
  # Save QR code to file
  prism project invitations shared qr WORKSHOP-NEURIPS-2025 --output workshop-qr.png

  # Display QR code URL
  prism project invitations shared qr WORKSHOP-NEURIPS-2025`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.projectInvitationsSharedQR(args, output)
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "Output file path (e.g., qrcode.png)")

	return cmd
}

// createSharedTokenExtendCommand creates the 'prism project invitations shared extend' command
func createSharedTokenExtendCommand(app *App) *cobra.Command {
	var addDays int

	cmd := &cobra.Command{
		Use:   "extend <token>",
		Short: "Extend token expiration",
		Long: `Extend the expiration of a shared token by N days.

Useful for extending workshop access for homework completion or follow-up work.

Examples:
  # Extend by 1 day (24 hours)
  prism project invitations shared extend WORKSHOP-NEURIPS-2025 --add-days 1

  # Extend by 7 days (1 week)
  prism project invitations shared extend WORKSHOP-NEURIPS-2025 --add-days 7`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.projectInvitationsSharedExtend(args, addDays)
		},
	}

	cmd.Flags().IntVar(&addDays, "add-days", 1, "Number of days to extend")

	return cmd
}

// createSharedTokenRevokeCommand creates the 'prism project invitations shared revoke' command
func createSharedTokenRevokeCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke <token>",
		Short: "Revoke a shared token",
		Long: `Revoke a shared token, preventing any further redemptions.

Existing redeemers retain their access, but no new redemptions are allowed.

Examples:
  prism project invitations shared revoke WORKSHOP-NEURIPS-2025`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.projectInvitationsSharedRevoke(args)
		},
	}

	return cmd
}

// Add shared command to invitations command
func init() {
	// This will be called when invitations command is initialized
	// The shared command will be added as a subcommand
}
