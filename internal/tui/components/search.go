package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Search represents a search input component
type Search struct {
	textInput textinput.Model
	active    bool
	query     string
}

// NewSearch creates a new search component
func NewSearch() *Search {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100

	return &Search{
		textInput: ti,
		active:    false,
		query:     "",
	}
}

// Active returns whether the search is currently active
func (s *Search) Active() bool {
	return s.active
}

// Query returns the current search query
func (s *Search) Query() string {
	return s.query
}

// SetActive sets the active state of the search
func (s *Search) SetActive(active bool) {
	s.active = active
	if active {
		s.textInput.Focus()
	} else {
		s.textInput.Blur()
	}
}

// SetQuery sets the search query
func (s *Search) SetQuery(query string) {
	s.query = query
	s.textInput.SetValue(query)
}

// Update handles updates to the search component
func (s *Search) Update(msg tea.Msg) (*Search, tea.Cmd) {
	var cmd tea.Cmd
	s.textInput, cmd = s.textInput.Update(msg)
	s.query = s.textInput.Value()
	return s, cmd
}

// View renders the search component
func (s *Search) View() string {
	if !s.active {
		return ""
	}
	return s.textInput.View()
}

// Filter filters a list of items based on the search query
func (s *Search) Filter(items []string) []string {
	if s.query == "" {
		return items
	}

	var filtered []string
	query := strings.ToLower(s.query)

	for _, item := range items {
		if strings.Contains(strings.ToLower(item), query) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// Clear clears the search query
func (s *Search) Clear() {
	s.SetQuery("")
}

// Reset resets the search component to its initial state
func (s *Search) Reset() {
	s.SetQuery("")
	s.SetActive(false)
}

// SetWidth sets the width of the search input
func (s *Search) SetWidth(width int) {
	s.textInput.Width = width
}
