// Package tui provides the terminal user interface for CloudWorkstation.
//
// This package implements a full-featured TUI using the BubbleTea framework,
// providing an interactive alternative to the command-line interface.
package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/models"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
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
	// ProjectsPage shows project management (Phase 4 Enterprise)
	ProjectsPage
	// BudgetPage shows budget management (Phase 4 Enterprise)
	BudgetPage
	// UsersPage shows user management (Phase 5A.2)
	UsersPage
	// PolicyPage shows policy framework management (Phase 5A+)
	PolicyPage
	// MarketplacePage shows template marketplace (Phase 5B)
	MarketplacePage
	// IdlePage shows idle detection and hibernation management (Phase 3)
	IdlePage
	// AMIPage shows AMI management
	AMIPage
	// RightsizingPage shows rightsizing recommendations
	RightsizingPage
	// LogsPage shows logs viewer
	LogsPage
	// DaemonPage shows daemon management
	DaemonPage
	// SettingsPage shows application settings
	SettingsPage
	// ProfilesPage shows profile management
	ProfilesPage
)

// AppModel represents the main application model
type AppModel struct {
	apiClient        *api.TUIClient
	currentPage      PageID
	dashboardModel   models.DashboardModel
	instancesModel   models.InstancesModel
	templatesModel   models.TemplatesModel
	storageModel     models.StorageModel
	projectsModel    models.ProjectsModel
	budgetModel      models.BudgetModel
	usersModel       models.UsersModel
	policyModel      models.PolicyModel
	marketplaceModel models.MarketplaceModel
	idleModel        models.IdleModel
	amiModel         models.AMIModel
	rightsizingModel models.RightsizingModel
	logsModel        models.LogsModel
	daemonModel      models.DaemonModel
	settingsModel    models.SettingsModel
	profilesModel    models.ProfilesModel
	width            int
	height           int
}

// NewApp creates a new TUI application
func NewApp() *App {
	// Get current profile for API client configuration
	profileManager, pmErr := profile.NewManagerEnhanced()
	var currentProfile *profile.Profile
	if pmErr != nil {
		// Use default profile if manager fails to initialize
		currentProfile = &profile.Profile{
			Name:       "default",
			AWSProfile: "",
			Region:     "",
		}
	} else {
		prof, err := profileManager.GetCurrentProfile()
		if err != nil {
			// Use default profile if none exists
			currentProfile = &profile.Profile{
				Name:       "default",
				AWSProfile: "",
				Region:     "",
			}
		} else {
			currentProfile = prof
		}
	}

	// Create API client with modern Options pattern
	apiClient := client.NewClientWithOptions("http://localhost:8947", client.Options{
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
		apiClient:        a.apiClient,
		currentPage:      DashboardPage,
		dashboardModel:   models.NewDashboardModel(a.apiClient),
		instancesModel:   models.NewInstancesModel(a.apiClient),
		templatesModel:   models.NewTemplatesModel(a.apiClient),
		storageModel:     models.NewStorageModel(a.apiClient),
		projectsModel:    models.NewProjectsModel(a.apiClient),
		budgetModel:      models.NewBudgetModel(a.apiClient),
		usersModel:       models.NewUsersModel(a.apiClient),
		policyModel:      models.NewPolicyModel(a.apiClient),
		marketplaceModel: models.NewMarketplaceModel(a.apiClient),
		idleModel:        models.NewIdleModel(a.apiClient),
		amiModel:         models.NewAMIModel(a.apiClient),
		rightsizingModel: models.NewRightsizingModel(a.apiClient),
		logsModel:        models.NewLogsModel(a.apiClient),
		daemonModel:      models.NewDaemonModel(a.apiClient),
		settingsModel:    models.NewSettingsModel(a.apiClient),
		profilesModel:    models.NewProfilesModel(a.apiClient),
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
	case ProjectsPage:
		return m.projectsModel.Init()
	case BudgetPage:
		return m.budgetModel.Init()
	case UsersPage:
		return m.usersModel.Init()
	case PolicyPage:
		return m.policyModel.Init()
	case MarketplacePage:
		return m.marketplaceModel.Init()
	case IdlePage:
		return m.idleModel.Init()
	case AMIPage:
		return m.amiModel.Init()
	case RightsizingPage:
		return m.rightsizingModel.Init()
	case LogsPage:
		return m.logsModel.Init()
	case DaemonPage:
		return m.daemonModel.Init()
	case SettingsPage:
		return m.settingsModel.Init()
	case ProfilesPage:
		return m.profilesModel.Init()
	default:
		return m.dashboardModel.Init()
	}
}

// AppMessageHandler interface for handling different app messages (Command Pattern - SOLID)
type AppMessageHandler interface {
	CanHandle(msg tea.Msg) bool
	Handle(m AppModel, msg tea.Msg) (AppModel, []tea.Cmd)
}

// WindowSizeHandler handles window size messages
type WindowSizeHandler struct{}

func (h *WindowSizeHandler) CanHandle(msg tea.Msg) bool {
	_, ok := msg.(tea.WindowSizeMsg)
	return ok
}

func (h *WindowSizeHandler) Handle(m AppModel, msg tea.Msg) (AppModel, []tea.Cmd) {
	windowMsg := msg.(tea.WindowSizeMsg)
	m.width = windowMsg.Width
	m.height = windowMsg.Height
	return m, nil
}

// QuitKeyHandler handles quit key messages
type QuitKeyHandler struct{}

func (h *QuitKeyHandler) CanHandle(msg tea.Msg) bool {
	keyMsg, ok := msg.(tea.KeyMsg)
	return ok && (keyMsg.String() == "ctrl+c" || keyMsg.String() == "q")
}

func (h *QuitKeyHandler) Handle(m AppModel, msg tea.Msg) (AppModel, []tea.Cmd) {
	return m, []tea.Cmd{tea.Quit}
}

// PageNavigationHandler handles page navigation keys
type PageNavigationHandler struct{}

func (h *PageNavigationHandler) CanHandle(msg tea.Msg) bool {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}
	key := keyMsg.String()
	return key == "1" || key == "2" || key == "3" || key == "4" || key == "5" || key == "6" || key == "7" || key == "8" || key == "9" || key == "0" || key == "m" || key == "i" || key == "a" || key == "r" || key == "l" || key == "d"
}

func (h *PageNavigationHandler) Handle(m AppModel, msg tea.Msg) (AppModel, []tea.Cmd) {
	keyMsg := msg.(tea.KeyMsg)
	var cmds []tea.Cmd

	switch keyMsg.String() {
	case "1":
		m.currentPage = DashboardPage
	case "2":
		m.currentPage = InstancesPage
		cmds = append(cmds, m.instancesModel.Init())
	case "3":
		m.currentPage = TemplatesPage
		cmds = append(cmds, m.templatesModel.Init())
	case "4":
		m.currentPage = StoragePage
		cmds = append(cmds, m.storageModel.Init())
	case "5":
		m.currentPage = ProjectsPage
		cmds = append(cmds, m.projectsModel.Init())
	case "6":
		m.currentPage = BudgetPage
		cmds = append(cmds, m.budgetModel.Init())
	case "7":
		m.currentPage = UsersPage
		cmds = append(cmds, m.usersModel.Init())
	case "8":
		m.currentPage = PolicyPage
		cmds = append(cmds, m.policyModel.Init())
	case "m":
		m.currentPage = MarketplacePage
		cmds = append(cmds, m.marketplaceModel.Init())
	case "i":
		m.currentPage = IdlePage
		cmds = append(cmds, m.idleModel.Init())
	case "a":
		m.currentPage = AMIPage
		cmds = append(cmds, m.amiModel.Init())
	case "r":
		m.currentPage = RightsizingPage
		cmds = append(cmds, m.rightsizingModel.Init())
	case "l":
		m.currentPage = LogsPage
		cmds = append(cmds, m.logsModel.Init())
	case "d":
		m.currentPage = DaemonPage
		cmds = append(cmds, m.daemonModel.Init())
	case "9":
		m.currentPage = SettingsPage
		cmds = append(cmds, m.settingsModel.Init())
	case "0":
		m.currentPage = ProfilesPage
		m.profilesModel.SetSize(m.width, m.height)
		cmds = append(cmds, func() tea.Msg { return models.ProfileInitMsg{} })
	}

	return m, cmds
}

// PageModelUpdater handles updating the current page model
type PageModelUpdater struct{}

func (u *PageModelUpdater) UpdateCurrentPage(m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	switch m.currentPage {
	case DashboardPage:
		newModel, newCmd := m.dashboardModel.Update(msg)
		m.dashboardModel = newModel.(models.DashboardModel)
		return m, newCmd
	case InstancesPage:
		newModel, newCmd := m.instancesModel.Update(msg)
		m.instancesModel = newModel.(models.InstancesModel)
		return m, newCmd
	case TemplatesPage:
		newModel, newCmd := m.templatesModel.Update(msg)
		m.templatesModel = newModel.(models.TemplatesModel)
		return m, newCmd
	case StoragePage:
		newModel, newCmd := m.storageModel.Update(msg)
		m.storageModel = newModel.(models.StorageModel)
		return m, newCmd
	case ProjectsPage:
		newModel, newCmd := m.projectsModel.Update(msg)
		m.projectsModel = newModel.(models.ProjectsModel)
		return m, newCmd
	case BudgetPage:
		newModel, newCmd := m.budgetModel.Update(msg)
		m.budgetModel = newModel.(models.BudgetModel)
		return m, newCmd
	case UsersPage:
		newModel, newCmd := m.usersModel.Update(msg)
		m.usersModel = newModel.(models.UsersModel)
		return m, newCmd
	case PolicyPage:
		newModel, newCmd := m.policyModel.Update(msg)
		m.policyModel = newModel.(models.PolicyModel)
		return m, newCmd
	case MarketplacePage:
		newModel, newCmd := m.marketplaceModel.Update(msg)
		m.marketplaceModel = newModel.(models.MarketplaceModel)
		return m, newCmd
	case IdlePage:
		newModel, newCmd := m.idleModel.Update(msg)
		m.idleModel = newModel.(models.IdleModel)
		return m, newCmd
	case AMIPage:
		newModel, newCmd := m.amiModel.Update(msg)
		m.amiModel = newModel.(models.AMIModel)
		return m, newCmd
	case RightsizingPage:
		newModel, newCmd := m.rightsizingModel.Update(msg)
		m.rightsizingModel = newModel.(models.RightsizingModel)
		return m, newCmd
	case LogsPage:
		newModel, newCmd := m.logsModel.Update(msg)
		m.logsModel = newModel.(models.LogsModel)
		return m, newCmd
	case DaemonPage:
		newModel, newCmd := m.daemonModel.Update(msg)
		m.daemonModel = newModel.(models.DaemonModel)
		return m, newCmd
	case SettingsPage:
		newModel, newCmd := m.settingsModel.Update(msg)
		m.settingsModel = newModel.(models.SettingsModel)
		return m, newCmd
	case ProfilesPage:
		newModel, newCmd := m.profilesModel.Update(msg)
		m.profilesModel = newModel.(models.ProfilesModel)
		return m, newCmd
	}
	return m, nil
}

// AppMessageDispatcher manages app message handlers (Command Pattern - SOLID)
type AppMessageDispatcher struct {
	handlers []AppMessageHandler
	updater  *PageModelUpdater
}

// NewAppMessageDispatcher creates app message dispatcher
func NewAppMessageDispatcher() *AppMessageDispatcher {
	return &AppMessageDispatcher{
		handlers: []AppMessageHandler{
			&WindowSizeHandler{},
			&QuitKeyHandler{},
			&PageNavigationHandler{},
		},
		updater: &PageModelUpdater{},
	}
}

// Dispatch processes message using appropriate handler
func (d *AppMessageDispatcher) Dispatch(m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	var allCmds []tea.Cmd

	// Try global handlers first
	for _, handler := range d.handlers {
		if handler.CanHandle(msg) {
			newModel, cmds := handler.Handle(m, msg)
			if cmds != nil {
				allCmds = append(allCmds, cmds...)
			}
			m = newModel
			break
		}
	}

	// Update current page model
	newModel, pageCmd := d.updater.UpdateCurrentPage(m, msg)
	if pageCmd != nil {
		allCmds = append(allCmds, pageCmd)
	}

	return newModel, tea.Batch(allCmds...)
}

// Update handles messages using Command Pattern (SOLID: Single Responsibility)
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	dispatcher := NewAppMessageDispatcher()
	newModel, cmd := dispatcher.Dispatch(m, msg)
	return newModel, cmd
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
	case ProjectsPage:
		return m.projectsModel.View()
	case BudgetPage:
		return m.budgetModel.View()
	case UsersPage:
		return m.usersModel.View()
	case PolicyPage:
		return m.policyModel.View()
	case MarketplacePage:
		return m.marketplaceModel.View()
	case IdlePage:
		return m.idleModel.View()
	case AMIPage:
		return m.amiModel.View()
	case RightsizingPage:
		return m.rightsizingModel.View()
	case LogsPage:
		return m.logsModel.View()
	case DaemonPage:
		return m.daemonModel.View()
	case SettingsPage:
		return m.settingsModel.View()
	case ProfilesPage:
		return m.profilesModel.View()
	default:
		return fmt.Sprintf("CloudWorkstation v%s\n\nUnknown page", version.GetVersion())
	}
}
