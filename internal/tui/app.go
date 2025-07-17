// Package tui provides the terminal user interface for CloudWorkstation.
//
// This package implements a full-featured TUI using the BubbleTea framework,
// providing an interactive alternative to the command-line interface.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	// "github.com/scttfrdmn/cloudworkstation/internal/tui/api/mock" // Temporarily commented out
	"github.com/scttfrdmn/cloudworkstation/internal/tui/models"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

// App represents the TUI application
type App struct {
	apiClient *api.TUIClient
	program   *tea.Program
}

// PageID represents different pages in the TUI
type PageID int

const (
	// DashboardPage shows the main dashboard
	DashboardPage PageID = iota
	// InstancesPage shows instance management
	InstancesPage
	// TemplatesPage shows template selection
	TemplatesPage
	// StoragePage shows storage management
	StoragePage
	// SettingsPage shows application settings
	SettingsPage
	// ProfilesPage shows profile management
	ProfilesPage
)

// AppModel represents the main application model
type AppModel struct {
	apiClient     *api.TUIClient
	currentPage   PageID
	dashboardModel models.DashboardModel
	templatesModel models.TemplatesModel
	profilesModel models.ProfilesModel
	// Add other page models here
	width         int
	height        int
}

// NewApp creates a new TUI application
func NewApp() *App {
	// Create a mock client for now since we're refactoring
	// Temporarily commenting out until we fix the API client issues
	/*
	mockClient := mock.NewMockClient()
	apiClient := api.NewTUIClient(mockClient)
	*/
	
	return &App{
		apiClient: nil, // Temporarily nil until we fix API client issues
		program:   nil,
	}
}

// Run starts the TUI application
func (a *App) Run() error {
	// Create initial model
	model := AppModel{
		apiClient:     a.apiClient,
		currentPage:   DashboardPage,
		dashboardModel: models.NewDashboardModel(a.apiClient),
		templatesModel: models.NewTemplatesModel(a.apiClient),
		profilesModel: models.NewProfilesModel(a.apiClient),
	}

	// Create program with model
	a.program = tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use the full terminal screen
	)

	// Run the application
	_, err := a.program.Run()
	return err
}

// Init initializes the application model
func (m AppModel) Init() tea.Cmd {
	switch m.currentPage {
	case DashboardPage:
		return m.dashboardModel.Init()
	case TemplatesPage:
		return m.templatesModel.Init()
	case ProfilesPage:
		return m.profilesModel.Init()
	default:
		return m.dashboardModel.Init()
	}
}

// Update handles messages and updates the model
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	
	// Handle global messages
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Quit application
			return m, tea.Quit
		case "1":
			// Switch to dashboard
			m.currentPage = DashboardPage
		case "2":
			// Switch to instances page
			m.currentPage = InstancesPage
			// TODO: Initialize instances page
		case "3":
			// Switch to templates page
			m.currentPage = TemplatesPage
			{
				cmds = append(cmds, m.templatesModel.Init())
			}
		case "4":
			// Switch to storage page
			m.currentPage = StoragePage
			// TODO: Initialize storage page
		case "5":
			// Switch to settings page
			m.currentPage = SettingsPage
			// TODO: Initialize settings page
		case "6":
			// Switch to profiles page
			m.currentPage = ProfilesPage
			{
				// Initialize profiles page
				m.profilesModel.SetSize(m.width, m.height)
				cmds = append(cmds, func() tea.Msg { return models.ProfileInitMsg{} })
			}
		}
	}

	// Update current page model based on active page
	switch m.currentPage {
	case DashboardPage:
		newModel, newCmd := m.dashboardModel.Update(msg)
		m.dashboardModel = newModel.(models.DashboardModel)
		cmd = newCmd
	case TemplatesPage:
		newModel, newCmd := m.templatesModel.Update(msg)
		m.templatesModel = newModel.(models.TemplatesModel)
		cmd = newCmd
	case ProfilesPage:
		newModel, newCmd := m.profilesModel.Update(msg)
		m.profilesModel = newModel.(models.ProfilesModel)
		cmd = newCmd
	// Handle other pages here
	}

	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// View renders the application
func (m AppModel) View() string {
	// Render current page based on active page
	switch m.currentPage {
	case DashboardPage:
		return m.dashboardModel.View()
	case InstancesPage:
		return "Instances Page" // TODO: Implement instances page view
	case TemplatesPage:
		return m.templatesModel.View()
	case StoragePage:
		return "Storage Page" // TODO: Implement storage page view
	case SettingsPage:
		return "Settings Page" // TODO: Implement settings page view
	case ProfilesPage:
		return m.profilesModel.View()
	default:
		return fmt.Sprintf("CloudWorkstation v%s\n\nUnknown page", version.GetVersion())
	}
}
