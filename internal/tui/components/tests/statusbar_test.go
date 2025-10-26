package tests

import (
	"testing"
	"time"

	"github.com/scttfrdmn/prism/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

// TestStatusBarCreation tests creating a new status bar
func TestStatusBarCreation(t *testing.T) {
	// Create status bar
	version := "1.0.0"
	region := "us-west-2"
	statusBar := components.NewStatusBar(version, region)

	// Verify status bar was created
	view := statusBar.View()
	assert.NotEmpty(t, view, "Status bar view should not be empty")
	assert.Contains(t, view, version, "Status bar should contain version")
	assert.Contains(t, view, region, "Status bar should contain region")
}

// TestSetStatus tests setting a status message
func TestSetStatus(t *testing.T) {
	// Create status bar
	statusBar := components.NewStatusBar("1.0.0", "us-west-2")

	// Set status
	status := "Test status"
	statusBar.SetStatus(status, components.StatusSuccess)

	// Verify status was set
	view := statusBar.View()
	assert.Contains(t, view, status, "Status bar should contain the status message")
}

// TestStatusTypes tests different status types
func TestStatusTypes(t *testing.T) {
	// Create status bar
	statusBar := components.NewStatusBar("1.0.0", "us-west-2")

	// Test success status
	statusBar.SetStatus("Success", components.StatusSuccess)
	successView := statusBar.View()
	assert.Contains(t, successView, "Success", "Status bar should contain success message")

	// Test warning status
	statusBar.SetStatus("Warning", components.StatusWarning)
	warningView := statusBar.View()
	assert.Contains(t, warningView, "Warning", "Status bar should contain warning message")

	// Test error status
	statusBar.SetStatus("Error", components.StatusError)
	errorView := statusBar.View()
	assert.Contains(t, errorView, "Error", "Status bar should contain error message")
}

// TestSetWidth tests setting the width of the status bar
func TestSetWidth(t *testing.T) {
	// Create status bar
	statusBar := components.NewStatusBar("1.0.0", "us-west-2")

	// Set width
	statusBar.SetWidth(100)

	// The actual effect is on rendering, but we can at least ensure it doesn't crash
	view := statusBar.View()
	assert.NotEmpty(t, view, "Status bar view should not be empty after setting width")
}

// TestSetRegion tests setting the region
func TestSetRegion(t *testing.T) {
	// Create status bar
	statusBar := components.NewStatusBar("1.0.0", "us-west-2")

	// Set new region
	newRegion := "us-east-1"
	statusBar.SetRegion(newRegion)

	// Verify region was updated
	view := statusBar.View()
	assert.Contains(t, view, newRegion, "Status bar should contain the new region")
}

// TestSetConnections tests setting the number of connections
func TestSetConnections(t *testing.T) {
	// Create status bar
	statusBar := components.NewStatusBar("1.0.0", "us-west-2")

	// Set connections
	connections := 5
	statusBar.SetConnections(connections)

	// Verify connections were updated
	view := statusBar.View()
	assert.Contains(t, view, "Connections: •••••", "Status bar should show connection indicators")
}

// TestStatusWithLastUpdated tests the last updated time
func TestStatusWithLastUpdated(t *testing.T) {
	// Create status bar
	statusBar := components.NewStatusBar("1.0.0", "us-west-2")

	// Set status, which should update last updated time
	statusBar.SetStatus("Test status", components.StatusSuccess)

	// Last updated time should be recent
	diff := time.Since(statusBar.LastUpdated())
	assert.Less(t, diff, 1*time.Second, "Last updated time should be recent")
}
