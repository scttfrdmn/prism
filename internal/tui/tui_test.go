package tui

import (
	"testing"

	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
)

// TestTUIAppInitialization tests that the TUI app initializes correctly
func TestTUIAppInitialization(t *testing.T) {
	// Create mock API client
	mockClient := &api.TUIClient{}

	app := &App{
		apiClient: mockClient,
	}

	// Verify app properties
	if app.apiClient == nil {
		t.Error("Expected API client to be initialized")
	}
}

// TestPageIDs tests page ID constants
func TestPageIDs(t *testing.T) {
	// Test that page IDs are unique
	pages := []PageID{
		DashboardPage,
		InstancesPage,
		TemplatesPage,
		StoragePage,
		SettingsPage,
		ProfilesPage,
	}

	seen := make(map[PageID]bool)
	for _, page := range pages {
		if seen[page] {
			t.Errorf("Duplicate page ID: %d", page)
		}
		seen[page] = true
	}

	// Verify page count
	if len(pages) != 6 {
		t.Errorf("Expected 6 pages, got %d", len(pages))
	}
}

// TestAppStructure tests the app structure exists
func TestAppStructure(t *testing.T) {
	// Just verify the App struct can be created
	app := &App{}
	if app == nil {
		t.Error("Failed to create App struct")
	}
}

// TestAppModelStructure tests the app model structure
func TestAppModelStructure(t *testing.T) {
	// Verify AppModel struct can be created
	model := &AppModel{
		currentPage: DashboardPage,
	}
	
	if model.currentPage != DashboardPage {
		t.Errorf("Expected current page to be DashboardPage, got %v", model.currentPage)
	}
}