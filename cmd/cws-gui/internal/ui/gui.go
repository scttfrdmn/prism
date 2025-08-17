package ui

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	fynecontainer "fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/internal/ui/dashboard"
	"github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/internal/ui/instances"
	"github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/internal/ui/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// GUI represents the main GUI application using SOLID principles
type GUI struct {
	*CloudWorkstationGUI
	sectionManager *SectionManager
}

// NewGUI creates a new GUI application
func NewGUI() (*GUI, error) {
	// Create Fyne app
	fyneApp := app.NewWithID("com.cloudworkstation.gui")
	fyneApp.SetIcon(fyne.NewStaticResource("icon", nil))
	
	// Create main window
	window := fyneApp.NewWindow("CloudWorkstation")
	window.SetMaster()
	window.Resize(fyne.NewSize(1200, 800))
	
	// Initialize API client
	apiClient := client.NewClient("http://localhost:8947")
	
	// Create main GUI struct
	gui := &CloudWorkstationGUI{
		app:       fyneApp,
		window:    window,
		apiClient: apiClient,
	}
	
	// Load profiles
	if err := gui.loadProfiles(); err != nil {
		log.Printf("Warning: Failed to load profiles: %v", err)
	}
	
	// Create section manager
	sectionManager := NewSectionManager(gui)
	
	// Register sections (Open/Closed Principle - easy to add new sections)
	sectionManager.RegisterSection(SectionDashboard, dashboard.NewDashboard(apiClient))
	sectionManager.RegisterSection(SectionInstances, instances.NewInstances(apiClient, window))
	sectionManager.RegisterSection(SectionTemplates, templates.NewTemplates(apiClient, window))
	// TODO: Add volumes and settings sections
	
	wrappedGUI := &GUI{
		CloudWorkstationGUI: gui,
		sectionManager:      sectionManager,
	}
	
	// Setup UI
	if err := wrappedGUI.setupUI(); err != nil {
		return nil, fmt.Errorf("failed to setup UI: %w", err)
	}
	
	return wrappedGUI, nil
}

// Run starts the GUI application
func (g *GUI) Run() {
	// Setup system tray if supported
	if desk, ok := g.app.(desktop.App); ok {
		g.setupSystemTray(desk)
	}
	
	// Start auto-refresh
	g.startAutoRefresh()
	
	// Show window and run
	g.window.ShowAndRun()
}

// setupUI initializes the main UI layout
func (g *GUI) setupUI() error {
	// Create main layout containers
	g.sidebar = fynecontainer.NewVBox()
	g.content = fynecontainer.NewVBox()
	g.notification = fynecontainer.NewVBox()
	
	// Setup navigation sidebar
	g.setupSidebar()
	
	// Setup initial content (dashboard)
	g.navigateToSection(SectionDashboard)
	
	// Main layout
	mainContent := fynecontainer.NewBorder(
		g.notification, // top
		nil,            // bottom
		g.sidebar,      // left
		nil,            // right
		g.content,      // center
	)
	
	g.window.SetContent(mainContent)
	
	return nil
}

// setupSidebar creates the navigation sidebar
func (g *GUI) setupSidebar() {
	// Navigation buttons
	dashboardBtn := g.createNavButton("📊 Dashboard", SectionDashboard)
	instancesBtn := g.createNavButton("🖥️ Instances", SectionInstances)
	templatesBtn := g.createNavButton("📋 Templates", SectionTemplates)
	volumesBtn := g.createNavButton("💾 Volumes", SectionVolumes)
	settingsBtn := g.createNavButton("⚙️ Settings", SectionSettings)
	
	// Add to sidebar
	g.sidebar.Add(widget.NewLabel("Navigation"))
	g.sidebar.Add(widget.NewSeparator())
	g.sidebar.Add(dashboardBtn)
	g.sidebar.Add(instancesBtn)
	g.sidebar.Add(templatesBtn)
	g.sidebar.Add(volumesBtn)
	g.sidebar.Add(settingsBtn)
}

// createNavButton creates a navigation button
func (g *GUI) createNavButton(label string, section NavigationSection) *widget.Button {
	btn := widget.NewButton(label, func() {
		g.navigateToSection(section)
	})
	
	return btn
}

// navigateToSection switches to the specified section
func (g *GUI) navigateToSection(section NavigationSection) {
	g.currentSection = section
	
	// Use section manager to handle navigation
	if err := g.sectionManager.NavigateToSection(section); err != nil {
		// Fallback to placeholder for unimplemented sections
		g.showPlaceholder(section)
	}
}

// showPlaceholder shows a placeholder for unimplemented sections
func (g *GUI) showPlaceholder(section NavigationSection) {
	sectionName := g.getSectionName(section)
	placeholder := widget.NewLabel(fmt.Sprintf("%s section coming soon!", sectionName))
	
	g.content.Objects = []fyne.CanvasObject{placeholder}
	g.content.Refresh()
}

// getSectionName returns the display name for a section
func (g *GUI) getSectionName(section NavigationSection) string {
	switch section {
	case SectionDashboard:
		return "Dashboard"
	case SectionInstances:
		return "Instances"
	case SectionTemplates:
		return "Templates"
	case SectionVolumes:
		return "Volumes"
	case SectionSettings:
		return "Settings"
	default:
		return "Unknown"
	}
}

// setupSystemTray configures system tray integration
func (g *GUI) setupSystemTray(desk desktop.App) {
	// Create system tray menu
	menu := fyne.NewMenu("CloudWorkstation",
		fyne.NewMenuItem("Show", func() {
			g.window.Show()
		}),
		fyne.NewMenuItem("Dashboard", func() {
			g.navigateToSection(SectionDashboard)
			g.window.Show()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			g.app.Quit()
		}),
	)
	
	desk.SetSystemTrayMenu(menu)
}

// startAutoRefresh starts automatic data refresh
func (g *GUI) startAutoRefresh() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				g.refreshCurrentSection()
			}
		}
	}()
}

// refreshCurrentSection refreshes the currently active section
func (g *GUI) refreshCurrentSection() {
	section := g.sectionManager.GetSection(g.currentSection)
	if section != nil {
		if err := section.UpdateView(); err != nil {
			log.Printf("Failed to refresh %s: %v", section.GetTitle(), err)
		}
	}
}

// loadProfiles loads available CloudWorkstation profiles
func (g *CloudWorkstationGUI) loadProfiles() error {
	var err error
	g.profileManager, err = profile.NewManagerEnhanced()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}
	
	g.stateManager, err = profile.NewProfileAwareStateManager(g.profileManager)
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}
	
	// Get current profile
	currentProfile, err := profile.GetCurrentProfile()
	if err != nil {
		log.Printf("Warning: No active profile found, using defaults: %v", err)
		// Create default profile
		g.activeProfile = &profile.Profile{
			Name:       "default",
			AWSProfile: "default",
			Region:     "us-west-2",
		}
	} else {
		// Convert from core.Profile to profile.Profile
		g.activeProfile = &profile.Profile{
			Name:       currentProfile.Name,
			AWSProfile: currentProfile.AWSProfile,
			Region:     currentProfile.Region,
		}
	}
	
	// Configure API client with profile
	if httpClient, ok := g.apiClient.(*client.HTTPClient); ok {
		httpClient.SetOptions(client.Options{
			AWSProfile: g.activeProfile.AWSProfile,
			AWSRegion:  g.activeProfile.Region,
		})
	}
	
	return nil
}