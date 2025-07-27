package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/spf13/cobra"
)

// AddBatchDeviceCommands adds batch device management commands to the CLI
func AddBatchDeviceCommands(invitationsCmd *cobra.Command, config *Config) {
	// Create a devices subcommand
	devicesCmd := &cobra.Command{
		Use:   "devices",
		Short: "Manage invitation devices",
		Long:  `Manage devices associated with invitations.`,
	}
	invitationsCmd.AddCommand(devicesCmd)

	// Create batch device manager
	createBatchDeviceManager := func() (*profile.BatchDeviceManager, error) {
		profileManager, err := createProfileManager(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile manager: %w", err)
		}

		secureManager, err := profile.NewSecureInvitationManager(profileManager)
		if err != nil {
			return nil, fmt.Errorf("failed to create secure invitation manager: %w", err)
		}

		return profile.NewBatchDeviceManager(secureManager), nil
	}

	// Batch device operation command
	batchDeviceCmd := &cobra.Command{
		Use:   "batch-operation --csv-file [path] --operation [operation] --output-file [path]",
		Short: "Execute batch device operations",
		Long: `Execute operations on multiple devices across invitations.

CSV format should have the following columns (first row can be a header):
- Device ID: Required, the unique identifier of the device
- Token: Required, the invitation token
- Name: Optional, a descriptive name (for reporting)
- Operation: Optional, one of: revoke, validate, info

Example CSV:
Device ID,Token,Name,Operation
d1234567890abcdef,inv-abcdefg,User Device,revoke
d2345678901bcdefg,inv-bcdefgh,Other Device,validate`,
		Run: func(cmd *cobra.Command, args []string) {
			csvFile, _ := cmd.Flags().GetString("csv-file")
			operation, _ := cmd.Flags().GetString("operation")
			outputFile, _ := cmd.Flags().GetString("output-file")
			concurrency, _ := cmd.Flags().GetInt("concurrency")
			hasHeader, _ := cmd.Flags().GetBool("has-header")

			// Validate operation
			if operation != "" && operation != "revoke" && operation != "validate" && operation != "info" {
				fmt.Fprintf(os.Stderr, "Error: Invalid operation '%s'. Must be one of: revoke, validate, info\n", operation)
				os.Exit(1)
			}

			// Create batch device manager
			batchManager, err := createBatchDeviceManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Process device operations
			fmt.Printf("Processing batch device operations from %s...\n", csvFile)
			results, err := batchManager.ExecuteBatchDeviceOperation(
				csvFile,
				operation,
				concurrency,
				hasHeader,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error executing batch device operations: %v\n", err)
				os.Exit(1)
			}

			// Print summary
			fmt.Printf("\nProcessed %d device operation(s):\n", results.TotalProcessed)
			fmt.Printf("  - Successful: %d\n", results.TotalSuccessful)
			fmt.Printf("  - Failed: %d\n", results.TotalFailed)

			// Export results if requested
			if outputFile != "" {
				err := batchManager.ExportDeviceInfoToCSVFile(outputFile, results)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error exporting results: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("\nResults exported to %s\n", outputFile)
			} else {
				// Print results to console if not exporting
				if results.TotalSuccessful > 0 {
					fmt.Printf("\nSuccessful operations:\n")
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
					fmt.Fprintln(w, "DEVICE ID\tTOKEN\tOPERATION\tDETAILS")
					for _, device := range results.Successful {
						details := ""
						if device.Operation == "info" && device.Details != nil {
							if reg, ok := device.Details["registered_at"]; ok {
								details = fmt.Sprintf("Registered: %v", reg)
							}
						}
						fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
							device.DeviceID, device.Token, device.Operation, details)
					}
					w.Flush()
				}

				if results.TotalFailed > 0 {
					fmt.Printf("\nFailed operations:\n")
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
					fmt.Fprintln(w, "DEVICE ID\tTOKEN\tOPERATION\tERROR")
					for _, device := range results.Failed {
						errMsg := ""
						if device.Error != nil {
							errMsg = device.Error.Error()
						}
						fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
							device.DeviceID, device.Token, device.Operation, errMsg)
					}
					w.Flush()
				}
			}
		},
	}

	// Add flags for batch device command
	batchDeviceCmd.Flags().String("csv-file", "", "Path to CSV file containing device operations")
	batchDeviceCmd.Flags().String("operation", "", "Operation to perform (revoke, validate, info)")
	batchDeviceCmd.Flags().String("output-file", "", "Path to export results to a CSV file")
	batchDeviceCmd.Flags().Int("concurrency", 5, "Number of concurrent operations")
	batchDeviceCmd.Flags().Bool("has-header", true, "Whether the CSV file has a header row")
	batchDeviceCmd.MarkFlagRequired("csv-file")

	// Export device info command
	exportDeviceInfoCmd := &cobra.Command{
		Use:   "export-info --output-file [path]",
		Short: "Export device information for all invitations",
		Long:  `Export information about all devices registered with invitations.`,
		Run: func(cmd *cobra.Command, args []string) {
			outputFile, _ := cmd.Flags().GetString("output-file")
			concurrency, _ := cmd.Flags().GetInt("concurrency")

			// Create batch device manager
			batchManager, err := createBatchDeviceManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

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

			// Get all invitations
			invitations := invitationManager.ListInvitations()
			if len(invitations) == 0 {
				fmt.Println("No invitations found")
				os.Exit(0)
			}

			fmt.Printf("Exporting device information for %d invitation(s)...\n", len(invitations))

			// Get device info for all invitations
			results := batchManager.BatchGetDeviceInfo(invitations, concurrency)

			// Print summary
			totalDevices := results.TotalSuccessful
			fmt.Printf("\nFound %d device(s) across %d invitation(s)\n", totalDevices, len(invitations))

			// Export results
			if outputFile != "" {
				err := batchManager.ExportDeviceInfoToCSVFile(outputFile, results)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error exporting device info: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("\nDevice information exported to %s\n", outputFile)
			} else {
				// Print results to console
				if totalDevices > 0 {
					fmt.Printf("\nRegistered devices:\n")
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
					fmt.Fprintln(w, "DEVICE ID\tINVITATION\tREGISTERED AT")
					for _, device := range results.Successful {
						registeredAt := ""
						if reg, ok := device.Details["registered_at"]; ok {
							registeredAt = fmt.Sprintf("%v", reg)
						}
						fmt.Fprintf(w, "%s\t%s\t%s\n",
							device.DeviceID, device.Name, registeredAt)
					}
					w.Flush()
				} else {
					fmt.Println("No devices found")
				}
			}
		},
	}

	// Add flags for export device info command
	exportDeviceInfoCmd.Flags().String("output-file", "device_info.csv", "Path to export device info to a CSV file")
	exportDeviceInfoCmd.Flags().Int("concurrency", 5, "Number of concurrent operations")

	// Batch revoke all devices command
	batchRevokeAllCmd := &cobra.Command{
		Use:   "batch-revoke-all --confirm",
		Short: "Revoke all devices across all invitations",
		Long:  `Revoke all registered devices across all invitations.`,
		Run: func(cmd *cobra.Command, args []string) {
			confirm, _ := cmd.Flags().GetBool("confirm")
			concurrency, _ := cmd.Flags().GetInt("concurrency")
			outputFile, _ := cmd.Flags().GetString("output-file")

			if !confirm {
				fmt.Println("This operation will revoke ALL devices across ALL invitations.")
				fmt.Println("To confirm, use the --confirm flag")
				os.Exit(1)
			}

			// Create batch device manager
			batchManager, err := createBatchDeviceManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

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

			// Get all invitations
			invitations := invitationManager.ListInvitations()
			if len(invitations) == 0 {
				fmt.Println("No invitations found")
				os.Exit(0)
			}

			fmt.Printf("Gathering device information for %d invitation(s)...\n", len(invitations))

			// Get device info for all invitations
			infoResults := batchManager.BatchGetDeviceInfo(invitations, concurrency)
			if len(infoResults.Successful) == 0 {
				fmt.Println("No devices found")
				os.Exit(0)
			}

			// Convert to revoke operations
			revokeOperations := make([]profile.DeviceOperationResult, 0)
			for _, device := range infoResults.Successful {
				revokeOperations = append(revokeOperations, profile.DeviceOperationResult{
					DeviceID:  device.DeviceID,
					Token:     device.Token,
					Name:      device.Name,
					Operation: "revoke",
				})
			}

			fmt.Printf("Revoking %d device(s)...\n", len(revokeOperations))

			// Revoke all devices
			results := batchManager.BatchRevokeDevices(revokeOperations, concurrency)

			// Print summary
			fmt.Printf("\nProcessed %d device revocation(s):\n", results.TotalProcessed)
			fmt.Printf("  - Successful: %d\n", results.TotalSuccessful)
			fmt.Printf("  - Failed: %d\n", results.TotalFailed)

			// Export results if requested
			if outputFile != "" {
				err := batchManager.ExportDeviceInfoToCSVFile(outputFile, results)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error exporting results: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("\nResults exported to %s\n", outputFile)
			}
		},
	}

	// Add flags for batch revoke all command
	batchRevokeAllCmd.Flags().Bool("confirm", false, "Confirm revocation of all devices")
	batchRevokeAllCmd.Flags().Int("concurrency", 5, "Number of concurrent operations")
	batchRevokeAllCmd.Flags().String("output-file", "", "Path to export results to a CSV file")

	// Add commands to devices command
	devicesCmd.AddCommand(batchDeviceCmd)
	devicesCmd.AddCommand(exportDeviceInfoCmd)
	devicesCmd.AddCommand(batchRevokeAllCmd)
}