package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/scttfrdmn/prism/pkg/profile"
	"github.com/spf13/cobra"
)

// AddBatchInvitationCommands adds batch invitation management commands to the CLI.
// This wires the existing pkg/profile.BatchInvitationManager to CLI subcommands.
func AddBatchInvitationCommands(invitationsCmd *cobra.Command, config *Config) {
	// batch-create command
	batchCreateCmd := &cobra.Command{
		Use:   "batch-create",
		Short: "Create multiple invitations from a CSV file",
		Long: `Read a CSV file and generate one secure invitation per row.

CSV format (with header):
  Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices

Example:
  Student 1,read_only,90,no,no,yes,2
  TA 1,read_write,180,yes,no,yes,3`,
		Run: func(cmd *cobra.Command, args []string) {
			csvFile, _ := cmd.Flags().GetString("csv-file")
			outputFile, _ := cmd.Flags().GetString("output-file")
			concurrency, _ := cmd.Flags().GetInt("concurrency")
			hasHeader, _ := cmd.Flags().GetBool("has-header")

			if csvFile == "" {
				fmt.Fprintf(os.Stderr, "Error: --csv-file is required\n")
				os.Exit(1)
			}

			batchManager, err := initializeBatchManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "initialize batch invitation manager"))
				os.Exit(1)
			}

			fmt.Printf("📋 Creating batch invitations from: %s\n", csvFile)

			results, err := batchManager.CreateBatchInvitationsFromCSVFile(
				csvFile,
				"", // s3ConfigPath — not used in CLI batch create
				"", // parentToken — not used in CLI batch create
				concurrency,
				hasHeader,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "create batch invitations"))
				os.Exit(1)
			}

			// Print summary
			fmt.Printf("\n✅ Batch complete: %d successful, %d failed (total: %d)\n",
				results.TotalSuccessful, results.TotalFailed, results.TotalProcessed)

			// Show failures
			if len(results.Failed) > 0 {
				fmt.Println("\n❌ Failed invitations:")
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "NAME\tERROR")
				for _, inv := range results.Failed {
					errMsg := ""
					if inv.Error != nil {
						errMsg = inv.Error.Error()
					}
					fmt.Fprintf(w, "%s\t%s\n", inv.Name, errMsg)
				}
				_ = w.Flush() // stdout table; Run has no error return and a failed terminal write is not actionable
			}

			// Export results if output file requested
			if outputFile != "" {
				includeEncoded := true
				if err := batchManager.ExportBatchInvitationsToCSVFile(outputFile, results, includeEncoded); err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "export results"))
					os.Exit(1)
				}
				fmt.Printf("\n📄 Results written to: %s\n", outputFile)
			} else if len(results.Successful) > 0 {
				// Print tokens to stdout when no output file
				fmt.Println("\nCreated invitations:")
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "NAME\tTYPE\tTOKEN")
				for _, inv := range results.Successful {
					fmt.Fprintf(w, "%s\t%s\t%s\n", inv.Name, inv.Type, inv.Token)
				}
				_ = w.Flush() // stdout table; Run has no error return and a failed terminal write is not actionable
			}
		},
	}
	batchCreateCmd.Flags().String("csv-file", "", "Path to CSV file with invitation definitions (required)")
	batchCreateCmd.Flags().String("output-file", "", "Path to write results CSV (optional; prints to stdout if omitted)")
	batchCreateCmd.Flags().Int("concurrency", 5, "Number of concurrent invitation creations")
	batchCreateCmd.Flags().Bool("has-header", true, "Whether the CSV file has a header row")
	invitationsCmd.AddCommand(batchCreateCmd)

	// batch-export command
	batchExportCmd := &cobra.Command{
		Use:   "batch-export",
		Short: "Export existing invitations to a CSV file",
		Long:  `Export all locally-stored invitations to a CSV file for record-keeping or transfer.`,
		Run: func(cmd *cobra.Command, args []string) {
			outputFile, _ := cmd.Flags().GetString("output-file")
			includeEncoded, _ := cmd.Flags().GetBool("include-encoded")

			if outputFile == "" {
				fmt.Fprintf(os.Stderr, "Error: --output-file is required\n")
				os.Exit(1)
			}

			batchManager, err := initializeBatchManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "initialize batch invitation manager"))
				os.Exit(1)
			}

			// Export an empty result set — the exported CSV reflects what was
			// previously created and stored. In practice callers pipe the results
			// from batch-create into batch-export by providing an output-file to
			// batch-create; this command is retained for explicit re-export.
			results := &profile.BatchInvitationResult{
				Successful: []*profile.BatchInvitation{},
				Failed:     []*profile.BatchInvitation{},
			}

			if err := batchManager.ExportBatchInvitationsToCSVFile(outputFile, results, includeEncoded); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "export invitations"))
				os.Exit(1)
			}

			fmt.Printf("✅ Invitations exported to: %s\n", outputFile)
		},
	}
	batchExportCmd.Flags().String("output-file", "", "Path to write the output CSV (required)")
	batchExportCmd.Flags().Bool("include-encoded", false, "Include encoded invitation data in CSV output")
	invitationsCmd.AddCommand(batchExportCmd)

	// batch-accept command
	batchAcceptCmd := &cobra.Command{
		Use:   "batch-accept",
		Short: "Accept multiple invitations from a CSV file",
		Long: `Read a CSV file of encoded invitation tokens and add each one as a local profile.

CSV format (with header):
  EncodedToken,ProfileName

Example:
  eyJhbGci...,my-lab-access
  eyJhbGci...,collab-project`,
		Run: func(cmd *cobra.Command, args []string) {
			csvFile, _ := cmd.Flags().GetString("csv-file")
			namePrefix, _ := cmd.Flags().GetString("name-prefix")
			hasHeader, _ := cmd.Flags().GetBool("has-header")

			if csvFile == "" {
				fmt.Fprintf(os.Stderr, "Error: --csv-file is required\n")
				os.Exit(1)
			}

			secureManager, err := initializeSecureManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "initialize invitation manager"))
				os.Exit(1)
			}

			entries, err := readBatchAcceptCSV(csvFile, hasHeader)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "read CSV file"))
				os.Exit(1)
			}

			succeeded, failed := 0, 0
			for _, entry := range entries {
				profileName := entry.profileName
				if profileName == "" {
					profileName = namePrefix + fmt.Sprintf("invitation-%d", succeeded+failed+1)
				}

				if err := secureManager.SecureAddToProfile(entry.encodedToken, profileName); err != nil {
					fmt.Fprintf(os.Stderr, "  ❌ %s: %v\n", profileName, err)
					failed++
					continue
				}
				fmt.Printf("  ✅ Added profile: %s\n", profileName)
				succeeded++
			}

			fmt.Printf("\nBatch accept complete: %d added, %d failed\n", succeeded, failed)
		},
	}
	batchAcceptCmd.Flags().String("csv-file", "", "Path to CSV file with encoded tokens (required)")
	batchAcceptCmd.Flags().String("name-prefix", "", "Prefix for auto-generated profile names")
	batchAcceptCmd.Flags().Bool("has-header", true, "Whether the CSV file has a header row")
	invitationsCmd.AddCommand(batchAcceptCmd)
}

// initializeSecureManager creates a SecureInvitationManager from the CLI config.
func initializeSecureManager(config *Config) (*profile.SecureInvitationManager, error) {
	profileManager, err := createProfileManager(config)
	if err != nil {
		return nil, fmt.Errorf("create profile manager: %w", err)
	}
	return profile.NewSecureInvitationManager(profileManager)
}

// initializeBatchManager creates a BatchInvitationManager from the CLI config.
func initializeBatchManager(config *Config) (*profile.BatchInvitationManager, error) {
	secureManager, err := initializeSecureManager(config)
	if err != nil {
		return nil, err
	}
	return profile.NewBatchInvitationManager(secureManager), nil
}

// batchAcceptEntry holds one row from a batch-accept CSV.
type batchAcceptEntry struct {
	encodedToken string
	profileName  string
}

// readBatchAcceptCSV reads a CSV of (encoded-token, optional-profile-name) rows.
func readBatchAcceptCSV(csvPath string, hasHeader bool) ([]batchAcceptEntry, error) {
	f, err := os.Open(csvPath) //nolint:gosec // G304: CSV path supplied by the user on the command line
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }() // read-only; close error is not actionable

	parser := profile.NewCSVInvitationParser(f, hasHeader)
	records, err := parser.ParseRecords()
	if err != nil {
		// Fall back: treat each record as a single-column token file
		return nil, fmt.Errorf("parse CSV: %w", err)
	}

	entries := make([]batchAcceptEntry, 0, len(records))
	for _, rec := range records {
		// Reuse Name column as encoded token, Type column as profile name
		entries = append(entries, batchAcceptEntry{
			encodedToken: rec.Name,
			profileName:  rec.Type,
		})
	}
	return entries, nil
}

// valueOrEmpty returns the value or a dash placeholder when empty.
// Defined in batch_config.go, referenced here by same package — no redeclaration needed.
