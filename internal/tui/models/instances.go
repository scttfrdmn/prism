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
)

// InstancesModel represents the instances management view
type InstancesModel struct {
	apiClient      apiClient
	instancesTable components.Table
	statusBar      components.StatusBar
	spinner        components.Spinner
	width          int
	height         int
	loading        bool
	error          string
	instances      []api.InstanceResponse
	selected       int
	showingActions bool
	actionMessage  string

	// Command dispatcher for SOLID architecture
	dispatcher *CommandDispatcher
}

// InstanceRefreshMsg is sent when instance data should be refreshed
type InstanceRefreshMsg struct{}

// Note: InstanceActionMsg is defined in instance_action.go

// NewInstancesModel creates a new instances model
func NewInstancesModel(apiClient apiClient) InstancesModel {
	// Create instances table
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "TEMPLATE", Width: 15},
		{Title: "STATUS", Width: 12},
		{Title: "TYPE", Width: 4},
		{Title: "COST/DAY", Width: 10},
		{Title: "PUBLIC IP", Width: 15},
		{Title: "LAUNCH TIME", Width: 12},
	}

	instancesTable := components.NewTable(columns, []table.Row{}, 80, 10, true)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Instances", "")
	spinner := components.NewSpinner("Loading instances...")

	// Initialize command dispatcher with instance commands
	dispatcher := NewCommandDispatcher()
	dispatcher.RegisterCommand(InstanceWindowResizeCommand{})
	dispatcher.RegisterCommand(InstanceRefreshCommand{})
	dispatcher.RegisterCommand(InstanceActionsCommand{})
	dispatcher.RegisterCommand(InstanceConnectionCommand{})
	dispatcher.RegisterCommand(InstanceActionExecuteCommand{})
	dispatcher.RegisterCommand(InstanceActionCancelCommand{})
	dispatcher.RegisterCommand(InstanceTableNavigationCommand{})

	return InstancesModel{
		apiClient:      apiClient,
		instancesTable: instancesTable,
		statusBar:      statusBar,
		spinner:        spinner,
		width:          80,
		height:         24,
		loading:        true,
		selected:       0,
		dispatcher:     dispatcher,
	}
}

// Init initializes the model
func (m InstancesModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchInstances,
		m.refreshTicker(),
	)
}

// fetchInstances retrieves instance data from the API
func (m InstancesModel) fetchInstances() tea.Msg {
	resp, err := m.apiClient.ListInstances(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}
	return resp
}

// refreshTicker creates a ticker for auto-refresh
func (m InstancesModel) refreshTicker() tea.Cmd {
	return tea.Every(30*time.Second, func(t time.Time) tea.Msg {
		return InstanceRefreshMsg{}
	})
}

// performAction performs an action on the selected instance
func (m InstancesModel) performAction(action string) tea.Cmd {
	if len(m.instances) == 0 || m.selected >= len(m.instances) {
		return nil
	}

	instance := m.instances[m.selected]

	return func() tea.Msg {
		var err error
		switch action {
		case "start":
			err = m.apiClient.StartInstance(context.Background(), instance.Name)
		case "stop":
			err = m.apiClient.StopInstance(context.Background(), instance.Name)
		case "delete":
			err = m.apiClient.DeleteInstance(context.Background(), instance.Name)
		}

		if err != nil {
			return InstanceActionMsg{
				Action:  action,
				Success: false,
				Message: fmt.Sprintf("Failed to %s instance %s: %v", action, instance.Name, err),
			}
		}
		return InstanceActionMsg{
			Action:  action,
			Success: true,
			Message: fmt.Sprintf("Successfully %sed instance %s", action, instance.Name),
		}
	}
}

// Update handles messages and updates the model using Command Pattern (SOLID)
func (m InstancesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle global quit command
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "q" && !m.showingActions {
		return m, tea.Quit
	}

	// Try to dispatch command using Command Pattern
	newModel, cmd := m.dispatcher.Dispatch(msg, m)
	if cmd != nil {
		return newModel, cmd
	}
	// Check if model was updated
	if newInstanceModel, ok := newModel.(InstancesModel); ok && (newInstanceModel.loading != m.loading || newInstanceModel.showingActions != m.showingActions) {
		return newModel, cmd
	}

	// Handle non-command messages
	switch msg := msg.(type) {
	case InstanceRefreshMsg:
		if !m.loading {
			return m, m.fetchInstances
		}

	case InstanceActionMsg:
		m.loading = false
		if !msg.Success {
			m.error = msg.Message
			m.statusBar.SetStatus(m.error, components.StatusError)
		} else {
			m.statusBar.SetStatus(msg.Message, components.StatusSuccess)
			return m, m.fetchInstances
		}

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)

	case *api.ListInstancesResponse:
		m.loading = false
		m.instances = msg.Instances
		m.updateInstancesTable()
		m.statusBar.SetStatus(fmt.Sprintf("Loaded %d instances", len(m.instances)), components.StatusSuccess)
		cmds = append(cmds, m.refreshTicker())

	default:
		// Handle spinner updates
		if m.loading {
			var spinnerCmd tea.Cmd
			m.spinner, spinnerCmd = m.spinner.Update(msg)
			cmds = append(cmds, spinnerCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateInstancesTable updates the instances table with current data (Single Responsibility)
func (m *InstancesModel) updateInstancesTable() {
	rows := []table.Row{}
	for _, instance := range m.instances {
		status := strings.ToUpper(instance.State)
		launchTime := "N/A"
		if !instance.LaunchTime.IsZero() {
			launchTime = instance.LaunchTime.Format("01/02 15:04")
		}

		// Format spot/on-demand indicator
		typeIndicator := "OD"
		if instance.InstanceLifecycle == "spot" {
			typeIndicator = "SP"
		}

		rows = append(rows, table.Row{
			instance.Name,
			instance.Template,
			status,
			typeIndicator,
			fmt.Sprintf("$%.3f/hr", instance.HourlyRate),
			instance.PublicIP,
			launchTime,
		})
	}
	m.instancesTable.SetRows(rows)
}

// updateSelectedIndex updates the selected index based on table selection (Single Responsibility)
func (m *InstancesModel) updateSelectedIndex() {
	selectedRow := m.instancesTable.SelectedRow()
	if len(selectedRow) > 0 {
		// Find the index of the selected instance by name
		for i, instance := range m.instances {
			if instance.Name == selectedRow[0] {
				m.selected = i
				break
			}
		}
	}
}

// View renders the instances view
func (m InstancesModel) View() string {
	theme := styles.CurrentTheme

	// Title section
	title := theme.Title.Render("CloudWorkstation Instances")

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
	} else if len(m.instances) == 0 {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render("No instances found. Use 'cws launch' to create one.")
	} else {
		// Main instances table
		content = m.instancesTable.View()

		// Show action menu if requested
		if m.showingActions && m.selected < len(m.instances) {
			instance := m.instances[m.selected]
			actionMenu := theme.Panel.
				Width(40).
				Render(lipgloss.JoinVertical(
					lipgloss.Left,
					theme.PanelHeader.Render(fmt.Sprintf("Actions for %s", instance.Name)),
					"",
					"s - Start instance",
					"p - Stop instance",
					"d - Delete instance",
					"",
					"esc - Cancel",
				))

			// Overlay action menu
			content = lipgloss.Place(
				m.width, m.height-4,
				lipgloss.Center, lipgloss.Center,
				actionMenu,
				lipgloss.WithWhitespaceChars(""),
			)
		}

		// Show action message if present
		if m.actionMessage != "" {
			messageBox := theme.Panel.
				Width(min(len(m.actionMessage)+4, m.width-4)).
				Render(m.actionMessage)

			content += "\n" + messageBox
			m.actionMessage = "" // Clear after showing
		}
	}

	// Help text
	var help string
	if m.showingActions {
		help = theme.Help.Render("s: start • p: stop • d: delete • esc: cancel")
	} else {
		help = theme.Help.Render("r: refresh • a: actions • c: connect info • q: quit • ↑/↓: navigate")
	}

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

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
