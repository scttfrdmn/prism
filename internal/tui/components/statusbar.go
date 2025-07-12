package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Status constants
const (
	StatusNone    = iota
	StatusSuccess
	StatusWarning
	StatusError
)

// StatusBar represents a status bar component
type StatusBar struct {
	version string
	region  string
	status  string
	style   int
}

// NewStatusBar creates a new status bar
func NewStatusBar(version, region string) *StatusBar {
	return &StatusBar{
		version: version,
		region:  region,
		status:  "Ready",
		style:   StatusNone,
	}
}

// SetStatus updates the status message and style
func (s *StatusBar) SetStatus(message string, style int) {
	s.status = message
	s.style = style
}

// GetStatus returns the current status message
func (s *StatusBar) GetStatus() string {
	return s.status
}

// GetStatusStyle returns the current status style
func (s *StatusBar) GetStatusStyle() int {
	return s.style
}

// GetVersion returns the application version
func (s *StatusBar) GetVersion() string {
	return s.version
}

// GetRegion returns the current AWS region
func (s *StatusBar) GetRegion() string {
	return s.region
}

// SetRegion updates the current AWS region
func (s *StatusBar) SetRegion(region string) {
	s.region = region
}

// View renders the status bar
func (s *StatusBar) View() string {
	// Define styles
	baseStyle := lipgloss.NewStyle().
		Width(100).
		Padding(0, 1).
		Bold(true)

	versionStyle := baseStyle.Copy().
		Foreground(lipgloss.Color("#AAAAAA")).
		Align(lipgloss.Left)

	statusStyle := baseStyle.Copy().Align(lipgloss.Center)
	
	// Apply color based on style
	switch s.style {
	case StatusSuccess:
		statusStyle = statusStyle.Foreground(lipgloss.Color("10"))
	case StatusWarning:
		statusStyle = statusStyle.Foreground(lipgloss.Color("3"))
	case StatusError:
		statusStyle = statusStyle.Foreground(lipgloss.Color("9"))
	}
	
	regionStyle := baseStyle.Copy().
		Foreground(lipgloss.Color("#AAAAAA")).
		Align(lipgloss.Right)
	
	// Render components
	versionText := versionStyle.Render("v" + s.version)
	statusText := statusStyle.Render(s.status)
	regionText := regionStyle.Render(s.region)
	
	// Combine and return
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		versionText,
		statusText,
		regionText,
	)
}