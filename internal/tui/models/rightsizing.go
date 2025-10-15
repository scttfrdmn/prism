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

// RightsizingModel represents the rightsizing recommendations view
type RightsizingModel struct {
	apiClient              apiClient
	recommendationsTable   components.Table
	instancesTable         components.Table
	statusBar              components.StatusBar
	spinner                components.Spinner
	width                  int
	height                 int
	loading                bool
	error                  string
	recommendations        []api.RightsizingRecommendation
	instances              []api.InstanceResponse
	selectedTab            int // 0=recommendations, 1=instances, 2=savings
	selectedRecommendation int
	selectedInstance       int
	showDetailView         bool
	showApplyDialog        bool
	dialogRecommendationID string
}

// RightsizingDataMsg represents rightsizing data retrieved from the API
type RightsizingDataMsg struct {
	Recommendations []api.RightsizingRecommendation
	Instances       []api.InstanceResponse
	Error           error
}

// RightsizingActionMsg represents rightsizing action result
type RightsizingActionMsg struct {
	Success bool
	Message string
	Error   error
}

// NewRightsizingModel creates a new rightsizing model
func NewRightsizingModel(apiClient apiClient) RightsizingModel {
	// Create recommendations table
	recommendationColumns := []table.Column{
		{Title: "INSTANCE", Width: 25},
		{Title: "CURRENT", Width: 15},
		{Title: "RECOMMENDED", Width: 15},
		{Title: "SAVINGS", Width: 12},
		{Title: "STATUS", Width: 15},
	}

	recommendationsTable := components.NewTable(recommendationColumns, []table.Row{}, 80, 8, true)

	// Create instances table
	instanceColumns := []table.Column{
		{Title: "INSTANCE", Width: 25},
		{Title: "TYPE", Width: 15},
		{Title: "CPU UTIL", Width: 12},
		{Title: "MEM UTIL", Width: 12},
		{Title: "STATUS", Width: 15},
	}

	instancesTable := components.NewTable(instanceColumns, []table.Row{}, 80, 8, true)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Rightsizing", "")
	spinner := components.NewSpinner("Loading rightsizing data...")

	return RightsizingModel{
		apiClient:            apiClient,
		recommendationsTable: recommendationsTable,
		instancesTable:       instancesTable,
		statusBar:            statusBar,
		spinner:              spinner,
		width:                80,
		height:               24,
		loading:              true,
		selectedTab:          0,
	}
}

// Init initializes the model
func (m RightsizingModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchRightsizingData,
	)
}

// fetchRightsizingData retrieves rightsizing data from the API
func (m RightsizingModel) fetchRightsizingData() tea.Msg {
	// Fetch recommendations
	recommendationsResp, err := m.apiClient.GetRightsizingRecommendations(context.Background())
	if err != nil {
		return RightsizingDataMsg{Error: fmt.Errorf("failed to get recommendations: %w", err)}
	}

	// Fetch instances for utilization data
	instancesResp, err := m.apiClient.ListInstances(context.Background())
	if err != nil {
		return RightsizingDataMsg{Error: fmt.Errorf("failed to list instances: %w", err)}
	}

	return RightsizingDataMsg{
		Recommendations: recommendationsResp.Recommendations,
		Instances:       instancesResp.Instances,
		Error:           nil,
	}
}

// Update handles messages and updates the model
func (m RightsizingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recommendationsTable.SetSize(msg.Width-4, msg.Height-18)
		m.instancesTable.SetSize(msg.Width-4, msg.Height-18)
		return m, nil

	case RightsizingDataMsg:
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.loading = false
			return m, nil
		}

		m.recommendations = msg.Recommendations
		m.instances = msg.Instances
		m.loading = false
		m.error = ""

		// Update tables with data
		m.updateRecommendationsTable()
		m.updateInstancesTable()
		return m, nil

	case RightsizingActionMsg:
		m.showApplyDialog = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			return m, nil
		}
		// Refresh data after action
		return m, m.fetchRightsizingData

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r", "f5":
			// Refresh rightsizing data
			m.loading = true
			return m, m.fetchRightsizingData

		case "tab":
			// Cycle through tabs
			m.selectedTab = (m.selectedTab + 1) % 3
			return m, nil

		case "v":
			// Toggle detail view
			if m.selectedTab == 0 && len(m.recommendations) > 0 {
				m.showDetailView = !m.showDetailView
				return m, nil
			}

		case "a":
			// Apply recommendation
			if m.selectedTab == 0 && m.selectedRecommendation < len(m.recommendations) {
				rec := m.recommendations[m.selectedRecommendation]
				m.dialogRecommendationID = rec.InstanceName
				m.showApplyDialog = true
				return m, nil
			}

		case "enter":
			// Handle dialog confirmation
			if m.showApplyDialog {
				// Apply recommendation
				return m, m.applyRecommendation(m.dialogRecommendationID)
			}

		case "esc":
			// Close dialogs or detail view
			if m.showApplyDialog {
				m.showApplyDialog = false
				return m, nil
			}
			if m.showDetailView {
				m.showDetailView = false
				return m, nil
			}

		case "up", "k":
			if m.selectedTab == 0 && m.selectedRecommendation > 0 {
				m.selectedRecommendation--
			} else if m.selectedTab == 1 && m.selectedInstance > 0 {
				m.selectedInstance--
			}
			return m, nil

		case "down", "j":
			if m.selectedTab == 0 && m.selectedRecommendation < len(m.recommendations)-1 {
				m.selectedRecommendation++
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
func (m RightsizingModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("📊 Rightsizing & Optimization")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0:
		b.WriteString(m.renderRecommendations())
	case 1:
		b.WriteString(m.renderInstances())
	case 2:
		b.WriteString(m.renderSavings())
	}

	// Show apply dialog if active
	if m.showApplyDialog {
		dialog := m.renderApplyDialog()
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
func (m RightsizingModel) renderTabs() string {
	theme := styles.CurrentTheme
	tabs := []string{"Recommendations", "Instances", "Savings"}

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

// renderRecommendations displays the recommendations view
func (m RightsizingModel) renderRecommendations() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Rightsizing Recommendations"))
	b.WriteString("\n\n")

	if len(m.recommendations) == 0 {
		b.WriteString("No recommendations available. All instances are optimally sized.\n")
		return b.String()
	}

	// Recommendations table
	b.WriteString(m.recommendationsTable.View())

	// Detail view or summary
	if m.showDetailView && m.selectedRecommendation < len(m.recommendations) {
		rec := m.recommendations[m.selectedRecommendation]
		b.WriteString("\n\n")
		b.WriteString(m.renderRecommendationDetails(rec))
	} else if m.selectedRecommendation < len(m.recommendations) {
		rec := m.recommendations[m.selectedRecommendation]
		b.WriteString("\n\n")
		b.WriteString(theme.SubTitle.Render("Quick Summary") + "\n\n")
		b.WriteString(fmt.Sprintf("Instance: %s\n", rec.InstanceName))
		b.WriteString(fmt.Sprintf("Recommendation: %s → %s\n", rec.CurrentType, rec.RecommendedType))
		b.WriteString(fmt.Sprintf("Monthly Savings: $%.2f\n", rec.MonthlySavings))
		b.WriteString(fmt.Sprintf("Confidence: %s\n", rec.Confidence))
		b.WriteString("\nPress 'v' for detailed analysis, 'a' to apply recommendation\n")
	}

	return b.String()
}

// renderRecommendationDetails displays detailed recommendation analysis
func (m RightsizingModel) renderRecommendationDetails(rec api.RightsizingRecommendation) string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SubTitle.Render("Detailed Analysis") + "\n\n")

	// Current vs Recommended comparison
	b.WriteString("📋 Instance Comparison\n")
	b.WriteString(fmt.Sprintf("  Current:     %s\n", rec.CurrentType))
	b.WriteString(fmt.Sprintf("  Recommended: %s\n", rec.RecommendedType))
	b.WriteString("\n")

	// Resource utilization
	b.WriteString("📊 Resource Utilization (30-day average)\n")
	b.WriteString(fmt.Sprintf("  CPU:    %.1f%% (current) → %.1f%% (recommended)\n", rec.CPUUtilization, rec.CPUUtilization*1.3))
	b.WriteString(fmt.Sprintf("  Memory: %.1f%% (current) → %.1f%% (recommended)\n", rec.MemoryUtilization, rec.MemoryUtilization*1.3))
	b.WriteString("\n")

	// Cost impact
	b.WriteString("💰 Cost Impact\n")
	b.WriteString(fmt.Sprintf("  Current Cost:    $%.2f/month\n", rec.CurrentCost))
	b.WriteString(fmt.Sprintf("  Recommended Cost: $%.2f/month\n", rec.RecommendedCost))
	savingsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	b.WriteString(savingsStyle.Render(fmt.Sprintf("  Monthly Savings:  $%.2f (%.1f%%)\n", rec.MonthlySavings, rec.SavingsPercentage)))
	b.WriteString("\n")

	// Recommendation confidence
	b.WriteString(fmt.Sprintf("🎯 Confidence: %s\n", rec.Confidence))
	if rec.Reason != "" {
		b.WriteString(fmt.Sprintf("   Reason: %s\n", rec.Reason))
	}

	b.WriteString("\nPress 'v' to close, 'a' to apply recommendation\n")

	return b.String()
}

// renderInstances displays the instances utilization view
func (m RightsizingModel) renderInstances() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Instance Utilization Analysis"))
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
		b.WriteString(theme.SubTitle.Render("Instance Details") + "\n\n")
		b.WriteString(fmt.Sprintf("Name: %s\n", instance.Name))
		b.WriteString(fmt.Sprintf("State: %s\n", instance.State))
		b.WriteString(fmt.Sprintf("Template: %s\n", instance.Template))
		b.WriteString(fmt.Sprintf("Hourly Rate: $%.2f\n", instance.HourlyRate))
	}

	return b.String()
}

// renderSavings displays the savings summary view
func (m RightsizingModel) renderSavings() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Cost Optimization Summary"))
	b.WriteString("\n\n")

	// Calculate total potential savings
	totalSavings := 0.0
	totalCurrent := 0.0
	for _, rec := range m.recommendations {
		totalSavings += rec.MonthlySavings
		totalCurrent += rec.CurrentCost
	}

	// Savings summary
	b.WriteString("💰 Potential Monthly Savings\n\n")
	savingsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	b.WriteString(savingsStyle.Render(fmt.Sprintf("  Total Savings: $%.2f/month\n", totalSavings)))
	b.WriteString(fmt.Sprintf("  Current Spend: $%.2f/month\n", totalCurrent))
	if totalCurrent > 0 {
		savingsPercent := (totalSavings / totalCurrent) * 100
		b.WriteString(fmt.Sprintf("  Reduction:     %.1f%%\n", savingsPercent))
	}
	b.WriteString("\n")

	// Breakdown by recommendation
	if len(m.recommendations) > 0 {
		b.WriteString("📊 Savings Breakdown\n\n")
		for _, rec := range m.recommendations {
			b.WriteString(fmt.Sprintf("  %s: $%.2f/month\n", rec.InstanceName, rec.MonthlySavings))
		}
	}

	b.WriteString("\n")
	b.WriteString(theme.SubTitle.Render("Optimization Strategy") + "\n\n")
	b.WriteString("CloudWorkstation analyzes 30-day utilization patterns to identify:\n")
	b.WriteString("• Over-provisioned instances (low CPU/memory usage)\n")
	b.WriteString("• Under-utilized resources (inefficient workload placement)\n")
	b.WriteString("• Cost-effective alternatives (ARM instances, spot instances)\n")
	b.WriteString("\nApply recommendations from the Recommendations tab to optimize costs.\n")

	return b.String()
}

// renderApplyDialog displays the apply recommendation confirmation dialog
func (m RightsizingModel) renderApplyDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("Apply Recommendation") + "\n\n")

	if m.selectedRecommendation < len(m.recommendations) {
		rec := m.recommendations[m.selectedRecommendation]
		content.WriteString(fmt.Sprintf("Instance: %s\n", rec.InstanceName))
		content.WriteString(fmt.Sprintf("Change: %s → %s\n", rec.CurrentType, rec.RecommendedType))
		content.WriteString(fmt.Sprintf("Savings: $%.2f/month\n\n", rec.MonthlySavings))
	}

	content.WriteString("This will stop and resize the instance.\n")
	content.WriteString("Data will be preserved.\n\n")
	content.WriteString("Press Enter to apply, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m RightsizingModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showApplyDialog {
		helps = []string{"enter: apply", "esc: cancel"}
	} else if m.selectedTab == 0 {
		helps = []string{"↑/↓: select", "tab: switch tabs", "v: details", "a: apply", "r: refresh", "q: quit"}
	} else if m.selectedTab == 1 {
		helps = []string{"↑/↓: select", "tab: switch tabs", "r: refresh", "q: quit"}
	} else {
		helps = []string{"tab: switch tabs", "r: refresh", "q: quit"}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// applyRecommendation applies a rightsizing recommendation
func (m RightsizingModel) applyRecommendation(instanceName string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.ApplyRightsizingRecommendation(context.Background(), instanceName)
		if err != nil {
			return RightsizingActionMsg{Error: fmt.Errorf("failed to apply recommendation: %w", err)}
		}

		return RightsizingActionMsg{
			Success: true,
			Message: fmt.Sprintf("Successfully applied recommendation for %s", instanceName),
			Error:   nil,
		}
	}
}

// updateRecommendationsTable updates the recommendations table with current data
func (m *RightsizingModel) updateRecommendationsTable() {
	rows := []table.Row{}
	for i, rec := range m.recommendations {
		// Selection indicator
		name := rec.InstanceName
		if i == m.selectedRecommendation {
			name = "> " + name
		}

		// Format savings
		savings := fmt.Sprintf("$%.2f", rec.MonthlySavings)

		row := table.Row{
			name,
			rec.CurrentType,
			rec.RecommendedType,
			savings,
			rec.Confidence,
		}
		rows = append(rows, row)
	}

	m.recommendationsTable.SetRows(rows)
}

// updateInstancesTable updates the instances table with current data
func (m *RightsizingModel) updateInstancesTable() {
	rows := []table.Row{}
	for i, instance := range m.instances {
		// Selection indicator
		name := instance.Name
		if i == m.selectedInstance {
			name = "> " + name
		}

		// Find utilization data from recommendations if available
		cpuUtil := "N/A"
		memUtil := "N/A"
		instanceType := "-"
		for _, rec := range m.recommendations {
			if rec.InstanceName == instance.Name {
				cpuUtil = fmt.Sprintf("%.1f%%", rec.CPUUtilization)
				memUtil = fmt.Sprintf("%.1f%%", rec.MemoryUtilization)
				instanceType = rec.CurrentType
				break
			}
		}

		row := table.Row{
			name,
			instanceType,
			cpuUtil,
			memUtil,
			instance.State,
		}
		rows = append(rows, row)
	}

	m.instancesTable.SetRows(rows)
}
