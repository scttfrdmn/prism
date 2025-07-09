package models

import (
	"fmt"
	"strings"
	"time"
	
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

// DashboardModel represents the dashboard view
type DashboardModel struct {
	apiClient      api.CloudWorkstationAPI
	instancesTable components.Table
	statusBar      components.StatusBar
	spinner        components.Spinner
	width          int
	height         int
	loading        bool
	error          string
	instances      []types.Instance
	totalCost      float64
}

// RefreshMsg is sent when data should be refreshed
type RefreshMsg struct{}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(apiClient api.CloudWorkstationAPI) DashboardModel {
	// Create instances table
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "TEMPLATE", Width: 15},
		{Title: "STATUS", Width: 10},
		{Title: "COST/DAY", Width: 10},
	}
	
	instancesTable := components.NewTable(columns, []table.Row{}, 60, 5, true)
	
	// Create status bar with version and placeholder region
	statusBar := components.NewStatusBar(version.GetVersion(), "us-west-2")
	
	// Create spinner for loading state
	spinner := components.NewSpinner("Loading instances...")
	
	return DashboardModel{
		apiClient:      apiClient,
		instancesTable: instancesTable,
		statusBar:      statusBar,
		spinner:        spinner,
		width:          80,
		height:         24,
		loading:        true,
		instances:      []types.Instance{},
	}
}

// Init initializes the model
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchInstances,
	)
}

// fetchInstances retrieves instance data from the API
func (m DashboardModel) fetchInstances() tea.Msg {
	response, err := m.apiClient.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}
	return response
}

// Update handles messages and updates the model
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.loading = true
			m.error = ""
			return m, m.fetchInstances
			
		case "q", "esc":
			return m, tea.Quit
		}

	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchInstances

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
		
	case *types.ListResponse:
		m.loading = false
		m.instances = msg.Instances
		m.totalCost = msg.TotalCost

		// Update instances table
		rows := []table.Row{}
		for _, instance := range m.instances {
			status := strings.ToUpper(instance.State)
			rows = append(rows, table.Row{
				instance.Name,
				instance.Template,
				status,
				fmt.Sprintf("$%.2f", instance.EstimatedDailyCost),
			})
		}
		m.instancesTable.SetRows(rows)
		m.statusBar.SetStatus("Ready", components.StatusSuccess)

		// Schedule next refresh
		return m, tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
			return RefreshMsg{}
		})
	}

	// Update components
	if m.loading {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	instancesTable, tableCmd := m.instancesTable.Update(msg)
	m.instancesTable = instancesTable
	cmds = append(cmds, tableCmd)

	return m, tea.Batch(cmds...)
}

// View renders the dashboard
func (m DashboardModel) View() string {
	theme := styles.CurrentTheme
	
	// Title section
	title := theme.Title.Render("CloudWorkstation Dashboard")
	
	// Content area
	var content string
	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 4). // Account for title and status bar
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else {
		// System status panel
		systemPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("System Status"),
				fmt.Sprintf("Region: %s", "us-west-2"), // TODO: Get from API
				fmt.Sprintf("Daemon: %s", "Running"),   // TODO: Get from API
			),
		)
		
		// Build instances panel
		instancesPanel := theme.Panel.Copy().Width(m.width / 2 - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Running Instances"),
				m.instancesTable.View(),
			),
		)
		
		// Build cost panel
		costPanel := theme.Panel.Copy().Width(m.width / 2 - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Cost Overview"),
				fmt.Sprintf("Daily Cost: $%.2f", m.totalCost),
				fmt.Sprintf("Monthly Estimate: $%.2f", m.totalCost*30),
			),
		)
		
		// Quick actions
		actions := []string{
			theme.Button.Render("Launch"),
			theme.Button.Render("Templates"),
			theme.Button.Render("Storage"),
		}
		
		quickActions := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Quick Actions"),
				lipgloss.JoinHorizontal(lipgloss.Center, actions...),
			),
		)
		
		// Join all panels
		middleSection := lipgloss.JoinHorizontal(
			lipgloss.Top,
			instancesPanel,
			costPanel,
		)
		
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			systemPanel,
			middleSection,
			quickActions,
		)
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
