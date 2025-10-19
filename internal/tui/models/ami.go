package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// AMIModel represents the AMI management view
type AMIModel struct {
	apiClient          apiClient
	amisTable          components.Table
	buildsTable        components.Table
	statusBar          components.StatusBar
	spinner            components.Spinner
	width              int
	height             int
	loading            bool
	error              string
	amis               []api.AMIResponse
	builds             []api.AMIBuildResponse
	selectedTab        int // 0=amis, 1=builds, 2=regions
	selectedAMI        int
	selectedBuild      int
	showBuildDialog    bool
	showDeleteDialog   bool
	dialogTemplateName string
	dialogAMIID        string
	regions            []api.AMIRegionResponse
}

// AMIDataMsg represents AMI data retrieved from the API
type AMIDataMsg struct {
	AMIs    []api.AMIResponse
	Builds  []api.AMIBuildResponse
	Regions []api.AMIRegionResponse
	Error   error
}

// AMIActionMsg represents AMI action result
type AMIActionMsg struct {
	Success bool
	Message string
	Error   error
}

// NewAMIModel creates a new AMI management model
func NewAMIModel(apiClient apiClient) AMIModel {
	// Create AMIs table
	amiColumns := []table.Column{
		{Title: "AMI ID", Width: 22},
		{Title: "TEMPLATE", Width: 25},
		{Title: "REGION", Width: 15},
		{Title: "STATE", Width: 12},
		{Title: "CREATED", Width: 18},
	}

	amisTable := components.NewTable(amiColumns, []table.Row{}, 80, 8, true)

	// Create builds table
	buildColumns := []table.Column{
		{Title: "BUILD ID", Width: 20},
		{Title: "TEMPLATE", Width: 25},
		{Title: "STATUS", Width: 15},
		{Title: "PROGRESS", Width: 12},
		{Title: "STARTED", Width: 18},
	}

	buildsTable := components.NewTable(buildColumns, []table.Row{}, 80, 8, true)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation AMI Management", "")
	spinner := components.NewSpinner("Loading AMI data...")

	return AMIModel{
		apiClient:   apiClient,
		amisTable:   amisTable,
		buildsTable: buildsTable,
		statusBar:   statusBar,
		spinner:     spinner,
		width:       80,
		height:      24,
		loading:     true,
		selectedTab: 0,
	}
}

// Init initializes the model
func (m AMIModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchAMIData,
	)
}

// fetchAMIData retrieves AMI data from the API
func (m AMIModel) fetchAMIData() tea.Msg {
	// Fetch AMIs
	amisResp, err := m.apiClient.ListAMIs(context.Background())
	if err != nil {
		return AMIDataMsg{Error: fmt.Errorf("failed to list AMIs: %w", err)}
	}

	// Fetch builds
	buildsResp, err := m.apiClient.ListAMIBuilds(context.Background())
	if err != nil {
		return AMIDataMsg{Error: fmt.Errorf("failed to list AMI builds: %w", err)}
	}

	// Fetch regions
	regionsResp, err := m.apiClient.ListAMIRegions(context.Background())
	if err != nil {
		return AMIDataMsg{Error: fmt.Errorf("failed to list AMI regions: %w", err)}
	}

	return AMIDataMsg{
		AMIs:    amisResp.AMIs,
		Builds:  buildsResp.Builds,
		Regions: regionsResp.Regions,
		Error:   nil,
	}
}

// Update handles messages and updates the model
func (m AMIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case AMIDataMsg:
		return m.handleAMIData(msg)
	case AMIActionMsg:
		return m.handleActionResult(msg)
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleWindowSize handles window resize events
func (m AMIModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.amisTable.SetSize(msg.Width-4, msg.Height-18)
	m.buildsTable.SetSize(msg.Width-4, msg.Height-18)
	return m, nil
}

// handleAMIData handles AMI data response from API
func (m AMIModel) handleAMIData(msg AMIDataMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.error = msg.Error.Error()
		m.loading = false
		return m, nil
	}

	m.amis = msg.AMIs
	m.builds = msg.Builds
	m.regions = msg.Regions
	m.loading = false
	m.error = ""

	// Update tables with data
	m.updateAMIsTable()
	m.updateBuildsTable()
	return m, nil
}

// handleActionResult handles AMI action result
func (m AMIModel) handleActionResult(msg AMIActionMsg) (tea.Model, tea.Cmd) {
	m.showBuildDialog = false
	m.showDeleteDialog = false
	if msg.Error != nil {
		m.error = msg.Error.Error()
		return m, nil
	}
	// Refresh data after action
	return m, m.fetchAMIData
}

// handleKeyPress handles keyboard input
func (m AMIModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "b":
		return m.handleBuildAMI()
	case "d":
		return m.handleDeleteAMI()
	case "enter":
		return m.handleEnterKey()
	case "esc":
		return m.handleEscKey()
	case "up", "k":
		return m.handleUpKey()
	case "down", "j":
		return m.handleDownKey()
	}

	return m, nil
}

// handleRefresh refreshes AMI data
func (m AMIModel) handleRefresh() (tea.Model, tea.Cmd) {
	m.loading = true
	return m, m.fetchAMIData
}

// handleTabSwitch cycles through tabs
func (m AMIModel) handleTabSwitch() (tea.Model, tea.Cmd) {
	m.selectedTab = (m.selectedTab + 1) % 3
	return m, nil
}

// handleBuildAMI shows build AMI dialog
func (m AMIModel) handleBuildAMI() (tea.Model, tea.Cmd) {
	if m.selectedTab == 0 {
		m.showBuildDialog = true
	}
	return m, nil
}

// handleDeleteAMI shows delete AMI dialog
func (m AMIModel) handleDeleteAMI() (tea.Model, tea.Cmd) {
	if m.selectedTab == 0 && m.selectedAMI < len(m.amis) {
		ami := m.amis[m.selectedAMI]
		m.dialogAMIID = ami.ID
		m.showDeleteDialog = true
	}
	return m, nil
}

// handleEnterKey handles Enter key press (dialog confirmation)
func (m AMIModel) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.showBuildDialog {
		m.showBuildDialog = false
		return m, nil
	}
	if m.showDeleteDialog {
		return m, m.deleteAMI(m.dialogAMIID)
	}
	return m, nil
}

// handleEscKey handles Escape key press (close dialogs)
func (m AMIModel) handleEscKey() (tea.Model, tea.Cmd) {
	if m.showBuildDialog {
		m.showBuildDialog = false
		return m, nil
	}
	if m.showDeleteDialog {
		m.showDeleteDialog = false
		return m, nil
	}
	return m, nil
}

// handleUpKey handles up arrow navigation
func (m AMIModel) handleUpKey() (tea.Model, tea.Cmd) {
	if m.selectedTab == 0 && m.selectedAMI > 0 {
		m.selectedAMI--
	} else if m.selectedTab == 1 && m.selectedBuild > 0 {
		m.selectedBuild--
	}
	return m, nil
}

// handleDownKey handles down arrow navigation
func (m AMIModel) handleDownKey() (tea.Model, tea.Cmd) {
	if m.selectedTab == 0 && m.selectedAMI < len(m.amis)-1 {
		m.selectedAMI++
	} else if m.selectedTab == 1 && m.selectedBuild < len(m.builds)-1 {
		m.selectedBuild++
	}
	return m, nil
}

// View renders the model
func (m AMIModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("💿 AMI Management")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0:
		b.WriteString(m.renderAMIs())
	case 1:
		b.WriteString(m.renderBuilds())
	case 2:
		b.WriteString(m.renderRegions())
	}

	// Show build dialog if active
	if m.showBuildDialog {
		dialog := m.renderBuildDialog()
		b.WriteString("\n\n")
		b.WriteString(dialog)
	}

	// Show delete dialog if active
	if m.showDeleteDialog {
		dialog := m.renderDeleteDialog()
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

// renderTabs displays the tab navigation
func (m AMIModel) renderTabs() string {
	theme := styles.CurrentTheme
	tabs := []string{"AMIs", "Builds", "Regions"}

	var renderedTabs []string
	for i, tab := range tabs {
		if i == m.selectedTab {
			renderedTabs = append(renderedTabs, theme.Tab.Active.Render(" "+tab+" "))
		} else {
			renderedTabs = append(renderedTabs, theme.Tab.Inactive.Render(" "+tab+" "))
		}
	}

	return strings.Join(renderedTabs, " ")
}

// renderAMIs displays the AMIs view
func (m AMIModel) renderAMIs() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Available AMIs"))
	b.WriteString("\n\n")

	if len(m.amis) == 0 {
		b.WriteString("No AMIs available. Build an AMI to enable fast instance launching.\n")
		b.WriteString("\n")
		b.WriteString("AMIs reduce launch times from 5-8 minutes to under 30 seconds.\n")
		return b.String()
	}

	// AMIs table
	b.WriteString(m.amisTable.View())

	// AMI details
	if m.selectedAMI < len(m.amis) {
		ami := m.amis[m.selectedAMI]
		b.WriteString("\n\n")
		b.WriteString(theme.SubTitle.Render("AMI Details") + "\n\n")
		b.WriteString(fmt.Sprintf("ID: %s\n", ami.ID))
		b.WriteString(fmt.Sprintf("Template: %s\n", ami.TemplateName))
		b.WriteString(fmt.Sprintf("Region: %s\n", ami.Region))
		b.WriteString(fmt.Sprintf("Architecture: %s\n", ami.Architecture))
		b.WriteString(fmt.Sprintf("Size: %.2f GB\n", ami.SizeGB))
		if ami.Description != "" {
			b.WriteString(fmt.Sprintf("Description: %s\n", ami.Description))
		}
	}

	return b.String()
}

// renderBuilds displays the AMI builds view
func (m AMIModel) renderBuilds() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("AMI Build Status"))
	b.WriteString("\n\n")

	if len(m.builds) == 0 {
		b.WriteString("No active AMI builds\n")
		return b.String()
	}

	// Builds table
	b.WriteString(m.buildsTable.View())

	// Build details
	if m.selectedBuild < len(m.builds) {
		build := m.builds[m.selectedBuild]
		b.WriteString("\n\n")
		b.WriteString(theme.SubTitle.Render("Build Details") + "\n\n")
		b.WriteString(fmt.Sprintf("Build ID: %s\n", build.ID))
		b.WriteString(fmt.Sprintf("Template: %s\n", build.TemplateName))
		b.WriteString(fmt.Sprintf("Status: %s\n", build.Status))
		b.WriteString(fmt.Sprintf("Progress: %d%%\n", build.Progress))
		if build.CurrentStep != "" {
			b.WriteString(fmt.Sprintf("Current Step: %s\n", build.CurrentStep))
		}
		if build.Error != "" {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
			b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s\n", build.Error)))
		}
	}

	return b.String()
}

// renderRegions displays the AMI regions view
func (m AMIModel) renderRegions() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("AMI Regional Coverage"))
	b.WriteString("\n\n")

	if len(m.regions) == 0 {
		b.WriteString("No regional data available\n")
		return b.String()
	}

	// Regions list
	for _, region := range m.regions {
		statusIcon := "✅"
		if region.AMICount == 0 {
			statusIcon = "⚪"
		}
		b.WriteString(fmt.Sprintf("%s %s - %d AMIs\n", statusIcon, region.Name, region.AMICount))
	}

	b.WriteString("\n")
	b.WriteString(theme.SubTitle.Render("Regional Distribution") + "\n\n")
	b.WriteString("AMIs are region-specific and must be copied to other regions for use.\n")
	b.WriteString("Use 'cws ami publish' to copy AMIs across regions.\n")

	return b.String()
}

// renderBuildDialog displays the build AMI dialog
func (m AMIModel) renderBuildDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("Build New AMI") + "\n\n")
	content.WriteString("Select a template to build an AMI.\n")
	content.WriteString("This process takes 10-15 minutes but enables 30-second launches.\n\n")
	content.WriteString("Press Enter to continue, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderDeleteDialog displays the delete AMI confirmation dialog
func (m AMIModel) renderDeleteDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("9")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("⚠️  Delete AMI") + "\n\n")
	content.WriteString(fmt.Sprintf("AMI ID: %s\n\n", m.dialogAMIID))
	content.WriteString("This will permanently delete the AMI and associated snapshots.\n")
	content.WriteString("This action cannot be undone.\n\n")
	content.WriteString("Press Enter to confirm, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m AMIModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showBuildDialog || m.showDeleteDialog {
		helps = []string{"enter: confirm", "esc: cancel"}
	} else if m.selectedTab == 0 {
		helps = []string{"↑/↓: select", "tab: switch tabs", "b: build", "d: delete", "r: refresh", "q: quit"}
	} else if m.selectedTab == 1 {
		helps = []string{"↑/↓: select", "tab: switch tabs", "r: refresh", "q: quit"}
	} else {
		helps = []string{"tab: switch tabs", "r: refresh", "q: quit"}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// deleteAMI deletes an AMI
func (m AMIModel) deleteAMI(amiID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.DeleteAMI(context.Background(), amiID)
		if err != nil {
			return AMIActionMsg{Error: fmt.Errorf("failed to delete AMI: %w", err)}
		}

		return AMIActionMsg{
			Success: true,
			Message: fmt.Sprintf("Successfully deleted AMI %s", amiID),
			Error:   nil,
		}
	}
}

// updateAMIsTable updates the AMIs table with current data
func (m *AMIModel) updateAMIsTable() {
	rows := []table.Row{}
	for i, ami := range m.amis {
		// Selection indicator
		id := ami.ID
		if i == m.selectedAMI {
			id = "> " + id
		}

		// Format created time
		created := ami.CreatedAt.Format("2006-01-02 15:04")

		row := table.Row{
			id,
			ami.TemplateName,
			ami.Region,
			ami.State,
			created,
		}
		rows = append(rows, row)
	}

	m.amisTable.SetRows(rows)
}

// updateBuildsTable updates the builds table with current data
func (m *AMIModel) updateBuildsTable() {
	rows := []table.Row{}
	for i, build := range m.builds {
		// Selection indicator
		buildID := build.ID
		if i == m.selectedBuild {
			buildID = "> " + buildID
		}

		// Format started time
		started := build.StartedAt.Format("2006-01-02 15:04")

		row := table.Row{
			buildID,
			build.TemplateName,
			build.Status,
			fmt.Sprintf("%d%%", build.Progress),
			started,
		}
		rows = append(rows, row)
	}

	m.buildsTable.SetRows(rows)
}
