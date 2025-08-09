// Package tui provides the terminal user interface for CloudWorkstation.
//
// This package implements a full-featured TUI using the BubbleTea framework,
// providing an interactive alternative to the command-line interface.
package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile/core"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/models"
	pkgapi "github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
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
	instancesModel models.InstancesModel
	templatesModel models.TemplatesModel
	storageModel   models.StorageModel
	settingsModel  models.SettingsModel
	profilesModel models.ProfilesModel
	width         int
	height        int
}

// NewApp creates a new TUI application
func NewApp() *App {
	// Get current profile for API client configuration
	currentProfile, err := profile.GetCurrentProfile()
	if err != nil {
		// Use default profile if none exists
		currentProfile = &core.Profile{
			Name:       "default",
			AWSProfile: "",
			Region:     "",
		}
	}
	
	// Create API client with modern Options pattern
	apiClient := pkgapi.NewClientWithOptions("http://localhost:8947", client.Options{
		AWSProfile: currentProfile.AWSProfile,
		AWSRegion:  currentProfile.Region,
	})
	
	// Wrap with TUI client
	tuiClient := api.NewTUIClient(apiClient)
	
	return &App{
		apiClient: tuiClient,
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
		instancesModel: models.NewInstancesModel(a.apiClient),
		templatesModel: models.NewTemplatesModel(a.apiClient),
		storageModel:   models.NewStorageModel(a.apiClient),
		settingsModel:  models.NewSettingsModel(a.apiClient),
		profilesModel: models.NewProfilesModel(a.apiClient),
	}

	// Create program with explicit input/output streams for maximum compatibility
	program := tea.NewProgram(
		model,
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr), // Use stderr to avoid conflicts with stdout
	)
	
	// Store program reference
	a.program = program

	// Run the application
	_, err := program.Run()
	return err
}

// Init initializes the application model
func (m AppModel) Init() tea.Cmd {
	switch m.currentPage {
	case DashboardPage:
		return m.dashboardModel.Init()
	case InstancesPage:
		return m.instancesModel.Init()
	case TemplatesPage:
		return m.templatesModel.Init()
	case StoragePage:
		return m.storageModel.Init()
	case SettingsPage:
		return m.settingsModel.Init()
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
			{
				cmds = append(cmds, m.instancesModel.Init())
			}
		case "3":
			// Switch to templates page
			m.currentPage = TemplatesPage
			{
				cmds = append(cmds, m.templatesModel.Init())
			}
		case "4":
			// Switch to storage page
			m.currentPage = StoragePage
			{
				cmds = append(cmds, m.storageModel.Init())
			}
		case "5":
			// Switch to settings page
			m.currentPage = SettingsPage
			{
				cmds = append(cmds, m.settingsModel.Init())
			}
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
	case InstancesPage:
		newModel, newCmd := m.instancesModel.Update(msg)
		m.instancesModel = newModel.(models.InstancesModel)
		cmd = newCmd
	case TemplatesPage:
		newModel, newCmd := m.templatesModel.Update(msg)
		m.templatesModel = newModel.(models.TemplatesModel)
		cmd = newCmd
	case StoragePage:
		newModel, newCmd := m.storageModel.Update(msg)
		m.storageModel = newModel.(models.StorageModel)
		cmd = newCmd
	case SettingsPage:
		newModel, newCmd := m.settingsModel.Update(msg)
		m.settingsModel = newModel.(models.SettingsModel)
		cmd = newCmd
	case ProfilesPage:
		newModel, newCmd := m.profilesModel.Update(msg)
		m.profilesModel = newModel.(models.ProfilesModel)
		cmd = newCmd
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
		return m.instancesModel.View()
	case TemplatesPage:
		return m.templatesModel.View()
	case StoragePage:
		return m.storageModel.View()
	case SettingsPage:
		return m.settingsModel.View()
	case ProfilesPage:
		return m.profilesModel.View()
	default:
		return fmt.Sprintf("CloudWorkstation v%s\n\nUnknown page", version.GetVersion())
	}
}
