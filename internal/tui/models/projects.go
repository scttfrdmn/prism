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

// ProjectsModel represents the project management view
type ProjectsModel struct {
	apiClient         apiClient
	projectsTable     components.Table
	statusBar         components.StatusBar
	spinner           components.Spinner
	width             int
	height            int
	loading           bool
	error             string
	projects          []api.ProjectResponse
	selectedProject   int
	selectedTab       int // 0=list, 1=members, 2=instances, 3=budget
	showCreateDialog  bool
	createName        string
	createDescription string
}

// ProjectDataMsg represents project data retrieved from the API
type ProjectDataMsg struct {
	Projects []api.ProjectResponse
	Error    error
}

// NewProjectsModel creates a new projects model
func NewProjectsModel(apiClient apiClient) ProjectsModel {
	// Create projects table
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "OWNER", Width: 15},
		{Title: "STATUS", Width: 10},
		{Title: "MEMBERS", Width: 8},
		{Title: "INSTANCES", Width: 10},
		{Title: "COST", Width: 12},
		{Title: "BUDGET", Width: 12},
	}

	projectsTable := components.NewTable(columns, []table.Row{}, 80, 10, true)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Project Management", "")
	spinner := components.NewSpinner("Loading projects...")

	return ProjectsModel{
		apiClient:       apiClient,
		projectsTable:   projectsTable,
		statusBar:       statusBar,
		spinner:         spinner,
		width:           80,
		height:          24,
		loading:         true,
		selectedTab:     0,
		selectedProject: 0,
	}
}

// Init initializes the model
func (m ProjectsModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchProjects,
	)
}

// fetchProjects retrieves project data from the API
func (m ProjectsModel) fetchProjects() tea.Msg {
	resp, err := m.apiClient.ListProjects(context.Background(), nil)
	if err != nil {
		return ProjectDataMsg{Error: fmt.Errorf("failed to list projects: %w", err)}
	}

	return ProjectDataMsg{
		Projects: resp.Projects,
		Error:    nil,
	}
}

// Update handles messages and updates the model
func (m ProjectsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.projectsTable.SetSize(msg.Width-4, msg.Height-12)
		return m, nil

	case ProjectDataMsg:
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.loading = false
			return m, nil
		}

		m.projects = msg.Projects
		m.loading = false
		m.error = ""

		// Update table with project data
		m.updateProjectsTable()
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r", "f5":
			// Refresh projects
			m.loading = true
			return m, m.fetchProjects

		case "tab":
			// Switch between tabs
			m.selectedTab = (m.selectedTab + 1) % 4
			return m, nil

		case "n":
			// Create new project (show dialog)
			if m.selectedTab == 0 {
				m.showCreateDialog = true
				return m, nil
			}

		case "esc":
			// Close dialog
			if m.showCreateDialog {
				m.showCreateDialog = false
				m.createName = ""
				m.createDescription = ""
				return m, nil
			}

		case "enter":
			// Handle project selection or dialog submission
			if m.showCreateDialog {
				// Submit create project dialog
				return m, m.createProject
			}

		case "up", "k":
			if m.selectedProject > 0 {
				m.selectedProject--
			}
			return m, nil

		case "down", "j":
			if m.selectedProject < len(m.projects)-1 {
				m.selectedProject++
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
func (m ProjectsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("📁 Project Management")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Tab bar
	tabs := []string{"Overview", "Members", "Instances", "Budget"}
	tabBar := renderProjectTabBar(tabs, m.selectedTab, theme)
	b.WriteString(tabBar)
	b.WriteString("\n\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0: // Overview
		b.WriteString(m.renderOverview())
	case 1: // Members
		b.WriteString(m.renderMembers())
	case 2: // Instances
		b.WriteString(m.renderInstances())
	case 3: // Budget
		b.WriteString(m.renderBudget())
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

// renderOverview displays the project overview list
func (m ProjectsModel) renderOverview() string {
	if len(m.projects) == 0 {
		return "No projects found.\n\nPress 'n' to create a new project."
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Summary statistics
	activeCount := 0
	totalCost := 0.0
	totalBudget := 0.0

	for _, proj := range m.projects {
		if proj.Status == "active" {
			activeCount++
		}
		totalCost += proj.TotalCost
		if proj.BudgetStatus != nil {
			totalBudget += proj.BudgetStatus.TotalBudget
		}
	}

	summary := fmt.Sprintf("Total Projects: %d | Active: %d | Total Cost: $%.2f | Total Budget: $%.2f",
		len(m.projects), activeCount, totalCost, totalBudget)
	b.WriteString(theme.SubTitle.Render(summary))
	b.WriteString("\n\n")

	// Projects table
	rows := []table.Row{}
	for i, proj := range m.projects {
		// Budget status
		budgetStr := "-"
		if proj.BudgetStatus != nil {
			budgetStr = fmt.Sprintf("$%.2f", proj.BudgetStatus.TotalBudget)
		}

		// Selection indicator
		projectName := proj.Name
		if i == m.selectedProject {
			projectName = "> " + projectName
		}

		row := table.Row{
			projectName,
			proj.Owner,
			proj.Status,
			fmt.Sprintf("%d", proj.MemberCount),
			fmt.Sprintf("%d", proj.ActiveInstances),
			fmt.Sprintf("$%.2f", proj.TotalCost),
			budgetStr,
		}
		rows = append(rows, row)
	}

	// Update table rows
	m.projectsTable.SetRows(rows)
	b.WriteString(m.projectsTable.View())

	return b.String()
}

// renderMembers displays project members
func (m ProjectsModel) renderMembers() string {
	if m.selectedProject >= len(m.projects) {
		return "Select a project to view members."
	}

	project := m.projects[m.selectedProject]
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Members for project '%s'\n\n", project.Name))
	b.WriteString(fmt.Sprintf("Owner: %s\n", project.Owner))
	b.WriteString(fmt.Sprintf("Total Members: %d\n\n", project.MemberCount))

	// Design Decision: TUI shows member summary; detailed member list requires CLI
	// Rationale: Member list can be long; CLI provides better formatting for detailed operations
	// Future Enhancement: Add paginated member list view if TUI space permits
	b.WriteString("Member management:\n")
	b.WriteString("  • Add members: cws project members " + project.Name + " add <email> <role>\n")
	b.WriteString("  • Remove members: cws project members " + project.Name + " remove <email>\n")
	b.WriteString("  • List members: cws project members " + project.Name + " list\n\n")

	b.WriteString("💡 Detailed member management available via CLI commands\n")

	return b.String()
}

// renderInstances displays project instances
func (m ProjectsModel) renderInstances() string {
	if m.selectedProject >= len(m.projects) {
		return "Select a project to view instances."
	}

	project := m.projects[m.selectedProject]
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Instances for project '%s'\n\n", project.Name))
	b.WriteString(fmt.Sprintf("Active Instances: %d\n", project.ActiveInstances))
	b.WriteString(fmt.Sprintf("Total Cost: $%.2f\n\n", project.TotalCost))

	// Design Decision: TUI shows instance summary; detailed instance list requires CLI/Instance view
	// Rationale: Instance details are available in main Instances view (tab 3)
	// Project-filtered instance list would duplicate existing TUI functionality
	b.WriteString("💡 View project instances: cws project instances " + project.Name + "\n")

	return b.String()
}

// renderBudget displays project budget information
func (m ProjectsModel) renderBudget() string {
	if m.selectedProject >= len(m.projects) {
		return "Select a project to view budget."
	}

	project := m.projects[m.selectedProject]
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Budget for project '%s'\n\n", project.Name))

	if project.BudgetStatus == nil || project.BudgetStatus.TotalBudget <= 0 {
		b.WriteString("No budget configured for this project.\n\n")
		b.WriteString("💡 Set budget: cws project budget set " + project.Name + " <amount>\n")
		return b.String()
	}

	budget := project.BudgetStatus
	remaining := budget.TotalBudget - budget.SpentAmount
	if remaining < 0 {
		remaining = 0
	}

	b.WriteString(fmt.Sprintf("Total Budget: $%.2f\n", budget.TotalBudget))
	b.WriteString(fmt.Sprintf("Spent: $%.2f (%.1f%%)\n", budget.SpentAmount, budget.SpentPercentage))
	b.WriteString(fmt.Sprintf("Remaining: $%.2f\n\n", remaining))

	// Alert status
	if len(budget.ActiveAlerts) > 0 {
		b.WriteString(fmt.Sprintf("⚠️  Active Alerts: %d\n", len(budget.ActiveAlerts)))
		for _, alert := range budget.ActiveAlerts {
			b.WriteString(fmt.Sprintf("  • %s\n", alert))
		}
		b.WriteString("\n")
	}

	// Projected spending
	if budget.ProjectedMonthlySpend > 0 {
		b.WriteString(fmt.Sprintf("Projected Monthly: $%.2f\n", budget.ProjectedMonthlySpend))
	}

	b.WriteString("\n💡 Budget management: cws project budget status " + project.Name + "\n")

	return b.String()
}

// renderCreateDialog displays the project creation dialog
func (m ProjectsModel) renderCreateDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("Create Project") + "\n\n")
	content.WriteString("Project Name: " + m.createName + "\n")
	content.WriteString("Description: " + m.createDescription + "\n\n")
	content.WriteString("Press Enter to create, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m ProjectsModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showCreateDialog {
		helps = []string{"esc: cancel", "enter: create"}
	} else {
		helps = []string{
			"tab: switch tabs",
			"↑/↓: select",
			"n: new project",
			"r: refresh",
			"q: quit",
		}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// createProject creates a new project via the API
func (m ProjectsModel) createProject() tea.Msg {
	// Design Decision: Project creation requires CLI for proper input validation
	// Rationale: TUI input forms are complex; CLI provides better error handling and validation
	// Future Enhancement: Add TUI form dialog if demand exists
	// Use CLI command: cws project create <name> --owner <email> [--description "..."]
	return ProjectDataMsg{Error: fmt.Errorf("project creation via TUI not implemented - use CLI: cws project create <name> --owner <email>")}
}

// renderProjectTabBar renders a tab bar for navigation
func renderProjectTabBar(tabs []string, selected int, theme styles.Theme) string {
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

// updateProjectsTable updates the projects table with current data
func (m *ProjectsModel) updateProjectsTable() {
	// This method updates the table rows with current project data
	// The actual update happens in renderOverview()
}
