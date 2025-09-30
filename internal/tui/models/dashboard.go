package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

// CostData represents cost information for display
type CostData struct {
	DailyCost   float64
	MonthlyCost float64
	ByTemplate  map[string]float64
	ByInstance  map[string]float64
	Storage     float64
	Volumes     float64
}

// DashboardModel represents the dashboard view
type DashboardModel struct {
	apiClient      apiClient
	instancesTable components.Table
	statusBar      components.StatusBar
	spinner        components.Spinner
	tabs           components.TabBar
	width          int
	height         int
	loading        bool
	error          string
	instances      []api.InstanceResponse
	costData       CostData
	activeTab      string
	refreshTicker  tea.Cmd
}

// DashboardRefreshMsg is sent when dashboard data should be refreshed
type DashboardRefreshMsg struct{}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(apiClient apiClient) DashboardModel {

	// Create instances table
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "TEMPLATE", Width: 15},
		{Title: "STATUS", Width: 10},
		{Title: "COST/DAY", Width: 10},
	}

	instancesTable := components.NewTable(columns, []table.Row{}, 60, 5, true)

	// Create tabs
	tabs := components.NewTabBar(
		[]string{"Overview", "Instances", "Storage", "Costs"},
		0,
	)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Dashboard", version.GetVersion())
	spinner := components.NewSpinner("Loading dashboard data...")

	return DashboardModel{
		apiClient:      apiClient,
		instancesTable: instancesTable,
		statusBar:      statusBar,
		spinner:        spinner,
		tabs:           tabs,
		width:          80,
		height:         24,
		loading:        true,
		activeTab:      "Overview",
		refreshTicker:  refreshRoutine(30 * time.Second),
		costData: CostData{
			ByTemplate: make(map[string]float64),
			ByInstance: make(map[string]float64),
		},
	}
}

// Init initializes the model
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchDashboardData,
		m.refreshTicker,
	)
}

// fetchDashboardData retrieves instance data from the API
func (m DashboardModel) fetchDashboardData() tea.Msg {
	response, err := m.apiClient.ListInstances(context.Background())
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
			return m, m.fetchDashboardData

		case "q", "esc":
			return m, tea.Quit
		}

	case DashboardRefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchDashboardData

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)

	case *api.ListInstancesResponse:
		m.loading = false
		m.instances = msg.Instances
		m.costData.DailyCost = msg.TotalCost

		// Update instances table
		rows := []table.Row{}
		for _, instance := range m.instances {
			status := strings.ToUpper(instance.State)
			rows = append(rows, table.Row{
				instance.Name,
				instance.Template,
				status,
				fmt.Sprintf("$%.3f/hr eff:$%.3f", instance.HourlyRate, instance.EffectiveRate),
			})
		}
		m.instancesTable.SetRows(rows)
		m.statusBar.SetStatus("Ready", components.StatusSuccess)

		// Schedule next refresh
		return m, tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
			return DashboardRefreshMsg{}
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
			Height(m.height-4). // Account for title and status bar
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else {
		// System status panel
		systemPanel := theme.Panel.Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("System Status"),
				fmt.Sprintf("Region: %s", "us-west-2"), // TODO: Get from API
				fmt.Sprintf("Daemon: %s", "Running"),   // TODO: Get from API
			),
		)

		// Build instances panel
		instancesPanel := theme.Panel.Width(m.width/2 - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Running Instances"),
				m.instancesTable.View(),
			),
		)

		// Build cost panel
		costPanel := theme.Panel.Width(m.width/2 - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Cost Overview"),
				fmt.Sprintf("Daily Cost: $%.2f", m.costData.DailyCost),
				fmt.Sprintf("Monthly Estimate: $%.2f", m.costData.DailyCost*30),
			),
		)

		// Quick actions
		actions := []string{
			theme.Button.Render("Launch"),
			theme.Button.Render("Templates"),
			theme.Button.Render("Storage"),
		}

		quickActions := theme.Panel.Width(m.width - 4).Render(
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

	// Help text with navigation
	help := theme.Help.Render("Navigation: 1: Dashboard • 2: Instances • 3: Templates • 4: Storage • 5: Users • 6: Settings\n" +
		"Actions: r: refresh • q: quit")

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
