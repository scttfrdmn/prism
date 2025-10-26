package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/prism/internal/tui/api"
	"github.com/scttfrdmn/prism/internal/tui/components"
	"github.com/scttfrdmn/prism/internal/tui/styles"
)

// PolicyModel represents the policy management view
type PolicyModel struct {
	apiClient         apiClient
	policySetsTable   components.Table
	statusBar         components.StatusBar
	spinner           components.Spinner
	width             int
	height            int
	loading           bool
	error             string
	status            *api.PolicyStatusResponse
	policySets        []api.PolicySetResponse
	selectedPolicySet int
	showCheckDialog   bool
	checkTemplateName string
}

// PolicyDataMsg represents policy data retrieved from the API
type PolicyDataMsg struct {
	Status     *api.PolicyStatusResponse
	PolicySets []api.PolicySetResponse
	Error      error
}

// PolicyCheckResultMsg represents template access check result
type PolicyCheckResultMsg struct {
	Result *api.TemplateAccessResponse
	Error  error
}

// NewPolicyModel creates a new policy model
func NewPolicyModel(apiClient apiClient) PolicyModel {
	// Create policy sets table
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "DESCRIPTION", Width: 40},
		{Title: "POLICIES", Width: 10},
		{Title: "STATUS", Width: 15},
	}

	policySetsTable := components.NewTable(columns, []table.Row{}, 80, 10, true)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("Prism Policy Framework", "")
	spinner := components.NewSpinner("Loading policy information...")

	return PolicyModel{
		apiClient:         apiClient,
		policySetsTable:   policySetsTable,
		statusBar:         statusBar,
		spinner:           spinner,
		width:             80,
		height:            24,
		loading:           true,
		selectedPolicySet: 0,
	}
}

// Init initializes the model
func (m PolicyModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchPolicyData,
	)
}

// fetchPolicyData retrieves policy data from the API
func (m PolicyModel) fetchPolicyData() tea.Msg {
	// Fetch policy status
	status, err := m.apiClient.GetPolicyStatus(context.Background())
	if err != nil {
		return PolicyDataMsg{Error: fmt.Errorf("failed to get policy status: %w", err)}
	}

	// Fetch policy sets
	policySets, err := m.apiClient.ListPolicySets(context.Background())
	if err != nil {
		return PolicyDataMsg{Error: fmt.Errorf("failed to list policy sets: %w", err)}
	}

	return PolicyDataMsg{
		Status:     status,
		PolicySets: policySets.PolicySets,
		Error:      nil,
	}
}

// Update handles messages and updates the model
func (m PolicyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.policySetsTable.SetSize(msg.Width-4, msg.Height-12)
		return m, nil

	case PolicyDataMsg:
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.loading = false
			return m, nil
		}

		m.status = msg.Status
		m.policySets = msg.PolicySets
		m.loading = false
		m.error = ""

		// Update table with policy set data
		m.updatePolicySetsTable()
		return m, nil

	case PolicyCheckResultMsg:
		// Handle template access check result
		if msg.Error != nil {
			m.error = msg.Error.Error()
			return m, nil
		}

		m.showCheckDialog = false
		m.checkTemplateName = ""
		// Display result (could enhance with a result dialog)
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r", "f5":
			// Refresh policy data
			m.loading = true
			return m, m.fetchPolicyData

		case "e":
			// Enable policy enforcement
			if m.status != nil && !m.status.Enabled {
				return m, m.enableEnforcement
			}

		case "d":
			// Disable policy enforcement
			if m.status != nil && m.status.Enabled {
				return m, m.disableEnforcement
			}

		case "a":
			// Assign selected policy set
			if m.selectedPolicySet < len(m.policySets) {
				policySet := m.policySets[m.selectedPolicySet]
				return m, m.assignPolicySet(policySet.ID)
			}

		case "c":
			// Check template access (show dialog)
			m.showCheckDialog = true
			return m, nil

		case "esc":
			// Close dialog
			if m.showCheckDialog {
				m.showCheckDialog = false
				m.checkTemplateName = ""
				return m, nil
			}

		case "enter":
			// Handle check dialog submission
			if m.showCheckDialog && m.checkTemplateName != "" {
				return m, m.checkTemplateAccess(m.checkTemplateName)
			}

		case "up", "k":
			if m.selectedPolicySet > 0 {
				m.selectedPolicySet--
			}
			return m, nil

		case "down", "j":
			if m.selectedPolicySet < len(m.policySets)-1 {
				m.selectedPolicySet++
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
func (m PolicyModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("🔒 Policy Framework")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Policy status section
	b.WriteString(m.renderStatus())
	b.WriteString("\n\n")

	// Policy sets section
	b.WriteString(m.renderPolicySets())

	// Show check dialog if active
	if m.showCheckDialog {
		dialog := m.renderCheckDialog()
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

// renderStatus displays the policy enforcement status
func (m PolicyModel) renderStatus() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Policy Status"))
	b.WriteString("\n\n")

	if m.status == nil {
		b.WriteString("No policy status available\n")
		return b.String()
	}

	// Enforcement status
	enforcementStatus := "DISABLED"
	enforcementStyle := theme.StatusError
	if m.status.Enabled {
		enforcementStatus = "ENABLED"
		enforcementStyle = theme.StatusOK
	}

	b.WriteString("Enforcement: ")
	b.WriteString(enforcementStyle.Render(enforcementStatus))
	b.WriteString("\n")

	// Assigned policies
	if len(m.status.AssignedPolicies) > 0 {
		b.WriteString("Assigned Policies: ")
		b.WriteString(theme.SubTitle.Render(strings.Join(m.status.AssignedPolicies, ", ")))
		b.WriteString("\n")
	} else {
		b.WriteString("Assigned Policies: ")
		b.WriteString(theme.Warning.Render("None (default allow)"))
		b.WriteString("\n")
	}

	// Status message
	if m.status.Message != "" {
		b.WriteString("\n")
		b.WriteString(m.status.Message)
		b.WriteString("\n")
	}

	return b.String()
}

// renderPolicySets displays the available policy sets
func (m PolicyModel) renderPolicySets() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Available Policy Sets"))
	b.WriteString("\n\n")

	if len(m.policySets) == 0 {
		b.WriteString("No policy sets available\n")
		return b.String()
	}

	// Policy sets table
	rows := []table.Row{}
	for i, policySet := range m.policySets {
		// Selection indicator
		name := policySet.ID
		if i == m.selectedPolicySet {
			name = "> " + name
		}

		row := table.Row{
			name,
			policySet.Description,
			fmt.Sprintf("%d", policySet.PolicyCount),
			policySet.Status,
		}
		rows = append(rows, row)
	}

	// Update table rows
	m.policySetsTable.SetRows(rows)
	b.WriteString(m.policySetsTable.View())

	return b.String()
}

// renderCheckDialog displays the template access check dialog
func (m PolicyModel) renderCheckDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(theme.SubTitle.Render("Check Template Access") + "\n\n")
	content.WriteString("Template Name: " + m.checkTemplateName + "\n\n")
	content.WriteString("Press Enter to check, Esc to cancel\n")

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m PolicyModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showCheckDialog {
		helps = []string{"esc: cancel", "enter: check"}
	} else {
		helps = []string{
			"↑/↓: select",
			"a: assign",
			"e: enable",
			"d: disable",
			"c: check access",
			"r: refresh",
			"q: quit",
		}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// enableEnforcement enables policy enforcement
func (m PolicyModel) enableEnforcement() tea.Msg {
	err := m.apiClient.SetPolicyEnforcement(context.Background(), true)
	if err != nil {
		return PolicyDataMsg{Error: fmt.Errorf("failed to enable enforcement: %w", err)}
	}

	// Refresh policy data
	return m.fetchPolicyData()
}

// disableEnforcement disables policy enforcement
func (m PolicyModel) disableEnforcement() tea.Msg {
	err := m.apiClient.SetPolicyEnforcement(context.Background(), false)
	if err != nil {
		return PolicyDataMsg{Error: fmt.Errorf("failed to disable enforcement: %w", err)}
	}

	// Refresh policy data
	return m.fetchPolicyData()
}

// assignPolicySet assigns a policy set to the current user
func (m PolicyModel) assignPolicySet(policySetID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.AssignPolicySet(context.Background(), policySetID)
		if err != nil {
			return PolicyDataMsg{Error: fmt.Errorf("failed to assign policy set: %w", err)}
		}

		// Refresh policy data
		return m.fetchPolicyData()
	}
}

// checkTemplateAccess checks if the user has access to a template
func (m PolicyModel) checkTemplateAccess(templateName string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.apiClient.CheckTemplateAccess(context.Background(), templateName)
		if err != nil {
			return PolicyCheckResultMsg{Error: fmt.Errorf("failed to check template access: %w", err)}
		}

		return PolicyCheckResultMsg{Result: result, Error: nil}
	}
}

// updatePolicySetsTable updates the policy sets table with current data
func (m *PolicyModel) updatePolicySetsTable() {
	// This method updates the table rows with current policy set data
	// The actual update happens in renderPolicySets()
}
