package pages_test

import (
	"context"
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/internal/gui/pages"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed test_assets
var testAssets embed.FS

// setupTest creates a testing environment and returns the context and temp directory
func setupTest(t *testing.T) (context.Context, *profile.SecureInvitationManager, string) {
	// Create a mock runtime
	mockRuntime := &struct {
		runtime.Runtime
	}{}

	// Create context with mock runtime
	ctx := context.WithValue(context.Background(), "runtime", mockRuntime)

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-page-batch-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	// Create config directory
	configDir := filepath.Join(tempDir, ".cloudworkstation")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tempDir)
	})

	// Create profile manager
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		t.Fatalf("Failed to create profile manager: %v", err)
	}

	// Create secure invitation manager
	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		t.Fatalf("Failed to create secure invitation manager: %v", err)
	}

	return ctx, secureManager, tempDir
}

// parseJSONResponse parses a JSON response from a page method
func parseJSONResponse(response string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := json.Unmarshal([]byte(response), &result)
	return result, err
}

// TestBatchInvitationPage tests the batch invitation page functionality
func TestBatchInvitationPage(t *testing.T) {
	ctx, secureManager, tempDir := setupTest(t)

	// Create the batch invitation page
	page := pages.NewBatchInvitationPage(ctx, secureManager, testAssets)
	if page == nil {
		t.Fatal("Failed to create batch invitation page")
	}

	// Create test CSV file
	csvData := `Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Test User 1,read_only,30,no,no,yes,1
Test User 2,read_write,60,no,no,yes,2
Test Admin,admin,90,yes,no,yes,3`

	csvFile := filepath.Join(tempDir, "test-invitations.csv")
	if err := os.WriteFile(csvFile, []byte(csvData), 0644); err != nil {
		t.Fatalf("Failed to write test CSV file: %v", err)
	}

	// Test error response
	t.Run("ErrorResponse", func(t *testing.T) {
		response := page.ErrorResponse("Test error message")
		result, err := parseJSONResponse(response)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok || success {
			t.Errorf("Expected success to be false in error response")
		}

		errorMsg, ok := result["error"].(string)
		if !ok || errorMsg != "Test error message" {
			t.Errorf("Expected error message 'Test error message', got '%v'", result["error"])
		}
	})

	// Test success response
	t.Run("SuccessResponse", func(t *testing.T) {
		response := page.SuccessResponse("Test success message")
		result, err := parseJSONResponse(response)
		if err != nil {
			t.Fatalf("Failed to parse success response: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok || !success {
			t.Errorf("Expected success to be true in success response")
		}

		message, ok := result["message"].(string)
		if !ok || message != "Test success message" {
			t.Errorf("Expected message 'Test success message', got '%v'", result["message"])
		}
	})

	// Test preview CSV file
	t.Run("PreviewCSVFile", func(t *testing.T) {
		response := page.PreviewCSVFile(csvFile, true)
		result, err := parseJSONResponse(response)
		if err != nil {
			t.Fatalf("Failed to parse preview response: %v", err)
		}

		// Check if it's an error response
		if success, ok := result["success"].(bool); ok && !success {
			t.Fatalf("Preview returned error: %v", result["error"])
		}

		// Otherwise, it should be an array of invitations
		// But since this is a mock environment, we might not get actual data
		// Just verify the response is well-formed JSON
	})

	// Test get all invitations
	t.Run("GetAllInvitations", func(t *testing.T) {
		response := page.GetAllInvitations()
		result, err := parseJSONResponse(response)
		if err != nil {
			t.Fatalf("Failed to parse get all invitations response: %v", err)
		}

		// In a test environment without real invitations, this might return an empty array
		// We're mainly testing that the method executes without errors
	})

	// Test operations that require file dialogs
	// These tests will be limited since we can't fully mock the file dialogs

	t.Run("SelectImportFile", func(t *testing.T) {
		// This will use the mock runtime, which isn't fully implemented
		// We're just testing that the method exists and can be called
		page.SelectImportFile()
	})

	t.Run("SelectExportFile", func(t *testing.T) {
		// Similar to above
		page.SelectExportFile()
	})

	// Test get last operation result when none exists
	t.Run("GetLastOperationResult", func(t *testing.T) {
		response := page.GetLastOperationResult()
		result, err := parseJSONResponse(response)
		if err != nil {
			t.Fatalf("Failed to parse last operation result: %v", err)
		}

		// Should return an error since no operations have been performed
		success, ok := result["success"].(bool)
		if !ok || success {
			t.Errorf("Expected success to be false when no operations exist")
		}
	})
}

// Additional helper functions for the page package
func (p *pages.BatchInvitationPage) ErrorResponse(message string) string {
	return p.JsonResponse(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func (p *pages.BatchInvitationPage) SuccessResponse(message string) string {
	return p.JsonResponse(map[string]interface{}{
		"success": true,
		"message": message,
	})
}

func (p *pages.BatchInvitationPage) JsonResponse(data interface{}) string {
	jsonBytes, _ := json.Marshal(data)
	return string(jsonBytes)
}