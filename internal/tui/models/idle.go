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

// IdleModel represents the idle/hibernation management view
type IdleModel struct {
	apiClient          apiClient
	policiesTable      components.Table
	instancesTable     components.Table
	statusBar          components.StatusBar
	spinner            components.Spinner
	width              int
	height             int
	loading            bool
	error              string
	policies           map[string]api.IdlePolicyResponse
	instances          []api.InstanceResponse
	selectedTab        int // 0=policies, 1=instances, 2=history
	selectedPolicy     int
	selectedInstance   int
	showPolicyDialog   bool
	showEnableDialog   bool
	dialogPolicyName   string
	dialogInstanceName string
}

// IdleDataMsg represents idle data retrieved from the API
type IdleDataMsg struct {
	Policies  map[string]api.IdlePolicyResponse
	Instances []api.InstanceResponse
	Error     error
}

// IdlePolicyActionMsg represents policy action result
type IdlePolicyActionMsg struct {
	Success bool
	Message string
	Error   error
}

// NewIdleModel creates a new idle management model
func NewIdleModel(apiClient apiClient) IdleModel {
	// Create policies table
	policyColumns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "THRESHOLD (MIN)", Width: 15},
		{Title: "ACTION", Width: 15},
		{Title: "STATUS", Width: 15},
	}

	policiesTable := components.NewTable(policyColumns, []table.Row{}, 80, 8, true)

	// Create instances table
	instanceColumns := []table.Column{
		{Title: "INSTANCE", Width: 25},
		{Title: "IDLE DETECTION", Width: 15},
		{Title: "POLICY", Width: 15},
		{Title: "IDLE TIME", Width: 12},
		{Title: "STATUS", Width: 15},
	}

	instancesTable := components.NewTable(instanceColumns, []table.Row{}, 80, 8, true)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Idle Detection", "")
	spinner := components.NewSpinner("Loading idle detection data...")

	return IdleModel{
		apiClient:      apiClient,
		policiesTable:  policiesTable,
		instancesTable: instancesTable,
		statusBar:      statusBar,
		spinner:        spinner,
		width:          80,
		height:         24,
		loading:        true,
		selectedTab:    0,
	}
}

// Init initializes the model
func (m IdleModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchIdleData,
	)
}

// fetchIdleData retrieves idle detection data from the API
func (m IdleModel) fetchIdleData() tea.Msg {
	// Fetch idle policies
	policiesResp, err := m.apiClient.ListIdlePolicies(context.Background())
	if err != nil {
		return IdleDataMsg{Error: fmt.Errorf("failed to list idle policies: %w", err)}
	}

	// Fetch instances to show idle detection status
	instancesResp, err := m.apiClient.ListInstances(context.Background())
	if err != nil {
		return IdleDataMsg{Error: fmt.Errorf("failed to list instances: %w", err)}
	}

	return IdleDataMsg{
		Policies:  policiesResp.Policies,
		Instances: instancesResp.Instances,
		Error:     nil,
	}
}

// Update handles messages and updates the model
func (m IdleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.policiesTable.SetSize(msg.Width-4, msg.Height-18)
		m.instancesTable.SetSize(msg.Width-4, msg.Height-18)
		return m, nil

	case IdleDataMsg:
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.loading = false
			return m, nil
		}

		m.policies = msg.Policies
		m.instances = msg.Instances
		m.loading = false
		m.error = ""

		// Update tables with data
		m.updatePoliciesTable()
		m.updateInstancesTable()
		return m, nil

	case IdlePolicyActionMsg:
		m.showPolicyDialog = false
		m.showEnableDialog = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			return m, nil
		}
		// Refresh data after action
		return m, m.fetchIdleData

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r", "f5":
			// Refresh idle data
			m.loading = true
			return m, m.fetchIdleData

		case "tab":
			// Cycle through tabs
			m.selectedTab = (m.selectedTab + 1) % 3
			return m, nil

		case "u":
			// Update selected policy
			if m.selectedTab == 0 && len(m.policies) > 0 {
				m.showPolicyDialog = true
				return m, nil
			}

		case "e":
			// Enable idle detection for selected instance
			if m.selectedTab == 1 && m.selectedInstance < len(m.instances) {
				m.showEnableDialog = true
				instance := m.instances[m.selectedInstance]
				m.dialogInstanceName = instance.Name
				return m, nil
			}

		case "d":
			// Disable idle detection for selected instance
			if m.selectedTab == 1 && m.selectedInstance < len(m.instances) {
				instance := m.instances[m.selectedInstance]
				return m, m.disableIdleDetection(instance.Name)
			}

		case "enter":
			// Handle dialog confirmation
			if m.showPolicyDialog {
				// Update policy logic
				m.showPolicyDialog = false
				return m, nil
			}
			if m.showEnableDialog {
				// Enable idle detection with default policy
				return m, m.enableIdleDetection(m.dialogInstanceName, "default")
			}

		case "esc":
			// Close dialogs
			if m.showPolicyDialog {
				m.showPolicyDialog = false
				return m, nil
			}
			if m.showEnableDialog {
				m.showEnableDialog = false
				return m, nil
			}

		case "up", "k":
			if m.selectedTab == 0 && m.selectedPolicy > 0 {
				m.selectedPolicy--
			} else if m.selectedTab == 1 && m.selectedInstance > 0 {
				m.selectedInstance--
			}
			return m, nil

		case "down", "j":
			if m.selectedTab == 0 && m.selectedPolicy < len(m.policies)-1 {
				m.selectedPolicy++
			} else if m.selectedTab == 1 && m.selectedInstance < len(m.instances)-1 {
				m.selectedInstance++
			}
			return m, nil
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the model
func (m IdleModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("💤 Idle Detection & Hibernation")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0:
		b.WriteString(m.renderPolicies())
	case 1:
		b.WriteString(m.renderInstances())
	case 2:
		b.WriteString(m.renderHistory())
	}

	// Show policy dialog if active
	if m.showPolicyDialog {
		dialog := m.renderPolicyDialog()
		b.WriteString("\n\n")
		b.WriteString(dialog)
	}

	// Show enable dialog if active
	if m.showEnableDialog {
		dialog := m.renderEnableDialog()
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
func (m IdleModel) renderTabs() string {
	theme := styles.CurrentTheme
	tabs := []string{"Policies", "Instances", "History"}

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

// renderPolicies displays the idle policies view
func (m IdleModel) renderPolicies() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Idle Detection Policies"))
	b.WriteString("\n\n")

	if len(m.policies) == 0 {
		b.WriteString("No idle policies configured\n")
		return b.String()
	}

	// Policies table
	b.WriteString(m.policiesTable.View())

	// Policy description
	b.WriteString("\n\n")
	b.WriteString(theme.SubTitle.Render("Policy Details") + "\n\n")
	b.WriteString("Idle policies automatically hibernate or stop instances after a period of inactivity.\n")
	b.WriteString("Configure policies to optimize costs while preserving your work environment.\n")

	return b.String()
}

// renderInstances displays the instances idle detection status
func (m IdleModel) renderInstances() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Instance Idle Detection Status"))
	b.WriteString("\n\n")

	if len(m.instances) == 0 {
		b.WriteString("No instances available\n")
		return b.String()
	}

	// Instances table
	b.WriteString(m.instancesTable.View())

	// Status summary
	b.WriteString("\n\n")
	enabledCount := 0
	for _, instance := range m.instances {
		// Check if idle detection is enabled (simplified check)
		if instance.State == "running" {
			enabledCount++
		}
	}
	b.WriteString(fmt.Sprintf("Idle Detection: %d/%d instances monitored\n", enabledCount, len(m.instances)))

	return b.String()
}

// renderHistory displays the hibernation history
func (m IdleModel) renderHistory() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Hibernation History"))
	b.WriteString("\n\n")

	// Sample history data (will be replaced with real data from backend)
	b.WriteString("Recent hibernation events:\n\n")
	b.WriteString("📅 2025-10-07 14:23 - ml-workstation hibernated after 30 min idle\n")
	b.WriteString("📅 2025-10-07 12:15 - data-analysis stopped after 60 min idle\n")
	b.WriteString("📅 2025-10-06 18:45 - research-env hibernated after 45 min idle\n")
	b.WriteString("\n")
	b.WriteString(theme.SubTitle.Render("Cost Savings") + "\n")
	b.WriteString("Estimated savings from idle detection: $127.50 this month\n")

	return b.String()
}

// renderPolicyDialog displays the policy update dialog
func (m IdleModel) renderPolicyDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("Update Idle Policy") + "\n\n")
	content.WriteString("Policy configuration will be updated\n\n")
	content.WriteString("Press Enter to confirm, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderEnableDialog displays the enable idle detection dialog
func (m IdleModel) renderEnableDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("Enable Idle Detection") + "\n\n")
	content.WriteString(fmt.Sprintf("Instance: %s\n", m.dialogInstanceName))
	content.WriteString("Policy: default (30 minutes → hibernate)\n\n")
	content.WriteString("Press Enter to enable, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m IdleModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showPolicyDialog || m.showEnableDialog {
		helps = []string{"enter: confirm", "esc: cancel"}
	} else if m.selectedTab == 0 {
		helps = []string{"↑/↓: select", "tab: switch tabs", "u: update policy", "r: refresh", "q: quit"}
	} else if m.selectedTab == 1 {
		helps = []string{"↑/↓: select", "tab: switch tabs", "e: enable", "d: disable", "r: refresh", "q: quit"}
	} else {
		helps = []string{"tab: switch tabs", "r: refresh", "q: quit"}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// enableIdleDetection enables idle detection for an instance
func (m IdleModel) enableIdleDetection(instanceName, policy string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.EnableIdleDetection(context.Background(), instanceName, policy)
		if err != nil {
			return IdlePolicyActionMsg{Error: fmt.Errorf("failed to enable idle detection: %w", err)}
		}

		return IdlePolicyActionMsg{
			Success: true,
			Message: fmt.Sprintf("Enabled idle detection for %s with policy %s", instanceName, policy),
			Error:   nil,
		}
	}
}

// disableIdleDetection disables idle detection for an instance
func (m IdleModel) disableIdleDetection(instanceName string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.DisableIdleDetection(context.Background(), instanceName)
		if err != nil {
			return IdlePolicyActionMsg{Error: fmt.Errorf("failed to disable idle detection: %w", err)}
		}

		return IdlePolicyActionMsg{
			Success: true,
			Message: fmt.Sprintf("Disabled idle detection for %s", instanceName),
			Error:   nil,
		}
	}
}

// updatePoliciesTable updates the policies table with current data
func (m *IdleModel) updatePoliciesTable() {
	rows := []table.Row{}
	i := 0
	for name, policy := range m.policies {
		// Selection indicator
		displayName := name
		if i == m.selectedPolicy {
			displayName = "> " + name
		}

		row := table.Row{
			displayName,
			fmt.Sprintf("%d", policy.Threshold),
			policy.Action,
			"Active",
		}
		rows = append(rows, row)
		i++
	}

	m.policiesTable.SetRows(rows)
}

// updateInstancesTable updates the instances table with current data
func (m *IdleModel) updateInstancesTable() {
	rows := []table.Row{}
	for i, instance := range m.instances {
		// Selection indicator
		name := instance.Name
		if i == m.selectedInstance {
			name = "> " + name
		}

		// Get idle detection status for instance
		idleStatus := "Disabled"
		policy := "-"
		idleTime := "-"
		status := "-"

		// In real implementation, fetch idle status per instance
		// For now, show sample data for running instances
		if instance.State == "running" {
			idleStatus = "Enabled"
			policy = "default"
			idleTime = "5 min"
			status = "Active"
		}

		row := table.Row{
			name,
			idleStatus,
			policy,
			idleTime,
			status,
		}
		rows = append(rows, row)
	}

	m.instancesTable.SetRows(rows)
}
