package profile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

func TestBatchInvitationIntegration(t *testing.T) {
	// Skip integration tests in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration tests in CI environment")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-batch-invitation-integration-test")
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

	// Create a profile manager for testing
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		t.Fatalf("Failed to create profile manager: %v", err)
	}

	// Create a secure invitation manager
	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		t.Fatalf("Failed to create secure invitation manager: %v", err)
	}

	// Create a batch invitation manager
	batchManager := profile.NewBatchInvitationManager(secureManager)

	// Test end-to-end batch invitation flow
	t.Run("EndToEndBatchFlow", func(t *testing.T) {
		// 1. Create CSV file with test data
		csvData := `Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Integration Test User 1,read_only,30,no,no,yes,1
Integration Test User 2,read_write,60,no,no,yes,2
Integration Test Admin,admin,90,yes,no,yes,3`

		csvFile := filepath.Join(tempDir, "integration-test.csv")
		if err := os.WriteFile(csvFile, []byte(csvData), 0644); err != nil {
			t.Fatalf("Failed to write test CSV file: %v", err)
		}

		// 2. Import invitations from CSV
		invitations, err := batchManager.ImportBatchInvitationsFromCSVFile(csvFile, true)
		if err != nil {
			t.Fatalf("Failed to import invitations from CSV file: %v", err)
		}

		if len(invitations) != 3 {
			t.Errorf("Expected 3 imported invitations, got %d", len(invitations))
		}

		// 3. Create batch invitations
		results := batchManager.CreateBatchInvitations(invitations, "", "", 2)

		// 4. Check results
		if results.TotalProcessed != 3 {
			t.Errorf("Expected 3 processed invitations, got %d", results.TotalProcessed)
		}

		// The actual success count may vary based on the environment, especially in CI
		t.Logf("Processed: %d, Successful: %d, Failed: %d", 
			results.TotalProcessed, results.TotalSuccessful, results.TotalFailed)

		// 5. Export results to CSV
		outputFile := filepath.Join(tempDir, "integration-results.csv")
		err = batchManager.ExportBatchInvitationsToCSVFile(outputFile, results, true)
		if err != nil {
			t.Fatalf("Failed to export batch invitations to CSV: %v", err)
		}

		// 6. Verify output file exists
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output CSV file was not created")
		}

		// 7. Read output CSV to verify content
		outputData, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output CSV file: %v", err)
		}

		outputContent := string(outputData)
		if !strings.Contains(outputContent, "Name,Type,Token,Valid Days,Can Invite,Transferable,Device Bound,Max Devices,Status") {
			t.Errorf("Expected CSV header in output")
		}

		if !strings.Contains(outputContent, "Integration Test User 1,read_only,") {
			t.Errorf("Expected 'Integration Test User 1' in CSV output")
		}
	})

	// Test error handling with invalid CSV
	t.Run("InvalidCSVHandling", func(t *testing.T) {
		// Create invalid CSV file
		invalidCsv := `Name,Type,Invalid
User,read_only,not-a-number`

		invalidFile := filepath.Join(tempDir, "invalid-test.csv")
		if err := os.WriteFile(invalidFile, []byte(invalidCsv), 0644); err != nil {
			t.Fatalf("Failed to write invalid CSV file: %v", err)
		}

		// Try to import invitations
		_, err := batchManager.ImportBatchInvitationsFromCSVFile(invalidFile, true)
		if err == nil {
			t.Errorf("Expected error when importing invalid CSV, but got none")
		}
	})

	// Test export functionality
	t.Run("BatchExportFunctionality", func(t *testing.T) {
		// Create test invitations
		invitations := []*profile.BatchInvitation{
			{
				Name:        "Export Test 1",
				Type:        profile.InvitationTypeReadOnly,
				ValidDays:   30,
				DeviceBound: true,
				Token:       "test-token-1",
				EncodedData: "encoded-data-1",
			},
			{
				Name:        "Export Test 2",
				Type:        profile.InvitationTypeReadWrite,
				ValidDays:   60,
				DeviceBound: true,
				Token:       "test-token-2",
				EncodedData: "encoded-data-2",
				Error:       nil,
			},
		}

		// Create result structure
		results := &profile.BatchInvitationResult{
			Successful:     invitations,
			Failed:         []*profile.BatchInvitation{},
			TotalProcessed: len(invitations),
			TotalSuccessful: len(invitations),
			TotalFailed:    0,
		}

		// Export to buffer for testing
		var buf bytes.Buffer
		err := batchManager.ExportBatchInvitationsToCSV(&buf, results, true)
		if err != nil {
			t.Fatalf("Failed to export batch invitations to CSV: %v", err)
		}

		// Verify content
		output := buf.String()
		if !strings.Contains(output, "Export Test 1,read_only,test-token-1,30,no,no,yes,1,Success") {
			t.Errorf("Expected Export Test 1 data in CSV output")
		}
		if !strings.Contains(output, "encoded-data-1") {
			t.Errorf("Expected encoded data in CSV output")
		}
	})
}