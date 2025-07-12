package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
)

// ActionItem represents an action in the action list
type ActionItem struct {
	name        string
	description string
	action      string
	dangerous   bool
}

// FilterValue returns the value to filter on in the list
func (a ActionItem) FilterValue() string { return a.name }

// Title returns the name of the action
func (a ActionItem) Title() string { 
	if a.dangerous {
		return a.name + " (!)"
	}
	return a.name 
}

// Description returns a short description of the action
func (a ActionItem) Description() string { return a.description }

// InstanceActionModel represents a model for instance actions
type InstanceActionModel struct {
	apiClient   apiClient
	actionList  list.Model
	statusBar   components.StatusBar
	spinner     components.Spinner
	width       int
	height      int
	loading     bool
	error       string
	instance    string
	confirmStep bool
}

// NewInstanceActionModel creates a new instance action model
func NewInstanceActionModel(apiClient apiClient, instance string) InstanceActionModel {
	theme := styles.CurrentTheme
	
	// Set up action list
	actionList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	actionList.Title = "Instance Actions"
	actionList.Styles.Title = theme.Title
	actionList.Styles.PaginationStyle = theme.Pagination
	actionList.Styles.HelpStyle = theme.Help
	
	// Create status bar and spinner
	statusBar := components.NewStatusBar("Instance Actions", "")
	spinner := components.NewSpinner("Loading instance information...")
	
	// Create the model
	model := InstanceActionModel{
		apiClient:   apiClient,
		actionList:  actionList,
		statusBar:   statusBar,
		spinner:     spinner,
		width:       80,
		height:      24,
		loading:     true,
		instance:    instance,
		confirmStep: false,
	}
	
	return model
}

// Init initializes the model
func (m InstanceActionModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchInstanceDetails,
	)
}

// fetchInstanceDetails retrieves instance details from the API
func (m InstanceActionModel) fetchInstanceDetails() tea.Msg {
	response, err := m.apiClient.GetInstance(context.Background(), m.instance)
	if err != nil {
		return fmt.Errorf("failed to get instance details: %w", err)
	}
	return response
}

// performAction executes the selected action
func (m InstanceActionModel) performAction(action string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		
		switch action {
		case "start":
			err = m.apiClient.StartInstance(ctx, m.instance)
			if err != nil {
				return fmt.Errorf("failed to start instance: %w", err)
			}
			return InstanceActionMsg{
				Action:  "start",
				Success: true,
				Message: fmt.Sprintf("Started instance %s", m.instance),
			}
			
		case "stop":
			err = m.apiClient.StopInstance(ctx, m.instance)
			if err != nil {
				return fmt.Errorf("failed to stop instance: %w", err)
			}
			return InstanceActionMsg{
				Action:  "stop",
				Success: true,
				Message: fmt.Sprintf("Stopped instance %s", m.instance),
			}
			
		case "delete":
			err = m.apiClient.DeleteInstance(ctx, m.instance)
			if err != nil {
				return fmt.Errorf("failed to delete instance: %w", err)
			}
			return InstanceActionMsg{
				Action:  "delete",
				Success: true,
				Message: fmt.Sprintf("Deleted instance %s", m.instance),
			}
			
		default:
			return fmt.Errorf("unknown action: %s", action)
		}
	}
}

// InstanceActionMsg represents a message about an instance action
type InstanceActionMsg struct {
	Action  string
	Success bool
	Message string
}

// Update handles messages and updates the model
func (m InstanceActionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		
		// Update list dimensions
		listHeight := m.height - 10 // Account for title, status bar, help
		if listHeight < 3 {
			listHeight = 3
		}
		m.actionList.SetHeight(listHeight)
		m.actionList.SetWidth(m.width - 4)
		
		return m, nil
		
	case tea.KeyMsg:
		// Handle key presses
		if m.confirmStep {
			switch msg.String() {
			case "y", "Y":
				// Get the selected action
				if i, ok := m.actionList.SelectedItem().(ActionItem); ok {
					m.loading = true
					m.statusBar.SetStatus(fmt.Sprintf("Performing action: %s", i.name), components.StatusWarning)
					return m, m.performAction(i.action)
				}
			case "n", "N", "esc", "q":
				m.confirmStep = false
				return m, nil
			}
		} else {
			switch msg.String() {
			case "enter":
				// Get the selected action
				if i, ok := m.actionList.SelectedItem().(ActionItem); ok {
					if i.dangerous {
						m.confirmStep = true
						return m, nil
					}
					
					// Non-dangerous actions can be performed immediately
					m.loading = true
					m.statusBar.SetStatus(fmt.Sprintf("Performing action: %s", i.name), components.StatusWarning)
					return m, m.performAction(i.action)
				}
				
			case "q", "esc":
				return m, tea.Quit
			}
		}
		
		// If not loading, process list navigation
		if !m.loading {
			var cmd tea.Cmd
			m.actionList, cmd = m.actionList.Update(msg)
			cmds = append(cmds, cmd)
		}
		
	case api.InstanceResponse:
		m.loading = false
		
		// Build list of actions based on instance state
		var items []list.Item
		
		// Format state for display
		state := strings.ToLower(msg.State)
		
		// Add actions based on instance state
		if state == "running" {
			items = append(items, ActionItem{
				name:        "Stop Instance",
				description: "Stops the instance but preserves data (can be restarted)",
				action:      "stop",
				dangerous:   false,
			})
			
			items = append(items, ActionItem{
				name:        "Connect via SSH",
				description: fmt.Sprintf("SSH to instance: ssh %s@%s", msg.Username, msg.PublicIP),
				action:      "ssh",
				dangerous:   false,
			})
			
			if msg.HasWebInterface {
				items = append(items, ActionItem{
					name:        "Open Web Interface",
					description: fmt.Sprintf("Open web interface at: http://%s:%d", msg.PublicIP, msg.WebPort),
					action:      "web",
					dangerous:   false,
				})
			}
		} else if state == "stopped" {
			items = append(items, ActionItem{
				name:        "Start Instance",
				description: "Starts the stopped instance",
				action:      "start",
				dangerous:   false,
			})
		}
		
		// These actions are available regardless of state
		items = append(items, ActionItem{
			name:        "Delete Instance",
			description: "Permanently deletes the instance and all data (cannot be undone)",
			action:      "delete",
			dangerous:   true,
		})
		
		m.actionList.SetItems(items)
		m.statusBar.SetStatus("Select an action to perform", components.StatusSuccess)
		
	case InstanceActionMsg:
		m.loading = false
		if msg.Success {
			m.statusBar.SetStatus(msg.Message, components.StatusSuccess)
		} else {
			m.statusBar.SetStatus(fmt.Sprintf("Error: %s", msg.Message), components.StatusError)
		}
		
		// Add a delay and then quit
		return m, tea.Sequence(
			tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return nil
			}),
			func() tea.Msg {
				return tea.Quit()
			},
		)
		
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

// View renders the instance action view
func (m InstanceActionModel) View() string {
	theme := styles.CurrentTheme
	
	// Title section
	title := theme.Title.Render(fmt.Sprintf("Instance Actions: %s", m.instance))
	
	// Content area
	var content string
	
	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8). // Account for title, status bar, help
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else {
		if m.confirmStep {
			// Confirmation dialog
			selected, ok := m.actionList.SelectedItem().(ActionItem)
			if !ok {
				selected = ActionItem{name: "Unknown Action"}
			}
			
			confirmPanel := theme.Panel.Copy().Width(m.width - 10).Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					theme.PanelHeader.Render("Confirm Action"),
					"",
					fmt.Sprintf("Are you sure you want to perform this action: %s?", selected.name),
					"",
					theme.WarningText.Render("This action may have permanent consequences."),
					"",
					"Press Y to confirm, N to cancel",
				),
			)
			
			content = lipgloss.NewStyle().
				Width(m.width).
				Height(m.height - 8).
				Align(lipgloss.Center, lipgloss.Center).
				Render(confirmPanel)
		} else {
			// Normal action list
			content = theme.Panel.Copy().Width(m.width - 4).Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					m.actionList.View(),
				),
			)
		}
	}
	
	// Help text
	var help string
	if m.confirmStep {
		help = theme.Help.Render("y: confirm • n: cancel")
	} else {
		help = theme.Help.Render("↑/↓: navigate • enter: select • q: quit")
	}
	
	// Join everything together
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		m.statusBar.View(),
		help,
	)
}