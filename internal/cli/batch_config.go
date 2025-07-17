package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
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
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Get current configuration
			config := configManager.GetConfig()

			// Display configuration
			fmt.Println("Batch Invitation Configuration:")
			fmt.Println()
			
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			
			// General settings
			fmt.Fprintln(w, "General Settings:")
			fmt.Fprintf(w, "  Default Concurrency:\t%d\n", config.DefaultConcurrency)
			fmt.Fprintf(w, "  Default Valid Days:\t%d\n", config.DefaultValidDays)
			fmt.Fprintln(w)
			
			// Security settings
			fmt.Fprintln(w, "Security Settings:")
			fmt.Fprintf(w, "  Default Device Bound:\t%t\n", config.DefaultDeviceBound)
			fmt.Fprintf(w, "  Default Max Devices:\t%d\n", config.DefaultMaxDevices)
			fmt.Fprintf(w, "  Default Can Invite:\t%t\n", config.DefaultCanInvite)
			fmt.Fprintf(w, "  Default Transferable:\t%t\n", config.DefaultTransferable)
			fmt.Fprintln(w)
			
			// CSV settings
			fmt.Fprintln(w, "CSV Settings:")
			fmt.Fprintf(w, "  Default Has Header:\t%t\n", config.DefaultHasHeader)
			fmt.Fprintf(w, "  Default Delimiter:\t%s\n", config.DefaultDelimiter)
			fmt.Fprintf(w, "  Include Encoded Data:\t%t\n", config.IncludeEncodedData)
			fmt.Fprintf(w, "  Default Output Directory:\t%s\n", valueOrEmpty(config.DefaultOutputDirectory))
			fmt.Fprintln(w)
			
			// Admin settings
			fmt.Fprintln(w, "Admin Settings:")
			fmt.Fprintf(w, "  Require Admin Auth:\t%t\n", config.RequireAdminAuth)
			fmt.Fprintf(w, "  Admin Invitation Token:\t%s\n", maskIfNotEmpty(config.AdminInvitationToken))
			fmt.Fprintf(w, "  Notification Webhook:\t%s\n", maskIfNotEmpty(config.NotificationWebhook))
			fmt.Fprintf(w, "  Audit Logging Enabled:\t%t\n", config.AuditLoggingEnabled)
			fmt.Fprintf(w, "  Log Directory:\t%s\n", valueOrEmpty(config.LogDirectory))
			fmt.Fprintln(w)
			
			// Performance settings
			fmt.Fprintln(w, "Performance Settings:")
			fmt.Fprintf(w, "  Batch Size Limit:\t%d\n", config.BatchSizeLimit)
			fmt.Fprintf(w, "  Enable Rate Limiting:\t%t\n", config.EnableRateLimiting)
			fmt.Fprintf(w, "  Max Operations Per Hour:\t%d\n", config.MaxOperationsPerHour)
			fmt.Fprintln(w)
			
			// Last updated
			fmt.Fprintf(w, "Last Updated:\t%s\n", config.LastUpdated.Format("2006-01-02 15:04:05"))
			
			w.Flush()
		},
	})

	// Set configuration command
	setCmd := &cobra.Command{
		Use:   "set [setting] [value]",
		Short: "Set a batch invitation configuration value",
		Long:  `Update a specific batch invitation configuration setting.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			setting := args[0]
			value := args[1]

			// Create batch config manager
			configManager, err := createBatchConfigManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Get current configuration
			config := configManager.GetConfig()

			// Update the specified setting
			updated, err := updateConfigSetting(config, setting, value)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Save the updated configuration
			if err := configManager.UpdateConfig(config); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Updated %s to %s\n", setting, updated)
		},
	})
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
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
				fmt.Fprintf(os.Stderr, "Error resetting configuration: %v\n", err)
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

// updateConfigSetting updates a specific configuration setting
func updateConfigSetting(config *profile.BatchInvitationConfig, setting, value string) (string, error) {
	switch setting {
	// General settings
	case "defaultConcurrency":
		val, err := parseInt(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for defaultConcurrency: %w", err)
		}
		if val <= 0 {
			return "", fmt.Errorf("defaultConcurrency must be greater than 0")
		}
		config.DefaultConcurrency = val
		return fmt.Sprintf("%d", val), nil

	case "defaultValidDays":
		val, err := parseInt(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for defaultValidDays: %w", err)
		}
		if val <= 0 {
			return "", fmt.Errorf("defaultValidDays must be greater than 0")
		}
		config.DefaultValidDays = val
		return fmt.Sprintf("%d", val), nil

	// Security settings
	case "defaultDeviceBound":
		val, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for defaultDeviceBound: %w", err)
		}
		config.DefaultDeviceBound = val
		return fmt.Sprintf("%t", val), nil

	case "defaultMaxDevices":
		val, err := parseInt(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for defaultMaxDevices: %w", err)
		}
		if val <= 0 {
			return "", fmt.Errorf("defaultMaxDevices must be greater than 0")
		}
		config.DefaultMaxDevices = val
		return fmt.Sprintf("%d", val), nil

	case "defaultCanInvite":
		val, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for defaultCanInvite: %w", err)
		}
		config.DefaultCanInvite = val
		return fmt.Sprintf("%t", val), nil

	case "defaultTransferable":
		val, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for defaultTransferable: %w", err)
		}
		config.DefaultTransferable = val
		return fmt.Sprintf("%t", val), nil

	// CSV settings
	case "defaultHasHeader":
		val, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for defaultHasHeader: %w", err)
		}
		config.DefaultHasHeader = val
		return fmt.Sprintf("%t", val), nil

	case "defaultDelimiter":
		if value != "," && value != ";" && value != "\t" {
			return "", fmt.Errorf("defaultDelimiter must be one of: ',', ';', '\\t'")
		}
		config.DefaultDelimiter = value
		return value, nil

	case "includeEncodedData":
		val, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for includeEncodedData: %w", err)
		}
		config.IncludeEncodedData = val
		return fmt.Sprintf("%t", val), nil

	case "defaultOutputDirectory":
		config.DefaultOutputDirectory = value
		return value, nil

	// Admin settings
	case "requireAdminAuth":
		val, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for requireAdminAuth: %w", err)
		}
		config.RequireAdminAuth = val
		return fmt.Sprintf("%t", val), nil

	case "adminInvitationToken":
		config.AdminInvitationToken = value
		return "********", nil

	case "notificationWebhook":
		config.NotificationWebhook = value
		return "********", nil

	case "auditLoggingEnabled":
		val, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for auditLoggingEnabled: %w", err)
		}
		config.AuditLoggingEnabled = val
		return fmt.Sprintf("%t", val), nil

	case "logDirectory":
		config.LogDirectory = value
		return value, nil

	// Performance settings
	case "batchSizeLimit":
		val, err := parseInt(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for batchSizeLimit: %w", err)
		}
		if val <= 0 {
			return "", fmt.Errorf("batchSizeLimit must be greater than 0")
		}
		config.BatchSizeLimit = val
		return fmt.Sprintf("%d", val), nil

	case "enableRateLimiting":
		val, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for enableRateLimiting: %w", err)
		}
		config.EnableRateLimiting = val
		return fmt.Sprintf("%t", val), nil

	case "maxOperationsPerHour":
		val, err := parseInt(value)
		if err != nil {
			return "", fmt.Errorf("invalid value for maxOperationsPerHour: %w", err)
		}
		if val <= 0 {
			return "", fmt.Errorf("maxOperationsPerHour must be greater than 0")
		}
		config.MaxOperationsPerHour = val
		return fmt.Sprintf("%d", val), nil

	default:
		return "", fmt.Errorf("unknown configuration setting: %s", setting)
	}
}

// parseInt parses a string to an integer
func parseInt(value string) (int, error) {
	return strconv.Atoi(value)
}

// parseBool parses a string to a boolean
func parseBool(value string) (bool, error) {
	value = strings.ToLower(value)
	if value == "true" || value == "yes" || value == "y" || value == "1" || value == "t" {
		return true, nil
	} else if value == "false" || value == "no" || value == "n" || value == "0" || value == "f" {
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean value: %s", value)
}