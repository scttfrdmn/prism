package tests

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/prism/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

// TestSearchCreation tests the creation of a new search component
func TestSearchCreation(t *testing.T) {
	search := components.NewSearch()

	// Verify initial state
	assert.False(t, search.Active(), "Search should not be active initially")
	assert.Equal(t, "", search.Query(), "Search query should be empty initially")
}

// TestSearchActivation tests activating search programmatically
func TestSearchActivation(t *testing.T) {
	search := components.NewSearch()

	// Activate search programmatically (this is how the parent handles it)
	search.SetActive(true)

	// Verify search is activated
	assert.True(t, search.Active(), "Search should be activated")
}

// TestSearchDeactivation tests deactivating search programmatically
func TestSearchDeactivation(t *testing.T) {
	search := components.NewSearch()

	// First activate search
	search.SetActive(true)
	assert.True(t, search.Active(), "Search should be activated")

	// Deactivate search
	search.SetActive(false)

	// Verify search is deactivated
	assert.False(t, search.Active(), "Search should be deactivated")
}

// TestSearchQuery tests setting and getting search query
func TestSearchQuery(t *testing.T) {
	search := components.NewSearch()

	// Set query programmatically
	search.SetQuery("test")

	// Verify query is set correctly
	assert.Equal(t, "test", search.Query(), "Search query should be 'test'")
}

// TestSearchReset tests resetting the search query
func TestSearchReset(t *testing.T) {
	search := components.NewSearch()

	// Activate search and enter query
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	// Reset search
	search.Reset()

	// Verify query is reset
	assert.Equal(t, "", search.Query(), "Search query should be empty after reset")
}

// TestSearchWidth tests setting the search width
func TestSearchWidth(t *testing.T) {
	search := components.NewSearch()

	// Set width and activate to test rendering
	search.SetWidth(100)
	search.SetActive(true)

	// The actual effect is on rendering, but we can at least ensure it doesn't crash
	view := search.View()
	assert.NotEmpty(t, view, "Active search view should not be empty")
}

// TestSearchEnterKey tests text input functionality
func TestSearchEnterKey(t *testing.T) {
	search := components.NewSearch()

	// Activate search and set query
	search.SetActive(true)
	search.SetQuery("test")

	// Press Enter (should not crash)
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _ = search.Update(enterMsg)

	// Should handle gracefully
	assert.Equal(t, "test", search.Query(), "Query should be preserved after Enter")
}

// TestSearchView tests that the view method works correctly
func TestSearchView(t *testing.T) {
	search := components.NewSearch()

	// Test inactive view (should be empty)
	inactiveView := search.View()
	assert.Empty(t, inactiveView, "Inactive search view should be empty")

	// Activate search
	search.SetActive(true)

	// Test active view (should not be empty)
	activeView := search.View()
	assert.NotEmpty(t, activeView, "Active search view should not be empty")

	// The views should be different
	assert.NotEqual(t, inactiveView, activeView, "Active and inactive views should be different")
}
