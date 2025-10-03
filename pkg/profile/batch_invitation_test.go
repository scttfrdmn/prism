package profile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

func TestBatchInvitationImportExport(t *testing.T) {
	batchManager := setupBatchInvitationTest(t)

	t.Run("csv_import_and_validation", func(t *testing.T) {
		testCSVImport(t, batchManager)
	})

	t.Run("batch_creation_process", func(t *testing.T) {
		invitations := createTestInvitations(t, batchManager)
		testBatchCreation(t, batchManager, invitations)
	})

	t.Run("csv_export_functionality", func(t *testing.T) {
		invitations := createTestInvitations(t, batchManager)
		results := batchManager.CreateBatchInvitations(invitations, "", "", 2)
		testCSVExport(t, batchManager, results)
	})

	t.Run("file_based_operations", func(t *testing.T) {
		testFileOperations(t, batchManager)
	})
}

func TestBatchInvitationEdgeCases(t *testing.T) {
	batchManager := setupBatchInvitationTest(t)

	t.Run("minimal_csv_with_defaults", func(t *testing.T) {
		testMinimalCSVDefaults(t, batchManager)
	})

	t.Run("invalid_csv_format_handling", func(t *testing.T) {
		testInvalidCSVFormats(t, batchManager)
	})

	t.Run("case_insensitive_type_parsing", func(t *testing.T) {
		testCaseInsensitiveTypes(t, batchManager)
	})

	t.Run("boolean_field_parsing_variations", func(t *testing.T) {
		testBooleanFieldParsing(t, batchManager)
	})
}

// Helper functions for batch invitation testing

func setupBatchInvitationTest(t *testing.T) *profile.BatchInvitationManager {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-batch-invitation-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Create config directory
	configDir := filepath.Join(tempDir, ".cloudworkstation")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	t.Cleanup(func() { _ = os.Setenv("HOME", originalHome) })

	// Create managers
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		t.Fatalf("Failed to create profile manager: %v", err)
	}

	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		t.Fatalf("Failed to create secure invitation manager: %v", err)
	}

	return profile.NewBatchInvitationManager(secureManager)
}

func getTestCSVData() string {
	return `Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Test User 1,read_only,30,no,no,yes,1
Test User 2,read_write,60,no,no,yes,2
Test Admin,admin,90,yes,no,yes,3`
}

func createTestInvitations(t *testing.T, batchManager *profile.BatchInvitationManager) []*profile.BatchInvitation {
	invitations, err := batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(getTestCSVData()), true)
	if err != nil {
		t.Fatalf("Failed to import invitations from CSV: %v", err)
	}
	return invitations
}

func testCSVImport(t *testing.T, batchManager *profile.BatchInvitationManager) {
	invitations := createTestInvitations(t, batchManager)

	// Check imported invitations
	if len(invitations) != 3 {
		t.Errorf("Expected 3 invitations, got %d", len(invitations))
	}

	// Test cases for invitation validation
	testCases := []struct {
		index        int
		name         string
		invType      profile.InvitationType
		validDays    int
		canInvite    bool
		transferable bool
		deviceBound  bool
		maxDevices   int
	}{
		{0, "Test User 1", profile.InvitationTypeReadOnly, 30, false, false, true, 1},
		{2, "Test Admin", profile.InvitationTypeAdmin, 90, true, false, true, 3},
	}

	for _, tc := range testCases {
		inv := invitations[tc.index]
		if inv.Name != tc.name {
			t.Errorf("Expected name to be '%s', got '%s'", tc.name, inv.Name)
		}
		if inv.Type != tc.invType {
			t.Errorf("Expected type to be %s, got %s", tc.invType, inv.Type)
		}
		if inv.ValidDays != tc.validDays {
			t.Errorf("Expected valid days to be %d, got %d", tc.validDays, inv.ValidDays)
		}
		if inv.CanInvite != tc.canInvite {
			t.Errorf("Expected canInvite to be %v, got %v", tc.canInvite, inv.CanInvite)
		}
		if inv.MaxDevices != tc.maxDevices {
			t.Errorf("Expected maxDevices to be %d, got %d", tc.maxDevices, inv.MaxDevices)
		}
	}
}

func testBatchCreation(t *testing.T, batchManager *profile.BatchInvitationManager, invitations []*profile.BatchInvitation) {
	results := batchManager.CreateBatchInvitations(invitations, "", "", 2)

	// Check results
	expectedCounts := map[string]int{
		"total":      3,
		"successful": 3,
		"failed":     0,
	}

	if results.TotalProcessed != expectedCounts["total"] {
		t.Errorf("Expected %d processed invitations, got %d", expectedCounts["total"], results.TotalProcessed)
	}
	if results.TotalSuccessful != expectedCounts["successful"] {
		t.Errorf("Expected %d successful invitations, got %d", expectedCounts["successful"], results.TotalSuccessful)
	}
	if results.TotalFailed != expectedCounts["failed"] {
		t.Errorf("Expected %d failed invitations, got %d", expectedCounts["failed"], results.TotalFailed)
	}

	// Check successful invitations
	for _, inv := range results.Successful {
		if inv.Token == "" {
			t.Errorf("Expected token to be set for successful invitation")
		}
		if inv.EncodedData == "" {
			t.Errorf("Expected encodedData to be set for successful invitation")
		}
	}
}

func testCSVExport(t *testing.T, batchManager *profile.BatchInvitationManager, results *profile.BatchInvitationResult) {
	var buf bytes.Buffer
	err := batchManager.ExportBatchInvitationsToCSV(&buf, results, false)
	if err != nil {
		t.Fatalf("Failed to export invitations to CSV: %v", err)
	}

	csvOutput := buf.String()
	expectedContent := []string{
		"Name,Type,Token,Valid Days,Can Invite,Transferable,Device Bound,Max Devices,Status,Error",
		"Test User 1,read_only,",
		"Test Admin,admin,",
		"Success",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(csvOutput, expected) {
			t.Errorf("Expected '%s' in CSV output", expected)
		}
	}
}

func testFileOperations(t *testing.T, batchManager *profile.BatchInvitationManager) {
	tempDir, err := os.MkdirTemp("", "cws-file-ops-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Test file-based import/export
	csvFilePath := filepath.Join(tempDir, "test-invitations.csv")
	if err := os.WriteFile(csvFilePath, []byte(getTestCSVData()), 0644); err != nil {
		t.Fatalf("Failed to write test CSV file: %v", err)
	}

	// Test file-based import and creation
	fileResults, err := batchManager.CreateBatchInvitationsFromCSVFile(csvFilePath, "", "", 2, true)
	if err != nil {
		t.Fatalf("Failed to create invitations from CSV file: %v", err)
	}

	if fileResults.TotalSuccessful != 3 {
		t.Errorf("Expected 3 successful invitations from file, got %d", fileResults.TotalSuccessful)
	}

	// Test file-based export
	outputCSVPath := filepath.Join(tempDir, "output-invitations.csv")
	err = batchManager.ExportBatchInvitationsToCSVFile(outputCSVPath, fileResults, true)
	if err != nil {
		t.Fatalf("Failed to export invitations to CSV file: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputCSVPath); os.IsNotExist(err) {
		t.Errorf("Output CSV file was not created")
	}
}

func testMinimalCSVDefaults(t *testing.T, batchManager *profile.BatchInvitationManager) {
	minimalCSV := `Test User 1,read_only
Test User 2,read_write
Test Admin,admin`

	minimalInvitations, err := batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(minimalCSV), false)
	if err != nil {
		t.Fatalf("Failed to import minimal invitations from CSV: %v", err)
	}

	if len(minimalInvitations) != 3 {
		t.Errorf("Expected 3 invitations from minimal CSV, got %d", len(minimalInvitations))
	}

	// Test default values using table-driven approach
	defaultTests := []struct {
		index       int
		validDays   int
		canInvite   bool
		deviceBound bool
		maxDevices  int
	}{
		{0, 30, false, true, 1}, // read_only defaults
		{2, 30, true, true, 1},  // admin defaults
	}

	for _, test := range defaultTests {
		inv := minimalInvitations[test.index]
		if inv.ValidDays != test.validDays {
			t.Errorf("Expected ValidDays to be %d, got %d", test.validDays, inv.ValidDays)
		}
		if inv.CanInvite != test.canInvite {
			t.Errorf("Expected CanInvite to be %v, got %v", test.canInvite, inv.CanInvite)
		}
		if inv.DeviceBound != test.deviceBound {
			t.Errorf("Expected DeviceBound to be %v, got %v", test.deviceBound, inv.DeviceBound)
		}
		if inv.MaxDevices != test.maxDevices {
			t.Errorf("Expected MaxDevices to be %d, got %d", test.maxDevices, inv.MaxDevices)
		}
	}
}

func testInvalidCSVFormats(t *testing.T, batchManager *profile.BatchInvitationManager) {
	invalidTests := []struct {
		name    string
		csvData string
		desc    string
	}{
		{"header_only", `Name`, "invalid CSV format"},
		{"empty_name", `,read_only`, "empty name"},
		{"invalid_type", `Test User,invalid_type`, "invalid invitation type"},
	}

	for _, test := range invalidTests {
		t.Run(test.name, func(t *testing.T) {
			_, err := batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(test.csvData), false)
			if err == nil {
				t.Errorf("Expected error for %s", test.desc)
			}
		})
	}
}

func testCaseInsensitiveTypes(t *testing.T, batchManager *profile.BatchInvitationManager) {
	caseInsensitiveCSV := `Test User 1,READ_ONLY
Test User 2,ReadWrite
Test Admin,Admin`

	caseInvitations, err := batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(caseInsensitiveCSV), false)
	if err != nil {
		t.Fatalf("Failed to import case-insensitive invitations from CSV: %v", err)
	}

	if len(caseInvitations) != 3 {
		t.Errorf("Expected 3 invitations from case-insensitive CSV, got %d", len(caseInvitations))
	}

	// Test case-insensitive type mapping
	expectedTypes := []profile.InvitationType{
		profile.InvitationTypeReadOnly,
		profile.InvitationTypeReadWrite,
		profile.InvitationTypeAdmin,
	}

	for i, expectedType := range expectedTypes {
		if caseInvitations[i].Type != expectedType {
			t.Errorf("Expected type to be %s for index %d, got %s", expectedType, i, caseInvitations[i].Type)
		}
	}
}

func testBooleanFieldParsing(t *testing.T, batchManager *profile.BatchInvitationManager) {
	boolCSV := `Test User 1,read_only,30,true,yes,y,1
Test User 2,read_write,60,false,no,n,2
Test User 3,read_only,90,0,1,True,3`

	boolInvitations, err := batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(boolCSV), false)
	if err != nil {
		t.Fatalf("Failed to import boolean invitations from CSV: %v", err)
	}

	// Test boolean parsing with table-driven tests
	boolTests := []struct {
		index        int
		canInvite    bool
		transferable bool
		deviceBound  bool
	}{
		{0, true, true, true},    // true, yes, y
		{1, false, false, false}, // false, no, n
		{2, false, true, true},   // 0, 1, True
	}

	for _, test := range boolTests {
		inv := boolInvitations[test.index]
		if inv.CanInvite != test.canInvite {
			t.Errorf("Expected CanInvite to be %v for index %d, got %v", test.canInvite, test.index, inv.CanInvite)
		}
		if inv.Transferable != test.transferable {
			t.Errorf("Expected Transferable to be %v for index %d, got %v", test.transferable, test.index, inv.Transferable)
		}
		if inv.DeviceBound != test.deviceBound {
			t.Errorf("Expected DeviceBound to be %v for index %d, got %v", test.deviceBound, test.index, inv.DeviceBound)
		}
	}
}
