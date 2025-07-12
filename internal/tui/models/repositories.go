package models

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"
	
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// RepoItem represents a repository in the list
type RepoItem struct {
	name         string
	url          string
	priority     int
	enabled      bool
	templateCount int
	lastSync     time.Time
}

// FilterValue returns the value to filter on in the list
func (r RepoItem) FilterValue() string { return r.name }

// Title returns the name of the repository
func (r RepoItem) Title() string { 
	if !r.enabled {
		return r.name + " (disabled)"
	}
	return r.name
}

// Description returns a short description of the repository
func (r RepoItem) Description() string {
	status := "Enabled"
	if !r.enabled {
		status = "Disabled"
	}
	
	// Format the last sync time
	syncTime := "Never"
	if !r.lastSync.IsZero() {
		syncTime = r.lastSync.Format("2006-01-02 15:04")
	}
	
	return fmt.Sprintf("Priority: %d | Templates: %d | Last Sync: %s | %s",
		r.priority, r.templateCount, syncTime, status)
}

// RepositoriesModel represents the repository management view
type RepositoriesModel struct {
	apiClient     api.CloudWorkstationAPI
	repoList      list.Model
	statusBar     components.StatusBar
	spinner       components.Spinner
	tabs          components.TabBar
	width         int
	height        int
	loading       bool
	error         string
	repos         []types.TemplateRepository
	selected      string
	editing       bool
	nameInput     textinput.Model
	urlInput      textinput.Model
	priorityInput textinput.Model
	enabledInput  textinput.Model
	currentInput  int
	mode          string // view, add, edit
}

// NewRepositoriesModel creates a new repositories model
func NewRepositoriesModel(apiClient api.CloudWorkstationAPI) RepositoriesModel {
	theme := styles.CurrentTheme
	
	// Set up repository list
	repoList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	repoList.Title = "Template Repositories"
	repoList.Styles.Title = theme.Title
	repoList.Styles.PaginationStyle = theme.Pagination
	repoList.Styles.HelpStyle = theme.Help
	
	// Create input fields
	nameInput := textinput.New()
	nameInput.Placeholder = "Repository Name"
	nameInput.Width = 30
	
	urlInput := textinput.New()
	urlInput.Placeholder = "Repository URL (e.g., https://github.com/username/repo)"
	urlInput.Width = 50
	
	priorityInput := textinput.New()
	priorityInput.Placeholder = "Priority (0-100, lower is higher priority)"
	priorityInput.Width = 10
	
	enabledInput := textinput.New()
	enabledInput.Placeholder = "Enabled (true/false)"
	enabledInput.Width = 10
	
	// Create status bar and spinner
	statusBar := components.NewStatusBar("Repository Management", "")
	spinner := components.NewSpinner("Loading repositories...")
	
	// Create tabs
	tabs := components.NewTabBar(
		[]components.TabItem{
			{ID: "view", Title: "View"},
			{ID: "add", Title: "Add Repository"},
			{ID: "sync", Title: "Sync"},
		},
		"view",
	)
	
	return RepositoriesModel{
		apiClient:     apiClient,
		repoList:      repoList,
		statusBar:     statusBar,
		spinner:       spinner,
		tabs:          tabs,
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
	repos, err := m.apiClient.ListRepositories(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}
	return repos
}

// syncRepositories triggers a repository sync
func (m RepositoriesModel) syncRepositories() tea.Msg {
	err := m.apiClient.SyncRepositories(context.Background())
	if err != nil {
		return fmt.Errorf("failed to sync repositories: %w", err)
	}
	
	// Refresh the repository list after sync
	return RefreshMsg{}
}

// addRepository adds a new repository
func (m RepositoriesModel) addRepository() tea.Msg {
	// Parse priority value
	priority, err := strconv.Atoi(m.priorityInput.Value())
	if err != nil {
		return fmt.Errorf("invalid priority value: %w", err)
	}
	
	// Parse enabled value
	enabled := true
	if m.enabledInput.Value() == "false" {
		enabled = false
	}
	
	// Validate inputs
	if m.nameInput.Value() == "" {
		return fmt.Errorf("repository name is required")
	}
	
	if m.urlInput.Value() == "" {
		return fmt.Errorf("repository URL is required")
	}
	
	// Create update request
	req := types.TemplateRepositoryUpdate{
		Name:     m.nameInput.Value(),
		URL:      m.urlInput.Value(),
		Priority: priority,
		Enabled:  enabled,
	}
	
	// Add or update repository
	var err error
	if m.mode == "add" {
		err = m.apiClient.AddRepository(context.Background(), req)
		if err != nil {
			return fmt.Errorf("failed to add repository: %w", err)
		}
	} else {
		err = m.apiClient.UpdateRepository(context.Background(), req)
		if err != nil {
			return fmt.Errorf("failed to update repository: %w", err)
		}
	}
	
	// Reset inputs
	m.nameInput.SetValue("")
	m.urlInput.SetValue("")
	m.priorityInput.SetValue("0")
	m.enabledInput.SetValue("true")
	
	// Switch back to view mode
	m.mode = "view"
	m.tabs.SetActiveTab("view")
	
	// Refresh the repository list
	return RefreshMsg{}
}

// removeRepository removes a repository
func (m RepositoriesModel) removeRepository() tea.Msg {
	if m.selected == "" {
		return fmt.Errorf("no repository selected")
	}
	
	err := m.apiClient.RemoveRepository(context.Background(), m.selected)
	if err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}
	
	// Reset selection
	m.selected = ""
	
	// Refresh the repository list
	return RefreshMsg{}
}

// Update handles messages and updates the model
func (m RepositoriesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		m.tabs.SetWidth(msg.Width - 4)
		
		// Update list height to account for tabs and details panel
		listHeight := m.height - 15 // tabs + details + status bar + help
		if listHeight < 3 {
			listHeight = 3
		}
		m.repoList.SetHeight(listHeight)
		m.repoList.SetWidth(m.width - 4)
		
		return m, nil
		
	case tea.KeyMsg:
		// Handle form editing keys
		if m.mode == "add" || m.mode == "edit" {
			switch msg.String() {
			case "enter":
				// Move to next input or submit form
				if m.currentInput < 3 {
					m.currentInput++
					m.focusCurrentInput()
					return m, nil
				}
				
				// Submit form
				return m, m.addRepository
				
			case "esc":
				// Cancel editing
				m.mode = "view"
				m.tabs.SetActiveTab("view")
				return m, nil
				
			case "tab":
				// Switch between inputs
				m.currentInput = (m.currentInput + 1) % 4
				m.focusCurrentInput()
				return m, nil
				
			case "shift+tab":
				// Switch between inputs (reverse)
				m.currentInput = (m.currentInput + 3) % 4
				m.focusCurrentInput()
				return m, nil
			}
			
			// Handle input updates
			var cmd tea.Cmd
			switch m.currentInput {
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
			
			return m, tea.Batch(cmds...)
		}
		
		// Handle view mode keys
		switch msg.String() {
		case "r":
			m.loading = true
			m.error = ""
			return m, m.fetchRepositories
			
		case "s":
			if m.mode == "view" {
				m.loading = true
				m.error = ""
				return m, m.syncRepositories
			}
			
		case "a":
			if m.mode == "view" {
				m.mode = "add"
				m.tabs.SetActiveTab("add")
				m.currentInput = 0
				m.nameInput.SetValue("")
				m.urlInput.SetValue("")
				m.priorityInput.SetValue("0")
				m.enabledInput.SetValue("true")
				m.focusCurrentInput()
				return m, nil
			}
			
		case "e":
			if m.mode == "view" && m.selected != "" {
				m.mode = "edit"
				
				// Find the selected repository
				for _, repo := range m.repos {
					if repo.Name == m.selected {
						m.nameInput.SetValue(repo.Name)
						m.urlInput.SetValue(repo.URL)
						m.priorityInput.SetValue(strconv.Itoa(repo.Priority))
						m.enabledInput.SetValue(strconv.FormatBool(repo.Enabled))
						break
					}
				}
				
				m.tabs.SetActiveTab("add") // Reuse add tab for editing
				m.currentInput = 0
				m.focusCurrentInput()
				return m, nil
			}
			
		case "d":
			if m.mode == "view" && m.selected != "" {
				return m, m.removeRepository
			}
			
		case "q", "esc":
			if m.mode != "view" {
				m.mode = "view"
				m.tabs.SetActiveTab("view")
				return m, nil
			}
			return m, tea.Quit
		}
		
		// Update list selection
		if m.mode == "view" && !m.loading {
			var cmd tea.Cmd
			m.repoList, cmd = m.repoList.Update(msg)
			cmds = append(cmds, cmd)
			
			if i, ok := m.repoList.SelectedItem().(RepoItem); ok {
				m.selected = i.name
			}
		}
		
	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchRepositories
		
	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
		
	case []types.TemplateRepository:
		m.loading = false
		m.repos = msg
		
		// Sort repositories by priority (lowest first)
		sort.Slice(m.repos, func(i, j int) bool {
			return m.repos[i].Priority < m.repos[j].Priority
		})
		
		// Update repository list items
		var items []list.Item
		for _, repo := range m.repos {
			items = append(items, RepoItem{
				name:         repo.Name,
				url:          repo.URL,
				priority:     repo.Priority,
				enabled:      repo.Enabled,
				templateCount: repo.TemplateCount,
				lastSync:     repo.LastSync,
			})
		}
		
		m.repoList.SetItems(items)
		m.statusBar.SetStatus("Repositories loaded", components.StatusSuccess)
		
		// Select first item if none selected
		if len(items) > 0 && m.selected == "" {
			m.selected = items[0].(RepoItem).name
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

// focusCurrentInput focuses the current input field
func (m *RepositoriesModel) focusCurrentInput() {
	m.nameInput.Blur()
	m.urlInput.Blur()
	m.priorityInput.Blur()
	m.enabledInput.Blur()
	
	switch m.currentInput {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.urlInput.Focus()
	case 2:
		m.priorityInput.Focus()
	case 3:
		m.enabledInput.Focus()
	}
}

// View renders the repositories view
func (m RepositoriesModel) View() string {
	theme := styles.CurrentTheme
	
	// Title section
	title := theme.Title.Render("CloudWorkstation Template Repositories")
	
	// Tab bar
	tabBar := m.tabs.View()
	
	// Content area
	var content string
	
	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8). // Account for title, tabs, and status
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else if m.mode == "add" || m.mode == "edit" {
		// Form view for adding or editing repositories
		modeText := "Add New Repository"
		if m.mode == "edit" {
			modeText = "Edit Repository: " + m.selected
		}
		
		formPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render(modeText),
				"",
				"Name: " + m.nameInput.View(),
				"",
				"URL: " + m.urlInput.View(),
				"",
				"Priority: " + m.priorityInput.View(),
				"",
				"Enabled: " + m.enabledInput.View(),
				"",
				"Press Enter to save, Esc to cancel",
			),
		)
		
		content = formPanel
	} else if m.mode == "view" {
		// Repository list view
		listPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.repoList.View(),
			),
		)
		
		// Detail panel for selected repository
		var detailPanel string
		for _, repo := range m.repos {
			if repo.Name == m.selected {
				syncTime := "Never"
				if !repo.LastSync.IsZero() {
					syncTime = repo.LastSync.Format("2006-01-02 15:04:05")
				}
				
				status := "Enabled"
				if !repo.Enabled {
					status = "Disabled"
				}
				
				detailPanel = theme.Panel.Copy().Width(m.width - 4).Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						theme.PanelHeader.Render("Repository Details"),
						"",
						"Name: " + repo.Name,
						"URL: " + repo.URL,
						"Priority: " + strconv.Itoa(repo.Priority),
						"Status: " + status,
						"Templates: " + strconv.Itoa(repo.TemplateCount),
						"Last Synchronized: " + syncTime,
						"",
						"Press 'e' to edit, 'd' to delete, 's' to sync all repositories",
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
	} else if m.mode == "sync" {
		// Sync view
		syncPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Repository Synchronization"),
				"",
				"Press 's' to synchronize all template repositories.",
				"",
				"Last Sync: " + (func() string {
					for _, repo := range m.repos {
						if !repo.LastSync.IsZero() {
							return repo.LastSync.Format("2006-01-02 15:04:05")
						}
					}
					return "Never"
				})(),
				"",
				"This will download the latest templates from all enabled repositories.",
			),
		)
		
		content = syncPanel
	}
	
	// Help text
	var help string
	if m.mode == "add" || m.mode == "edit" {
		help = theme.Help.Render("enter: next/save • esc: cancel • tab: next field • shift+tab: previous field")
	} else {
		help = theme.Help.Render("r: refresh • a: add • e: edit • d: delete • s: sync • q: quit")
	}
	
	// Join everything together
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		tabBar,
		content,
		"",
		m.statusBar.View(),
		help,
	)
}