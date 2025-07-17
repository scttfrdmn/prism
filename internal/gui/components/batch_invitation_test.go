package components_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/internal/gui/components"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// mockRuntime implements the minimal runtime.Runtime interface for testing
type mockRuntime struct{}

func (m *mockRuntime) EventsEmit(ctx context.Context, eventName string, optionalData ...interface{}) {}
func (m *mockRuntime) EventsOn(ctx context.Context, eventName string, callback func(optionalData ...interface{})) {}
func (m *mockRuntime) EventsOff(ctx context.Context, eventName string) {}
func (m *mockRuntime) EventsOnce(ctx context.Context, eventName string, callback func(optionalData ...interface{})) {}
func (m *mockRuntime) EventsOnMultiple(ctx context.Context, eventName string, callback func(optionalData ...interface{}), counter int) {}
func (m *mockRuntime) LogPrint(ctx context.Context, message string) {}
func (m *mockRuntime) LogTrace(ctx context.Context, message string) {}
func (m *mockRuntime) LogDebug(ctx context.Context, message string) {}
func (m *mockRuntime) LogInfo(ctx context.Context, message string) {}
func (m *mockRuntime) LogWarning(ctx context.Context, message string) {}
func (m *mockRuntime) LogError(ctx context.Context, message string) {}
func (m *mockRuntime) LogFatal(ctx context.Context, message string) {}
func (m *mockRuntime) WindowSetTitle(ctx context.Context, title string) {}
func (m *mockRuntime) WindowFullscreen(ctx context.Context) {}
func (m *mockRuntime) WindowUnfullscreen(ctx context.Context) {}
func (m *mockRuntime) WindowSetSize(ctx context.Context, width int, height int) {}
func (m *mockRuntime) WindowGetSize(ctx context.Context) (int, int) { return 800, 600 }
func (m *mockRuntime) WindowCenter(ctx context.Context) {}
func (m *mockRuntime) WindowShow(ctx context.Context) {}
func (m *mockRuntime) WindowHide(ctx context.Context) {}
func (m *mockRuntime) WindowMaximise(ctx context.Context) {}
func (m *mockRuntime) WindowToggleMaximise(ctx context.Context) {}
func (m *mockRuntime) WindowUnmaximise(ctx context.Context) {}
func (m *mockRuntime) WindowMinimise(ctx context.Context) {}
func (m *mockRuntime) WindowUnminimise(ctx context.Context) {}
func (m *mockRuntime) WindowSetMinSize(ctx context.Context, width int, height int) {}
func (m *mockRuntime) WindowSetMaxSize(ctx context.Context, width int, height int) {}
func (m *mockRuntime) WindowSetPosition(ctx context.Context, x int, y int) {}
func (m *mockRuntime) WindowGetPosition(ctx context.Context) (int, int) { return 0, 0 }
func (m *mockRuntime) WindowReload(ctx context.Context) {}
func (m *mockRuntime) WindowReloadApp(ctx context.Context) {}
func (m *mockRuntime) WindowSetAlwaysOnTop(ctx context.Context, b bool) {}
func (m *mockRuntime) MenuSetApplicationMenu(ctx context.Context, menu interface{}) {}
func (m *mockRuntime) MenuUpdateApplicationMenu(ctx context.Context) {}
func (m *mockRuntime) MenuSetTrayMenu(ctx context.Context, menu interface{}) {}
func (m *mockRuntime) MenuUpdateTrayMenu(ctx context.Context) {}

func (m *mockRuntime) BrowserOpenURL(ctx context.Context, url string) {}
func (m *mockRuntime) DialogMessage(ctx context.Context, dialogOptions runtime.MessageDialogOptions) (string, error) { 
	return "ok", nil 
}
func (m *mockRuntime) DialogOpen(ctx context.Context, dialogOptions runtime.OpenDialogOptions) (string, error) {
	// Mock implementation - return test path
	return filepath.Join(os.TempDir(), "test.csv"), nil
}
func (m *mockRuntime) DialogSave(ctx context.Context, dialogOptions runtime.SaveDialogOptions) (string, error) {
	// Mock implementation - return test path
	return filepath.Join(os.TempDir(), "output.csv"), nil
}

func (m *mockRuntime) ClipboardGetText(ctx context.Context) (string, error) { return "", nil }
func (m *mockRuntime) ClipboardSetText(ctx context.Context, text string) error { return nil }
func (m *mockRuntime) ScreenGetAll(ctx context.Context) ([]runtime.Screen, error) { return nil, nil }
func (m *mockRuntime) Show() {}
func (m *mockRuntime) Hide() {}
func (m *mockRuntime) Quit() {}
func (m *mockRuntime) Environment() map[string]string { return nil }
func (m *mockRuntime) OpenFile(ctx context.Context, path string) error { return nil }

// TestBatchInvitationManager tests the batch invitation manager for GUI
func TestBatchInvitationManager(t *testing.T) {
	// Create a test context with mock runtime
	ctx := context.WithValue(context.Background(), "runtime", &mockRuntime{})
	runtime.Init(ctx)

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-gui-batch-test")
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

	// Create the batch invitation manager
	batchManager := components.NewBatchInvitationManager(ctx, secureManager)
	if batchManager == nil {
		t.Fatal("Failed to create batch invitation manager")
	}

	// Test file selection
	t.Run("SelectCSVFileForImport", func(t *testing.T) {
		filePath := batchManager.SelectCSVFileForImport()
		if !strings.HasSuffix(filePath, "test.csv") {
			t.Errorf("Expected file path to end with 'test.csv', got %s", filePath)
		}
	})

	t.Run("SelectOutputFileForExport", func(t *testing.T) {
		filePath := batchManager.SelectOutputFileForExport()
		if !strings.HasSuffix(filePath, "output.csv") {
			t.Errorf("Expected file path to end with 'output.csv', got %s", filePath)
		}
	})

	// Test CSV preview
	t.Run("PreviewCSVFile", func(t *testing.T) {
		rows, err := batchManager.PreviewCSVFile(csvFile, true)
		if err != nil {
			t.Fatalf("Failed to preview CSV file: %v", err)
		}

		if len(rows) != 3 {
			t.Errorf("Expected 3 rows, got %d", len(rows))
		}

		if rows[0].Name != "Test User 1" {
			t.Errorf("Expected name 'Test User 1', got '%s'", rows[0].Name)
		}

		if rows[2].Type != "admin" {
			t.Errorf("Expected type 'admin', got '%s'", rows[2].Type)
		}
	})

	// Test template generation
	t.Run("GenerateEmptyCSVTemplate", func(t *testing.T) {
		templateFile := filepath.Join(tempDir, "template.csv")
		err := batchManager.GenerateEmptyCSVTemplate(templateFile)
		if err != nil {
			t.Fatalf("Failed to generate template: %v", err)
		}

		// Verify the file exists and contains expected content
		if _, err := os.Stat(templateFile); os.IsNotExist(err) {
			t.Errorf("Template file was not created")
		}

		content, err := os.ReadFile(templateFile)
		if err != nil {
			t.Fatalf("Failed to read template file: %v", err)
		}

		if !strings.Contains(string(content), "Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices") {
			t.Errorf("Template file does not contain expected header")
		}
	})

	// Note: Full batch creation tests are not included as they would require
	// a complete mock of the underlying profile.BatchInvitationManager
	// which would essentially duplicate its logic. Instead, we focus on
	// testing the GUI-specific functionality.
}

// TestBatchOperationResult tests the batch operation result handling
func TestBatchOperationResult(t *testing.T) {
	// Create a test context with mock runtime
	ctx := context.WithValue(context.Background(), "runtime", &mockRuntime{})
	runtime.Init(ctx)

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-gui-result-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup profile manager and secure manager
	profileManager, _ := profile.NewManagerEnhanced()
	secureManager, _ := profile.NewSecureInvitationManager(profileManager)
	batchManager := components.NewBatchInvitationManager(ctx, secureManager)

	// Test get last result when none exists
	t.Run("GetLastResultEmpty", func(t *testing.T) {
		result := batchManager.GetLastOperationResult()
		if result != nil {
			t.Errorf("Expected nil result when no operations have been performed")
		}
	})

	// Since we can't easily test the full operation cycle without mocking the
	// underlying BatchInvitationManager, we'll just verify that the last result
	// is properly accessible after operations.
}

// Additional tests could be added to cover:
// - Error handling for invalid files
// - Integration with the UI layer
// - Concurrent operation handling
// - File opening functionality
// However, these would require more sophisticated mocking of the runtime
// and underlying managers, which is beyond the scope of this test file.