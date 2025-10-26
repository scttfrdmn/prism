package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/prism/internal/tui/components"
	"github.com/scttfrdmn/prism/internal/tui/styles"
	"github.com/scttfrdmn/prism/pkg/types"
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
	dispatcher  *CommandDispatcher // Command Pattern for message handling
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

	// Create command dispatcher for message handling
	dispatcher := NewCommandDispatcher()
	dispatcher.RegisterCommand(&InstanceActionWindowSizeCommand{})
	dispatcher.RegisterCommand(&InstanceActionKeyCommand{})
	dispatcher.RegisterCommand(&InstanceActionInstanceCommand{})
	dispatcher.RegisterCommand(&InstanceActionResultCommand{})
	dispatcher.RegisterCommand(&InstanceActionErrorCommand{})

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
		dispatcher:  dispatcher,
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

// Update handles messages and updates the model using Command Pattern (SOLID: Single Responsibility)
func (m InstanceActionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Try to dispatch using Command Pattern (same as RepositoriesModel)
	if m.dispatcher != nil {
		newModel, cmd := m.dispatcher.Dispatch(msg, m)
		if cmd != nil {
			return newModel, cmd
		}
	}

	// Handle remaining message types
	return m.handleDefaultMessages(msg)
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
			Height(m.height-8). // Account for title, status bar, help
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-8).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else {
		if m.confirmStep {
			// Confirmation dialog
			selected, ok := m.actionList.SelectedItem().(ActionItem)
			if !ok {
				selected = ActionItem{name: "Unknown Action"}
			}

			confirmPanel := theme.Panel.Width(m.width - 10).Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					theme.PanelHeader.Render("Confirm Action"),
					"",
					fmt.Sprintf("Are you sure you want to perform this action: %s?", selected.name),
					"",
					theme.Warning.Render("This action may have permanent consequences."),
					"",
					"Press Y to confirm, N to cancel",
				),
			)

			content = lipgloss.NewStyle().
				Width(m.width).
				Height(m.height-8).
				Align(lipgloss.Center, lipgloss.Center).
				Render(confirmPanel)
		} else {
			// Normal action list
			content = theme.Panel.Width(m.width - 4).Render(
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

// handleDefaultMessages handles messages not processed by commands (Single Responsibility)
func (m InstanceActionModel) handleDefaultMessages(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case *types.Instance:
		// Handle instance data received
		m.loading = false
		m.error = ""
		// Update action list based on instance state
		actions := m.getAvailableActions(msg.State)
		items := make([]list.Item, len(actions))
		for i, action := range actions {
			items[i] = ActionItem{name: action, description: m.getActionDescription(action)}
		}
		m.actionList.SetItems(items)

	case error:
		// Handle error messages
		m.loading = false
		m.error = msg.Error()

	default:
		// Update spinner when loading
		if m.loading {
			var spinnerCmd tea.Cmd
			m.spinner, spinnerCmd = m.spinner.Update(msg)
			cmds = append(cmds, spinnerCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// Instance Action Commands (Command Pattern - SOLID: Single Responsibility + Open/Closed)

// InstanceActionWindowSizeCommand handles window resize messages
type InstanceActionWindowSizeCommand struct{}

func (c *InstanceActionWindowSizeCommand) CanExecute(msg tea.Msg) bool {
	_, ok := msg.(tea.WindowSizeMsg)
	return ok
}

func (c *InstanceActionWindowSizeCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	m := model.(InstanceActionModel)
	windowMsg := msg.(tea.WindowSizeMsg)

	m.width = windowMsg.Width
	m.height = windowMsg.Height
	m.statusBar.SetWidth(windowMsg.Width)

	// Update list dimensions
	listHeight := m.height - 10 // Account for title, status bar, help
	if listHeight < 3 {
		listHeight = 3
	}
	m.actionList.SetHeight(listHeight)
	m.actionList.SetWidth(m.width - 4)

	return m, nil
}

// InstanceActionKeyCommand handles keyboard input messages
type InstanceActionKeyCommand struct{}

func (c *InstanceActionKeyCommand) CanExecute(msg tea.Msg) bool {
	_, ok := msg.(tea.KeyMsg)
	return ok
}

func (c *InstanceActionKeyCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	m := model.(InstanceActionModel)
	keyMsg := msg.(tea.KeyMsg)

	if m.confirmStep {
		return c.handleConfirmationKeys(m, keyMsg)
	}
	return c.handleActionKeys(m, keyMsg)
}

func (c *InstanceActionKeyCommand) handleConfirmationKeys(m InstanceActionModel, keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyMsg.String() {
	case "y", "Y":
		if i, ok := m.actionList.SelectedItem().(ActionItem); ok {
			m.loading = true
			m.statusBar.SetStatus(fmt.Sprintf("Performing action: %s", i.name), components.StatusWarning)
			return m, m.performAction(i.action)
		}
	case "n", "N", "esc", "q":
		m.confirmStep = false
		return m, nil
	}
	return m, nil
}

func (c *InstanceActionKeyCommand) handleActionKeys(m InstanceActionModel, keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch keyMsg.String() {
	case "enter":
		if i, ok := m.actionList.SelectedItem().(ActionItem); ok {
			if i.dangerous {
				m.confirmStep = true
				return m, nil
			}

			m.loading = true
			m.statusBar.SetStatus(fmt.Sprintf("Performing action: %s", i.name), components.StatusWarning)
			return m, m.performAction(i.action)
		}
	case "q", "esc":
		return m, tea.Quit
	}

	// Process list navigation if not loading
	if !m.loading {
		var cmd tea.Cmd
		m.actionList, cmd = m.actionList.Update(keyMsg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// InstanceActionInstanceCommand handles instance detail messages
type InstanceActionInstanceCommand struct{}

func (c *InstanceActionInstanceCommand) CanExecute(msg tea.Msg) bool {
	_, ok := msg.(*types.Instance)
	return ok
}

func (c *InstanceActionInstanceCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	m := model.(InstanceActionModel)
	instance := msg.(*types.Instance)

	m.loading = false

	// Build action items using helper
	items := c.buildActionItems(instance)
	m.actionList.SetItems(items)
	m.statusBar.SetStatus("Select an action to perform", components.StatusSuccess)

	return m, nil
}

func (c *InstanceActionInstanceCommand) buildActionItems(instance *types.Instance) []list.Item {
	var items []list.Item
	state := strings.ToLower(instance.State)

	// Add state-specific actions
	switch state {
	case "running":
		items = append(items, c.createRunningInstanceActions(instance)...)
	case "stopped":
		items = append(items, c.createStoppedInstanceActions()...)
	}

	// Add universal actions
	items = append(items, c.createUniversalActions()...)

	return items
}

func (c *InstanceActionInstanceCommand) createRunningInstanceActions(instance *types.Instance) []list.Item {
	items := []list.Item{
		ActionItem{
			name:        "Stop Instance",
			description: "Stops the instance but preserves data (can be restarted)",
			action:      "stop",
			dangerous:   false,
		},
		ActionItem{
			name:        "Connect via SSH",
			description: fmt.Sprintf("SSH to instance: ssh %s@%s", instance.Username, instance.PublicIP),
			action:      "ssh",
			dangerous:   false,
		},
	}

	if instance.HasWebInterface {
		items = append(items, ActionItem{
			name:        "Open Web Interface",
			description: fmt.Sprintf("Open web interface at: http://%s:%d", instance.PublicIP, instance.WebPort),
			action:      "web",
			dangerous:   false,
		})
	}

	return items
}

func (c *InstanceActionInstanceCommand) createStoppedInstanceActions() []list.Item {
	return []list.Item{
		ActionItem{
			name:        "Start Instance",
			description: "Starts the stopped instance",
			action:      "start",
			dangerous:   false,
		},
	}
}

func (c *InstanceActionInstanceCommand) createUniversalActions() []list.Item {
	return []list.Item{
		ActionItem{
			name:        "Delete Instance",
			description: "Permanently deletes the instance and all data (cannot be undone)",
			action:      "delete",
			dangerous:   true,
		},
	}
}

// InstanceActionResultCommand handles action result messages
type InstanceActionResultCommand struct{}

func (c *InstanceActionResultCommand) CanExecute(msg tea.Msg) bool {
	_, ok := msg.(InstanceActionMsg)
	return ok
}

func (c *InstanceActionResultCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	m := model.(InstanceActionModel)
	actionMsg := msg.(InstanceActionMsg)

	m.loading = false

	if actionMsg.Success {
		m.statusBar.SetStatus(actionMsg.Message, components.StatusSuccess)
	} else {
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", actionMsg.Message), components.StatusError)
	}

	// Add delay and quit
	return m, tea.Sequence(
		tea.Tick(2*time.Second, func(time.Time) tea.Msg { return nil }),
		func() tea.Msg { return tea.Quit() },
	)
}

// InstanceActionErrorCommand handles error messages
type InstanceActionErrorCommand struct{}

func (c *InstanceActionErrorCommand) CanExecute(msg tea.Msg) bool {
	_, ok := msg.(error)
	return ok
}

func (c *InstanceActionErrorCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	m := model.(InstanceActionModel)
	err := msg.(error)

	m.loading = false
	m.error = err.Error()
	m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)

	return m, nil
}

// getAvailableActions returns actions available for the given instance state
func (m InstanceActionModel) getAvailableActions(state string) []string {
	switch state {
	case "running":
		return []string{"stop", "hibernate", "connect", "terminate"}
	case "stopped":
		return []string{"start", "terminate"}
	case "hibernated":
		return []string{"resume", "terminate"}
	case "stopping", "starting", "hibernating":
		return []string{} // No actions during transitions
	default:
		return []string{"start", "stop", "terminate"}
	}
}

// getActionDescription returns a description for the given action
func (m InstanceActionModel) getActionDescription(action string) string {
	switch action {
	case "start":
		return "Start the instance"
	case "stop":
		return "Stop the instance"
	case "hibernate":
		return "Hibernate the instance (preserves RAM)"
	case "resume":
		return "Resume from hibernation"
	case "connect":
		return "Connect to the instance via SSH"
	case "terminate":
		return "Permanently delete the instance"
	default:
		return "Perform action on instance"
	}
}
