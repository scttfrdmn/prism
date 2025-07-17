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
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-batch-invitation-test")
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

	// Test CSV import
	csvData := `Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Test User 1,read_only,30,no,no,yes,1
Test User 2,read_write,60,no,no,yes,2
Test Admin,admin,90,yes,no,yes,3`

	// Parse CSV
	invitations, err := batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(csvData), true)
	if err != nil {
		t.Fatalf("Failed to import invitations from CSV: %v", err)
	}

	// Check imported invitations
	if len(invitations) != 3 {
		t.Errorf("Expected 3 invitations, got %d", len(invitations))
	}

	// Verify first invitation
	if invitations[0].Name != "Test User 1" {
		t.Errorf("Expected name to be 'Test User 1', got '%s'", invitations[0].Name)
	}
	if invitations[0].Type != profile.InvitationTypeReadOnly {
		t.Errorf("Expected type to be read_only, got '%s'", invitations[0].Type)
	}
	if invitations[0].ValidDays != 30 {
		t.Errorf("Expected valid days to be 30, got %d", invitations[0].ValidDays)
	}
	if invitations[0].CanInvite {
		t.Errorf("Expected canInvite to be false")
	}
	if invitations[0].Transferable {
		t.Errorf("Expected transferable to be false")
	}
	if !invitations[0].DeviceBound {
		t.Errorf("Expected deviceBound to be true")
	}
	if invitations[0].MaxDevices != 1 {
		t.Errorf("Expected maxDevices to be 1, got %d", invitations[0].MaxDevices)
	}

	// Verify admin invitation
	if invitations[2].Name != "Test Admin" {
		t.Errorf("Expected name to be 'Test Admin', got '%s'", invitations[2].Name)
	}
	if invitations[2].Type != profile.InvitationTypeAdmin {
		t.Errorf("Expected type to be admin, got '%s'", invitations[2].Type)
	}
	if !invitations[2].CanInvite {
		t.Errorf("Expected canInvite to be true for admin")
	}
	if invitations[2].MaxDevices != 3 {
		t.Errorf("Expected maxDevices to be 3, got %d", invitations[2].MaxDevices)
	}

	// Test batch creation
	results := batchManager.CreateBatchInvitations(invitations, "", "", 2)

	// Check results
	if results.TotalProcessed != 3 {
		t.Errorf("Expected 3 processed invitations, got %d", results.TotalProcessed)
	}
	if results.TotalFailed != 0 {
		t.Errorf("Expected 0 failed invitations, got %d", results.TotalFailed)
	}
	if results.TotalSuccessful != 3 {
		t.Errorf("Expected 3 successful invitations, got %d", results.TotalSuccessful)
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

	// Test CSV export
	var buf bytes.Buffer
	err = batchManager.ExportBatchInvitationsToCSV(&buf, results, false)
	if err != nil {
		t.Fatalf("Failed to export invitations to CSV: %v", err)
	}

	// Check exported CSV
	csvOutput := buf.String()
	if !strings.Contains(csvOutput, "Name,Type,Token,Valid Days,Can Invite,Transferable,Device Bound,Max Devices,Status,Error") {
		t.Errorf("Expected CSV header in output")
	}
	if !strings.Contains(csvOutput, "Test User 1,read_only,") {
		t.Errorf("Expected Test User 1 in CSV output")
	}
	if !strings.Contains(csvOutput, "Test Admin,admin,") {
		t.Errorf("Expected Test Admin in CSV output")
	}
	if !strings.Contains(csvOutput, "Success") {
		t.Errorf("Expected Success status in CSV output")
	}

	// Test file-based import/export
	csvFilePath := filepath.Join(tempDir, "test-invitations.csv")
	if err := os.WriteFile(csvFilePath, []byte(csvData), 0644); err != nil {
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

func TestBatchInvitationEdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-batch-invitation-edge-test")
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

	// Test with minimal CSV (just name and type)
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

	// Check defaults are applied
	if minimalInvitations[0].ValidDays != 30 {
		t.Errorf("Expected default ValidDays to be 30, got %d", minimalInvitations[0].ValidDays)
	}
	if minimalInvitations[0].CanInvite {
		t.Errorf("Expected default CanInvite for read_only to be false")
	}
	if !minimalInvitations[0].DeviceBound {
		t.Errorf("Expected default DeviceBound to be true")
	}
	if minimalInvitations[0].MaxDevices != 1 {
		t.Errorf("Expected default MaxDevices to be 1, got %d", minimalInvitations[0].MaxDevices)
	}

	// Check admin defaults
	if !minimalInvitations[2].CanInvite {
		t.Errorf("Expected default CanInvite for admin to be true")
	}

	// Test invalid CSV format
	invalidCSV := `Name`
	_, err = batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(invalidCSV), false)
	if err == nil {
		t.Errorf("Expected error for invalid CSV format")
	}

	// Test empty name
	emptyNameCSV := `,read_only`
	_, err = batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(emptyNameCSV), false)
	if err == nil {
		t.Errorf("Expected error for empty name")
	}

	// Test invalid type
	invalidTypeCSV := `Test User,invalid_type`
	_, err = batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(invalidTypeCSV), false)
	if err == nil {
		t.Errorf("Expected error for invalid invitation type")
	}

	// Test case-insensitive type parsing
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

	if caseInvitations[0].Type != profile.InvitationTypeReadOnly {
		t.Errorf("Expected type to be read_only for case-insensitive 'READ_ONLY'")
	}

	if caseInvitations[1].Type != profile.InvitationTypeReadWrite {
		t.Errorf("Expected type to be read_write for case-insensitive 'ReadWrite'")
	}

	if caseInvitations[2].Type != profile.InvitationTypeAdmin {
		t.Errorf("Expected type to be admin for case-insensitive 'Admin'")
	}

	// Test boolean field parsing
	boolCSV := `Test User 1,read_only,30,true,yes,y,1
Test User 2,read_write,60,false,no,n,2
Test User 3,read_only,90,0,1,True,3`

	boolInvitations, err := batchManager.ImportBatchInvitationsFromCSV(strings.NewReader(boolCSV), false)
	if err != nil {
		t.Fatalf("Failed to import boolean invitations from CSV: %v", err)
	}

	if !boolInvitations[0].CanInvite {
		t.Errorf("Expected CanInvite to be true for 'true'")
	}
	if !boolInvitations[0].Transferable {
		t.Errorf("Expected Transferable to be true for 'yes'")
	}
	if !boolInvitations[0].DeviceBound {
		t.Errorf("Expected DeviceBound to be true for 'y'")
	}

	if boolInvitations[1].CanInvite {
		t.Errorf("Expected CanInvite to be false for 'false'")
	}
	if boolInvitations[1].Transferable {
		t.Errorf("Expected Transferable to be false for 'no'")
	}
	if boolInvitations[1].DeviceBound {
		t.Errorf("Expected DeviceBound to be false for 'n'")
	}

	if boolInvitations[2].CanInvite {
		t.Errorf("Expected CanInvite to be false for '0'")
	}
	if !boolInvitations[2].Transferable {
		t.Errorf("Expected Transferable to be true for '1'")
	}
	if !boolInvitations[2].DeviceBound {
		t.Errorf("Expected DeviceBound to be true for 'True'")
	}
}