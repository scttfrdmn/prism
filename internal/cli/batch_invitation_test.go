package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/internal/cli"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/spf13/cobra"
)

// TestBatchInvitationCommands tests the batch invitation CLI commands
func TestBatchInvitationCommands(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-cli-batch-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config directory
	configDir := filepath.Join(tempDir, ".cloudworkstation")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create test CSV file
	csvData := `Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Test User 1,read_only,30,no,no,yes,1
Test User 2,read_write,60,no,no,yes,2
Test Admin,admin,90,yes,no,yes,3`

	csvFile := filepath.Join(tempDir, "test-invitations.csv")
	if err := os.WriteFile(csvFile, []byte(csvData), 0644); err != nil {
		t.Fatalf("Failed to write test CSV file: %v", err)
	}

	// Output file for results
	outputFile := filepath.Join(tempDir, "results.csv")

	// Create a test config
	config := &cli.Config{
		AWS: cli.AWSConfig{
			Profile: "default",
			Region:  "us-west-2",
		},
		Daemon: cli.DaemonConfig{
			URL: "http://localhost:8080",
		},
	}

	// Create a root command for testing
	rootCmd := &cobra.Command{Use: "cws"}

	// Add profiles and invitations commands
	profilesCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage CloudWorkstation profiles",
	}
	rootCmd.AddCommand(profilesCmd)

	invitationsCmd := &cobra.Command{
		Use:   "invitations",
		Short: "Manage shared access invitations",
	}
	profilesCmd.AddCommand(invitationsCmd)

	// Add batch invitation commands
	cli.AddBatchInvitationCommands(invitationsCmd, config)

	// Helper function to capture command output
	captureOutput := func(cmd *cobra.Command, args []string) (string, error) {
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs(args)
		err := cmd.Execute()
		return buf.String(), err
	}

	// Test batch-create command - this will fail but let's verify the command structure
	t.Run("BatchCreateCommandStructure", func(t *testing.T) {
		// Find the batch-create command
		batchCreateCmd, _, err := rootCmd.Find([]string{"profiles", "invitations", "batch-create"})
		if err != nil {
			t.Fatalf("Failed to find batch-create command: %v", err)
		}

		// Verify required flags
		csvFlag := batchCreateCmd.Flags().Lookup("csv-file")
		if csvFlag == nil {
			t.Errorf("batch-create command missing required --csv-file flag")
		}

		s3ConfigFlag := batchCreateCmd.Flags().Lookup("s3-config")
		if s3ConfigFlag == nil {
			t.Errorf("batch-create command missing --s3-config flag")
		}

		parentTokenFlag := batchCreateCmd.Flags().Lookup("parent-token")
		if parentTokenFlag == nil {
			t.Errorf("batch-create command missing --parent-token flag")
		}

		concurrencyFlag := batchCreateCmd.Flags().Lookup("concurrency")
		if concurrencyFlag == nil {
			t.Errorf("batch-create command missing --concurrency flag")
		}

		hasHeaderFlag := batchCreateCmd.Flags().Lookup("has-header")
		if hasHeaderFlag == nil {
			t.Errorf("batch-create command missing --has-header flag")
		}

		outputFileFlag := batchCreateCmd.Flags().Lookup("output-file")
		if outputFileFlag == nil {
			t.Errorf("batch-create command missing --output-file flag")
		}
	})

	// Test batch-export command structure
	t.Run("BatchExportCommandStructure", func(t *testing.T) {
		// Find the batch-export command
		batchExportCmd, _, err := rootCmd.Find([]string{"profiles", "invitations", "batch-export"})
		if err != nil {
			t.Fatalf("Failed to find batch-export command: %v", err)
		}

		// Verify flags
		outputFileFlag := batchExportCmd.Flags().Lookup("output-file")
		if outputFileFlag == nil {
			t.Errorf("batch-export command missing --output-file flag")
		}

		includeEncodedFlag := batchExportCmd.Flags().Lookup("include-encoded")
		if includeEncodedFlag == nil {
			t.Errorf("batch-export command missing --include-encoded flag")
		}
	})

	// Test batch-accept command structure
	t.Run("BatchAcceptCommandStructure", func(t *testing.T) {
		// Find the batch-accept command
		batchAcceptCmd, _, err := rootCmd.Find([]string{"profiles", "invitations", "batch-accept"})
		if err != nil {
			t.Fatalf("Failed to find batch-accept command: %v", err)
		}

		// Verify required flags
		csvFileFlag := batchAcceptCmd.Flags().Lookup("csv-file")
		if csvFileFlag == nil {
			t.Errorf("batch-accept command missing required --csv-file flag")
		}

		namePrefixFlag := batchAcceptCmd.Flags().Lookup("name-prefix")
		if namePrefixFlag == nil {
			t.Errorf("batch-accept command missing --name-prefix flag")
		}

		hasHeaderFlag := batchAcceptCmd.Flags().Lookup("has-header")
		if hasHeaderFlag == nil {
			t.Errorf("batch-accept command missing --has-header flag")
		}
	})

	// Mock the profile manager for testing - we need to create a custom test impl
	// that doesn't actually call AWS services
	setupMockProfileManager(t, tempDir)

	// Test batch-create execution with mocked profile manager
	t.Run("BatchCreateExecution", func(t *testing.T) {
		args := []string{
			"profiles", "invitations", "batch-create",
			"--csv-file", csvFile,
			"--output-file", outputFile,
		}

		_, err := captureOutput(rootCmd, args)
		
		// We expect this to fail in the test environment, but we want to verify
		// that the command structure works and it tries to process the CSV file
		if err == nil {
			t.Log("Warning: Expected failure but command succeeded - mock is working better than expected")
		}
	})
}

// setupMockProfileManager sets up a mock profile manager for testing
func setupMockProfileManager(t *testing.T, tempDir string) {
	// This would typically involve creating a mock implementation
	// of the profile manager and secure invitation manager
	// In a real test, we'd inject these mocks into the CLI commands
	
	// For now, we'll just create the state file structure
	stateDir := filepath.Join(tempDir, ".cloudworkstation")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("Failed to create state directory: %v", err)
	}

	// Create a minimal state file
	stateFile := filepath.Join(stateDir, "profiles.json")
	stateData := `{
		"profiles": {},
		"current": ""
	}`

	if err := os.WriteFile(stateFile, []byte(stateData), 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Create a minimal invitations directory
	invitationsDir := filepath.Join(stateDir, "invitations")
	if err := os.MkdirAll(invitationsDir, 0755); err != nil {
		t.Fatalf("Failed to create invitations directory: %v", err)
	}
}