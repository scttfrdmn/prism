package tests

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

// TestSearchCreation tests the creation of a new search component
func TestSearchCreation(t *testing.T) {
	search := components.NewSearch()
	
	// Verify initial state
	assert.False(t, search.Active(), "Search should not be active initially")
	assert.Equal(t, "", search.Query(), "Search query should be empty initially")
}

// TestSearchActivation tests activating search with the "/" key
func TestSearchActivation(t *testing.T) {
	search := components.NewSearch()
	
	// Send "/" key to activate search
	activateMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedSearch, cmd := search.Update(activateMsg)
	
	// Verify search is activated
	assert.True(t, updatedSearch.Active(), "Search should be activated after '/' key")
	assert.NotNil(t, cmd, "Command should be returned to focus the input")
}

// TestSearchDeactivation tests deactivating search with the Escape key
func TestSearchDeactivation(t *testing.T) {
	search := components.NewSearch()
	
	// First activate search
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, search.Active(), "Search should be activated")
	
	// Send Escape key to deactivate search
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedSearch, cmd := search.Update(escMsg)
	
	// Verify search is deactivated
	assert.False(t, updatedSearch.Active(), "Search should be deactivated after Escape key")
	assert.NotNil(t, cmd, "Command should be returned to blur the input")
}

// TestSearchQuery tests entering a search query
func TestSearchQuery(t *testing.T) {
	search := components.NewSearch()
	
	// Activate search
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	
	// Enter search query "test"
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	search, cmd := search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	
	// Verify query and command
	assert.Equal(t, "test", search.Query(), "Search query should be 'test'")
	assert.NotNil(t, cmd, "Command should be returned to update search results")
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
	
	// Set width
	search.SetWidth(100)
	
	// The actual effect is on rendering, but we can at least ensure it doesn't crash
	view := search.View()
	assert.NotEmpty(t, view, "Search view should not be empty")
}

// TestSearchEnterKey tests pressing Enter to submit a search
func TestSearchEnterKey(t *testing.T) {
	search := components.NewSearch()
	
	// Activate search and enter query
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	
	// Press Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := search.Update(enterMsg)
	
	// Command should be returned
	assert.NotNil(t, cmd, "Command should be returned when Enter is pressed")
}

// TestSearchView tests that the view method doesn't crash
func TestSearchView(t *testing.T) {
	search := components.NewSearch()
	
	// Test inactive view
	inactiveView := search.View()
	assert.NotEmpty(t, inactiveView, "Inactive search view should not be empty")
	
	// Activate search
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	
	// Test active view
	activeView := search.View()
	assert.NotEmpty(t, activeView, "Active search view should not be empty")
	
	// The views should be different
	assert.NotEqual(t, inactiveView, activeView, "Active and inactive views should be different")
}