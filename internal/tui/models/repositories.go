package models

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// RepositoriesModel manages template repository configuration
type RepositoriesModel struct {
	apiClient apiClient
	repoList  list.Model
	statusBar components.StatusBar
	spinner   components.Spinner
	tabBar    components.TabBar
	width     int
	height    int
	loading   bool
	error     string
	repos     []types.TemplateRepository
	selected  string

	// Editor fields
	nameInput     textinput.Model
	urlInput      textinput.Model
	priorityInput textinput.Model
	enabledInput  textinput.Model
	mode          string // "view", "add", "edit"
	focusIndex    int

	// Command dispatcher for SOLID architecture
	dispatcher *CommandDispatcher
}

// RepositoryItem represents a repository in the list
type RepositoryItem struct {
	Name     string
	URL      string
	Priority int
	Enabled  bool
}

func (i RepositoryItem) FilterValue() string {
	return i.Name
}

func (i RepositoryItem) Title() string {
	return i.Name
}

func (i RepositoryItem) Description() string {
	status := "Disabled"
	if i.Enabled {
		status = "Enabled"
	}
	return fmt.Sprintf("%s • Priority: %d • %s", i.URL, i.Priority, status)
}

// NewRepositoriesModel creates a new repositories model
func NewRepositoriesModel(apiClient apiClient) RepositoriesModel {
	theme := styles.CurrentTheme

	// Set up repository list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(theme.AccentColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(theme.AccentColor)

	repoList := list.New([]list.Item{}, delegate, 0, 0)
	repoList.Title = "Template Repositories"
	repoList.SetShowHelp(false)

	// Set up form inputs
	nameInput := textinput.New()
	nameInput.Placeholder = "Repository Name"
	nameInput.CharLimit = 32

	urlInput := textinput.New()
	urlInput.Placeholder = "Repository URL"
	urlInput.CharLimit = 256

	priorityInput := textinput.New()
	priorityInput.Placeholder = "Priority (0-100)"
	priorityInput.CharLimit = 3

	enabledInput := textinput.New()
	enabledInput.Placeholder = "Enabled (true/false)"
	enabledInput.CharLimit = 5

	// Status bar
	statusBar := components.NewStatusBar("Repository Management", "↑/↓: Navigate • Enter: Select • a: Add • e: Edit • r: Refresh • d: Delete")

	// Spinner for loading states
	spinner := components.NewSpinner("Loading repositories...")

	// Tab bar
	tabBar := components.NewTabBar([]string{"Instances", "Templates", "Repositories", "Volumes", "Storage"}, 2)

	// Initialize command dispatcher with all commands
	dispatcher := NewCommandDispatcher()
	dispatcher.RegisterCommand(WindowResizeCommand{})
	dispatcher.RegisterCommand(RepositoryAddCommand{})
	dispatcher.RegisterCommand(RepositoryEditCommand{})
	dispatcher.RegisterCommand(RepositoryDeleteCommand{})
	dispatcher.RegisterCommand(RepositoryRefreshCommand{})
	dispatcher.RegisterCommand(RepositorySyncCommand{})
	dispatcher.RegisterCommand(FormCancelCommand{})
	dispatcher.RegisterCommand(FormSubmitCommand{})
	dispatcher.RegisterCommand(FormNavigationCommand{})
	dispatcher.RegisterCommand(FormInputCommand{})

	return RepositoriesModel{
		apiClient:     apiClient,
		repoList:      repoList,
		statusBar:     statusBar,
		spinner:       spinner,
		tabBar:        tabBar,
		width:         80,
		height:        24,
		loading:       true,
		repos:         []types.TemplateRepository{},
		nameInput:     nameInput,
		urlInput:      urlInput,
		priorityInput: priorityInput,
		enabledInput:  enabledInput,
		mode:          "view",
		dispatcher:    dispatcher,
	}
}

// Init initializes the model
func (m RepositoriesModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchRepositories,
	)
}

// fetchRepositories retrieves repository data from the API
func (m RepositoriesModel) fetchRepositories() tea.Msg {
	// This would normally call the API client
	// Since it's not fully implemented, we'll return mock data

	repos := []types.TemplateRepository{
		{
			Name:     "default",
			URL:      "https://cloudworkstation.example.com/templates",
			Priority: 100,
			Enabled:  true,
		},
		{
			Name:     "community",
			URL:      "https://community.example.com/templates",
			Priority: 50,
			Enabled:  true,
		},
	}

	return repos
}

// syncRepositories triggers a repository sync
func (m RepositoriesModel) syncRepositories() tea.Msg {
	// This would normally call the API client
	// Since it's not fully implemented, we'll just return success
	return RefreshMsg{}
}

// refreshRepositoryList updates the list items
func (m *RepositoriesModel) refreshRepositoryList() {
	var items []list.Item

	for _, repo := range m.repos {
		items = append(items, RepositoryItem{
			Name:     repo.Name,
			URL:      repo.URL,
			Priority: repo.Priority,
			Enabled:  repo.Enabled,
		})
	}

	m.repoList.SetItems(items)
}

// Update handles UI events using Command Pattern (SOLID: Single Responsibility)
func (m RepositoriesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle global quit command
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	// Handle tab navigation to other views
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "tab" && m.mode == "view" {
		return tea.Model(nil), tea.Quit
	}

	// Try to dispatch command using Command Pattern
	newModel, cmd := m.dispatcher.Dispatch(msg, m)
	if cmd != nil {
		return newModel, cmd
	}
	// Check if model was updated by comparing mode or other simple fields
	if newRepoModel, ok := newModel.(RepositoriesModel); ok && (newRepoModel.mode != m.mode || newRepoModel.loading != m.loading) {
		return newModel, cmd
	}

	// Handle non-command messages
	switch msg := msg.(type) {
	case []types.TemplateRepository:
		m.loading = false
		m.repos = msg
		m.refreshRepositoryList()
		m.statusBar.SetStatus("Repository list updated", components.StatusSuccess)

	case RefreshMsg:
		m.loading = true
		return m, m.fetchRepositories

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus("Error: "+m.error, components.StatusError)
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)

	default:
		// Handle list updates in view mode
		if m.mode == "view" {
			var cmd tea.Cmd
			m.repoList, cmd = m.repoList.Update(msg)
			cmds = append(cmds, cmd)
		}

		// Handle spinner updates
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	// Keep spinner going if loading
	if m.loading {
		cmds = append(cmds, func() tea.Msg { return nil })
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m RepositoriesModel) View() string {
	theme := styles.CurrentTheme

	// Render tab bar at top
	view := m.tabBar.View() + "\n\n"

	// Main content area
	contentStyle := lipgloss.NewStyle().
		Width(m.width).
		AlignHorizontal(lipgloss.Left)

	if m.loading {
		// Show spinner while loading
		spinnerView := lipgloss.NewStyle().
			Padding(2).
			Width(m.width).
			Align(lipgloss.Center).
			Render(m.spinner.View() + " " + "Loading repositories...")

		view += contentStyle.Render(spinnerView)
	} else if m.mode == "view" {
		// Show repository list
		view += contentStyle.Render(m.repoList.View())
	} else {
		// Show form
		formTitle := "Add Repository"
		if m.mode == "edit" {
			formTitle = "Edit Repository"
		}

		formStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderColor).
			Padding(1).
			Width(m.width - 10).
			AlignHorizontal(lipgloss.Center)

		// Build form
		form := theme.Title.Render(formTitle) + "\n\n"

		// Inputs
		inputStyle := lipgloss.NewStyle().MarginBottom(1)
		form += inputStyle.Render("Name:") + "\n"
		form += inputStyle.Render(m.nameInput.View()) + "\n"

		form += inputStyle.Render("URL:") + "\n"
		form += inputStyle.Render(m.urlInput.View()) + "\n"

		form += inputStyle.Render("Priority:") + "\n"
		form += inputStyle.Render(m.priorityInput.View()) + "\n"

		form += inputStyle.Render("Enabled:") + "\n"
		form += inputStyle.Render(m.enabledInput.View())

		formView := lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center).
			Render(formStyle.Render(form))

		view += formView
	}

	// Add status bar at bottom
	view += "\n" + m.statusBar.View()

	return view
}
