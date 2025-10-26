package models

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/prism/internal/tui/components"
	"github.com/scttfrdmn/prism/internal/tui/styles"
	"github.com/scttfrdmn/prism/pkg/profile"
)

// ProfilesModel represents a simplified profiles view
type ProfilesModel struct {
	apiClient      apiClient
	statusBar      components.StatusBar
	width          int
	height         int
	currentProfile *profile.Profile
	error          string
}

// ProfileInitMsg is sent when the profile page is initialized
type ProfileInitMsg struct{}

// NewProfilesModel creates a new simplified profiles model
func NewProfilesModel(apiClient apiClient) ProfilesModel {
	statusBar := components.NewStatusBar("Prism Profiles", "")

	return ProfilesModel{
		apiClient: apiClient,
		statusBar: statusBar,
		width:     80,
		height:    24,
	}
}

// Init initializes the model
func (m ProfilesModel) Init() tea.Cmd {
	return func() tea.Msg { return ProfileInitMsg{} }
}

// SetSize sets the dimensions of the model
func (m *ProfilesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.statusBar.SetWidth(width)
}

// Update handles messages and updates the model
func (m ProfilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		return m, nil

	case ProfileInitMsg:
		// Load current profile using ManagerEnhanced
		profileManager, pmErr := profile.NewManagerEnhanced()
		if pmErr != nil {
			m.error = pmErr.Error()
			m.statusBar.SetStatus("Failed to initialize profile manager", components.StatusError)
		} else {
			currentProfile, err := profileManager.GetCurrentProfile()
			if err != nil {
				m.error = err.Error()
				m.statusBar.SetStatus("Failed to load profile", components.StatusError)
			} else {
				m.currentProfile = currentProfile
				m.statusBar.SetStatus("Profile loaded", components.StatusSuccess)
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return m, func() tea.Msg { return ProfileInitMsg{} }

		case "q", "esc":
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the profiles view
func (m ProfilesModel) View() string {
	theme := styles.CurrentTheme

	// Title section
	title := theme.Title.Render("Prism Profiles")

	// Content
	var content string
	if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else if m.currentProfile == nil {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Loading profile information...")
	} else {
		// Show current profile information
		profileInfo := "Current Profile:\n"
		profileInfo += fmt.Sprintf("  Name: %s\n", m.currentProfile.Name)
		profileInfo += fmt.Sprintf("  AWS Profile: %s\n", m.currentProfile.AWSProfile)
		profileInfo += fmt.Sprintf("  Region: %s\n", m.currentProfile.Region)

		// Profile management commands
		profileInfo += "\nProfile Management:\n"
		profileInfo += "  Use CLI commands to manage profiles:\n"
		profileInfo += "  cws config profile <name>     # Set AWS profile\n"
		profileInfo += "  cws config region <region>    # Set AWS region\n"
		profileInfo += "  cws config show               # Show current config\n"

		content = lipgloss.NewStyle().
			Width(m.width-4).
			Padding(1, 2).
			Render(profileInfo)
	}

	// Help text
	help := theme.Help.Render("r: refresh • q: quit")

	// Join everything together
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		"",
		m.statusBar.View(),
		help,
	)
}
