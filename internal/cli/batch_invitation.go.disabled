package cli

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/fatih/color"
)

// AddBatchInvitationCommands adds batch invitation commands to the CLI
func AddBatchInvitationCommands(invitationsCmd *cobra.Command, config *Config) {
	// Create batch invitation manager
	createBatchInvitationManager := func() (*profile.BatchInvitationManager, error) {
		profileManager, err := createProfileManager(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile manager: %w", err)
		}

		secureManager, err := profile.NewSecureInvitationManager(profileManager)
		if err != nil {
			return nil, fmt.Errorf("failed to create secure invitation manager: %w", err)
		}

		return profile.NewBatchInvitationManager(secureManager), nil
	}

	// Batch create command
	batchCreateCmd := &cobra.Command{
		Use:   "batch-create --csv-file [path] --s3-config [s3-path] --parent-token [token] --concurrency [num]",
		Short: "Create multiple invitations from a CSV file",
		Long: `Create multiple invitations at once from a CSV file containing recipient details.

CSV format should have the following columns (first row can be a header):
- Name: Required, the recipient's name
- Type: Required, one of: read_only, read_write, admin
- ValidDays: Optional, number of days the invitation is valid (default: 30)
- CanInvite: Optional, whether the recipient can invite others (default: false, true for admin)
- Transferable: Optional, whether the invitation can be transferred (default: false)
- DeviceBound: Optional, whether the invitation is bound to the device (default: true)
- MaxDevices: Optional, maximum number of devices allowed (default: 1)

Example CSV:
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Test User 1,read_only,30,no,no,yes,1
Test User 2,read_write,60,no,no,yes,2
Test Admin,admin,90,yes,no,yes,3`,
		Run: func(cmd *cobra.Command, args []string) {
			csvFile, _ := cmd.Flags().GetString("csv-file")
			s3ConfigPath, _ := cmd.Flags().GetString("s3-config")
			parentToken, _ := cmd.Flags().GetString("parent-token")
			concurrency, _ := cmd.Flags().GetInt("concurrency")
			hasHeader, _ := cmd.Flags().GetBool("has-header")
			outputFile, _ := cmd.Flags().GetString("output-file")
			includeEncoded, _ := cmd.Flags().GetBool("include-encoded")

			// Create batch invitation manager
			batchManager, err := createBatchInvitationManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Process invitations
			fmt.Printf("Processing batch invitations from %s...\n", csvFile)
			results, err := batchManager.CreateBatchInvitationsFromCSVFile(
				csvFile,
				s3ConfigPath,
				parentToken,
				concurrency,
				hasHeader,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating batch invitations: %v\n", err)
				os.Exit(1)
			}

			// Print summary
			fmt.Printf("\nProcessed %d invitation(s):\n", results.TotalProcessed)
			fmt.Printf("  - Successful: %d\n", results.TotalSuccessful)
			fmt.Printf("  - Failed: %d\n", results.TotalFailed)

			// Export results if requested
			if outputFile != "" {
				err := batchManager.ExportBatchInvitationsToCSVFile(outputFile, results, includeEncoded)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error exporting results: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("\nResults exported to %s\n", outputFile)
			} else {
				// Print results to console if not exporting
				if results.TotalSuccessful > 0 {
					fmt.Printf("\nSuccessful invitations:\n")
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
					fmt.Fprintln(w, "NAME\tTYPE\tTOKEN\tVALID DAYS")
					for _, inv := range results.Successful {
						fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
							inv.Name, inv.Type, inv.Token, inv.ValidDays)
					}
					w.Flush()
				}

				if results.TotalFailed > 0 {
					fmt.Printf("\nFailed invitations:\n")
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
					fmt.Fprintln(w, "NAME\tTYPE\tERROR")
					for _, inv := range results.Failed {
						fmt.Fprintf(w, "%s\t%s\t%s\n",
							inv.Name, inv.Type, inv.Error)
					}
					w.Flush()
				}
			}
		},
	}

	// Add flags for batch create command
	batchCreateCmd.Flags().String("csv-file", "", "Path to CSV file containing invitation details")
	batchCreateCmd.Flags().String("s3-config", "", "Optional S3 path to configuration")
	batchCreateCmd.Flags().String("parent-token", "", "Optional parent invitation token")
	batchCreateCmd.Flags().Int("concurrency", 5, "Number of concurrent invitation creations")
	batchCreateCmd.Flags().Bool("has-header", true, "Whether the CSV file has a header row")
	batchCreateCmd.Flags().String("output-file", "", "Path to export results to a CSV file")
	batchCreateCmd.Flags().Bool("include-encoded", false, "Include encoded invitation data in the output")
	batchCreateCmd.MarkFlagRequired("csv-file")

	// Export batch invitation results
	batchExportCmd := &cobra.Command{
		Use:   "batch-export --result-file [path] --input-file [csv-path]",
		Short: "Export invitation results to a CSV file",
		Long: `Export batch invitation results to a CSV file for distribution.

This command takes a list of previously created invitations and exports
them to a CSV file suitable for sharing with recipients.`,
		Run: func(cmd *cobra.Command, args []string) {
			outputFile, _ := cmd.Flags().GetString("output-file")
			includeEncoded, _ := cmd.Flags().GetBool("include-encoded")

			// Create batch invitation manager
			batchManager, err := createBatchInvitationManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Get all invitations
			profileManager, err := createProfileManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			
			invitationManager, err := createInvitationManager(profileManager)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// List all invitations
			allInvitations := invitationManager.ListInvitations()
			if len(allInvitations) == 0 {
				fmt.Fprintf(os.Stderr, "Error: No invitations found\n")
				os.Exit(1)
			}

			// Convert to batch format
			batchInvitations := make([]*profile.BatchInvitation, len(allInvitations))
			for i, inv := range allInvitations {
				// Create encoded form for sharing
				encodedData, err := inv.EncodeToString()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to encode invitation %s: %v\n", inv.Token, err)
				}

				batchInvitations[i] = &profile.BatchInvitation{
					Name:        inv.Name,
					Type:        inv.Type,
					ValidDays:   int(inv.GetExpirationDuration().Hours() / 24),
					CanInvite:   inv.CanInvite,
					Transferable: inv.Transferable,
					DeviceBound: inv.DeviceBound,
					MaxDevices:  inv.MaxDevices,
					Token:       inv.Token,
					EncodedData: encodedData,
				}
			}

			// Create batch result from existing invitations
			results := &profile.BatchInvitationResult{
				Successful:     batchInvitations,
				Failed:         []*profile.BatchInvitation{},
				TotalProcessed: len(batchInvitations),
				TotalSuccessful: len(batchInvitations),
				TotalFailed:    0,
			}

			// Export to CSV
			err = batchManager.ExportBatchInvitationsToCSVFile(outputFile, results, includeEncoded)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting invitations: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully exported %d invitation(s) to %s\n", len(allInvitations), outputFile)
		},
	}

	// Add flags for batch export command
	batchExportCmd.Flags().String("output-file", "invitations.csv", "Path to export results to a CSV file")
	batchExportCmd.Flags().Bool("include-encoded", true, "Include encoded invitation data in the output")

	// Batch accept command
	batchAcceptCmd := &cobra.Command{
		Use:   "batch-accept --csv-file [path] --name-prefix [prefix]",
		Short: "Accept multiple invitations from a CSV file",
		Long: `Accept multiple invitations at once from a CSV file.

The CSV file should contain encoded invitations generated by batch-export.
Each invitation will be accepted and a profile created.`,
		Run: func(cmd *cobra.Command, args []string) {
			csvFile, _ := cmd.Flags().GetString("csv-file")
			namePrefix, _ := cmd.Flags().GetString("name-prefix")
			hasHeader, _ := cmd.Flags().GetBool("has-header")

			// Read the CSV file
			file, err := os.Open(csvFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening CSV file: %v\n", err)
				os.Exit(1)
			}
			defer file.Close()

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

			// Create secure invitation manager
			secureManager, err := profile.NewSecureInvitationManager(profileManager)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Process CSV file
			csvReader := csv.NewReader(file)
			
			// Skip header if needed
			if hasHeader {
				_, err = csvReader.Read()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading CSV header: %v\n", err)
					os.Exit(1)
				}
			}

			// Read and process rows
			successful := 0
			failed := 0
			
			for rowNum := 1; ; rowNum++ {
				record, err := csvReader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading CSV: %v\n", err)
					os.Exit(1)
				}

				// We need at least Name and EncodedData columns
				if len(record) < 10 {
					fmt.Fprintf(os.Stderr, "Invalid CSV format at row %d\n", rowNum)
					failed++
					continue
				}

				name := record[0]
				encodedData := record[9] // 10th column should be the encoded data

				if encodedData == "" {
					fmt.Fprintf(os.Stderr, "Missing encoded invitation at row %d\n", rowNum)
					failed++
					continue
				}

				// Generate profile name from invitation name
				profileName := fmt.Sprintf("%s-%s", namePrefix, name)
				if namePrefix == "" {
					profileName = name
				}

				// Accept the invitation
				err = secureManager.SecureAddToProfile(encodedData, profileName)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error accepting invitation for %s: %v\n", name, err)
					failed++
				} else {
					fmt.Printf("Accepted invitation for %s\n", name)
					successful++
				}
			}

			// Print summary
			fmt.Printf("\nProcessed %d invitation(s):\n", successful+failed)
			fmt.Printf("  - Successful: %d\n", successful)
			fmt.Printf("  - Failed: %d\n", failed)
		},
	}

	// Add flags for batch accept command
	batchAcceptCmd.Flags().String("csv-file", "", "Path to CSV file containing encoded invitations")
	batchAcceptCmd.Flags().String("name-prefix", "", "Optional prefix for created profile names")
	batchAcceptCmd.Flags().Bool("has-header", true, "Whether the CSV file has a header row")
	batchAcceptCmd.MarkFlagRequired("csv-file")

	// Add commands to invitations command
	invitationsCmd.AddCommand(batchCreateCmd)
	invitationsCmd.AddCommand(batchExportCmd)
	invitationsCmd.AddCommand(batchAcceptCmd)
}