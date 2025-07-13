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
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// IdlePolicy represents an idle policy configuration item
type IdlePolicy struct {
	name        string
	description string
	threshold   int    // in minutes
	action      string // stop, hibernate, notify
	// Added uppercase for external compatibility
	Name        string
	Desc        string // Using Desc instead of Description to avoid conflict with the method
	Threshold   int
	Action      string
}

// FilterValue returns the value to filter on in the list
func (p IdlePolicy) FilterValue() string { return p.name }

// Title returns the name of the policy
func (p IdlePolicy) Title() string { return p.name }

// Description returns the description of the policy
func (p IdlePolicy) Description() string {
	return fmt.Sprintf("%s (%d minutes → %s)", 
		p.description, p.threshold, strings.ToUpper(p.action))
}

// IdleSettingsModel represents the idle detection settings view
type IdleSettingsModel struct {
	apiClient   apiClient
	policyList  list.Model
	statusBar   components.StatusBar
	spinner     components.Spinner
	width       int
	height      int
	loading     bool
	error       string
	policies    []IdlePolicy
	selected    string
	editing     bool
	threshInput textinput.Model
	actionInput textinput.Model
}

// NewIdleSettingsModel creates a new idle settings model
func NewIdleSettingsModel(apiClient apiClient) IdleSettingsModel {
	theme := styles.CurrentTheme
	
	// Set up policy list
	policyList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	policyList.Title = "Idle Detection Policies"
	policyList.Styles.Title = theme.Title
	policyList.Styles.PaginationStyle = theme.Pagination
	policyList.Styles.HelpStyle = theme.Help
	
	// Create threshold input field
	threshInput := textinput.New()
	threshInput.Placeholder = "Threshold (minutes)"
	threshInput.Width = 20
	threshInput.CharLimit = 4
	
	// Create action input field
	actionInput := textinput.New()
	actionInput.Placeholder = "Action (stop/hibernate/notify)"
	actionInput.Width = 20
	actionInput.CharLimit = 10
	
	// Create status bar and spinner
	statusBar := components.NewStatusBar("Idle Settings", "")
	spinner := components.NewSpinner("Loading idle policies...")
	
	return IdleSettingsModel{
		apiClient:   apiClient,
		policyList:  policyList,
		statusBar:   statusBar,
		spinner:     spinner,
		width:       80,
		height:      24,
		loading:     true,
		policies:    []IdlePolicy{},
		threshInput: threshInput,
		actionInput: actionInput,
		editing:     false,
	}
}

// Init initializes the model
func (m IdleSettingsModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchIdlePolicies,
	)
}

// fetchIdlePolicies retrieves idle policy data from the API
func (m IdleSettingsModel) fetchIdlePolicies() tea.Msg {
	policies, err := m.apiClient.ListIdlePolicies(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list idle policies: %w", err)
	}
	return policies
}

// savePolicyChanges saves changes to a policy
func (m IdleSettingsModel) savePolicyChanges() tea.Msg {
	if m.selected == "" {
		return fmt.Errorf("no policy selected")
	}
	
	// Parse threshold value
	threshold, err := strconv.Atoi(m.threshInput.Value())
	if err != nil {
		return fmt.Errorf("invalid threshold value: %w", err)
	}
	
	// Validate action
	action := strings.ToLower(m.actionInput.Value())
	if action != "stop" && action != "hibernate" && action != "notify" {
		return fmt.Errorf("invalid action: must be stop, hibernate, or notify")
	}
	
	// Create update request
	req := api.IdlePolicyUpdateRequest{
		Name:      m.selected,
		Threshold: threshold,
		Action:    action,
	}
	
	err = m.apiClient.UpdateIdlePolicy(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to update idle policy: %w", err)
	}
	
	// Refresh policies after update
	return RefreshMsg{}
}

// Update handles messages and updates the model
func (m IdleSettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		m.policyList.SetSize(m.width, m.height-10)
		return m, nil
		
	case tea.KeyMsg:
		// Handle key presses when editing
		if m.editing {
			switch msg.String() {
			case "enter":
				// Save changes
				return m, m.savePolicyChanges
				
			case "esc":
				// Cancel editing
				m.editing = false
				return m, nil
			}
			
			// Handle inputs
			var cmd tea.Cmd
			
			if m.threshInput.Focused() {
				m.threshInput, cmd = m.threshInput.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				m.actionInput, cmd = m.actionInput.Update(msg)
				cmds = append(cmds, cmd)
			}
			
			// Handle tab to switch between inputs
			if msg.String() == "tab" {
				if m.threshInput.Focused() {
					m.threshInput.Blur()
					m.actionInput.Focus()
				} else {
					m.actionInput.Blur()
					m.threshInput.Focus()
				}
			}
			
			return m, tea.Batch(cmds...)
		}
		
		// Handle general key presses
		switch msg.String() {
		case "r":
			m.loading = true
			m.error = ""
			return m, m.fetchIdlePolicies
			
		case "e":
			if i, ok := m.policyList.SelectedItem().(IdlePolicy); ok {
				m.selected = i.name
				m.editing = true
				m.threshInput.SetValue(strconv.Itoa(i.threshold))
				m.threshInput.Focus()
				m.actionInput.SetValue(i.action)
				return m, nil
			}
			
		case "q", "esc":
			return m, tea.Quit
		}
		
		// Update list selection
		if !m.loading {
			var cmd tea.Cmd
			m.policyList, cmd = m.policyList.Update(msg)
			cmds = append(cmds, cmd)
			
			if i, ok := m.policyList.SelectedItem().(IdlePolicy); ok {
				m.selected = i.name
			}
		}
		
	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchIdlePolicies
		
	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
		
	case []IdlePolicy:
		m.loading = false
		m.policies = msg
		
		// Update policy list items
		var items []list.Item
		for _, policy := range m.policies {
			items = append(items, IdlePolicy{
				name:        policy.name,
				description: policy.description,
				threshold:   policy.threshold,
				action:      policy.action,
				// Set uppercase fields for external compatibility
				Name:        policy.name,
				Desc:        policy.description,
				Threshold:   policy.threshold,
				Action:      policy.action,
			})
		}
		
		m.policyList.SetItems(items)
		m.statusBar.SetStatus("Idle policies loaded", components.StatusSuccess)
		
		// Select first item
		if len(items) > 0 && m.selected == "" {
			m.selected = items[0].(IdlePolicy).name
		}
	}
	
	// Update spinner when loading
	if m.loading {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}
	
	return m, tea.Batch(cmds...)
}

// View renders the idle settings view
func (m IdleSettingsModel) View() string {
	theme := styles.CurrentTheme
	
	// Title section
	title := theme.Title.Render("CloudWorkstation Idle Detection Settings")
	
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
	} else if m.editing {
		// Editing view
		editPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Edit Idle Policy: " + m.selected),
				"",
				"Threshold (minutes): " + m.threshInput.View(),
				"",
				"Action (stop/hibernate/notify): " + m.actionInput.View(),
				"",
				"Press Enter to save, Esc to cancel",
			),
		)
		
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			editPanel,
		)
	} else {
		// Standard view with policy list
		listPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.policyList.View(),
			),
		)
		
		// Description panel for selected policy
		var descPanel string
		for _, policy := range m.policies {
			if policy.Name == m.selected {
				descPanel = theme.Panel.Copy().Width(m.width - 4).Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						theme.PanelHeader.Render("Policy Details"),
						"",
						"Name: " + policy.name,
						"Description: " + policy.description,
						"Threshold: " + strconv.Itoa(policy.threshold) + " minutes",
						"Action: " + strings.ToUpper(policy.action),
						"Instance Types: All",
						"",
						"Press 'e' to edit this policy",
					),
				)
				break
			}
		}
		
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			listPanel,
			descPanel,
		)
	}
	
	// Help text
	var help string
	if m.editing {
		help = theme.Help.Render("enter: save • esc: cancel • tab: next field")
	} else {
		help = theme.Help.Render("r: refresh • e: edit • q: quit • ↑/↓: navigate")
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