package models

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/prism/internal/tui/components"
	"github.com/scttfrdmn/prism/internal/tui/styles"
	"github.com/scttfrdmn/prism/pkg/version"
)

// SettingsModel represents the settings view
type SettingsModel struct {
	apiClient    apiClient
	statusBar    components.StatusBar
	width        int
	height       int
	daemonStatus string
	error        string
}

// NewSettingsModel creates a new settings model
func NewSettingsModel(apiClient apiClient) SettingsModel {
	// Create status bar
	statusBar := components.NewStatusBar("Prism Settings", "")

	return SettingsModel{
		apiClient:    apiClient,
		statusBar:    statusBar,
		width:        80,
		height:       24,
		daemonStatus: "unknown",
	}
}

// Init initializes the model
func (m SettingsModel) Init() tea.Cmd {
	return m.checkDaemonStatus
}

// checkDaemonStatus checks if the daemon is responding
func (m SettingsModel) checkDaemonStatus() tea.Msg {
	_, err := m.apiClient.ListIdlePolicies(context.Background())
	if err != nil {
		return DaemonStatusMsg{Status: "disconnected", Error: err}
	}
	return DaemonStatusMsg{Status: "connected", Error: nil}
}

// DaemonStatusMsg represents daemon status information
type DaemonStatusMsg struct {
	Status string
	Error  error
}

// Update handles messages and updates the model
func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.error = ""
			return m, m.checkDaemonStatus

		case "q", "esc":
			return m, tea.Quit
		}

	case DaemonStatusMsg:
		m.daemonStatus = msg.Status
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.statusBar.SetStatus("Daemon connection failed", components.StatusError)
		} else {
			m.statusBar.SetStatus("Settings loaded", components.StatusSuccess)
		}
	}

	return m, nil
}

// View renders the settings view
func (m SettingsModel) View() string {
	theme := styles.CurrentTheme

	// Title section
	title := theme.Title.Render("Prism Settings")

	// Build settings content
	var content string

	// System Information
	systemInfo := "System Information:\n"
	systemInfo += fmt.Sprintf("  Version: %s\n", version.GetVersion())
	systemInfo += fmt.Sprintf("  Daemon Status: %s\n", m.daemonStatus)
	systemInfo += "  API Endpoint: http://localhost:8947\n"

	// Configuration
	configInfo := "\nConfiguration:\n"
	configInfo += "  Profile system is managed via CLI commands:\n"
	configInfo += "  cws config profile <name>     # Set AWS profile\n"
	configInfo += "  cws config region <region>    # Set AWS region\n"
	configInfo += "  cws config show               # Show current config\n"

	// Daemon Management
	daemonInfo := "\nDaemon Management:\n"
	daemonInfo += "  cws daemon start              # Start daemon\n"
	daemonInfo += "  cws daemon stop               # Stop daemon\n"
	daemonInfo += "  cws daemon status             # Check daemon status\n"

	// TUI Controls
	tuiInfo := "\nTUI Navigation:\n"
	tuiInfo += "  1: Dashboard    2: Instances    3: Templates\n"
	tuiInfo += "  4: Storage      5: Settings     6: Profiles\n"
	tuiInfo += "  q: Quit         r: Refresh\n"

	// Error display
	errorInfo := ""
	if m.error != "" {
		errorInfo = fmt.Sprintf("\nConnection Error:\n  %s\n", m.error)
	}

	// Combine all content
	fullContent := systemInfo + configInfo + daemonInfo + tuiInfo + errorInfo

	content = lipgloss.NewStyle().
		Width(m.width-4).
		Padding(1, 2).
		Render(fullContent)

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
