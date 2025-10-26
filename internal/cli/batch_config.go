package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/scttfrdmn/prism/pkg/profile"
	"github.com/spf13/cobra"
)

// AddBatchConfigCommands adds batch configuration commands to the CLI
func AddBatchConfigCommands(invitationsCmd *cobra.Command, config *Config) {
	// Create batch config command
	batchConfigCmd := &cobra.Command{
		Use:   "config",
		Short: "Configure batch invitation system",
		Long:  `View and modify batch invitation configuration settings.`,
	}
	invitationsCmd.AddCommand(batchConfigCmd)

	// Create batch config manager
	createBatchConfigManager := func() (*profile.BatchConfigManager, error) {
		return profile.NewBatchConfigManager()
	}

	// Show configuration command
	batchConfigCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show batch invitation configuration",
		Long:  `Display the current batch invitation configuration settings.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create batch config manager
			configManager, err := createBatchConfigManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create batch config manager"))
				os.Exit(1)
			}

			// Get current configuration
			config := configManager.GetConfig()

			// Display configuration
			fmt.Println("Batch Invitation Configuration:")
			fmt.Println()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			// General settings
			_, _ = fmt.Fprintln(w, "General Settings:")
			_, _ = fmt.Fprintf(w, "  Default Concurrency:\t%d\n", config.DefaultConcurrency)
			_, _ = fmt.Fprintf(w, "  Default Valid Days:\t%d\n", config.DefaultValidDays)
			_, _ = fmt.Fprintln(w)

			// Security settings
			_, _ = fmt.Fprintln(w, "Security Settings:")
			_, _ = fmt.Fprintf(w, "  Default Device Bound:\t%t\n", config.DefaultDeviceBound)
			_, _ = fmt.Fprintf(w, "  Default Max Devices:\t%d\n", config.DefaultMaxDevices)
			_, _ = fmt.Fprintf(w, "  Default Can Invite:\t%t\n", config.DefaultCanInvite)
			_, _ = fmt.Fprintf(w, "  Default Transferable:\t%t\n", config.DefaultTransferable)
			_, _ = fmt.Fprintln(w)

			// CSV settings
			_, _ = fmt.Fprintln(w, "CSV Settings:")
			_, _ = fmt.Fprintf(w, "  Default Has Header:\t%t\n", config.DefaultHasHeader)
			_, _ = fmt.Fprintf(w, "  Default Delimiter:\t%s\n", config.DefaultDelimiter)
			_, _ = fmt.Fprintf(w, "  Include Encoded Data:\t%t\n", config.IncludeEncodedData)
			_, _ = fmt.Fprintf(w, "  Default Output Directory:\t%s\n", valueOrEmpty(config.DefaultOutputDirectory))
			_, _ = fmt.Fprintln(w)

			// Admin settings
			_, _ = fmt.Fprintln(w, "Admin Settings:")
			_, _ = fmt.Fprintf(w, "  Require Admin Auth:\t%t\n", config.RequireAdminAuth)
			_, _ = fmt.Fprintf(w, "  Admin Invitation Token:\t%s\n", maskIfNotEmpty(config.AdminInvitationToken))
			_, _ = fmt.Fprintf(w, "  Notification Webhook:\t%s\n", maskIfNotEmpty(config.NotificationWebhook))
			_, _ = fmt.Fprintf(w, "  Audit Logging Enabled:\t%t\n", config.AuditLoggingEnabled)
			_, _ = fmt.Fprintf(w, "  Log Directory:\t%s\n", valueOrEmpty(config.LogDirectory))
			_, _ = fmt.Fprintln(w)

			// Performance settings
			_, _ = fmt.Fprintln(w, "Performance Settings:")
			_, _ = fmt.Fprintf(w, "  Batch Size Limit:\t%d\n", config.BatchSizeLimit)
			_, _ = fmt.Fprintf(w, "  Enable Rate Limiting:\t%t\n", config.EnableRateLimiting)
			_, _ = fmt.Fprintf(w, "  Max Operations Per Hour:\t%d\n", config.MaxOperationsPerHour)
			_, _ = fmt.Fprintln(w)

			// Last updated
			_, _ = fmt.Fprintf(w, "Last Updated:\t%s\n", config.LastUpdated.Format("2006-01-02 15:04:05"))

			_ = w.Flush()
		},
	})

	// Set configuration command
	setCmd := &cobra.Command{
		Use:   "set [setting] [value]",
		Short: "Set a batch invitation configuration value",
		Long:  `Update a specific batch invitation configuration setting.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// Batch config commands disabled during Phase 1 simplification
			fmt.Println("⚠️  Batch configuration commands temporarily disabled during Phase 1.")
			fmt.Println("Core profile commands are available:")
			fmt.Println("  cws profiles list    # List available profiles")
			fmt.Println("  cws profiles create  # Create new profile")
			fmt.Println("  cws profiles set     # Set current profile")
			fmt.Println("")
			fmt.Println("Advanced batch features will return in Phase 2.")
		},
	}
	batchConfigCmd.AddCommand(setCmd)

	// Reset configuration command
	batchConfigCmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Reset batch invitation configuration to defaults",
		Long:  `Reset all batch invitation configuration settings to their default values.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create batch config manager
			configManager, err := createBatchConfigManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create batch config manager"))
				os.Exit(1)
			}

			// Confirm reset
			confirm, _ := cmd.Flags().GetBool("confirm")
			if !confirm {
				fmt.Println("This will reset all batch invitation configuration settings to their default values.")
				fmt.Println("To confirm, use the --confirm flag.")
				os.Exit(1)
			}

			// Reset configuration
			if err := configManager.ResetToDefaults(); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "reset batch configuration"))
				os.Exit(1)
			}

			fmt.Println("Batch invitation configuration reset to defaults.")
		},
	})
	batchConfigCmd.Commands()[2].Flags().Bool("confirm", false, "Confirm configuration reset")
}

// Helper functions

// maskIfNotEmpty returns a masked string if the value is not empty
func maskIfNotEmpty(value string) string {
	if value == "" {
		return "-"
	}
	return "********"
}
