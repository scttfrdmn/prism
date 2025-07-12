package components

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Spinner represents a loading spinner component
type Spinner struct {
	message   string
	frames    []string
	current   int
	lastTick  time.Time
	interval  time.Duration
	active    bool
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		current:  0,
		lastTick: time.Now(),
		interval: 80 * time.Millisecond,
		active:   true,
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

// Update advances the spinner animation if enough time has passed
func (s *Spinner) Update() {
	if !s.active {
		return
	}

	now := time.Now()
	if now.Sub(s.lastTick) >= s.interval {
		s.current = (s.current + 1) % len(s.frames)
		s.lastTick = now
	}
}

// View renders the spinner
func (s *Spinner) View() string {
	if !s.active {
		return ""
	}

	spinnerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15"))

	spinnerText := spinnerStyle.Render(s.frames[s.current])
	messageText := messageStyle.Render(s.message)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		spinnerText,
		" ",
		messageText,
	)
}