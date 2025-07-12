// Package tui provides the terminal user interface for CloudWorkstation.
//
// This package implements a full-featured TUI using the BubbleTea framework,
// providing an interactive alternative to the command-line interface.
package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/models"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

// App represents the TUI application
type App struct {
	apiClient api.CloudWorkstationAPI
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
)

// AppModel represents the main application model
type AppModel struct {
	apiClient     api.CloudWorkstationAPI
	currentPage   PageID
	dashboardModel models.DashboardModel
	templatesModel models.TemplatesModel
	// Add other page models here
	width         int
	height        int
}

// NewApp creates a new TUI application
func NewApp() *App {
	// Check for custom API URL from environment
	apiURL := os.Getenv("CWSD_URL")
	apiClient := api.NewClient(apiURL) // Uses default or environment URL
	
	return &App{
		apiClient: apiClient,
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
	default:
		return fmt.Sprintf("CloudWorkstation v%s\n\nUnknown page", version.GetVersion())
	}
}
