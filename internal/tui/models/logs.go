package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// LogsModel represents the logs viewer
type LogsModel struct {
	apiClient            apiClient
	instancesTable       components.Table
	viewport             viewport.Model
	statusBar            components.StatusBar
	spinner              components.Spinner
	width                int
	height               int
	loading              bool
	error                string
	instances            []api.InstanceResponse
	selectedTab          int // 0=instances, 1=viewer
	selectedInstance     int
	selectedInstanceName string
	logType              string // console, cloud-init, etc.
	logLines             []string
	showTypeDialog       bool
}

// LogsDataMsg represents logs data retrieved from the API
type LogsDataMsg struct {
	Instances []api.InstanceResponse
	Error     error
}

// LogLinesMsg represents log lines retrieved from the API
type LogLinesMsg struct {
	Lines []string
	Error error
}

// NewLogsModel creates a new logs viewer model
func NewLogsModel(apiClient apiClient) LogsModel {
	// Create instances table
	instanceColumns := []table.Column{
		{Title: "INSTANCE", Width: 25},
		{Title: "STATE", Width: 12},
		{Title: "TEMPLATE", Width: 25},
		{Title: "IP ADDRESS", Width: 18},
	}

	instancesTable := components.NewTable(instanceColumns, []table.Row{}, 80, 8, true)

	// Create viewport for log display
	vp := viewport.New(80, 15)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingLeft(1)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Logs Viewer", "")
	spinner := components.NewSpinner("Loading logs...")

	return LogsModel{
		apiClient:      apiClient,
		instancesTable: instancesTable,
		viewport:       vp,
		statusBar:      statusBar,
		spinner:        spinner,
		width:          80,
		height:         24,
		loading:        true,
		selectedTab:    0,
		logType:        "console",
	}
}

// Init initializes the model
func (m LogsModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		func() tea.Msg { return m.fetchInstances() },
	)
}

// fetchInstances retrieves instance list from the API
func (m LogsModel) fetchInstances() tea.Msg {
	instancesResp, err := m.apiClient.ListInstances(context.Background())
	if err != nil {
		return LogsDataMsg{Error: fmt.Errorf("failed to list instances: %w", err)}
	}

	return LogsDataMsg{
		Instances: instancesResp.Instances,
		Error:     nil,
	}
}

// fetchLogs retrieves logs for the selected instance
func (m LogsModel) fetchLogs(instanceName, logType string) tea.Cmd {
	return func() tea.Msg {
		logsResp, err := m.apiClient.GetLogs(context.Background(), instanceName, logType)
		if err != nil {
			return LogLinesMsg{Error: fmt.Errorf("failed to get logs: %w", err)}
		}

		return LogLinesMsg{
			Lines: logsResp.Lines,
			Error: nil,
		}
	}
}

// Update handles messages and updates the model
func (m LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case LogsDataMsg:
		return m.handleLogsData(msg)
	case LogLinesMsg:
		return m.handleLogLines(msg)
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleWindowSize handles window resize events
func (m LogsModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.instancesTable.SetSize(msg.Width-4, msg.Height-18)
	m.viewport.Width = msg.Width - 4
	m.viewport.Height = msg.Height - 10
	return m, nil
}

// handleLogsData handles instance list response from API
func (m LogsModel) handleLogsData(msg LogsDataMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.error = msg.Error.Error()
		m.loading = false
		return m, nil
	}

	m.instances = msg.Instances
	m.loading = false
	m.error = ""
	m.updateInstancesTable()
	return m, nil
}

// handleLogLines handles log lines response from API
func (m LogsModel) handleLogLines(msg LogLinesMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.error = msg.Error.Error()
		m.loading = false
		return m, nil
	}

	m.logLines = msg.Lines
	m.loading = false
	m.error = ""

	// Update viewport content
	m.viewport.SetContent(strings.Join(m.logLines, "\n"))
	return m, nil
}

// handleKeyPress handles keyboard input
func (m LogsModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "r", "f5":
		return m.handleRefresh()
	case "tab":
		return m.handleTabSwitch()
	case "enter":
		return m.handleEnterKey()
	case "t":
		return m.handleChangeLogType()
	case "esc":
		return m.handleEscKey()
	case "up", "k":
		return m.handleUpKey(msg)
	case "down", "j":
		return m.handleDownKey(msg)
	case "pgup", "pgdown", "home", "end":
		return m.handleViewportNav(msg)
	}

	return m, nil
}

// handleRefresh refreshes data based on current tab
func (m LogsModel) handleRefresh() (tea.Model, tea.Cmd) {
	m.loading = true
	if m.selectedTab == 0 {
		return m, func() tea.Msg { return m.fetchInstances() }
	} else if m.selectedInstanceName != "" {
		return m, m.fetchLogs(m.selectedInstanceName, m.logType)
	}
	return m, nil
}

// handleTabSwitch cycles through tabs
func (m LogsModel) handleTabSwitch() (tea.Model, tea.Cmd) {
	m.selectedTab = (m.selectedTab + 1) % 2
	return m, nil
}

// handleEnterKey handles Enter key press (view logs or close dialog)
func (m LogsModel) handleEnterKey() (tea.Model, tea.Cmd) {
	// View logs for selected instance
	if m.selectedTab == 0 && m.selectedInstance < len(m.instances) {
		instance := m.instances[m.selectedInstance]
		m.selectedInstanceName = instance.Name
		m.selectedTab = 1
		m.loading = true
		return m, m.fetchLogs(instance.Name, m.logType)
	}
	// Close type dialog
	if m.showTypeDialog {
		m.showTypeDialog = false
	}
	return m, nil
}

// handleChangeLogType shows log type selection dialog
func (m LogsModel) handleChangeLogType() (tea.Model, tea.Cmd) {
	if m.selectedTab == 1 {
		m.showTypeDialog = true
	}
	return m, nil
}

// handleEscKey handles Escape key press (close dialogs or go back)
func (m LogsModel) handleEscKey() (tea.Model, tea.Cmd) {
	if m.showTypeDialog {
		m.showTypeDialog = false
		return m, nil
	}
	if m.selectedTab == 1 {
		m.selectedTab = 0
		m.selectedInstanceName = ""
		m.logLines = nil
	}
	return m, nil
}

// handleUpKey handles up arrow navigation
func (m LogsModel) handleUpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.selectedTab == 0 && m.selectedInstance > 0 {
		m.selectedInstance--
		return m, nil
	} else if m.selectedTab == 1 {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleDownKey handles down arrow navigation
func (m LogsModel) handleDownKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.selectedTab == 0 && m.selectedInstance < len(m.instances)-1 {
		m.selectedInstance++
		return m, nil
	} else if m.selectedTab == 1 {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleViewportNav handles viewport navigation keys
func (m LogsModel) handleViewportNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.selectedTab == 1 {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the model
func (m LogsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("📜 Instance Logs Viewer")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0:
		b.WriteString(m.renderInstancesList())
	case 1:
		b.WriteString(m.renderLogsViewer())
	}

	// Show type dialog if active
	if m.showTypeDialog {
		dialog := m.renderTypeDialog()
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

// renderInstancesList displays the instances selection list
func (m LogsModel) renderInstancesList() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Select Instance"))
	b.WriteString("\n\n")

	if len(m.instances) == 0 {
		b.WriteString("No instances available\n")
		return b.String()
	}

	// Instances table
	b.WriteString(m.instancesTable.View())

	// Instance details
	if m.selectedInstance < len(m.instances) {
		instance := m.instances[m.selectedInstance]
		b.WriteString("\n\n")
		b.WriteString(theme.SubTitle.Render("Instance Info") + "\n\n")
		b.WriteString(fmt.Sprintf("Name: %s\n", instance.Name))
		b.WriteString(fmt.Sprintf("State: %s\n", instance.State))
		b.WriteString(fmt.Sprintf("Template: %s\n", instance.Template))
		b.WriteString("\nPress Enter to view logs\n")
	}

	return b.String()
}

// renderLogsViewer displays the logs viewer
func (m LogsModel) renderLogsViewer() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render(fmt.Sprintf("Logs: %s (%s)", m.selectedInstanceName, m.logType)))
	b.WriteString("\n\n")

	if len(m.logLines) == 0 {
		b.WriteString("No logs available\n")
		return b.String()
	}

	// Log viewport
	b.WriteString(m.viewport.View())

	b.WriteString("\n\n")
	b.WriteString(theme.SubTitle.Render("Log Controls") + "\n")
	b.WriteString("↑/↓: scroll • PgUp/PgDn: page • Home/End: top/bottom • t: change type • esc: back\n")

	return b.String()
}

// renderTypeDialog displays the log type selection dialog
func (m LogsModel) renderTypeDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("Select Log Type") + "\n\n")
	content.WriteString("Available log types:\n\n")
	content.WriteString("  • console      - System console output\n")
	content.WriteString("  • cloud-init   - Cloud-init logs\n")
	content.WriteString("  • messages     - System messages\n")
	content.WriteString("  • secure       - Security logs\n")
	content.WriteString("  • boot         - Boot logs\n")
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Current: %s\n\n", m.logType))
	content.WriteString("Press Esc to close\n")

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m LogsModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showTypeDialog {
		helps = []string{"esc: close"}
	} else if m.selectedTab == 0 {
		helps = []string{"↑/↓: select", "enter: view logs", "r: refresh", "q: quit"}
	} else {
		helps = []string{"↑/↓: scroll", "t: change type", "r: refresh", "esc: back", "q: quit"}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// updateInstancesTable updates the instances table with current data
func (m *LogsModel) updateInstancesTable() {
	rows := []table.Row{}
	for i, instance := range m.instances {
		// Selection indicator
		name := instance.Name
		if i == m.selectedInstance {
			name = "> " + name
		}

		row := table.Row{
			name,
			instance.State,
			instance.Template,
			instance.PublicIP,
		}
		rows = append(rows, row)
	}

	m.instancesTable.SetRows(rows)
}
