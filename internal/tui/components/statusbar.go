package components

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/prism/internal/tui/styles"
)

// StatusType represents the type of status message
type StatusType int

const (
	// StatusInfo is an informational status
	StatusInfo StatusType = iota
	// StatusSuccess is a success status
	StatusSuccess
	// StatusError is an error status
	StatusError
	// StatusWarning is a warning status
	StatusWarning
)

// StatusBar is a component for displaying status information
type StatusBar struct {
	status      string
	statusType  StatusType
	region      string
	version     string
	connections int
	lastUpdated time.Time
	width       int
}

// LastUpdated returns the time the status was last updated
func (s StatusBar) LastUpdated() time.Time {
	return s.lastUpdated
}

// NewStatusBar creates a new status bar component
func NewStatusBar(version, region string) StatusBar {
	return StatusBar{
		status:      "Ready",
		statusType:  StatusInfo,
		region:      region,
		version:     version,
		connections: 0,
		lastUpdated: time.Now(),
		width:       80,
	}
}

// SetStatus updates the status message
func (s *StatusBar) SetStatus(message string, statusType StatusType) {
	s.status = message
	s.statusType = statusType
	s.lastUpdated = time.Now()
}

// GetStatus returns the current status message
func (s *StatusBar) GetStatus() string {
	return s.status
}

// GetStatusStyle returns the current status style
func (s *StatusBar) GetStatusStyle() int {
	return int(s.statusType)
}

// SetRegion updates the AWS region
func (s *StatusBar) SetRegion(region string) {
	s.region = region
}

// GetRegion returns the current AWS region
func (s *StatusBar) GetRegion() string {
	return s.region
}

// SetConnections updates the number of active connections
func (s *StatusBar) SetConnections(count int) {
	s.connections = count
}

// SetWidth updates the width of the status bar
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// View renders the status bar
func (s *StatusBar) View() string {
	theme := styles.CurrentTheme

	// Status indicator
	var statusStyle lipgloss.Style
	var statusIndicator string

	switch s.statusType {
	case StatusSuccess:
		statusStyle = theme.StatusOK
		statusIndicator = "+"
	case StatusError:
		statusStyle = theme.StatusError
		statusIndicator = "x"
	case StatusWarning:
		statusStyle = theme.StatusWarning
		statusIndicator = "!"
	default:
		statusStyle = lipgloss.NewStyle().Foreground(theme.TextColor)
		statusIndicator = "*"
	}

	// Build the left part (status)
	left := statusStyle.Render(statusIndicator + " " + s.status)

	// Build the right part (region, connections, version)
	rightElements := []string{
		"Region: " + s.region,
	}

	if s.connections > 0 {
		rightElements = append(rightElements, "Connections: "+strings.Repeat("•", s.connections))
	}

	rightElements = append(rightElements, "v"+s.version)

	right := lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(theme.MutedColor).Render(strings.Join(rightElements, " | ")),
	)

	// Calculate padding to push the right side to the edge
	padding := s.width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 1 {
		padding = 1
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		left,
		strings.Repeat(" ", padding),
		right,
	)
}
