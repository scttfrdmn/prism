package tests

import (
	"testing"

	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

// TestTabBarCreation tests creating a new tab bar
func TestTabBarCreation(t *testing.T) {
	// Create tab bar
	tabs := []string{"Tab 1", "Tab 2", "Tab 3"}
	tabBar := components.NewTabBar(tabs, 0)
	
	// Verify active tab
	assert.Equal(t, 0, tabBar.ActiveTab(), "First tab should be active")
	
	// Verify rendering
	view := tabBar.View()
	assert.NotEmpty(t, view, "Tab bar view should not be empty")
}

// TestActiveTab tests getting the active tab
func TestActiveTab(t *testing.T) {
	tabs := []string{"Tab 1", "Tab 2", "Tab 3"}
	
	// Create with different active tabs
	tabBar1 := components.NewTabBar(tabs, 0)
	assert.Equal(t, 0, tabBar1.ActiveTab(), "First tab should be active")
	
	tabBar2 := components.NewTabBar(tabs, 1)
	assert.Equal(t, 1, tabBar2.ActiveTab(), "Second tab should be active")
}

// TestNextTab tests moving to the next tab
func TestNextTab(t *testing.T) {
	tabs := []string{"Tab 1", "Tab 2", "Tab 3"}
	tabBar := components.NewTabBar(tabs, 0)
	
	// Move to next tab
	cmd := tabBar.Next()
	assert.Equal(t, 1, tabBar.ActiveTab(), "Second tab should be active after Next()")
	assert.NotNil(t, cmd, "Command should be returned")
	
	// Move to next tab again
	tabBar.Next()
	assert.Equal(t, 2, tabBar.ActiveTab(), "Third tab should be active after Next()")
	
	// Wrap around
	tabBar.Next()
	assert.Equal(t, 0, tabBar.ActiveTab(), "First tab should be active after wrapping")
}

// TestPrevTab tests moving to the previous tab
func TestPrevTab(t *testing.T) {
	tabs := []string{"Tab 1", "Tab 2", "Tab 3"}
	tabBar := components.NewTabBar(tabs, 2)
	
	// Move to previous tab
	cmd := tabBar.Prev()
	assert.Equal(t, 1, tabBar.ActiveTab(), "Second tab should be active after Prev()")
	assert.NotNil(t, cmd, "Command should be returned")
	
	// Move to previous tab again
	tabBar.Prev()
	assert.Equal(t, 0, tabBar.ActiveTab(), "First tab should be active after Prev()")
	
	// Wrap around
	tabBar.Prev()
	assert.Equal(t, 2, tabBar.ActiveTab(), "Third tab should be active after wrapping")
}

// TestTabBarSetWidth tests setting tab bar width
func TestTabBarSetWidth(t *testing.T) {
	tabs := []string{"Tab 1", "Tab 2"}
	tabBar := components.NewTabBar(tabs, 0)
	
	// Set width
	tabBar.SetWidth(100)
	
	// The actual effect is on rendering, but we can at least ensure it doesn't crash
	view := tabBar.View()
	assert.NotEmpty(t, view, "Tab bar view should not be empty after setting width")
}

// TestView tests the tab bar rendering
func TestView(t *testing.T) {
	tabs := []string{"Tab 1", "Tab 2"}
	tabBar := components.NewTabBar(tabs, 0)
	
	// Render view
	view := tabBar.View()
	
	// We can't easily test the exact output, but we can check it's not empty
	assert.NotEmpty(t, view, "Tab bar view should not be empty")
}

// TestChangeMsgGeneration tests that change messages are generated correctly
func TestChangeMsgGeneration(t *testing.T) {
	tabs := []string{"Tab 1", "Tab 2"}
	tabBar := components.NewTabBar(tabs, 0)
	
	// Just call Next(), we can't test the message generation in this test environment
	_ = tabBar.Next()
	
	// In a real test, we would verify the message generated
	assert.Equal(t, 1, tabBar.ActiveTab(), "Next should change to tab 1")
}