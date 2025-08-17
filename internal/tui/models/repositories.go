package models

import (
	"fmt"
	"strings"

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

// Update handles UI events
func (m RepositoriesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate content area height (window height - status bar - tab bar)
		contentHeight := m.height - 2 - 3

		m.repoList.SetSize(m.width-2, contentHeight)

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "tab":
			// Cycle through tabs
			// Return to templates view
			return tea.Model(nil), tea.Quit
		}

		// Mode-specific keys
		if m.mode == "view" {
			switch msg.String() {
			case "a":
				// Switch to add mode
				m.mode = "add"
				m.nameInput.Focus()
				m.urlInput.Reset()
				m.priorityInput.SetValue("50")
				m.enabledInput.SetValue("true")
				m.focusIndex = 0
				m.statusBar.SetStatus("Enter: Submit • Esc: Cancel • Tab: Next Field", components.StatusInfo)
				return m, nil

			case "e":
				// Switch to edit mode if an item is selected
				if m.repoList.SelectedItem() != nil {
					item := m.repoList.SelectedItem().(RepositoryItem)
					m.mode = "edit"
					m.selected = item.Name
					m.nameInput.SetValue(item.Name)
					m.nameInput.Focus()
					m.urlInput.SetValue(item.URL)
					m.priorityInput.SetValue(fmt.Sprintf("%d", item.Priority))
					m.enabledInput.SetValue(fmt.Sprintf("%t", item.Enabled))
					m.focusIndex = 0
					m.statusBar.SetStatus("Enter: Submit • Esc: Cancel • Tab: Next Field", components.StatusInfo)
					return m, nil
				}

			case "d":
				// Delete the selected repository
				if m.repoList.SelectedItem() != nil {
					item := m.repoList.SelectedItem().(RepositoryItem)

					// Remove the repository
					var updatedRepos []types.TemplateRepository
					for _, repo := range m.repos {
						if repo.Name != item.Name {
							updatedRepos = append(updatedRepos, repo)
						}
					}
					m.repos = updatedRepos
					m.refreshRepositoryList()

					// Update status
					m.statusBar.SetStatus("Repository deleted: "+item.Name, components.StatusSuccess)
					return m, nil
				}

			case "r":
				// Refresh the list
				m.loading = true
				m.statusBar.SetStatus("Refreshing repository list...", components.StatusInfo)
				return m, tea.Batch(
					func() tea.Msg { return nil },
					m.fetchRepositories,
				)

			case "s":
				// Sync repositories
				m.loading = true
				m.statusBar.SetStatus("Syncing repositories...", components.StatusInfo)
				return m, tea.Batch(
					func() tea.Msg { return nil },
					m.syncRepositories,
				)
			}

			// Update the list
			var cmd tea.Cmd
			m.repoList, cmd = m.repoList.Update(msg)
			cmds = append(cmds, cmd)

		} else if m.mode == "add" || m.mode == "edit" {
			// Form editing mode
			switch msg.String() {
			case "esc":
				// Cancel and return to view mode
				m.mode = "view"
				m.statusBar.SetStatus("↑/↓: Navigate • Enter: Select • a: Add • e: Edit • r: Refresh • d: Delete", components.StatusInfo)
				return m, nil

			case "enter":
				// Submit the form
				newRepo := types.TemplateRepository{
					Name:     m.nameInput.Value(),
					URL:      m.urlInput.Value(),
					Priority: 50, // Default
					Enabled:  true,
				}

				// Parse priority if provided
				if m.priorityInput.Value() != "" {
					var priority int
					_, err := fmt.Sscanf(m.priorityInput.Value(), "%d", &priority)
					if err == nil {
						newRepo.Priority = priority
					}
				}

				// Parse enabled if provided
				if strings.ToLower(m.enabledInput.Value()) == "false" {
					newRepo.Enabled = false
				}

				if m.mode == "add" {
					// Add the new repository
					m.repos = append(m.repos, newRepo)
					m.statusBar.SetStatus("Repository added: "+newRepo.Name, components.StatusSuccess)
				} else {
					// Update the repository
					for i, repo := range m.repos {
						if repo.Name == m.selected {
							m.repos[i] = newRepo
							break
						}
					}
					m.statusBar.SetStatus("Repository updated: "+newRepo.Name, components.StatusSuccess)
				}

				m.refreshRepositoryList()
				m.mode = "view"
				m.statusBar.SetStatus("↑/↓: Navigate • Enter: Select • a: Add • e: Edit • r: Refresh • d: Delete", components.StatusInfo)
				return m, nil

			case "tab", "shift+tab":
				// Cycle through inputs
				inputs := []textinput.Model{
					m.nameInput,
					m.urlInput,
					m.priorityInput,
					m.enabledInput,
				}

				// Determine direction
				if msg.String() == "tab" {
					m.focusIndex = (m.focusIndex + 1) % len(inputs)
				} else {
					m.focusIndex = (m.focusIndex - 1 + len(inputs)) % len(inputs)
				}

				// Update focus
				for i := 0; i < len(inputs); i++ {
					if i == m.focusIndex {
						inputs[i].Focus()
					} else {
						inputs[i].Blur()
					}
				}

				m.nameInput = inputs[0]
				m.urlInput = inputs[1]
				m.priorityInput = inputs[2]
				m.enabledInput = inputs[3]

				return m, nil
			}

			// Handle input updates
			var cmd tea.Cmd
			switch m.focusIndex {
			case 0:
				m.nameInput, cmd = m.nameInput.Update(msg)
				cmds = append(cmds, cmd)
			case 1:
				m.urlInput, cmd = m.urlInput.Update(msg)
				cmds = append(cmds, cmd)
			case 2:
				m.priorityInput, cmd = m.priorityInput.Update(msg)
				cmds = append(cmds, cmd)
			case 3:
				m.enabledInput, cmd = m.enabledInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	// Handle fetch responses
	case []types.TemplateRepository:
		m.loading = false
		m.repos = msg
		m.refreshRepositoryList()
		m.statusBar.SetStatus("Repository list updated", components.StatusSuccess)

	case RefreshMsg:
		// Trigger a refresh after operations
		m.loading = true
		return m, m.fetchRepositories

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus("Error: "+m.error, components.StatusError)

		// Spinner updates
		// Skip spinner updates for now
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// If loading, keep spinner going
	if m.loading {
		spinnerCmd := func() tea.Msg { return nil }
		cmds = append(cmds, spinnerCmd)
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
