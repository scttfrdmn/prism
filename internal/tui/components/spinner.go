package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/prism/internal/tui/styles"
)

// Spinner is a loading spinner component
type Spinner struct {
	spinner spinner.Model
	message string
	active  bool
}

// NewSpinner creates a new spinner component with a message
func NewSpinner(message string) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.CurrentTheme.PrimaryColor)

	return Spinner{
		spinner: s,
		message: message,
		active:  true,
	}
}

// Start activates the spinner
func (s *Spinner) Start() {
	s.active = true
}

// Stop deactivates the spinner
func (s *Spinner) Stop() {
	s.active = false
}

// Update handles messages for the spinner
func (s *Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	if !s.active {
		return *s, nil
	}

	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return *s, cmd
}

// View renders the spinner
func (s *Spinner) View() string {
	if !s.active {
		return ""
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		s.spinner.View(),
		" "+s.message,
	)
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(message string) {
	s.message = message
}

// GetMessage returns the current spinner message
func (s *Spinner) GetMessage() string {
	return s.message
}

// IsActive returns whether the spinner is currently active
func (s *Spinner) IsActive() bool {
	return s.active
}

// Spinner returns the underlying spinner model
func (s *Spinner) Spinner() spinner.Model {
	return s.spinner
}

// InitialCmd returns the spinner's initial command
func (s *Spinner) InitialCmd() tea.Cmd {
	return s.spinner.Tick
}
