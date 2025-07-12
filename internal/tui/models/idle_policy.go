package models

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// IdleActionType represents different actions for idle policies
type IdleActionType string

const (
	IdleActionStop      IdleActionType = "stop"
	IdleActionHibernate IdleActionType = "hibernate"
	IdleActionNotify    IdleActionType = "notify"
)

// IdlePolicyItem represents an idle policy in the list
type IdlePolicyItem struct {
	name        string
	description string
	threshold   int
	action      string
	appliesTo   []string
}

// FilterValue returns the value to filter on in the list
func (i IdlePolicyItem) FilterValue() string { return i.name }

// Title returns the name of the policy
func (i IdlePolicyItem) Title() string { return i.name }

// Description returns a short description of the policy
func (i IdlePolicyItem) Description() string {
	return fmt.Sprintf("%s | Threshold: %d minutes | Action: %s",
		i.description, i.threshold, strings.ToUpper(i.action))
}

// IdlePolicyModel represents the model for managing idle policies
type IdlePolicyModel struct {
	apiClient       apiClient
	policyList      list.Model
	statusBar       components.StatusBar
	spinner         components.Spinner
	width           int
	height          int
	loading         bool
	error           string
	policies        []types.IdlePolicy
	selected        string
	editing         bool
	creating        bool
	nameInput       textinput.Model
	descInput       textinput.Model
	threshInput     textinput.Model
	actionInput     textinput.Model
	appliesToInput  textinput.Model
	currentInput    int
	confirmDelete   bool
}

// NewIdlePolicyModel creates a new idle policy model
func NewIdlePolicyModel(apiClient apiClient) IdlePolicyModel {
	theme := styles.CurrentTheme

	// Set up policy list
	policyList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	policyList.Title = "Idle Detection Policies"
	policyList.Styles.Title = theme.Title
	policyList.Styles.PaginationStyle = theme.Pagination
	policyList.Styles.HelpStyle = theme.Help

	// Create input fields
	nameInput := textinput.New()
	nameInput.Placeholder = "Policy Name"
	nameInput.Width = 30

	descInput := textinput.New()
	descInput.Placeholder = "Policy Description"
	descInput.Width = 50

	threshInput := textinput.New()
	threshInput.Placeholder = "Threshold (minutes)"
	threshInput.Width = 10

	actionInput := textinput.New()
	actionInput.Placeholder = "Action (stop, hibernate, notify)"
	actionInput.Width = 15

	appliesToInput := textinput.New()
	appliesToInput.Placeholder = "Applies To (comma-separated instance types, or 'all')"
	appliesToInput.Width = 50

	// Create status bar and spinner
	statusBar := components.NewStatusBar("Idle Policy Management", "")
	spinner := components.NewSpinner("Loading idle policies...")

	return IdlePolicyModel{
		apiClient:      apiClient,
		policyList:     policyList,
		statusBar:      statusBar,
		spinner:        spinner,
		width:          80,
		height:         24,
		loading:        true,
		policies:       []types.IdlePolicy{},
		nameInput:      nameInput,
		descInput:      descInput,
		threshInput:    threshInput,
		actionInput:    actionInput,
		appliesToInput: appliesToInput,
	}
}

// Init initializes the model
func (m IdlePolicyModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchPolicies,
	)
}

// fetchPolicies retrieves idle policies from the API
func (m IdlePolicyModel) fetchPolicies() tea.Msg {
	// In a real app, this would call the API
	// For now, we'll use sample data
	policies := []types.IdlePolicy{
		{
			Name:        "default",
			Description: "Default idle policy for all instances",
			Threshold:   30,
			Action:      "stop",
			AppliesTo:   []string{"all"},
		},
		{
			Name:        "gpu-instances",
			Description: "Aggressive idle policy for cost-intensive GPU instances",
			Threshold:   15,
			Action:      "stop",
			AppliesTo:   []string{"g4dn", "p3", "p4d"},
		},
		{
			Name:        "development",
			Description: "Lenient policy for development instances",
			Threshold:   60,
			Action:      "notify",
			AppliesTo:   []string{"t3", "t4g", "m5", "m6g"},
		},
	}
	return policies
}

// savePolicy saves changes to a policy or creates a new one
func (m IdlePolicyModel) savePolicy() tea.Cmd {
	return func() tea.Msg {
		// Parse threshold value
		threshold, err := strconv.Atoi(m.threshInput.Value())
		if err != nil {
			return fmt.Errorf("invalid threshold value: %w", err)
		}

		// Validate action
		action := strings.ToLower(m.actionInput.Value())
		if !isValidIdleAction(action) {
			return fmt.Errorf("invalid action: must be stop, hibernate, or notify")
		}

		// Parse applies to
		appliesTo := strings.Split(m.appliesToInput.Value(), ",")
		for i := range appliesTo {
			appliesTo[i] = strings.TrimSpace(appliesTo[i])
		}

		// Create policy object
		policy := types.IdlePolicy{
			Name:        m.nameInput.Value(),
			Description: m.descInput.Value(),
			Threshold:   threshold,
			Action:      action,
			AppliesTo:   appliesTo,
		}

		// If creating a new policy
		if m.creating {
			// Check for duplicate name
			for _, p := range m.policies {
				if p.Name == policy.Name {
					return fmt.Errorf("policy with name '%s' already exists", policy.Name)
				}
			}

			// Add to list
			m.policies = append(m.policies, policy)
		} else {
			// Update existing policy
			for i := range m.policies {
				if m.policies[i].Name == m.selected {
					m.policies[i] = policy
					break
				}
			}
		}

		// In a real app, this would call the API to save the policy

		return m.policies
	}
}

// deletePolicy deletes the selected policy
func (m IdlePolicyModel) deletePolicy() tea.Cmd {
	return func() tea.Msg {
		if m.selected == "" {
			return fmt.Errorf("no policy selected")
		}

		// Find and remove the policy
		var newPolicies []types.IdlePolicy
		for _, p := range m.policies {
			if p.Name != m.selected {
				newPolicies = append(newPolicies, p)
			}
		}

		// In a real app, this would call the API to delete the policy

		m.confirmDelete = false
		return newPolicies
	}
}

// isValidIdleAction checks if the action is valid
func isValidIdleAction(action string) bool {
	action = strings.ToLower(action)
	return action == string(IdleActionStop) || 
	       action == string(IdleActionHibernate) || 
	       action == string(IdleActionNotify)
}

// resetInputs clears all input fields
func (m *IdlePolicyModel) resetInputs() {
	m.nameInput.SetValue("")
	m.descInput.SetValue("")
	m.threshInput.SetValue("")
	m.actionInput.SetValue("")
	m.appliesToInput.SetValue("")
	m.currentInput = 0
}

// setupEditInputs prepares input fields for editing
func (m *IdlePolicyModel) setupEditInputs() {
	for _, policy := range m.policies {
		if policy.Name == m.selected {
			m.nameInput.SetValue(policy.Name)
			m.descInput.SetValue(policy.Description)
			m.threshInput.SetValue(strconv.Itoa(policy.Threshold))
			m.actionInput.SetValue(policy.Action)
			m.appliesToInput.SetValue(strings.Join(policy.AppliesTo, ", "))
			break
		}
	}
	m.focusInput(0)
}

// focusInput focuses a specific input field and blurs the others
func (m *IdlePolicyModel) focusInput(index int) {
	m.nameInput.Blur()
	m.descInput.Blur()
	m.threshInput.Blur()
	m.actionInput.Blur()
	m.appliesToInput.Blur()

	m.currentInput = index
	switch index {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.descInput.Focus()
	case 2:
		m.threshInput.Focus()
	case 3:
		m.actionInput.Focus()
	case 4:
		m.appliesToInput.Focus()
	}
}

// Update handles messages and updates the model
func (m IdlePolicyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)

		// Update list dimensions
		listHeight := m.height - 16 // Account for title, details, status bar, help
		if listHeight < 3 {
			listHeight = 3
		}
		m.policyList.SetHeight(listHeight)
		m.policyList.SetWidth(m.width - 4)

		return m, nil

	case tea.KeyMsg:
		// Handle confirmation dialog
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				return m, m.deletePolicy()
			case "n", "N", "esc", "q":
				m.confirmDelete = false
				return m, nil
			}
			return m, nil
		}

		// Handle form editing/creating
		if m.editing || m.creating {
			switch msg.String() {
			case "enter":
				// Move to next input or submit if on last input
				if m.currentInput < 4 {
					m.focusInput(m.currentInput + 1)
					return m, nil
				}
				// Submit the form
				m.editing = false
				m.creating = false
				return m, m.savePolicy()

			case "esc":
				// Cancel editing
				m.editing = false
				m.creating = false
				return m, nil

			case "tab":
				// Move to next input
				nextInput := (m.currentInput + 1) % 5
				m.focusInput(nextInput)
				return m, nil

			case "shift+tab":
				// Move to previous input
				prevInput := (m.currentInput + 4) % 5
				m.focusInput(prevInput)
				return m, nil
			}

			// Update the active input
			var cmd tea.Cmd
			switch m.currentInput {
			case 0:
				m.nameInput, cmd = m.nameInput.Update(msg)
			case 1:
				m.descInput, cmd = m.descInput.Update(msg)
			case 2:
				m.threshInput, cmd = m.threshInput.Update(msg)
			case 3:
				m.actionInput, cmd = m.actionInput.Update(msg)
			case 4:
				m.appliesToInput, cmd = m.appliesToInput.Update(msg)
			}
			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}

		// Handle view mode keys
		switch msg.String() {
		case "a":
			// Add new policy
			m.creating = true
			m.resetInputs()
			m.nameInput.Focus()
			return m, nil

		case "e":
			// Edit selected policy
			if m.selected != "" {
				m.editing = true
				m.setupEditInputs()
				return m, nil
			}

		case "d":
			// Delete selected policy
			if m.selected != "" {
				m.confirmDelete = true
				return m, nil
			}

		case "r":
			// Refresh policy list
			m.loading = true
			m.error = ""
			return m, m.fetchPolicies

		case "q", "esc":
			return m, tea.Quit
		}

		// Update list selection
		if !m.loading {
			var cmd tea.Cmd
			m.policyList, cmd = m.policyList.Update(msg)
			cmds = append(cmds, cmd)

			// Update selected policy
			if i, ok := m.policyList.SelectedItem().(IdlePolicyItem); ok {
				m.selected = i.name
			}
		}

	case []types.IdlePolicy:
		m.loading = false
		m.policies = msg

		// Update policy list items
		var items []list.Item
		for _, policy := range m.policies {
			items = append(items, IdlePolicyItem{
				name:        policy.Name,
				description: policy.Description,
				threshold:   policy.Threshold,
				action:      policy.Action,
				appliesTo:   policy.AppliesTo,
			})
		}

		m.policyList.SetItems(items)
		m.statusBar.SetStatus(fmt.Sprintf("Loaded %d idle policies", len(items)), components.StatusSuccess)

		// Select first item if none selected
		if len(items) > 0 && m.selected == "" {
			m.selected = items[0].(IdlePolicyItem).name
		}

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
	}

	// Update spinner when loading
	if m.loading {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the idle policy view
func (m IdlePolicyModel) View() string {
	theme := styles.CurrentTheme

	// Title section
	title := theme.Title.Render("CloudWorkstation Idle Policies")

	// Content area
	var content string

	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8). // Account for title and status
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else if m.confirmDelete {
		// Confirmation dialog
		confirmPanel := theme.Panel.Copy().Width(m.width - 20).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Confirm Delete"),
				"",
				fmt.Sprintf("Are you sure you want to delete the policy '%s'?", m.selected),
				"",
				"This action cannot be undone.",
				"",
				"Press Y to confirm, N to cancel",
			),
		)

		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8).
			Align(lipgloss.Center, lipgloss.Center).
			Render(confirmPanel)
	} else if m.editing || m.creating {
		// Form view
		var formTitle string
		if m.creating {
			formTitle = "Create New Policy"
		} else {
			formTitle = "Edit Policy: " + m.selected
		}

		formPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render(formTitle),
				"",
				"Policy Name: " + m.nameInput.View(),
				"",
				"Description: " + m.descInput.View(),
				"",
				"Threshold (minutes): " + m.threshInput.View(),
				"",
				"Action (stop, hibernate, notify): " + m.actionInput.View(),
				"",
				"Applies To (comma-separated): " + m.appliesToInput.View(),
				"",
				"Press Enter to save, Esc to cancel",
			),
		)

		content = formPanel
	} else {
		// List view
		listPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.policyList.View(),
			),
		)

		// Detail panel for selected policy
		var detailPanel string
		for _, policy := range m.policies {
			if policy.Name == m.selected {
				detailPanel = theme.Panel.Copy().Width(m.width - 4).Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						theme.PanelHeader.Render("Policy Details"),
						"",
						"Name: " + policy.Name,
						"Description: " + policy.Description,
						"Threshold: " + strconv.Itoa(policy.Threshold) + " minutes",
						"Action: " + strings.ToUpper(policy.Action),
						"Applies To: " + strings.Join(policy.AppliesTo, ", "),
						"",
						"Press 'e' to edit, 'd' to delete, 'a' to add new policy",
					),
				)
				break
			}
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			listPanel,
			detailPanel,
		)
	}

	// Help text
	var help string
	if m.editing || m.creating {
		help = theme.Help.Render("enter: next/save • esc: cancel • tab: next field • shift+tab: prev field")
	} else if m.confirmDelete {
		help = theme.Help.Render("y: confirm • n: cancel")
	} else {
		help = theme.Help.Render("a: add • e: edit • d: delete • r: refresh • q: quit • ↑/↓: navigate")
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