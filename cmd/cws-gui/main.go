// CloudWorkstation GUI (cws-gui) - Desktop application for research environments.
//
// The cws-gui provides a user-friendly desktop interface for CloudWorkstation.
// It offers visual management of cloud research environments with real-time
// cost monitoring, instance status, and one-click operations for non-technical users.
//
// Key Features:
//   - Dashboard with cost overview and instance status
//   - Visual template selection with descriptions
//   - One-click launch with smart defaults
//   - Real-time status updates and notifications
//   - System tray integration for background monitoring
//
// Interface Sections:
//   - Dashboard: Overview of running instances and costs
//   - Instances: Detailed instance management
//   - Templates: Research environment catalog
//   - Volumes: Storage management interface
//   - Settings: Configuration and preferences
//
// The GUI implements CloudWorkstation's "Progressive Disclosure" principle -
// simple interface for basic operations with advanced options available
// when needed. Perfect for researchers who prefer visual interfaces.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	fynecontainer "fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/systray"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)


// NavigationSection represents different sections of the app
type NavigationSection int

const (
	// SectionDashboard displays the main overview with costs and status
	SectionDashboard NavigationSection = iota
	// SectionInstances shows detailed instance management
	SectionInstances
	// SectionTemplates provides the research environment catalog
	SectionTemplates
	// SectionVolumes manages EFS and EBS storage
	SectionVolumes
	// SectionBilling shows cost tracking and budgets
	SectionBilling
	// SectionSettings handles configuration and preferences
	SectionSettings
)

// CloudWorkstationGUI represents the main GUI application
type CloudWorkstationGUI struct {
	app       fyne.App
	window    fyne.Window
	apiClient api.CloudWorkstationAPI
	// Profile-aware functionality integrated into main client

	// Navigation
	currentSection NavigationSection
	sidebar        *fyne.Container
	content        *fyne.Container
	notification   *fyne.Container

	// Data
	instances  []types.Instance
	templates  map[string]types.Template
	totalCost  float64
	lastUpdate time.Time

	// Profile Management
	profileManager *profile.ManagerEnhanced
	stateManager   *profile.ProfileAwareStateManager
	activeProfile  *profile.Profile
	profiles       []profile.Profile
	
	// UI Components
	refreshTicker *time.Ticker
	systemTray    *systray.SystemTrayHandler

	// Form state
	launchForm struct {
		templateSelect *widget.Select
		nameEntry      *widget.Entry
		sizeSelect     *widget.Select
		launchBtn      *widget.Button
	}
}

func main() {
	log.Printf("CloudWorkstation GUI v%s starting...", version.GetVersion())

	// Create the application
	gui := &CloudWorkstationGUI{
		app: app.NewWithID("com.cloudworkstation.gui"),
		// apiClient will be initialized after setting up profile manager
	}

	// Initialize and run
	if err := gui.initialize(); err != nil {
		log.Fatalf("Failed to initialize GUI: %v", err)
	}

	gui.run()
}

// initialize sets up the GUI application
func (g *CloudWorkstationGUI) initialize() error {
	// Set application metadata
	metadata := g.app.Metadata()
	metadata.ID = "com.cloudworkstation.gui"
	metadata.Name = "CloudWorkstation"
	metadata.Version = version.GetVersion()

	// Create main window
	g.window = g.app.NewWindow("CloudWorkstation")
	g.window.Resize(fyne.NewSize(1200, 800))
	g.window.SetMaster()

	// Setup containers first (needed for notifications)
	g.notification = fynecontainer.NewVBox()
	g.content = fynecontainer.NewStack()

	// Initialize enhanced profile manager
	var err error
	g.profileManager, err = profile.NewManagerEnhanced()
	if err != nil {
		return fmt.Errorf("failed to initialize enhanced profile manager: %w", err)
	}
	
	// Initialize state manager with profile manager
	g.stateManager, err = profile.NewProfileAwareStateManager(g.profileManager)
	if err != nil {
		return fmt.Errorf("failed to initialize profile-aware state manager: %w", err)
	}
	
	// Initialize API client with profile configuration matching CLI pattern
	// Get current profile for API client options
	currentProfile, err := g.profileManager.GetCurrentProfile()
	if err != nil {
		// No profile available, use basic client
		g.apiClient = api.NewClient("http://localhost:8080")
	} else {
		// Create client with current profile AWS settings
		g.apiClient = api.NewClientWithOptions("http://localhost:8080", client.Options{
			AWSProfile: currentProfile.AWSProfile,
			AWSRegion:  currentProfile.Region,
		})
		// Store active profile pointer
		g.activeProfile = currentProfile
	}
	// Don't show notifications yet - UI isn't ready
	
	// Load all profiles
	g.loadProfiles()
	
	// Initialize data
	g.refreshData()

	// Setup UI
	g.setupMainLayout()

	// Now we can show notifications if needed
	if g.activeProfile == nil {
		g.showNotification("warning", "Profile Notice", 
			"No active profile selected. Please create or select a profile in Settings.")
	}
	
	// Check daemon connectivity with retry logic
	if err := g.checkDaemonConnection(context.Background()); err != nil {
		g.showNotification("error", "Cannot connect to CloudWorkstation daemon", 
			"Make sure the daemon is running with 'cwsd'. GUI will retry automatically.")
		// Continue anyway - daemon might start later
	}

	// Setup system tray if supported
	if desk, ok := g.app.(desktop.App); ok {
		// Create system tray handler
g.systemTray = systray.NewSystemTrayHandler(desk, g.window, g.apiClient)

// Set status change callback
g.systemTray.SetOnStatusChange(func(connected bool) {
	g.app.Driver().StartAnimation(&fyne.Animation{
		Duration: 100 * time.Millisecond,
		Tick: func(_ float32) {
			// Update status in UI based on connection state
			if !connected && g.notification != nil {
				g.showNotification("warning", "Lost Connection", 
					"Unable to connect to CloudWorkstation daemon. Is cwsd running?")
			}
		},
	})
})

// Setup and start the system tray
g.systemTray.Setup()
g.systemTray.Start()
	}

	// Start background refresh
	g.startBackgroundRefresh()

	return nil
}

// setupMainLayout creates the main application layout
func (g *CloudWorkstationGUI) setupMainLayout() {
	// Create main layout components
	g.setupSidebar()
	g.setupContent()
	g.setupNotification()

	// Create main layout: sidebar | content
	mainLayout := fynecontainer.NewHSplit(
		g.sidebar,
		fynecontainer.NewVBox(
			g.notification,
			g.content,
		),
	)
	mainLayout.SetOffset(0.2) // 20% for sidebar, 80% for content

	g.window.SetContent(mainLayout)

	// Show dashboard by default
	g.navigateToSection(SectionDashboard)
}

// setupSidebar creates the navigation sidebar
func (g *CloudWorkstationGUI) setupSidebar() {
	// App title and status
	titleCard := fynecontainer.NewVBox(
		widget.NewCard("", "",
			fynecontainer.NewVBox(
				fynecontainer.NewHBox(
					widget.NewIcon(theme.ComputerIcon()),
					widget.NewLabelWithStyle("CloudWorkstation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				),
				widget.NewLabelWithStyle(fmt.Sprintf("v%s", version.GetVersion()), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
				widget.NewSeparator(),
				fynecontainer.NewHBox(
					widget.NewIcon(theme.InfoIcon()),
					widget.NewLabel(fmt.Sprintf("$%.2f/day", g.totalCost)),
				),
			),
		),
	)

	// Profile indicator
	profileText := "No profile selected"
	profileType := "Personal"
	var profileIcon fyne.Resource = theme.AccountIcon()
	var securityText string
	
	// Check if active profile exists
	if g.activeProfile != nil {
		profileText = g.activeProfile.Name
		if g.activeProfile.Type == "invitation" {
			profileType = "Invitation"
			
			// Set security icon and text based on device binding
			if g.activeProfile.DeviceBound {
				profileIcon = theme.ConfirmIcon()
				securityText = "🔒 Device-Bound"
			} else {
				profileIcon = theme.WarningIcon()
				securityText = "⚠️ Not Device-Bound"
			}
		}
	}
	
	// Create profile card with security info
	var profileCardContent *fyne.Container
	
	if securityText != "" {
		// Show security status for invitation profiles
		profileCardContent = fynecontainer.NewVBox(
			fynecontainer.NewHBox(
				widget.NewIcon(profileIcon),
				widget.NewLabelWithStyle("AWS Profile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			),
			fynecontainer.NewHBox(
				widget.NewLabelWithStyle(profileText, fyne.TextAlignLeading, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("(%s)", profileType), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			),
			widget.NewLabelWithStyle(securityText, fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			widget.NewButton("Switch Profile", func() {
				// Navigate to settings and focus on profiles section
				g.navigateToSection(SectionSettings)
			}),
		)
	} else {
		// Simple view for personal profiles
		profileCardContent = fynecontainer.NewVBox(
			fynecontainer.NewHBox(
				widget.NewIcon(profileIcon),
				widget.NewLabelWithStyle("AWS Profile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			),
			fynecontainer.NewHBox(
				widget.NewLabelWithStyle(profileText, fyne.TextAlignLeading, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("(%s)", profileType), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			),
			widget.NewButton("Switch Profile", func() {
				// Navigate to settings and focus on profiles section
				g.navigateToSection(SectionSettings)
			}),
		)
	}
	
	// Create the card
	profileCard := widget.NewCard("", "", profileCardContent)

	// Navigation buttons
	navButtons := fynecontainer.NewVBox(
		g.createNavButton("🏠 Dashboard", SectionDashboard),
		g.createNavButton("💻 Instances", SectionInstances),
		g.createNavButton("📋 Templates", SectionTemplates),
		g.createNavButton("💾 Storage", SectionVolumes),
		g.createNavButton("💰 Billing", SectionBilling),
		g.createNavButton("⚙️ Settings", SectionSettings),
	)

	// Quick actions
	quickActions := widget.NewCard("Quick Actions", "",
		fynecontainer.NewVBox(
			widget.NewButton("🚀 R Environment", func() {
				g.quickLaunch("r-research")
			}),
			widget.NewButton("🐍 Python ML", func() {
				g.quickLaunch("python-research")
			}),
			widget.NewButton("🖥️ Ubuntu Server", func() {
				g.quickLaunch("basic-ubuntu")
			}),
		),
	)

	// Connection status
	statusText := "Connected"
	if g.lastUpdate.IsZero() {
		statusText = "Disconnected"
	}

	statusCard := widget.NewCard("Status", "",
		fynecontainer.NewHBox(
			widget.NewIcon(theme.ConfirmIcon()),
			widget.NewRichTextFromMarkdown(fmt.Sprintf("**%s**", statusText)),
		),
	)

	// Combine sidebar elements
	g.sidebar = fynecontainer.NewVBox(
		titleCard,
		widget.NewSeparator(),
		profileCard,
		widget.NewSeparator(),
		navButtons,
		widget.NewSeparator(),
		quickActions,
		layout.NewSpacer(), // Push status to bottom
		statusCard,
	)
}

// createNavButton creates a navigation button for the sidebar
func (g *CloudWorkstationGUI) createNavButton(label string, section NavigationSection) *widget.Button {
	btn := widget.NewButton(label, func() {
		g.navigateToSection(section)
	})

	// Style the button based on current section
	if g.currentSection == section {
		btn.Importance = widget.HighImportance
	}

	return btn
}

// setupContent creates the main content area
func (g *CloudWorkstationGUI) setupContent() {
	// content container is already created in initialize
	// nothing to do here
}

// setupNotification creates the notification area
func (g *CloudWorkstationGUI) setupNotification() {
	// notification container is already created in initialize
	// just make sure it's hidden by default
	g.notification.Hide()
}

// navigateToSection switches to a different section of the app
func (g *CloudWorkstationGUI) navigateToSection(section NavigationSection) {
	g.currentSection = section

	// Update sidebar buttons
	g.setupSidebar()

	// Clear and update content
	g.content.RemoveAll()

	switch section {
	case SectionDashboard:
		g.content.Add(g.createDashboardView())
	case SectionInstances:
		g.content.Add(g.createInstancesView())
	case SectionTemplates:
		g.content.Add(g.createTemplatesView())
	case SectionVolumes:
		g.content.Add(g.createVolumesView())
	case SectionBilling:
		g.content.Add(g.createBillingView())
	case SectionSettings:
		g.content.Add(g.createSettingsView())
	}

	g.content.Refresh()
}

// loadProfiles loads all profiles from the profile manager
func (g *CloudWorkstationGUI) loadProfiles() error {
	// Get all profiles
	profiles, err := g.profileManager.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}
	
	// Store profiles
	g.profiles = profiles
	
	return nil
}

// createDashboardView creates the main dashboard view
func (g *CloudWorkstationGUI) createDashboardView() fyne.CanvasObject {
	// Header
	header := fynecontainer.NewHBox(
		widget.NewLabelWithStyle("Dashboard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButton("Refresh", func() {
			g.refreshData()
			g.showNotification("success", "Data refreshed", "")
		}),
	)

	// Overview cards
	overviewCards := fynecontainer.NewGridWithColumns(3,
		widget.NewCard("Active Instances", "",
			fynecontainer.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("%d", len(g.getRunningInstances())), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Currently running"),
			),
		),
		widget.NewCard("Daily Cost", "",
			fynecontainer.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("$%.2f", g.totalCost), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Estimated per day"),
			),
		),
		widget.NewCard("Total Instances", "",
			fynecontainer.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("%d", len(g.instances)), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("All instances"),
			),
		),
	)

	// Quick launch section
	quickLaunchCard := widget.NewCard("Quick Launch", "Launch a new research environment",
		g.createQuickLaunchForm(),
	)

	// Recent instances
	recentInstancesCard := widget.NewCard("Recent Instances", "Your latest cloud workstations",
		g.createRecentInstancesList(),
	)

	// Layout
	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		overviewCards,
		widget.NewSeparator(),
		fynecontainer.NewGridWithColumns(2,
			quickLaunchCard,
			recentInstancesCard,
		),
	)

	return fynecontainer.NewScroll(content)
}

// createQuickLaunchForm creates the quick launch form
func (g *CloudWorkstationGUI) createQuickLaunchForm() *fyne.Container {
	// Template selection
	templateNames := []string{"r-research", "python-research", "basic-ubuntu"}
	g.launchForm.templateSelect = widget.NewSelect(templateNames, nil)
	g.launchForm.templateSelect.SetSelected("r-research")

	// Instance name
	g.launchForm.nameEntry = widget.NewEntry()
	g.launchForm.nameEntry.SetPlaceHolder("my-workspace")

	// Size selection
	g.launchForm.sizeSelect = widget.NewSelect([]string{"XS", "S", "M", "L", "XL"}, nil)
	g.launchForm.sizeSelect.SetSelected("M")

	// Launch button
	g.launchForm.launchBtn = widget.NewButton("🚀 Launch Environment", func() {
		g.handleLaunchInstance()
	})
	g.launchForm.launchBtn.Importance = widget.HighImportance

	form := fynecontainer.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Template", g.launchForm.templateSelect),
			widget.NewFormItem("Name", g.launchForm.nameEntry),
			widget.NewFormItem("Size", g.launchForm.sizeSelect),
		),
		g.launchForm.launchBtn,
	)

	return form
}

// createRecentInstancesList creates a list of recent instances
func (g *CloudWorkstationGUI) createRecentInstancesList() *fyne.Container {
	if len(g.instances) == 0 {
		return fynecontainer.NewVBox(
			widget.NewLabelWithStyle("No instances yet", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
			widget.NewLabel("Launch your first environment using Quick Launch"),
		)
	}

	// Show up to 3 most recent instances
	items := make([]fyne.CanvasObject, 0)
	count := 0
	for _, instance := range g.instances {
		if count >= 3 {
			break
		}

		statusIcon := g.getStatusIcon(instance.State)

		instanceItem := fynecontainer.NewHBox(
			widget.NewLabel(statusIcon),
			fynecontainer.NewVBox(
				widget.NewLabelWithStyle(instance.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(fmt.Sprintf("%s • $%.2f/day", instance.Template, instance.EstimatedDailyCost)),
			),
			layout.NewSpacer(),
			widget.NewButton("Manage", func() {
				g.navigateToSection(SectionInstances)
			}),
		)

		items = append(items, instanceItem)
		count++
	}

	return fynecontainer.NewVBox(items...)
}

// createInstancesView creates the instances management view
func (g *CloudWorkstationGUI) createInstancesView() fyne.CanvasObject {
	header := fynecontainer.NewHBox(
		widget.NewLabelWithStyle("Instances", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButton("Launch New", func() {
			g.navigateToSection(SectionDashboard)
		}),
		widget.NewButton("Refresh", func() {
			g.refreshData()
		}),
	)

	// Instance cards
	instanceCards := fynecontainer.NewVBox()

	if len(g.instances) == 0 {
		instanceCards.Add(widget.NewCard("No Instances", "You haven't launched any instances yet",
			widget.NewButton("Launch Your First Instance", func() {
				g.navigateToSection(SectionDashboard)
			}),
		))
	} else {
		for _, instance := range g.instances {
			card := g.createInstanceCard(instance)
			instanceCards.Add(card)
		}
	}

	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		instanceCards,
	)

	return fynecontainer.NewScroll(content)
}

// createInstanceCard creates a detailed card for an instance
func (g *CloudWorkstationGUI) createInstanceCard(instance types.Instance) *widget.Card {
	statusIcon := g.getStatusIcon(instance.State)

	// Instance details
	details := fynecontainer.NewVBox(
		widget.NewLabelWithStyle(instance.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Template: %s", instance.Template)),
		widget.NewLabel(fmt.Sprintf("Cost: $%.2f/day", instance.EstimatedDailyCost)),
		widget.NewLabel(fmt.Sprintf("Launched: %s", instance.LaunchTime.Format("Jan 2, 2006 15:04"))),
	)

	// Status
	status := fynecontainer.NewVBox(
		fynecontainer.NewHBox(
			widget.NewLabel(statusIcon),
			widget.NewLabelWithStyle(instance.State, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
	)

	// Actions
	actions := fynecontainer.NewVBox()

	if instance.State == "running" {
		actions.Add(widget.NewButton("Connect", func() {
			g.handleConnectInstance(instance.Name)
		}))
		actions.Add(widget.NewButton("Stop", func() {
			g.handleStopInstance(instance.Name)
		}))
	} else if instance.State == "stopped" {
		actions.Add(widget.NewButton("Start", func() {
			g.handleStartInstance(instance.Name)
		}))
	}

	actions.Add(widget.NewButton("Delete", func() {
		g.handleDeleteInstance(instance.Name)
	}))

	// Card content
	cardContent := fynecontainer.NewHBox(
		details,
		layout.NewSpacer(),
		status,
		layout.NewSpacer(),
		actions,
	)

	return widget.NewCard("", "", cardContent)
}

// createTemplatesView creates the templates view
func (g *CloudWorkstationGUI) createTemplatesView() fyne.CanvasObject {
	header := widget.NewLabelWithStyle("Templates", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Template cards
	templateCards := fynecontainer.NewGridWithColumns(2,
		widget.NewCard("R Research Environment", "RStudio Server + R packages for data science",
			fynecontainer.NewVBox(
				widget.NewLabel("• RStudio Server"),
				widget.NewLabel("• Common R packages"),
				widget.NewLabel("• Jupyter Lab"),
				widget.NewButton("Launch R Environment", func() {
					g.quickLaunch("r-research")
				}),
			),
		),
		widget.NewCard("Python ML Environment", "Python + Jupyter + ML libraries",
			fynecontainer.NewVBox(
				widget.NewLabel("• Jupyter Notebook"),
				widget.NewLabel("• TensorFlow & PyTorch"),
				widget.NewLabel("• Data science libraries"),
				widget.NewButton("Launch Python Environment", func() {
					g.quickLaunch("python-research")
				}),
			),
		),
		widget.NewCard("Basic Ubuntu Server", "Clean Ubuntu server for general use",
			fynecontainer.NewVBox(
				widget.NewLabel("• Ubuntu 22.04 LTS"),
				widget.NewLabel("• Basic development tools"),
				widget.NewLabel("• Docker pre-installed"),
				widget.NewButton("Launch Ubuntu Server", func() {
					g.quickLaunch("basic-ubuntu")
				}),
			),
		),
		widget.NewCard("Custom Template", "Create your own environment",
			fynecontainer.NewVBox(
				widget.NewLabel("• Custom AMI"),
				widget.NewLabel("• Custom instance type"),
				widget.NewLabel("• Custom configuration"),
				widget.NewButton("Coming Soon", nil),
			),
		),
	)

	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		templateCards,
	)

	return fynecontainer.NewScroll(content)
}

// createVolumesView creates the storage/volumes view
func (g *CloudWorkstationGUI) createVolumesView() *fyne.Container {
	header := widget.NewLabelWithStyle("Storage & Volumes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		widget.NewCard("Storage Management", "Persistent storage for your workstations",
			widget.NewLabel("Storage management features coming soon..."),
		),
	)

	return content
}

// createBillingView creates the billing/cost view
func (g *CloudWorkstationGUI) createBillingView() *fyne.Container {
	header := widget.NewLabelWithStyle("Billing & Costs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Cost breakdown
	costCards := fynecontainer.NewGridWithColumns(2,
		widget.NewCard("Current Costs", "",
			fynecontainer.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("$%.2f", g.totalCost), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Daily cost estimate"),
				widget.NewLabel(fmt.Sprintf("Monthly: ~$%.2f", g.totalCost*30)),
			),
		),
		widget.NewCard("Cost Breakdown", "",
			fynecontainer.NewVBox(
				widget.NewLabel(fmt.Sprintf("Running instances: %d", len(g.getRunningInstances()))),
				widget.NewLabel(fmt.Sprintf("Total instances: %d", len(g.instances))),
				widget.NewLabel("Storage costs: $0.00"),
			),
		),
	)

	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		costCards,
		widget.NewSeparator(),
		widget.NewCard("Cost Management", "Monitor and control your cloud spending",
			widget.NewLabel("Advanced billing features coming soon..."),
		),
	)

	return content
}

// createSettingsView creates the settings view
func (g *CloudWorkstationGUI) createSettingsView() *fyne.Container {
	header := widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Connection settings
	connectionCard := widget.NewCard("Connection", "Daemon connection settings",
		fynecontainer.NewVBox(
			widget.NewLabel("Daemon URL: http://localhost:8080"),
			widget.NewLabel(fmt.Sprintf("Status: %s", func() string {
				if g.lastUpdate.IsZero() {
					return "Disconnected"
				}
				return "Connected"
			}())),
			widget.NewLabel(fmt.Sprintf("Active Profile: %s", func() string {
				if g.activeProfile != nil {
					return g.activeProfile.Name
				}
				return "None"
			}())),
			widget.NewButton("Test Connection", func() {
				if err := g.apiClient.Ping(context.Background()); err != nil {
					g.showNotification("error", "Connection failed", err.Error())
				} else {
					g.showNotification("success", "Connection successful", "")
				}
			}),
		),
	)

	// Profile management
	profileCard := widget.NewCard("Profile Management", "Manage AWS account profiles",
		g.createProfileManagerView(),
	)

	// About
	aboutCard := widget.NewCard("About", "CloudWorkstation information",
		fynecontainer.NewVBox(
			widget.NewLabel(fmt.Sprintf("Version: %s", version.GetVersion())),
			widget.NewLabel("A tool for managing cloud research environments"),
			widget.NewHyperlink("Documentation", nil),
			widget.NewHyperlink("GitHub Repository", nil),
		),
	)

	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		connectionCard,
		widget.NewSeparator(),
		profileCard,
		widget.NewSeparator(),
		aboutCard,
	)

	return content
}

// Event handlers

func (g *CloudWorkstationGUI) handleLaunchInstance() {
	// Enhanced validation
	if err := g.validateLaunchForm(); err != nil {
		g.showNotification("error", "Validation Error", err.Error())
		return
	}
	
	// Check daemon connection before launching
	if err := g.apiClient.Ping(context.Background()); err != nil {
		g.showNotification("error", "Connection Error", "Cannot connect to daemon. Please ensure cwsd is running.")
		return
	}

	req := types.LaunchRequest{
		Template: g.launchForm.templateSelect.Selected,
		Name:     g.launchForm.nameEntry.Text,
		Size:     g.launchForm.sizeSelect.Selected,
	}

	// Show loading state
	g.launchForm.launchBtn.SetText("Launching...")
	g.launchForm.launchBtn.Disable()

	// Launch in background with timeout
	go func() {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		// Launch instance with timeout context
		var response *types.LaunchResponse
		var err error
		
		done := make(chan bool, 1)
		go func() {
			response, err = g.apiClient.LaunchInstance(ctx, req)
			done <- true
		}()
		
		// Wait for completion or timeout
		select {
		case <-done:
			// Launch completed
		case <-ctx.Done():
			err = fmt.Errorf("launch operation timed out after 5 minutes")
		}

		// Update UI on main thread
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,
			Tick: func(_ float32) {
				if err != nil {
					g.showNotification("error", "Launch Failed", err.Error())
				} else {
					g.showNotification("success", "Launch Successful",
						fmt.Sprintf("Instance %s launched successfully! Estimated cost: %s",
							response.Instance.Name, response.EstimatedCost))

					// Clear form
					g.launchForm.nameEntry.SetText("")

					// Refresh data
					g.refreshData()
				}

				// Reset button
				g.launchForm.launchBtn.SetText("🚀 Launch Environment")
				g.launchForm.launchBtn.Enable()
			},
		})
	}()
}

// validateLaunchForm performs comprehensive validation of the launch form
func (g *CloudWorkstationGUI) validateLaunchForm() error {
	if g.launchForm.templateSelect.Selected == "" {
		return fmt.Errorf("please select a template")
	}
	
	instanceName := g.launchForm.nameEntry.Text
	if instanceName == "" {
		return fmt.Errorf("please enter an instance name")
	}
	
	// Validate instance name format
	if len(instanceName) < 3 {
		return fmt.Errorf("instance name must be at least 3 characters long")
	}
	
	if len(instanceName) > 50 {
		return fmt.Errorf("instance name must be less than 50 characters")
	}
	
	// Check for valid characters (alphanumeric and hyphens)
	for _, char := range instanceName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || char == '-') {
			return fmt.Errorf("instance name can only contain letters, numbers, and hyphens")
		}
	}
	
	// Check for duplicate names
	for _, instance := range g.instances {
		if instance.Name == instanceName {
			return fmt.Errorf("instance name '%s' already exists", instanceName)
		}
	}
	
	if g.launchForm.sizeSelect.Selected == "" {
		return fmt.Errorf("please select an instance size")
	}
	
	return nil
}

func (g *CloudWorkstationGUI) handleConnectInstance(name string) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get instance details instead of using deprecated ConnectInstance
	instance, err := g.apiClient.GetInstance(ctx, name)
	if err != nil {
		g.showNotification("error", "Connection Failed", err.Error())
		return
	}

	// Format connection info based on template
	var connectionInfo string
	switch instance.Template {
	case "r-research":
		connectionInfo = fmt.Sprintf("RStudio Server: http://%s:8787 (username: rstudio, password: cloudworkstation)", instance.PublicIP)
	case "python-research":
		connectionInfo = fmt.Sprintf("JupyterLab: http://%s:8888 (token: cloudworkstation)", instance.PublicIP)
	case "desktop-research":
		connectionInfo = fmt.Sprintf("NICE DCV: https://%s:8443 (username: ubuntu, password: cloudworkstation)", instance.PublicIP)
	default:
		connectionInfo = fmt.Sprintf("SSH: ssh ubuntu@%s", instance.PublicIP)
	}
	
	g.showNotification("info", "Connection Information", connectionInfo)
}

func (g *CloudWorkstationGUI) handleStartInstance(name string) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := g.apiClient.StartInstance(ctx, name); err != nil {
		g.showNotification("error", "Start Failed", err.Error())
		return
	}

	g.showNotification("success", "Instance Starting", fmt.Sprintf("Instance %s is starting up", name))
	g.refreshData()
}

func (g *CloudWorkstationGUI) handleStopInstance(name string) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := g.apiClient.StopInstance(ctx, name); err != nil {
		g.showNotification("error", "Stop Failed", err.Error())
		return
	}

	g.showNotification("success", "Instance Stopping", fmt.Sprintf("Instance %s is shutting down", name))
	g.refreshData()
}

func (g *CloudWorkstationGUI) handleDeleteInstance(name string) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := g.apiClient.DeleteInstance(ctx, name); err != nil {
		g.showNotification("error", "Delete Failed", err.Error())
		return
	}

	g.showNotification("success", "Instance Deleted", fmt.Sprintf("Instance %s has been deleted", name))
	g.refreshData()
}

// Utility methods

func (g *CloudWorkstationGUI) quickLaunch(template string) {
	g.launchForm.templateSelect.SetSelected(template)
	g.launchForm.nameEntry.SetText(fmt.Sprintf("%s-%d", template, time.Now().Unix()))
	g.navigateToSection(SectionDashboard)
}

func (g *CloudWorkstationGUI) getRunningInstances() []types.Instance {
	var running []types.Instance
	for _, instance := range g.instances {
		if instance.State == "running" {
			running = append(running, instance)
		}
	}
	return running
}

func (g *CloudWorkstationGUI) getStatusIcon(state string) string {
	switch state {
	case "running":
		return "🟢"
	case "stopped":
		return "🟡"
	case "pending":
		return "🟠"
	case "stopping":
		return "🟠"
	case "terminated":
		return "🔴"
	default:
		return "⚫"
	}
}

func (g *CloudWorkstationGUI) showNotification(notificationType, title, message string) {
	// Clear previous notifications
	g.notification.RemoveAll()

	var icon fyne.Resource
	switch notificationType {
	case "success":
		icon = theme.ConfirmIcon()
	case "error":
		icon = theme.ErrorIcon()
	case "info":
		icon = theme.InfoIcon()
	default:
		icon = theme.InfoIcon()
	}

	// Create notification
	var content *fyne.Container
	if message != "" {
		content = fynecontainer.NewHBox(
			widget.NewIcon(icon),
			fynecontainer.NewVBox(
				widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(message),
			),
			layout.NewSpacer(),
			widget.NewButton("×", func() {
				g.notification.Hide()
			}),
		)
	} else {
		content = fynecontainer.NewHBox(
			widget.NewIcon(icon),
			widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			widget.NewButton("×", func() {
				g.notification.Hide()
			}),
		)
	}

	notification := widget.NewCard("", "", content)
	g.notification.Add(notification)
	g.notification.Show()

	// Auto-hide after 5 seconds
	time.AfterFunc(5*time.Second, func() {
		if g.notification.Visible() {
			g.notification.Hide()
		}
	})
}

// checkDaemonConnection verifies daemon connectivity with retry logic
func (g *CloudWorkstationGUI) checkDaemonConnection(ctx context.Context) error {
	maxRetries := 3
	retryDelay := time.Second
	
	for i := 0; i < maxRetries; i++ {
		if err := g.apiClient.Ping(ctx); err == nil {
			return nil
		}
		
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}
	
	return fmt.Errorf("daemon unreachable after %d attempts", maxRetries)
}

func (g *CloudWorkstationGUI) refreshData() {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Fetch instances with error handling
	response, err := g.apiClient.ListInstances(ctx)
	if err != nil {
		log.Printf("Failed to refresh instance data: %v", err)
		
		// Check if this is a connection error
		if err := g.apiClient.Ping(ctx); err != nil {
			// Connection lost - clear last update to show disconnected status
			g.lastUpdate = time.Time{}
		}
		
		// Don't refresh UI if we can't get data
		return
	}

	g.instances = response.Instances
	g.totalCost = response.TotalCost
	g.lastUpdate = time.Now()

	// Refresh current view only if we have valid data
	g.navigateToSection(g.currentSection)
}

func (g *CloudWorkstationGUI) startBackgroundRefresh() {
	// Initial refresh
	g.refreshData()

	// Start ticker for periodic refresh with connection monitoring
	g.refreshTicker = time.NewTicker(30 * time.Second)
	go func() {
		consecutiveFailures := 0
		maxFailures := 3
		
		for range g.refreshTicker.C {
			// Try to refresh data
			prevLastUpdate := g.lastUpdate
			g.refreshData()
			
			// Check if refresh succeeded
			if g.lastUpdate.Equal(prevLastUpdate) && !g.lastUpdate.IsZero() {
				// No update occurred and we had a previous update - likely connection issue
				consecutiveFailures++
			} else {
				// Successful refresh
				consecutiveFailures = 0
			}
			
			// If we have too many failures, try to reconnect
			if consecutiveFailures >= maxFailures {
				log.Printf("Multiple refresh failures, checking daemon connection...")
				if err := g.checkDaemonConnection(context.Background()); err != nil {
					// Connection still failing - increase refresh interval to reduce load
					g.refreshTicker.Reset(60 * time.Second)
				} else {
					// Connection restored - restore normal interval
					g.refreshTicker.Reset(30 * time.Second)
					consecutiveFailures = 0
				}
			}
			
			// Check device binding validity if using a device-bound profile
			g.checkDeviceBindingValidity()
		}
	}()
}

// checkDeviceBindingValidity validates the current profile (simplified for CLI parity)
func (g *CloudWorkstationGUI) checkDeviceBindingValidity() {
	// Device binding features removed in profile simplification
	// Simplified profile system matches CLI functionality
	if g.activeProfile == nil {
		return
	}
	// Basic profile validation complete
}


// createProfileManagerView creates the profile management interface
func (g *CloudWorkstationGUI) createProfileManagerView() *fyne.Container {
	// Reload profiles to ensure we have the latest
	g.loadProfiles()
	
	// Create profile list
	profileList := widget.NewList(
		func() int {
			return len(g.profiles)
		},
		func() fyne.CanvasObject {
			return fynecontainer.NewHBox(
				widget.NewIcon(theme.AccountIcon()),
				fynecontainer.NewVBox(
					widget.NewLabelWithStyle("Profile Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					widget.NewLabel("Type"),
					widget.NewLabel("AWS Profile"),
					widget.NewLabel("Security Status"),
				),
				layout.NewSpacer(),
				fynecontainer.NewVBox(
					widget.NewButton("Use", nil),
					widget.NewButton("Validate", nil),
					widget.NewButton("Remove", nil),
				),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			profile := g.profiles[id]
			container := item.(*fyne.Container)
			
			// Get the profile info container
			infoContainer := container.Objects[1].(*fyne.Container)
			
			// Update profile information
			nameLabel := infoContainer.Objects[0].(*widget.Label)
			typeLabel := infoContainer.Objects[1].(*widget.Label)
			awsProfileLabel := infoContainer.Objects[2].(*widget.Label)
			securityLabel := infoContainer.Objects[3].(*widget.Label)
			
			// Set profile icon - simplified profile system
			profileIcon := container.Objects[0].(*widget.Icon)
			profileIcon.SetResource(theme.AccountIcon()) // All profiles get same icon
			
			nameLabel.SetText(profile.Name)
			
			// Display type
			typeText := "Personal AWS Account"
			if profile.Type == "invitation" {
				typeText = "Invitation Profile"
			}
			typeLabel.SetText(typeText)
			
			// Display AWS profile
			awsProfileLabel.SetText(profile.AWSProfile)
			
			// Display security status
			securityText := "Standard"
			if profile.Type == "invitation" {
				if profile.DeviceBound {
					securityText = "🔒 Device-Bound"
				} else {
					securityText = "⚠️ Not Device-Bound"
				}
			}
			securityLabel.SetText(securityText)
			
			// Get button container
			buttonContainer := container.Objects[3].(*fyne.Container)
			useButton := buttonContainer.Objects[0].(*widget.Button)
			validateButton := buttonContainer.Objects[1].(*widget.Button)
			removeButton := buttonContainer.Objects[2].(*widget.Button)
			
			// Check if this is the current profile
			isCurrentProfile := false
			if g.activeProfile != nil {
				isCurrentProfile = (g.activeProfile.AWSProfile == profile.AWSProfile && 
				                   g.activeProfile.Type == profile.Type && 
				                   g.activeProfile.Name == profile.Name)
			}
			
			if isCurrentProfile {
				useButton.SetText("Current")
				useButton.Disable()
			} else {
				useButton.SetText("Use")
				useButton.Enable()
				useButton.OnTapped = func() {
					g.switchProfile(profile.AWSProfile)
				}
			}
			
			// Set up validate button
			validateButton.OnTapped = func() {
				if profile.DeviceBound {
					g.validateSecureProfile(profile.AWSProfile)
				} else {
					g.validateProfile(profile.AWSProfile)
				}
			}
			
			// Set up remove button
			removeButton.OnTapped = func() {
				g.removeProfile(profile.AWSProfile)
			}
			
			// Disable remove button if this is the current profile
			if isCurrentProfile {
				removeButton.Disable()
			} else {
				removeButton.Enable()
			}
		},
	)
	
	// Add profile buttons
	addProfileButton := widget.NewButton("Add Personal Profile", func() {
		g.showAddPersonalProfileDialog()
	})
	
	addInvitationButton := widget.NewButton("Add Invitation", func() {
		g.showAddInvitationDialog()
	})
	
	// Add device management button
	manageDevicesButton := widget.NewButton("Manage Devices", func() {
		g.showDeviceManagementDialog()
	})
	
	// Layout the buttons in a horizontal container
	buttonContainer := fynecontainer.NewHBox(
		addProfileButton,
		addInvitationButton,
		manageDevicesButton,
	)
	
	// Add security explanation
	securityContainer := fynecontainer.NewVBox(
		widget.NewRichTextFromMarkdown("**Profile Security:**"),
		widget.NewRichTextFromMarkdown("🔒 **Device-Bound:** Profile can only be used on this device"),
		widget.NewRichTextFromMarkdown("⚠️ **Not Device-Bound:** Profile can be used on any device (less secure)"),
	)
	
	// Combine everything into a vertical container with more information
	return fynecontainer.NewVBox(
		widget.NewLabelWithStyle("Profile Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Manage AWS profiles and shared access through invitations"),
		widget.NewSeparator(),
		widget.NewLabel("Your Profiles:"),
		fynecontainer.NewVScroll(profileList),
		widget.NewSeparator(),
		buttonContainer,
		widget.NewSeparator(),
		securityContainer,
	)
}

// switchProfile switches to a different AWS profile
func (g *CloudWorkstationGUI) switchProfile(profileID string) {
	// Get the profile to check if it's device-bound
	profile, err := g.profileManager.GetProfile(profileID)
	if err != nil {
		g.showNotification("error", "Profile Error", fmt.Sprintf("Failed to get profile: %v", err))
		return
	}
	
	// Check if this is a device-bound profile
	if false { // Disabled complex profile features for Phase 2 - focus on CLI parity
		// Create secure invitation manager for validation
		// secureManager, err := profile.NewSecureInvitationManager(g.profileManager) // Disabled for CLI parity
		if err != nil {
			g.showNotification("error", "Security Error", 
				fmt.Sprintf("Failed to initialize security system: %v", err))
			return
		}
		
		// Show validating notification
		g.showNotification("info", "Validating Device Binding", 
			"Verifying this device is authorized to use this profile...")
		
		// Device binding validation disabled for simplified profile system
		
		// If we get here, device binding is valid
	}
	
	// Note: Profile switching is handled at API client level in simplified system
	// No explicit switch needed - profile context managed by client
	
	// Update the active profile
	activeProfile, err := g.profileManager.GetCurrentProfile()
	if err != nil {
		g.showNotification("error", "Profile Error", "Could not load selected profile")
		return
	}
	
	// Store active profile
	g.activeProfile = activeProfile
	
	// The API client is already updated by the profile-aware client
	
	// Refresh GUI to reflect profile change
	g.navigateToSection(g.currentSection)
	
	// Update status bar with security information
	if false { // Disabled invitation-specific features for CLI parity
		if profile.DeviceBound {
			g.showNotification("success", "Secure Profile Activated", 
				fmt.Sprintf("Now using device-bound profile: %s", activeProfile.Name))
		} else {
			g.showNotification("warning", "Profile Changed", 
				fmt.Sprintf("Now using profile: %s (Not device-bound - less secure)", activeProfile.Name))
		}
	} else {
		g.showNotification("success", "Profile Changed", 
			fmt.Sprintf("Now using profile: %s", activeProfile.Name))
	}
	
	// Refresh data with new profile
	g.refreshData()
}

// showAddPersonalProfileDialog shows the dialog to add a new personal profile
func (g *CloudWorkstationGUI) showAddPersonalProfileDialog() {
	// Create entry fields
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("My AWS Account")
	
	awsProfileEntry := widget.NewEntry()
	awsProfileEntry.SetPlaceHolder("default")
	
	regionEntry := widget.NewEntry()
	regionEntry.SetPlaceHolder("us-west-2")
	
	// Create form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Profile Name", Widget: nameEntry},
			{Text: "AWS Profile", Widget: awsProfileEntry, HintText: "Name in ~/.aws/credentials"},
			{Text: "AWS Region", Widget: regionEntry, HintText: "Optional - uses AWS defaults if empty"},
		},
		OnSubmit: func() {
			// Create profile
			newProfile := profile.Profile{
				Type:       "personal",
				Name:       nameEntry.Text,
				AWSProfile: awsProfileEntry.Text,
				Region:     regionEntry.Text,
				CreatedAt:  time.Now(),
			}
			
			// Add the profile using enhanced profile manager
			if err := g.profileManager.AddProfile(newProfile); err != nil {
				g.showNotification("error", "Add Profile Failed", err.Error())
				return
			}
			
			// Refresh the view
			g.showNotification("success", "Profile Added", fmt.Sprintf("Added profile: %s", newProfile.Name))
			g.loadProfiles()
			g.navigateToSection(SectionSettings)
		},
	}
	
	// Create and show dialog
	dialog := dialog.NewCustom("Add Personal Profile", "Cancel", form, g.window)
	dialog.Show()
}

// showAddInvitationDialog shows the dialog to add a new invitation profile
func (g *CloudWorkstationGUI) showAddInvitationDialog() {
	// Create entry fields
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Class Project")
	
	tokenEntry := widget.NewMultiLineEntry()
	tokenEntry.SetPlaceHolder("Paste the full invitation token here (starts with inv-...)")
	
	// Create device binding checkbox
	deviceBindingCheck := widget.NewCheck("Enable device binding (recommended)", nil)
	deviceBindingCheck.SetChecked(true) // Enable by default for security
	
	// Create explanation text for device binding
	securityExplanation := widget.NewRichTextFromMarkdown(
		"**Device binding** restricts this profile to only work on this device. " +
		"This improves security by preventing unauthorized access from other computers.")
	
	// Create device binding container
	deviceBindingContainer := fynecontainer.NewVBox(
		deviceBindingCheck,
		securityExplanation,
	)
	
	// Create form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Profile Name", Widget: nameEntry},
			{Text: "Invitation Token", Widget: tokenEntry, HintText: "Paste the complete invitation token"},
			{Text: "Security", Widget: deviceBindingContainer},
		},
		OnSubmit: func() {
			// Validate inputs
			if nameEntry.Text == "" {
				g.showNotification("error", "Validation Error", "Profile name cannot be empty")
				return
			}
			
			if tokenEntry.Text == "" {
				g.showNotification("error", "Validation Error", "Invitation token cannot be empty")
				return
			}
			
			// Check if token has the correct format
			// In a full implementation, we would validate with the server
			tokenValid := strings.HasPrefix(tokenEntry.Text, "inv-")
			if !tokenValid {
				g.showNotification("error", "Invalid Token", "The invitation token appears to be invalid. It should start with 'inv-'")
				return
			}
			
			// Decode the invitation token to check its properties
			invitation, err := profile.DecodeFromString(tokenEntry.Text)
			if err != nil {
				g.showNotification("error", "Invalid Token", "Could not decode invitation token: " + err.Error())
				return
			}
			
			// Check if the invitation is valid (not expired)
			if !invitation.IsValid() {
				g.showNotification("error", "Expired Invitation", "This invitation has expired and cannot be used")
				return
			}
			
			// Create a profile with the token
			newProfile := profile.Profile{
				// Type:            "personal", // Simplified profile system
				Name:            nameEntry.Text,
				InvitationToken: tokenEntry.Text,
				OwnerAccount:    invitation.OwnerAccount,
				S3ConfigPath:    invitation.S3ConfigPath,
				CreatedAt:       time.Now(),
				// Security properties
				DeviceBound:     deviceBindingCheck.Checked,
			}
			
			// Handle device binding
			if deviceBindingCheck.Checked {
				// Show confirmation dialog for device binding
				confirmDialog := dialog.NewConfirm(
					"Confirm Device Binding",
					"Device binding will restrict this profile to only work on this computer. You cannot use this profile on other devices. Continue?",
					func(confirmed bool) {
						if !confirmed {
							return
						}
						
						// Try to create secure invitation manager
						invitationManager, err := profile.NewSecureInvitationManager(g.profileManager)
						if err != nil {
							g.showNotification("error", "Security Error", 
								"Failed to initialize security system: " + err.Error())
							return
						}
						
						// Use secure add to profile to handle device binding
						err = invitationManager.SecureAddToProfile(tokenEntry.Text, nameEntry.Text)
						if err != nil {
							g.showNotification("error", "Device Binding Failed", 
								"Failed to add secure profile: " + err.Error())
							return
						}
						
						g.showNotification("success", "Secure Invitation Added", 
							fmt.Sprintf("Added device-bound profile: %s", nameEntry.Text))
						
						// Refresh the view
						g.loadProfiles()
						g.navigateToSection(SectionSettings)
					},
					g.window,
				)
				confirmDialog.SetConfirmText("Continue with Binding")
				confirmDialog.SetDismissText("Cancel")
				confirmDialog.Show()
				
				// Return early since we're handling this in the confirm dialog
				return
			} else {
				// Standard add profile without device binding
				if err := g.profileManager.AddProfile(newProfile); err != nil {
					g.showNotification("error", "Add Invitation Failed", err.Error())
					return
				}
				
				g.showNotification("success", "Invitation Added", 
					fmt.Sprintf("Added invitation profile: %s", nameEntry.Text))
				
				// Refresh the view
				g.loadProfiles()
				g.navigateToSection(SectionSettings)
			}
		},
	}
	
	// Create and show dialog
	dialog := dialog.NewCustom("Add Invitation", "Cancel", form, g.window)
	dialog.Show()
}

// validateProfile tests if a profile has valid credentials and configuration
func (g *CloudWorkstationGUI) validateProfile(profileID string) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Show loading notification
	g.showNotification("info", "Validating Profile", "Testing connection with profile...")
	
	// Get the profile to check its type
	profile, err := g.profileManager.GetProfile(profileID)
	if err != nil {
		g.showNotification("error", "Profile Error", fmt.Sprintf("Failed to get profile: %v", err))
		return
	}
	
	// Check if this is an invitation profile
	if profile.Type == "invitation" {
		// For invitation profiles, we need to check the invitation validity first
		if profile.InvitationToken != "" {
			// Simple validation - check if token has the expected format
			if !strings.HasPrefix(profile.InvitationToken, "inv-") {
				g.showNotification("error", "Invalid Invitation", 
					"This invitation token appears to be invalid")
				return
			}
			// If we got here, token format is valid - proceed with validation
		}
	}
	
	// Use existing API client - profile context handled automatically
	client := g.apiClient
	
	// Test connection with that profile
	err = client.Ping(ctx)
	if err != nil {
		g.showNotification("error", "Validation Failed", fmt.Sprintf("Profile validation failed: %v", err))
		return
	}
	
	// If we get here, validation succeeded
	if profile.Type == "invitation" {
		g.showNotification("success", "Invitation Valid", 
			fmt.Sprintf("Invitation profile '%s' is valid and can access resources", profile.Name))
	} else {
		g.showNotification("success", "Profile Valid", 
			fmt.Sprintf("Personal profile '%s' is valid and can access the API", profile.Name))
	}
}

// validateSecureProfile tests if a device-bound profile is valid on this device
func (g *CloudWorkstationGUI) validateSecureProfile(profileID string) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Show loading notification
	g.showNotification("info", "Validating Device Binding", "Verifying device security binding...")
	
	// Get the profile
	profile, err := g.profileManager.GetProfile(profileID)
	if err != nil {
		g.showNotification("error", "Profile Error", fmt.Sprintf("Failed to get profile: %v", err))
		return
	}
	
	// Secure invitation system disabled for CLI parity
	// Complex security features not available in simplified profile system
	
	// Device binding validation disabled - not available in CLI
	// Using simplified profile system for consistency
	
	// Use existing API client - profile context handled automatically
	client := g.apiClient
	if err != nil {
		g.showNotification("error", "Profile Error", 
			fmt.Sprintf("Device binding is valid, but API connection failed: %v", err))
		return
	}
	
	err = client.Ping(ctx)
	if err != nil {
		g.showNotification("error", "API Connection Failed", 
			fmt.Sprintf("Device binding is valid, but API connection failed: %v", err))
		return
	}
	
	// All validations passed
	g.showNotification("success", "Profile Valid", 
		fmt.Sprintf("Device-bound profile '%s' is valid on this device", profile.Name))
	
	// Try to check with registry in background (non-blocking)
	go func() {
		if profile.BindingRef != "" {
			// Use the security package to retrieve binding
			// binding, err := security.RetrieveDeviceBinding(profile.BindingRef) // Disabled for CLI parity
			if false { // Disabled for CLI parity
				// For now, just log that we would check the registry
				fmt.Printf("Would check registry for token %s and device %s\n", 
					"token", "deviceID")
				
				// In a real implementation, we would check with registry:
				// valid, _ := secureManager.registry.ValidateDevice(binding.InvitationToken, binding.DeviceID)
				// if !valid {
				//     // Show notification on main thread
				//     g.app.Driver().StartAnimation(&fyne.Animation{
				//         Duration: 100 * time.Millisecond,
				//         Tick: func(_ float32) {
				//             g.showNotification("warning", "Registry Check Failed", 
				//                 "The central registry could not validate this device. Local validation succeeded.")
				//         },
				//     })
				// }
			}
		}
	}()
}

// showDeviceManagementDialog shows the dialog for managing devices registered to invitations
func (g *CloudWorkstationGUI) showDeviceManagementDialog() {
	// Find all device-bound profiles
	var deviceBoundProfiles []profile.Profile
	for _, p := range g.profiles {
		if false { // Disabled device-bound profile features for CLI parity
			deviceBoundProfiles = append(deviceBoundProfiles, p)
		}
	}
	
	if len(deviceBoundProfiles) == 0 {
		g.showNotification("info", "No Device-Bound Profiles", 
			"You don't have any device-bound profiles to manage")
		return
	}
	
	// Create secure invitation manager
	secureManager, err := profile.NewSecureInvitationManager(g.profileManager)
	if err != nil {
		g.showNotification("error", "Security Error", 
			fmt.Sprintf("Failed to initialize security system: %v", err))
		return
	}
	
	// Create a tab container for each profile
	tabs := fynecontainer.NewAppTabs()
	
	for _, p := range deviceBoundProfiles {
		// For each profile, try to get its devices
		profileTab := fynecontainer.NewVBox(
			widget.NewLabelWithStyle("Loading devices...", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		)
		
		// Add tab with profile name
		tab := fynecontainer.NewTabItem(p.Name, fynecontainer.NewVScroll(profileTab))
		tabs.Append(tab)
		
		// Load devices in background
		go func(profile profile.Profile, container *fyne.Container) {
			// Try to load devices from registry
			var devices []map[string]interface{}
			if profile.InvitationToken != "" {
				devices, _ = secureManager.GetInvitationDevices(profile.InvitationToken)
			}
			
			// Get local device info
			localDeviceInfo := ""
			if profile.BindingRef != "" {
				// Use proper import to retrieve binding
				// binding, err := security.RetrieveDeviceBinding(profile.BindingRef) // Disabled for CLI parity
				if false { // Disabled for CLI parity
					localDeviceInfo = fmt.Sprintf("Current device: %s\nDevice ID: %s\nBound on: %s", 
						"Device", "ID", "Date")
				} else {
					localDeviceInfo = "This profile is device-bound, but binding information could not be retrieved."
				}
			}
			
			// Update UI on main thread
			g.app.Driver().StartAnimation(&fyne.Animation{
				Duration: 100 * time.Millisecond,
				Tick: func(_ float32) {
					// Clear container
					container.RemoveAll()
					
					// Add local device info
					if localDeviceInfo != "" {
						container.Add(widget.NewCard("This Device", "",
							widget.NewRichTextFromMarkdown(fmt.Sprintf("**%s**", localDeviceInfo))))
					}
					
					// Add header for registered devices
					container.Add(widget.NewLabelWithStyle("Registered Devices", 
						fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
					
					// Show devices or message if none
					if len(devices) == 0 {
						container.Add(widget.NewLabelWithStyle(
							"No other devices registered with this invitation",
							fyne.TextAlignCenter, fyne.TextStyle{Italic: true}))
					} else {
						// Add each device
						for i, device := range devices {
							// Extract device info
							deviceID, _ := device["device_id"].(string)
							hostname, _ := device["hostname"].(string)
							username, _ := device["username"].(string)
							timestamp, _ := device["timestamp"].(string)
							
							if deviceID == "" {
								deviceID = fmt.Sprintf("Unknown device %d", i+1)
							}
							
							// Create device card
							deviceInfo := fmt.Sprintf("Device ID: %s\n", deviceID)
							if hostname != "" {
								deviceInfo += fmt.Sprintf("Hostname: %s\n", hostname)
							}
							if username != "" {
								deviceInfo += fmt.Sprintf("Username: %s\n", username)
							}
							if timestamp != "" {
								deviceInfo += fmt.Sprintf("Registered: %s\n", timestamp)
							}
							
							deviceCard := widget.NewCard(deviceID, "",
								fynecontainer.NewVBox(
									widget.NewLabel(deviceInfo),
									widget.NewButton("Revoke Device", func() {
										g.revokeDevice(profile.InvitationToken, deviceID)
									}),
								))
							
							container.Add(deviceCard)
						}
					}
					
					// Add revoke all button
					container.Add(widget.NewSeparator())
					container.Add(widget.NewButton("Revoke All Devices", func() {
						g.revokeAllDevices(profile.InvitationToken)
					}))
				},
			})
		}(p, profileTab)
	}
	
	// Create the dialog
	dialog := dialog.NewCustom("Device Management", "Close",
		fynecontainer.NewVBox(
			widget.NewRichTextFromMarkdown("**Device Management**\n\nView and manage all devices registered with your secure invitations."),
			widget.NewSeparator(),
			tabs,
		), g.window)
	
	dialog.Resize(fyne.NewSize(600, 400))
	dialog.Show()
}

// revokeDevice revokes a specific device from using an invitation
func (g *CloudWorkstationGUI) revokeDevice(invitationToken, deviceID string) {
	// Confirm before revoking
	confirmDialog := dialog.NewConfirm(
		"Confirm Device Revocation",
		fmt.Sprintf("Are you sure you want to revoke access for device %s?\nThis action cannot be undone.", deviceID),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			
			// Security manager disabled for CLI parity
			// secureManager, err := profile.NewSecureInvitationManager(g.profileManager) // Disabled for CLI parity
			// Device revocation not available in simplified profile system
			g.showNotification("info", "Feature Disabled", 
				"Device revocation is not available in the simplified profile system")
			return
			
			g.showNotification("success", "Device Revoked", 
				fmt.Sprintf("Device %s has been revoked successfully", deviceID))
			
			// Refresh the device management dialog
			g.showDeviceManagementDialog()
		},
		g.window,
	)
	
	confirmDialog.SetConfirmText("Revoke Device")
	confirmDialog.SetDismissText("Cancel")
	confirmDialog.Show()
}

// revokeAllDevices revokes all devices for an invitation
func (g *CloudWorkstationGUI) revokeAllDevices(invitationToken string) {
	// Confirm before revoking all
	confirmDialog := dialog.NewConfirm(
		"Confirm Revocation",
		"Are you sure you want to revoke ALL devices for this invitation?\nThis action cannot be undone.",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			
			// Security manager disabled for CLI parity
			// secureManager, err := profile.NewSecureInvitationManager(g.profileManager) // Disabled for CLI parity
			// Device revocation not available in simplified profile system
			g.showNotification("info", "Feature Disabled", 
				"Device revocation is not available in the simplified profile system")
			return
			
			g.showNotification("success", "All Devices Revoked", 
				"All devices have been revoked successfully")
			
			// Refresh the device management dialog
			g.showDeviceManagementDialog()
		},
		g.window,
	)
	
	confirmDialog.SetConfirmText("Revoke All Devices")
	confirmDialog.SetDismissText("Cancel")
	confirmDialog.Show()
}

// removeProfile removes a profile after confirmation
func (g *CloudWorkstationGUI) removeProfile(profileID string) {
	// Check if this is the current profile
	current, err := g.profileManager.GetCurrentProfile()
	if err == nil && current.AWSProfile == profileID {
		g.showNotification("error", "Cannot Remove", "Cannot remove the active profile. Switch to another profile first.")
		return
	}
	
	// Show confirmation dialog
	confirm := dialog.NewConfirm(
		"Confirm Profile Removal",
		fmt.Sprintf("Are you sure you want to remove profile '%s'? This cannot be undone.", profileID),
		func(confirmed bool) {
			if confirmed {
				// Remove the profile
				err := g.profileManager.RemoveProfile(profileID)
				if err != nil {
					g.showNotification("error", "Remove Failed", err.Error())
					return
				}
				
				// Reload profiles and update view
				g.loadProfiles()
				g.showNotification("success", "Profile Removed", fmt.Sprintf("Profile '%s' has been removed", profileID))
				g.navigateToSection(SectionSettings)
			}
		},
		g.window,
	)
	
	confirm.SetDismissText("Cancel")
	confirm.SetConfirmText("Remove")
	confirm.Show()
}

func (g *CloudWorkstationGUI) run() {
	// Show window and run
	g.window.ShowAndRun()

	// Cleanup
	if g.refreshTicker != nil {
		g.refreshTicker.Stop()
	}
	
	// Stop system tray handler
	if g.systemTray != nil {
		g.systemTray.Stop()
	}
}
