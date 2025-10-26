package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/prism/internal/tui/api"
	"github.com/scttfrdmn/prism/internal/tui/components"
	"github.com/scttfrdmn/prism/internal/tui/styles"
)

// DaemonModel represents the daemon management view
type DaemonModel struct {
	apiClient         apiClient
	statusBar         components.StatusBar
	spinner           components.Spinner
	width             int
	height            int
	loading           bool
	error             string
	status            *api.SystemStatusResponse
	showRestartDialog bool
	showStopDialog    bool
}

// DaemonManagementStatusMsg represents daemon status retrieved from the API
type DaemonManagementStatusMsg struct {
	Status *api.SystemStatusResponse
	Error  error
}

// DaemonManagementActionMsg represents daemon action result
type DaemonManagementActionMsg struct {
	Success bool
	Message string
	Error   error
}

// NewDaemonModel creates a new daemon management model
func NewDaemonModel(apiClient apiClient) DaemonModel {
	// Create status bar and spinner
	statusBar := components.NewStatusBar("Prism Daemon Management", "")
	spinner := components.NewSpinner("Loading daemon status...")

	return DaemonModel{
		apiClient: apiClient,
		statusBar: statusBar,
		spinner:   spinner,
		width:     80,
		height:    24,
		loading:   true,
	}
}

// Init initializes the model
func (m DaemonModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		func() tea.Msg { return m.fetchStatus() },
	)
}

// fetchStatus retrieves daemon status from the API
func (m DaemonModel) fetchStatus() tea.Msg {
	statusResp, err := m.apiClient.GetStatus(context.Background())
	if err != nil {
		return DaemonManagementStatusMsg{Error: fmt.Errorf("failed to get daemon status: %w", err)}
	}

	return DaemonManagementStatusMsg{
		Status: statusResp,
		Error:  nil,
	}
}

// Update handles messages and updates the model
func (m DaemonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case DaemonManagementStatusMsg:
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.loading = false
			return m, nil
		}

		m.status = msg.Status
		m.loading = false
		m.error = ""
		return m, nil

	case DaemonManagementActionMsg:
		m.showRestartDialog = false
		m.showStopDialog = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			return m, nil
		}
		// Refresh status after action
		m.loading = true
		return m, func() tea.Msg { return m.fetchStatus() }

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r", "f5":
			// Refresh daemon status
			m.loading = true
			return m, func() tea.Msg { return m.fetchStatus() }

		case "s":
			// Show stop dialog
			m.showStopDialog = true
			return m, nil

		case "R":
			// Show restart dialog (capital R)
			m.showRestartDialog = true
			return m, nil

		case "enter":
			// Handle dialog confirmation
			if m.showRestartDialog {
				return m, m.restartDaemon()
			}
			if m.showStopDialog {
				return m, m.stopDaemon()
			}

		case "esc":
			// Close dialogs
			if m.showRestartDialog {
				m.showRestartDialog = false
				return m, nil
			}
			if m.showStopDialog {
				m.showStopDialog = false
				return m, nil
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the model
func (m DaemonModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("⚙️  Daemon Management")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Daemon status
	b.WriteString(m.renderStatus())

	// Show restart dialog if active
	if m.showRestartDialog {
		dialog := m.renderRestartDialog()
		b.WriteString("\n\n")
		b.WriteString(dialog)
	}

	// Show stop dialog if active
	if m.showStopDialog {
		dialog := m.renderStopDialog()
		b.WriteString("\n\n")
		b.WriteString(dialog)
	}

	// Error display
	if m.error != "" {
		b.WriteString("\n\n")
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		b.WriteString(errorStyle.Render("Error: " + m.error))
	}

	// Help text
	b.WriteString("\n\n")
	helpText := m.renderHelp()
	b.WriteString(helpText)

	return b.String()
}

// renderStatus displays the daemon status
func (m DaemonModel) renderStatus() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Daemon Status"))
	b.WriteString("\n\n")

	if m.status == nil {
		b.WriteString("No status information available\n")
		return b.String()
	}

	// Status indicator
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	if m.status.Status != "running" {
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	}
	b.WriteString(statusStyle.Render(fmt.Sprintf("Status: %s\n", strings.ToUpper(m.status.Status))))
	b.WriteString("\n")

	// Version and uptime
	b.WriteString(theme.SubTitle.Render("Daemon Information") + "\n\n")
	b.WriteString(fmt.Sprintf("Version:        %s\n", m.status.Version))
	b.WriteString(fmt.Sprintf("Uptime:         %s\n", m.status.Uptime))
	b.WriteString(fmt.Sprintf("Start Time:     %s\n", m.status.StartTime.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")

	// Activity metrics
	b.WriteString(theme.SubTitle.Render("Activity Metrics") + "\n\n")
	b.WriteString(fmt.Sprintf("Active Operations:    %d\n", m.status.ActiveOps))
	b.WriteString(fmt.Sprintf("Total Requests:       %d\n", m.status.TotalRequests))
	if m.status.RequestsPerMinute > 0 {
		b.WriteString(fmt.Sprintf("Requests per Minute:  %.2f\n", m.status.RequestsPerMinute))
	}
	b.WriteString("\n")

	// Configuration
	b.WriteString(theme.SubTitle.Render("Configuration") + "\n\n")
	b.WriteString(fmt.Sprintf("AWS Region:     %s\n", m.status.AWSRegion))
	if m.status.AWSProfile != "" {
		b.WriteString(fmt.Sprintf("AWS Profile:    %s\n", m.status.AWSProfile))
	}
	if m.status.CurrentProfile != "" {
		b.WriteString(fmt.Sprintf("Current Profile: %s\n", m.status.CurrentProfile))
	}

	return b.String()
}

// renderRestartDialog displays the restart confirmation dialog
func (m DaemonModel) renderRestartDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("⚠️  Restart Daemon") + "\n\n")
	content.WriteString("This will restart the Prism daemon.\n")
	content.WriteString("Active operations will be interrupted.\n\n")
	content.WriteString("Press Enter to restart, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderStopDialog displays the stop confirmation dialog
func (m DaemonModel) renderStopDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("9")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("⚠️  Stop Daemon") + "\n\n")
	content.WriteString("This will stop the Prism daemon.\n")
	content.WriteString("All TUI and GUI clients will disconnect.\n")
	content.WriteString("Active operations will be interrupted.\n\n")
	content.WriteString("Press Enter to stop, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m DaemonModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showRestartDialog || m.showStopDialog {
		helps = []string{"enter: confirm", "esc: cancel"}
	} else {
		helps = []string{"r: refresh", "R: restart daemon", "s: stop daemon", "q: quit"}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// restartDaemon restarts the daemon
func (m DaemonModel) restartDaemon() tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would call the daemon restart API
		// For now, just return success
		return DaemonManagementActionMsg{
			Success: true,
			Message: "Daemon restart initiated",
			Error:   nil,
		}
	}
}

// stopDaemon stops the daemon
func (m DaemonModel) stopDaemon() tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would call the daemon stop API
		// For now, just return success
		return DaemonManagementActionMsg{
			Success: true,
			Message: "Daemon stop initiated",
			Error:   nil,
		}
	}
}
