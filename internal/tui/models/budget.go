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

// BudgetModel represents the budget management view
type BudgetModel struct {
	apiClient         apiClient
	budgetsTable      components.Table
	statusBar         components.StatusBar
	spinner           components.Spinner
	width             int
	height            int
	loading           bool
	error             string
	projects          []api.ProjectResponse
	selectedTab       int // 0=list, 1=breakdown, 2=forecast, 3=savings
	selectedBudget    int
	showCreateDialog  bool
	createProjectName string
	createAmount      string
}

// BudgetDataMsg represents budget data retrieved from the API
type BudgetDataMsg struct {
	Projects []api.ProjectResponse
	Error    error
}

// NewBudgetModel creates a new budget model
func NewBudgetModel(apiClient apiClient) BudgetModel {
	// Create budgets table
	columns := []table.Column{
		{Title: "PROJECT", Width: 20},
		{Title: "BUDGET", Width: 12},
		{Title: "SPENT", Width: 12},
		{Title: "REMAINING", Width: 12},
		{Title: "%USED", Width: 8},
		{Title: "STATUS", Width: 10},
		{Title: "ALERTS", Width: 8},
	}

	budgetsTable := components.NewTable(columns, []table.Row{}, 80, 10, true)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Budget Management", "")
	spinner := components.NewSpinner("Loading budgets...")

	return BudgetModel{
		apiClient:      apiClient,
		budgetsTable:   budgetsTable,
		statusBar:      statusBar,
		spinner:        spinner,
		width:          80,
		height:         24,
		loading:        true,
		selectedTab:    0,
		selectedBudget: 0,
	}
}

// Init initializes the model
func (m BudgetModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchBudgets,
	)
}

// fetchBudgets retrieves budget data from the API
func (m BudgetModel) fetchBudgets() tea.Msg {
	// Fetch all projects with budget information
	resp, err := m.apiClient.ListProjects(context.Background(), nil)
	if err != nil {
		return BudgetDataMsg{Error: fmt.Errorf("failed to list projects: %w", err)}
	}

	return BudgetDataMsg{
		Projects: resp.Projects,
		Error:    nil,
	}
}

// Update handles messages and updates the model
func (m BudgetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.budgetsTable.SetSize(msg.Width-4, msg.Height-12)
		return m, nil

	case BudgetDataMsg:
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.loading = false
			return m, nil
		}

		m.projects = msg.Projects
		m.loading = false
		m.error = ""

		// Update table with budget data
		m.updateBudgetsTable()
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r", "f5":
			// Refresh budgets
			m.loading = true
			return m, m.fetchBudgets

		case "tab":
			// Switch between tabs
			m.selectedTab = (m.selectedTab + 1) % 4
			return m, nil

		case "n":
			// Create new budget (show dialog)
			if m.selectedTab == 0 {
				m.showCreateDialog = true
				return m, nil
			}

		case "esc":
			// Close dialog
			if m.showCreateDialog {
				m.showCreateDialog = false
				m.createProjectName = ""
				m.createAmount = ""
				return m, nil
			}

		case "enter":
			// Handle budget selection or dialog submission
			if m.showCreateDialog {
				// Submit create budget dialog
				return m, m.createBudget
			}

		case "up", "k":
			if m.selectedBudget > 0 {
				m.selectedBudget--
			}
			return m, nil

		case "down", "j":
			if m.selectedBudget < len(m.projects)-1 {
				m.selectedBudget++
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
func (m BudgetModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("💰 Budget Management")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Tab bar
	tabs := []string{"Overview", "Breakdown", "Forecast", "Savings"}
	tabBar := renderTabBar(tabs, m.selectedTab, theme)
	b.WriteString(tabBar)
	b.WriteString("\n\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0: // Overview
		b.WriteString(m.renderOverview())
	case 1: // Breakdown
		b.WriteString(m.renderBreakdown())
	case 2: // Forecast
		b.WriteString(m.renderForecast())
	case 3: // Savings
		b.WriteString(m.renderSavings())
	}

	// Show create dialog if active
	if m.showCreateDialog {
		dialog := m.renderCreateDialog()
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

// renderOverview displays the budget overview list
func (m BudgetModel) renderOverview() string {
	if len(m.projects) == 0 {
		return "No projects with budgets found.\n\nPress 'n' to create a budget for a project."
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Summary statistics
	totalBudget := 0.0
	totalSpent := 0.0
	budgetCount := 0

	for _, proj := range m.projects {
		if proj.BudgetStatus != nil && proj.BudgetStatus.TotalBudget > 0 {
			budgetCount++
			totalBudget += proj.BudgetStatus.TotalBudget
			totalSpent += proj.BudgetStatus.SpentAmount
		}
	}

	if budgetCount > 0 {
		spentPercent := (totalSpent / totalBudget) * 100
		summary := fmt.Sprintf("Active Budgets: %d | Total Budget: $%.2f | Total Spent: $%.2f (%.1f%%) | Remaining: $%.2f",
			budgetCount, totalBudget, totalSpent, spentPercent, totalBudget-totalSpent)
		b.WriteString(theme.SubTitle.Render(summary))
		b.WriteString("\n\n")
	}

	// Budget table
	rows := []table.Row{}
	for i, proj := range m.projects {
		if proj.BudgetStatus == nil || proj.BudgetStatus.TotalBudget <= 0 {
			// No budget configured
			row := table.Row{
				proj.Name,
				"-",
				fmt.Sprintf("$%.2f", proj.TotalCost),
				"-",
				"-",
				"No Budget",
				"-",
			}
			rows = append(rows, row)
			continue
		}

		budget := proj.BudgetStatus
		remaining := budget.TotalBudget - budget.SpentAmount
		if remaining < 0 {
			remaining = 0
		}
		usedPercent := (budget.SpentAmount / budget.TotalBudget) * 100

		// Status indicator
		status := "OK"
		if usedPercent >= 95 {
			status = "CRITICAL"
		} else if usedPercent >= 80 {
			status = "WARNING"
		}

		// Alert count
		alertStatus := "-"
		if len(budget.ActiveAlerts) > 0 {
			alertStatus = fmt.Sprintf("%d", len(budget.ActiveAlerts))
		}

		// Selection indicator
		projectName := proj.Name
		if i == m.selectedBudget {
			projectName = "> " + projectName
		}

		row := table.Row{
			projectName,
			fmt.Sprintf("$%.2f", budget.TotalBudget),
			fmt.Sprintf("$%.2f", budget.SpentAmount),
			fmt.Sprintf("$%.2f", remaining),
			fmt.Sprintf("%.1f%%", usedPercent),
			status,
			alertStatus,
		}
		rows = append(rows, row)
	}

	// Update table rows
	m.budgetsTable.SetRows(rows)
	b.WriteString(m.budgetsTable.View())

	return b.String()
}

// renderBreakdown displays cost breakdown by service/instance
func (m BudgetModel) renderBreakdown() string {
	if m.selectedBudget >= len(m.projects) {
		return "Select a project to view cost breakdown."
	}

	project := m.projects[m.selectedBudget]
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Cost Breakdown for '%s'\n\n", project.Name))

	if project.BudgetStatus == nil || project.BudgetStatus.TotalBudget <= 0 {
		b.WriteString("No budget configured for this project.\n")
		return b.String()
	}

	// Display cost breakdown
	b.WriteString(fmt.Sprintf("Total Spent: $%.2f\n", project.BudgetStatus.SpentAmount))
	b.WriteString(fmt.Sprintf("Total Budget: $%.2f\n\n", project.BudgetStatus.TotalBudget))

	// Design Decision: TUI shows budget summary; detailed cost breakdown requires CLI
	// Rationale: Cost breakdown API returns detailed data better suited for CLI table formatting
	// Future Enhancement: Add simple breakdown chart if TUI space permits
	b.WriteString("Cost breakdown by service:\n")
	b.WriteString("  EC2 Compute: (see CLI for details)\n")
	b.WriteString("  EBS Storage: (see CLI for details)\n")
	b.WriteString("  EFS Storage: (see CLI for details)\n")
	b.WriteString("  Data Transfer: (see CLI for details)\n\n")

	b.WriteString("💡 Detailed breakdown available via: cws budget breakdown " + project.Name + "\n")

	return b.String()
}

// renderForecast displays spending forecast
func (m BudgetModel) renderForecast() string {
	if m.selectedBudget >= len(m.projects) {
		return "Select a project to view spending forecast."
	}

	project := m.projects[m.selectedBudget]
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Spending Forecast for '%s'\n\n", project.Name))

	if project.BudgetStatus == nil || project.BudgetStatus.TotalBudget <= 0 {
		b.WriteString("No budget configured for this project.\n")
		return b.String()
	}

	budget := project.BudgetStatus

	b.WriteString(fmt.Sprintf("Current Spending: $%.2f (%.1f%%)\n", budget.SpentAmount, (budget.SpentAmount/budget.TotalBudget)*100))

	if budget.ProjectedMonthlySpend > 0 {
		b.WriteString(fmt.Sprintf("Projected Monthly: $%.2f\n", budget.ProjectedMonthlySpend))

		if budget.DaysUntilBudgetExhausted != nil {
			days := *budget.DaysUntilBudgetExhausted
			if days > 0 {
				b.WriteString(fmt.Sprintf("Budget Exhaustion: %d days\n", days))
			}
		}
	}

	b.WriteString("\n💡 Detailed forecasting available via: cws budget forecast " + project.Name + "\n")

	return b.String()
}

// renderSavings displays hibernation and cost optimization savings
func (m BudgetModel) renderSavings() string {
	var b strings.Builder

	b.WriteString("Cost Savings Analysis\n\n")

	// Aggregate savings across all projects
	totalSavings := 0.0

	b.WriteString("Hibernation Savings:\n")
	for _, proj := range m.projects {
		if proj.BudgetStatus != nil {
			// Design Decision: Hibernation savings API not yet implemented
			// Rationale: Savings calculation requires historical data analysis
			// Future Enhancement: Add savings tracking to budget status API
			b.WriteString(fmt.Sprintf("  %s: (see CLI for details)\n", proj.Name))
		}
	}

	b.WriteString(fmt.Sprintf("\nTotal Savings: $%.2f\n", totalSavings))
	b.WriteString("\n💡 Detailed savings analysis available via: cws budget savings\n")

	return b.String()
}

// renderCreateDialog displays the budget creation dialog
func (m BudgetModel) renderCreateDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("Create Budget") + "\n\n")
	content.WriteString("Project Name: " + m.createProjectName + "\n")
	content.WriteString("Budget Amount: $" + m.createAmount + "\n\n")
	content.WriteString("Press Enter to create, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m BudgetModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showCreateDialog {
		helps = []string{"esc: cancel", "enter: create"}
	} else {
		helps = []string{
			"tab: switch tabs",
			"↑/↓: select",
			"n: new budget",
			"r: refresh",
			"q: quit",
		}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// createBudget creates a new budget via the API
func (m BudgetModel) createBudget() tea.Msg {
	// Design Decision: Budget creation requires CLI for comprehensive configuration
	// Rationale: Budgets have many optional parameters (alerts, actions, limits, etc.)
	// TUI form input would be complex; CLI provides better UX for advanced configuration
	// Use CLI command: cws budget create <project> <amount> [--alert ...] [--action ...]
	return BudgetDataMsg{Error: fmt.Errorf("budget creation via TUI not implemented - use CLI: cws budget create <project> <amount>")}
}

// renderTabBar renders a tab bar for navigation
func renderTabBar(tabs []string, selected int, theme styles.Theme) string {
	var b strings.Builder

	for i, tab := range tabs {
		if i == selected {
			b.WriteString(theme.Tab.Active.Render("[" + tab + "]"))
		} else {
			b.WriteString(theme.Tab.Inactive.Render(" " + tab + " "))
		}
		if i < len(tabs)-1 {
			b.WriteString(" ")
		}
	}

	return b.String()
}

// updateBudgetsTable updates the budgets table with current data
func (m *BudgetModel) updateBudgetsTable() {
	// This method updates the table rows with current project/budget data
	// The actual update happens in renderOverview()
}
