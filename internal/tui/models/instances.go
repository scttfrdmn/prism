package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// InstanceActionType represents different actions that can be performed on instances
type InstanceActionType int

const (
	// InstanceActionNone means no action
	InstanceActionNone InstanceActionType = iota
	// InstanceActionStart starts an instance
	InstanceActionStart
	// InstanceActionStop stops an instance
	InstanceActionStop
	// InstanceActionReboot reboots an instance
	InstanceActionReboot
	// InstanceActionTerminate terminates an instance
	InstanceActionTerminate
	// InstanceActionConnect connects to an instance
	InstanceActionConnect
)

// InstanceActionResult is returned when an instance action is performed
type InstanceActionResult struct {
	Action  InstanceActionType
	Success bool
	Error   error
	Message string
}

// NotificationActionResult wraps InstanceActionResult with notification information
type NotificationActionResult struct {
	Result InstanceActionResult
	NotificationType components.NotificationType
}

// InstancesModel represents the instances view
type InstancesModel struct {
	apiClient      api.CloudWorkstationAPI
	instancesTable components.Table
	detailView     viewport.Model
	statusBar      components.StatusBar
	spinner        components.Spinner
	search         components.Search
	width          int
	height         int
	loading        bool
	error          string
	instances      []types.Instance
	filteredRows   []table.Row
	allRows        []table.Row
	totalCost      float64
	selected       string
	actionInProgress bool
	currentAction  InstanceActionType
	searchActive   bool
}

// NewInstancesModel creates a new instances model
func NewInstancesModel(apiClient api.CloudWorkstationAPI) InstancesModel {
	theme := styles.CurrentTheme

	// Create instances table
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "TEMPLATE", Width: 15},
		{Title: "STATUS", Width: 10},
		{Title: "IP ADDRESS", Width: 15},
		{Title: "COST/DAY", Width: 10},
	}

	instancesTable := components.NewTable(columns, []table.Row{}, 70, 5, true)

	// Set up detail view for instance information
	detailView := viewport.New(0, 0)
	detailView.Style = theme.Panel

	// Create status bar and spinner
	statusBar := components.NewStatusBar("", "")
	spinner := components.NewSpinner("Loading instances...")
	
	// Create search component
	search := components.NewSearch()

	return InstancesModel{
		apiClient:      apiClient,
		instancesTable: instancesTable,
		detailView:     detailView,
		statusBar:      statusBar,
		spinner:        spinner,
		search:         search,
		width:          80,
		height:         24,
		loading:        true,
		instances:      []types.Instance{},
		filteredRows:   []table.Row{},
		allRows:        []table.Row{},
		actionInProgress: false,
		currentAction:  InstanceActionNone,
		searchActive:   false,
	}
}

// Init initializes the model
func (m InstancesModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchInstances,
	)
}

// fetchInstances retrieves instance data from the API
func (m InstancesModel) fetchInstances() tea.Msg {
	response, err := m.apiClient.ListInstances(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}
	return response
}

// startInstance starts the selected instance
func (m InstancesModel) startInstance() tea.Cmd {
	return func() tea.Msg {
		// Find the instance
		var instance *types.Instance
		for i := range m.instances {
			if m.instances[i].Name == m.selected {
				instance = &m.instances[i]
				break
			}
		}

		if instance == nil {
			return InstanceActionResult{
				Action:  InstanceActionStart,
				Success: false,
				Error:   fmt.Errorf("instance not found: %s", m.selected),
			}
		}

		// Call API to start the instance
		err := m.apiClient.StartInstance(context.Background(), m.selected)
		if err != nil {
			return InstanceActionResult{
				Action:  InstanceActionStart,
				Success: false,
				Error:   err,
			}
		}

		return InstanceActionResult{
			Action:  InstanceActionStart,
			Success: true,
			Message: fmt.Sprintf("Started instance: %s", m.selected),
		}
	}
}

// stopInstance stops the selected instance
func (m InstancesModel) stopInstance() tea.Cmd {
	return func() tea.Msg {
		// Call API to stop the instance
		err := m.apiClient.StopInstance(context.Background(), m.selected)
		if err != nil {
			return InstanceActionResult{
				Action:  InstanceActionStop,
				Success: false,
				Error:   err,
			}
		}

		return InstanceActionResult{
			Action:  InstanceActionStop,
			Success: true,
			Message: fmt.Sprintf("Stopped instance: %s", m.selected),
		}
	}
}

// terminateInstance terminates the selected instance
func (m InstancesModel) terminateInstance() tea.Cmd {
	return func() tea.Msg {
		// Call API to terminate the instance
		err := m.apiClient.DeleteInstance(context.Background(), m.selected)
		if err != nil {
			return InstanceActionResult{
				Action:  InstanceActionTerminate,
				Success: false,
				Error:   err,
			}
		}

		return InstanceActionResult{
			Action:  InstanceActionTerminate,
			Success: true,
			Message: fmt.Sprintf("Terminated instance: %s", m.selected),
		}
	}
}

// Update handles messages and updates the model
func (m InstancesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)

		// Update table and detail view dimensions
		leftWidth := m.width / 3 * 2  // 2/3 of screen for table
		rightWidth := m.width - leftWidth - 2 // Account for separator
		contentHeight := m.height - 4 // Account for title and status

		m.instancesTable.SetSize(leftWidth, contentHeight)
		m.detailView.Width = rightWidth
		m.detailView.Height = contentHeight
		
		// Update search component width
		m.search.SetWidth(leftWidth)

	case components.SearchMsg:
		// Handle search queries
		if !m.loading {
			m.applySearch(msg.Query)
			return m, nil
		}
		
	case components.SearchActivateMsg:
		// Search was activated
		m.searchActive = true
		return m, nil
		
	case components.SearchDeactivateMsg:
		// Search was deactivated
		m.searchActive = false
		m.applySearch("") // Clear search
		return m, nil
		
	case tea.KeyMsg:
		// If search is active, let the search component handle the key press first
		if m.searchActive || msg.String() == "/" {
			newSearch, cmd := m.search.Update(msg)
			m.search = newSearch
			return m, cmd
		}
		
		// Handle key presses
		switch msg.String() {
		case "r":
			if !m.actionInProgress {
				m.loading = true
				m.error = ""
				return m, m.fetchInstances
			}

		case "s":
			if !m.actionInProgress && m.selected != "" {
				m.actionInProgress = true
				m.currentAction = InstanceActionStart
				m.statusBar.SetStatus(fmt.Sprintf("Starting instance: %s", m.selected), components.StatusWarning)
				return m, m.startInstance()
			}

		case "p":
			if !m.actionInProgress && m.selected != "" {
				m.actionInProgress = true
				m.currentAction = InstanceActionStop
				m.statusBar.SetStatus(fmt.Sprintf("Stopping instance: %s", m.selected), components.StatusWarning)
				return m, m.stopInstance()
			}

		case "t":
			if !m.actionInProgress && m.selected != "" {
				m.actionInProgress = true
				m.currentAction = InstanceActionTerminate
				m.statusBar.SetStatus(fmt.Sprintf("Terminating instance: %s", m.selected), components.StatusWarning)
				return m, m.terminateInstance()
			}

		case "c":
			if m.selected != "" {
				// Show connect information
				m.updateDetailView(true)
				return m, nil
			}

		case "q", "esc":
			return m, tea.Quit
		}

		// Only process table inputs when not loading or performing an action
		if !m.loading && !m.actionInProgress {
			var _ tea.Cmd
			tableModel, tableCmd := m.instancesTable.Update(msg)
			m.instancesTable = tableModel
			cmds = append(cmds, tableCmd)

			// Handle selection changes
			selectedRow := m.instancesTable.SelectedRow()
			if len(selectedRow) > 0 && selectedRow[0] != m.selected {
				m.selected = selectedRow[0]
				m.updateDetailView(false)
			}

			// Update detail view on scroll
			var viewportCmd tea.Cmd
			m.detailView, viewportCmd = m.detailView.Update(msg)
			cmds = append(cmds, viewportCmd)
		}

	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchInstances

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)

	case InstanceActionResult:
		m.actionInProgress = false
		// Create notification for the action result
		var notificationType components.NotificationType
		if msg.Success {
			m.statusBar.SetStatus(msg.Message, components.StatusSuccess)
			notificationType = components.NotificationSuccess
			// After a successful action, refresh the instance list
			return m, tea.Batch(
				m.fetchInstances,
				// Give user time to see the success message
				tea.Tick(2*time.Second, func(time.Time) tea.Msg {
					return RefreshMsg{}
				}),
				// Return notification action result
				func() tea.Msg {
					return NotificationActionResult{
						Result: msg,
						NotificationType: notificationType,
					}
				},
			)
		} else {
			m.error = msg.Error.Error()
			m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
			notificationType = components.NotificationError
			
			// Return notification for error
			return m, func() tea.Msg {
				return NotificationActionResult{
					Result: msg,
					NotificationType: notificationType,
				}
			}
		}

	case *types.ListResponse:
		m.loading = false
		m.instances = msg.Instances
		m.totalCost = msg.TotalCost

		// Update instances table
		m.allRows = []table.Row{}
		for _, instance := range m.instances {
			status := strings.ToUpper(instance.State)
			m.allRows = append(m.allRows, table.Row{
				instance.Name,
				instance.Template,
				status,
				instance.PublicIP,
				fmt.Sprintf("$%.2f", instance.EstimatedDailyCost),
			})
		}
		
		// Apply any active search filter
		query := m.search.Query()
		if query != "" {
			m.applySearch(query)
		} else {
			m.filteredRows = m.allRows
			m.instancesTable.SetRows(m.allRows)
		}
		m.statusBar.SetStatus("Ready", components.StatusSuccess)

		// If we had a selection before, try to restore it
		if m.selected != "" {
			found := false
			for _, instance := range m.instances {
				if instance.Name == m.selected {
					found = true
					break
				}
			}
			
			// If selected instance was not found (perhaps it was terminated),
			// reset selection to the first instance if available
			if !found {
				if len(m.instances) > 0 {
					m.selected = m.instances[0].Name
				} else {
					m.selected = ""
				}
			}
			
			m.updateDetailView(false)
		} else if len(m.instances) > 0 {
			// Select first instance if nothing was selected before
			m.selected = m.instances[0].Name
			m.updateDetailView(false)
		}

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

	return m, tea.Batch(cmds...)
}

// applySearch filters the instances based on a search query
func (m *InstancesModel) applySearch(query string) {
	// If query is empty, show all instances
	if query == "" {
		m.filteredRows = m.allRows
		m.instancesTable.SetRows(m.allRows)
		return
	}
	
	// Convert query to lowercase for case-insensitive search
	query = strings.ToLower(query)
	
	// Filter rows based on query
	m.filteredRows = []table.Row{}
	for _, row := range m.allRows {
		// Check if any field matches the query
		matched := false
		for _, field := range row {
			if strings.Contains(strings.ToLower(field), query) {
				matched = true
				break
			}
		}
		
		if matched {
			m.filteredRows = append(m.filteredRows, row)
		}
	}
	
	// Update table with filtered rows
	m.instancesTable.SetRows(m.filteredRows)
	m.statusBar.SetStatus(fmt.Sprintf("Found %d matching instances", len(m.filteredRows)), components.StatusSuccess)
}

// updateDetailView updates the content of the detail view with the selected instance
func (m *InstancesModel) updateDetailView(showConnect bool) {
	if m.selected == "" {
		return
	}

	theme := styles.CurrentTheme
	
	// Find the instance
	var instance *types.Instance
	for i := range m.instances {
		if m.instances[i].Name == m.selected {
			instance = &m.instances[i]
			break
		}
	}

	if instance == nil {
		return
	}

	// Format the instance details
	var content strings.Builder
	
	content.WriteString(theme.SectionTitle.Render(instance.Name) + "\n\n")
	
	content.WriteString(theme.SubTitle.Render("Instance Details:") + "\n")
	content.WriteString(fmt.Sprintf("ID: %s\n", instance.ID))
	content.WriteString(fmt.Sprintf("Template: %s\n", instance.Template))
	content.WriteString(fmt.Sprintf("Status: %s\n", strings.ToUpper(instance.State)))
	content.WriteString(fmt.Sprintf("Launch Time: %s\n\n", instance.LaunchTime.Format(time.RFC1123)))
	
	content.WriteString(theme.SubTitle.Render("Network:") + "\n")
	content.WriteString(fmt.Sprintf("Public IP: %s\n", instance.PublicIP))
	content.WriteString(fmt.Sprintf("Private IP: %s\n\n", instance.PrivateIP))
	
	content.WriteString(theme.SubTitle.Render("Cost:") + "\n")
	content.WriteString(fmt.Sprintf("Daily Cost: $%.2f\n", instance.EstimatedDailyCost))
	content.WriteString(fmt.Sprintf("Monthly Estimate: $%.2f\n\n", instance.EstimatedDailyCost*30))
	
	// Show connection details if requested
	if showConnect {
		content.WriteString(theme.SubTitle.Render("Connection:") + "\n")
		content.WriteString("SSH Command:\n")
		content.WriteString(fmt.Sprintf("  ssh ubuntu@%s\n\n", instance.PublicIP))
		
		// Determine ports based on template
		ports := []int{}
		if instance.Template == "r-research" {
			ports = []int{8787}
		} else if instance.Template == "python-research" {
			ports = []int{8888}
		} else if instance.Template == "desktop-research" {
			ports = []int{8443}
		}
		
		if len(ports) > 0 {
			content.WriteString("Open Ports:\n")
			for _, port := range ports {
				content.WriteString(fmt.Sprintf("  %d: http://%s:%d\n", port, instance.PublicIP, port))
			}
			content.WriteString("\n")
		}
	}
	
	content.WriteString(theme.SubTitle.Render("Actions:") + "\n")
	actions := []string{}
	
	// Show different actions based on instance state
	if strings.EqualFold(instance.State, "stopped") {
		actions = append(actions, "s: Start instance")
	} else if strings.EqualFold(instance.State, "running") {
		actions = append(actions, "p: Stop instance")
		actions = append(actions, "c: Show connection details")
	}
	
	actions = append(actions, "t: Terminate instance")
	content.WriteString(strings.Join(actions, "\n"))
	
	m.detailView.SetContent(content.String())
	m.detailView.GotoTop()
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
		// Create search component view
		searchView := m.search.View()
		
		// Split view with instances table on left and details on right
		tableView := m.instancesTable.View()
		
		// Combine search and table in left pane
		leftPane := lipgloss.JoinVertical(lipgloss.Left, searchView, tableView)
		rightPane := m.detailView.View()
		
		separator := lipgloss.NewStyle().
			Foreground(theme.MutedColor).
			Width(1).
			Height(m.height - 4).
			Render("│")
		
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, separator, rightPane)
	}
	
	// Help text
	help := components.CompactHelpView(components.HelpInstances)
	
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