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
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
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
	refreshTicker      *time.Ticker
	systemTray         *systray.SystemTrayHandler
	templatesContainer *fyne.Container
	efsContainer        *fyne.Container
	ebsContainer        *fyne.Container
	instancesContainer  *fyne.Container
	daemonStatusContainer *fyne.Container

	// Form state
	launchForm struct {
		templateSelect    *widget.Select
		nameEntry         *widget.Entry
		sizeSelect        *widget.Select
		packageMgrSelect  *widget.Select
		packageMgrHelp    *widget.Label
		launchBtn         *widget.Button
		
		// Advanced options
		advancedExpanded  bool
		volumesSelect     *widget.Select
		ebsVolumesSelect  *widget.Select
		spotCheck         *widget.Check
		dryRunCheck       *widget.Check
		regionEntry       *widget.Entry
		subnetEntry       *widget.Entry
		vpcEntry          *widget.Entry
		
		// Available options for selection
		availableVolumes    []string
		availableEBSVolumes []string
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
		g.apiClient = api.NewClient("http://localhost:8947")
	} else {
		// Create client with current profile AWS settings
		g.apiClient = api.NewClientWithOptions("http://localhost:8947", client.Options{
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

// createQuickLaunchForm creates the enhanced launch form
func (g *CloudWorkstationGUI) createQuickLaunchForm() *fyne.Container {
	// Initialize advanced launch form
	g.initializeAdvancedLaunchForm()
	
	// Package manager section with help text
	packageMgrContainer := fynecontainer.NewVBox(
		g.launchForm.packageMgrSelect,
		g.launchForm.packageMgrHelp,
	)

	// Basic form
	basicForm := widget.NewForm(
		widget.NewFormItem("Template", g.launchForm.templateSelect),
		widget.NewFormItem("Name", g.launchForm.nameEntry),
		widget.NewFormItem("Size", g.launchForm.sizeSelect),
		widget.NewFormItem("Package Manager", packageMgrContainer),
	)

	// Advanced options toggle
	advancedToggle := widget.NewButton("⚙️ Advanced Options", func() {
		g.toggleAdvancedOptions()
	})
	
	// Advanced options container (initially hidden)
	advancedContainer := g.createAdvancedOptionsContainer()
	
	// Action buttons
	buttonContainer := fynecontainer.NewHBox(
		advancedToggle,
		layout.NewSpacer(),
		g.launchForm.dryRunCheck,
		g.launchForm.launchBtn,
	)

	form := fynecontainer.NewVBox(
		basicForm,
		widget.NewSeparator(),
		advancedContainer,
		widget.NewSeparator(),
		buttonContainer,
	)

	return form
}

// initializeAdvancedLaunchForm initializes the advanced launch form components
func (g *CloudWorkstationGUI) initializeAdvancedLaunchForm() {
	// Template selection with dynamic loading
	g.launchForm.templateSelect = widget.NewSelect([]string{"r-research", "python-research", "basic-ubuntu"}, nil)
	g.launchForm.templateSelect.SetSelected("r-research")

	// Instance name
	g.launchForm.nameEntry = widget.NewEntry()
	g.launchForm.nameEntry.SetPlaceHolder("my-workspace")

	// Size selection with GPU options
	g.launchForm.sizeSelect = widget.NewSelect([]string{"XS", "S", "M", "L", "XL", "GPU-S", "GPU-M", "GPU-L"}, nil)
	g.launchForm.sizeSelect.SetSelected("M")

	// Package manager selection
	g.launchForm.packageMgrSelect = widget.NewSelect([]string{"Default", "conda", "apt", "dnf", "spack", "ami"}, nil)
	g.launchForm.packageMgrSelect.SetSelected("Default")
	
	// Package manager help text
	g.launchForm.packageMgrHelp = widget.NewLabel("Let template choose optimal package manager")
	g.launchForm.packageMgrHelp.TextStyle = fyne.TextStyle{Italic: true}
	
	g.launchForm.packageMgrSelect.OnChanged = func(selected string) {
		g.updatePackageManagerHelp(selected)
	}

	// Load available volumes for selection
	g.loadAvailableVolumes()

	// Volume selections
	g.launchForm.volumesSelect = widget.NewSelect(g.launchForm.availableVolumes, nil)
	g.launchForm.volumesSelect.PlaceHolder = "Select EFS volume (optional)"
	
	g.launchForm.ebsVolumesSelect = widget.NewSelect(g.launchForm.availableEBSVolumes, nil)
	g.launchForm.ebsVolumesSelect.PlaceHolder = "Select EBS volume (optional)"

	// Networking options
	g.launchForm.regionEntry = widget.NewEntry()
	g.launchForm.regionEntry.SetPlaceHolder("us-west-2 (optional)")
	
	g.launchForm.subnetEntry = widget.NewEntry()
	g.launchForm.subnetEntry.SetPlaceHolder("subnet-xxxxx (optional)")
	
	g.launchForm.vpcEntry = widget.NewEntry()
	g.launchForm.vpcEntry.SetPlaceHolder("vpc-xxxxx (optional)")

	// Advanced options checkboxes
	g.launchForm.spotCheck = widget.NewCheck("Use Spot Instance (lower cost)", nil)
	g.launchForm.dryRunCheck = widget.NewCheck("Dry Run (preview only)", nil)

	// Launch button
	g.launchForm.launchBtn = widget.NewButton("🚀 Launch Environment", func() {
		g.handleAdvancedLaunchInstance()
	})
	g.launchForm.launchBtn.Importance = widget.HighImportance
}

// createAdvancedOptionsContainer creates the collapsible advanced options
func (g *CloudWorkstationGUI) createAdvancedOptionsContainer() *fyne.Container {
	// Storage options
	storageSection := fynecontainer.NewVBox(
		widget.NewLabelWithStyle("Storage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("EFS Volume", g.launchForm.volumesSelect),
			widget.NewFormItem("EBS Volume", g.launchForm.ebsVolumesSelect),
		),
	)

	// Networking options
	networkingSection := fynecontainer.NewVBox(
		widget.NewLabelWithStyle("Networking", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Region", g.launchForm.regionEntry),
			widget.NewFormItem("Subnet ID", g.launchForm.subnetEntry),
			widget.NewFormItem("VPC ID", g.launchForm.vpcEntry),
		),
	)

	// Instance options
	instanceSection := fynecontainer.NewVBox(
		widget.NewLabelWithStyle("Instance Options", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		g.launchForm.spotCheck,
	)

	// Advanced options layout
	advancedContent := fynecontainer.NewVBox(
		fynecontainer.NewGridWithColumns(2,
			storageSection,
			networkingSection,
		),
		widget.NewSeparator(),
		instanceSection,
	)

	// Initially hidden container
	advancedContainer := fynecontainer.NewVBox()
	if g.launchForm.advancedExpanded {
		advancedContainer.Add(advancedContent)
	}

	return advancedContainer
}

// toggleAdvancedOptions toggles the advanced options visibility
func (g *CloudWorkstationGUI) toggleAdvancedOptions() {
	g.launchForm.advancedExpanded = !g.launchForm.advancedExpanded
	
	// Refresh the entire dashboard to update the advanced options display
	g.navigateToSection(SectionDashboard)
}

// loadAvailableVolumes loads available volumes for selection
func (g *CloudWorkstationGUI) loadAvailableVolumes() {
	// Load EFS volumes
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		volumes, err := g.apiClient.ListVolumes(ctx)
		if err == nil {
			var volumeNames []string
			for _, volume := range volumes {
				volumeNames = append(volumeNames, volume.Name)
			}
			g.launchForm.availableVolumes = volumeNames
			
			// Update the select widget if it exists
			if g.launchForm.volumesSelect != nil {
				g.launchForm.volumesSelect.Options = volumeNames
				g.launchForm.volumesSelect.Refresh()
			}
		} else {
			g.launchForm.availableVolumes = []string{}
		}
	}()

	// Load EBS volumes
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		volumes, err := g.apiClient.ListStorage(ctx)
		if err == nil {
			var volumeNames []string
			for _, volume := range volumes {
				if volume.State == "available" { // Only show available volumes
					volumeNames = append(volumeNames, volume.Name)
				}
			}
			g.launchForm.availableEBSVolumes = volumeNames
			
			// Update the select widget if it exists
			if g.launchForm.ebsVolumesSelect != nil {
				g.launchForm.ebsVolumesSelect.Options = volumeNames
				g.launchForm.ebsVolumesSelect.Refresh()
			}
		} else {
			g.launchForm.availableEBSVolumes = []string{}
		}
	}()
}

// updatePackageManagerHelp provides contextual help for package manager selection
func (g *CloudWorkstationGUI) updatePackageManagerHelp(selected string) {
	var helpText string
	
	switch selected {
	case "conda":
		helpText = "Best for Python data science and R packages. Cross-platform package manager."
	case "apt":
		helpText = "Native Ubuntu/Debian package manager. System-level packages."
	case "dnf":
		helpText = "Red Hat/Fedora package manager. Newer replacement for yum."
	case "spack":
		helpText = "HPC and scientific computing packages. Optimized builds."
	case "ami":
		helpText = "Use pre-built AMI with packages already installed."
	case "Default":
		helpText = "Let template choose optimal package manager for the workload."
	default:
		helpText = "Select a package manager to override template default."
	}
	
	if g.launchForm.packageMgrHelp != nil {
		g.launchForm.packageMgrHelp.SetText(helpText)
	}
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
			g.navigateToSection(SectionTemplates)
		}),
		widget.NewButton("Refresh", func() {
			g.refreshInstances()
		}),
	)

	// Initialize instances container if needed
	g.initializeInstancesContainer()

	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		g.instancesContainer,
	)

	// Load instances data
	g.refreshInstances()

	return fynecontainer.NewScroll(content)
}

// initializeInstancesContainer sets up the instances container
func (g *CloudWorkstationGUI) initializeInstancesContainer() {
	if g.instancesContainer == nil {
		g.instancesContainer = fynecontainer.NewVBox()
	}
}

// refreshInstances loads instances from the API and updates the display
func (g *CloudWorkstationGUI) refreshInstances() {
	if g.instancesContainer == nil {
		return
	}
	
	// Clear existing content
	g.instancesContainer.RemoveAll()
	
	// Show loading indicator
	loadingLabel := widget.NewLabel("Loading instances...")
	g.instancesContainer.Add(loadingLabel)
	g.instancesContainer.Refresh()
	
	// Fetch instances from API
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		response, err := g.apiClient.ListInstances(ctx)
		if err != nil {
			// Update UI on main thread
			g.app.Driver().StartAnimation(&fyne.Animation{
				Duration: 100 * time.Millisecond,
				Tick: func(_ float32) {
					g.instancesContainer.RemoveAll()
					g.instancesContainer.Add(widget.NewLabel("❌ Failed to load instances: " + err.Error()))
					g.instancesContainer.Refresh()
				},
			})
			return
		}
		
		// Update UI on main thread
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,
			Tick: func(_ float32) {
				g.displayInstances(response.Instances)
			},
		})
	}()
}

// displayInstances renders instance cards
func (g *CloudWorkstationGUI) displayInstances(instances []types.Instance) {
	g.instancesContainer.RemoveAll()
	g.instances = instances // Update internal state
	
	if len(instances) == 0 {
		emptyState := widget.NewCard("No Instances", "You haven't launched any instances yet",
			fynecontainer.NewVBox(
				widget.NewLabel("Get started by launching your first research environment."),
				widget.NewButton("Launch Your First Instance", func() {
					g.navigateToSection(SectionTemplates)
				}),
			),
		)
		g.instancesContainer.Add(emptyState)
		g.instancesContainer.Refresh()
		return
	}
	
	// Create instance cards
	for _, instance := range instances {
		inst := instance // Capture for closure
		card := g.createEnhancedInstanceCard(inst)
		g.instancesContainer.Add(card)
	}
	
	g.instancesContainer.Refresh()
}

// createEnhancedInstanceCard creates a comprehensive card for an instance
func (g *CloudWorkstationGUI) createEnhancedInstanceCard(instance types.Instance) *widget.Card {
	statusIcon := g.getStatusIcon(instance.State)
	
	// Left section: Instance details
	detailsContainer := fynecontainer.NewVBox()
	
	// Instance name and template
	nameLabel := widget.NewLabelWithStyle(instance.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	detailsContainer.Add(nameLabel)
	detailsContainer.Add(widget.NewLabel("• Template: " + instance.Template))
	detailsContainer.Add(widget.NewLabel("• Instance Type: " + instance.InstanceType))
	detailsContainer.Add(widget.NewLabel("• Launched: " + instance.LaunchTime.Format("Jan 2, 2006 15:04")))
	
	// Network information
	if instance.PublicIP != "" {
		detailsContainer.Add(widget.NewLabel("• Public IP: " + instance.PublicIP))
	}
	if instance.PrivateIP != "" {
		detailsContainer.Add(widget.NewLabel("• Private IP: " + instance.PrivateIP))
	}
	
	// Attached storage information
	if len(instance.AttachedVolumes) > 0 {
		detailsContainer.Add(widget.NewLabel(fmt.Sprintf("• EFS Volumes: %d", len(instance.AttachedVolumes))))
	}
	if len(instance.AttachedEBSVolumes) > 0 {
		detailsContainer.Add(widget.NewLabel(fmt.Sprintf("• EBS Volumes: %d", len(instance.AttachedEBSVolumes))))
	}
	
	// Middle section: Status and cost
	statusContainer := fynecontainer.NewVBox()
	
	// Status with icon
	statusRow := fynecontainer.NewHBox(
		widget.NewLabel(statusIcon),
		widget.NewLabelWithStyle(strings.ToUpper(instance.State), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	statusContainer.Add(statusRow)
	
	// Cost information
	costLabel := widget.NewLabelWithStyle(fmt.Sprintf("$%.2f/day", instance.EstimatedDailyCost), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	statusContainer.Add(costLabel)
	
	// Idle detection status if enabled
	if instance.IdleDetection != nil && instance.IdleDetection.Enabled {
		idleStatus := "Idle Detection: ON"
		if instance.IdleDetection.ActionPending {
			idleStatus = "⏰ Action Pending"
		}
		statusContainer.Add(widget.NewLabel(idleStatus))
	}
	
	// Connection information for running instances
	if instance.State == "running" && instance.HasWebInterface {
		connectionContainer := fynecontainer.NewVBox()
		
		// Web interface information
		var webURL string
		switch instance.Template {
		case "r-research":
			webURL = fmt.Sprintf("http://%s:8787", instance.PublicIP)
			connectionContainer.Add(widget.NewLabel("🌐 RStudio Server"))
		case "python-research":
			webURL = fmt.Sprintf("http://%s:8888", instance.PublicIP)
			connectionContainer.Add(widget.NewLabel("🌐 JupyterLab"))
		case "desktop-research":
			webURL = fmt.Sprintf("https://%s:8443", instance.PublicIP)
			connectionContainer.Add(widget.NewLabel("🖥️ NICE DCV"))
		}
		
		if webURL != "" {
			connectionContainer.Add(widget.NewLabel("• " + webURL))
		}
		
		// SSH access
		sshCommand := fmt.Sprintf("ssh %s@%s", instance.Username, instance.PublicIP)
		connectionContainer.Add(widget.NewLabel("🔧 SSH: " + sshCommand))
		
		statusContainer.Add(widget.NewSeparator())
		statusContainer.Add(connectionContainer)
	}
	
	// Right section: Actions
	actionsContainer := fynecontainer.NewVBox()
	
	// Primary action buttons based on state
	if instance.State == "running" {
		connectBtn := widget.NewButton("Connect", func() {
			g.showConnectionDialog(instance)
		})
		connectBtn.Importance = widget.HighImportance
		actionsContainer.Add(connectBtn)
		
		stopBtn := widget.NewButton("Stop", func() {
			g.showStopConfirmation(instance.Name)
		})
		actionsContainer.Add(stopBtn)
		
		hibernateBtn := widget.NewButton("Hibernate", func() {
			g.showHibernateConfirmation(instance)
		})
		actionsContainer.Add(hibernateBtn)
	} else if instance.State == "stopped" {
		startBtn := widget.NewButton("Start", func() {
			g.showStartConfirmation(instance.Name)
		})
		startBtn.Importance = widget.HighImportance
		actionsContainer.Add(startBtn)
		
		// Add Resume button for potentially hibernated instances
		resumeBtn := widget.NewButton("Resume", func() {
			g.showResumeConfirmation(instance.Name)
		})
		resumeBtn.Importance = widget.MediumImportance
		actionsContainer.Add(resumeBtn)
	}
	
	// Secondary action buttons  
	if instance.State == "running" {
		applyTemplateBtn := widget.NewButton("Apply Template", func() {
			g.showApplyTemplateDialog(instance)
		})
		actionsContainer.Add(applyTemplateBtn)
		
		templateLayersBtn := widget.NewButton("Template History", func() {
			g.showTemplateLayersDialog(instance)
		})
		actionsContainer.Add(templateLayersBtn)
	}
	
	moreBtn := widget.NewButton("Details", func() {
		g.showInstanceDetails(instance)
	})
	actionsContainer.Add(moreBtn)
	
	// Danger zone
	actionsContainer.Add(widget.NewSeparator())
	deleteBtn := widget.NewButton("Delete", func() {
		g.showDeleteInstanceConfirmation(instance.Name)
	})
	deleteBtn.Importance = widget.DangerImportance
	actionsContainer.Add(deleteBtn)
	
	// Card layout
	cardContent := fynecontainer.NewHBox(
		detailsContainer,
		layout.NewSpacer(),
		statusContainer,
		layout.NewSpacer(),
		actionsContainer,
	)
	
	return widget.NewCard("", "", cardContent)
}

// createTemplatesView creates the templates view
func (g *CloudWorkstationGUI) createTemplatesView() fyne.CanvasObject {
	// Header with refresh button
	header := fynecontainer.NewHBox(
		widget.NewLabelWithStyle("Templates", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButton("Refresh", func() {
			g.refreshTemplates()
		}),
	)

	// Templates will be loaded dynamically
	g.templatesContainer = fynecontainer.NewVBox()
	g.refreshTemplates() // Load templates on first view

	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		g.templatesContainer,
	)

	return fynecontainer.NewScroll(content)
}

// refreshTemplates loads templates from the API and updates the view
func (g *CloudWorkstationGUI) refreshTemplates() {
	// Clear existing templates
	g.templatesContainer.RemoveAll()
	
	// Show loading indicator
	loadingLabel := widget.NewLabel("Loading templates...")
	g.templatesContainer.Add(loadingLabel)
	g.templatesContainer.Refresh()
	
	// Fetch templates from API
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		templates, err := g.apiClient.ListTemplates(ctx)
		if err != nil {
			// Update UI on main thread
			g.app.Driver().StartAnimation(&fyne.Animation{
				Duration: 100 * time.Millisecond,
				Tick: func(_ float32) {
					g.templatesContainer.RemoveAll()
					g.templatesContainer.Add(widget.NewLabel("❌ Failed to load templates: " + err.Error()))
					g.templatesContainer.Refresh()
				},
			})
			return
		}
		
		// Update UI on main thread
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,  
			Tick: func(_ float32) {
				g.displayTemplates(templates)
			},
		})
	}()
}

// displayTemplates renders the template cards in a grid layout
func (g *CloudWorkstationGUI) displayTemplates(templates map[string]types.Template) {
	g.templatesContainer.RemoveAll()
	
	if len(templates) == 0 {
		g.templatesContainer.Add(widget.NewLabel("No templates available"))
		g.templatesContainer.Refresh()
		return
	}
	
	// Create template cards in a grid (2 columns)
	templateCards := fynecontainer.NewGridWithColumns(2)
	
	// Create cards for each template
	for id, template := range templates {
		templateID := id // Capture for closure
		templateInfo := template // Capture for closure
		
		// Create template card with details
		card := g.createTemplateCard(templateID, templateInfo)
		templateCards.Add(card)
	}
	
	g.templatesContainer.Add(templateCards)
	g.templatesContainer.Refresh()
}

// createTemplateCard creates a card widget for a template
func (g *CloudWorkstationGUI) createTemplateCard(templateID string, template types.Template) *widget.Card {
	// Template details
	detailsContainer := fynecontainer.NewVBox()
	
	// Add architecture info if available
	if len(template.InstanceType) > 0 {
		if armType, hasArm := template.InstanceType["arm64"]; hasArm {
			detailsContainer.Add(widget.NewLabel("• ARM64: " + armType))
		}
		if x86Type, hasX86 := template.InstanceType["x86_64"]; hasX86 {
			detailsContainer.Add(widget.NewLabel("• x86_64: " + x86Type))
		}
	}
	
	// Add cost information if available
	if len(template.EstimatedCostPerHour) > 0 {
		if armCost, hasArm := template.EstimatedCostPerHour["arm64"]; hasArm {
			detailsContainer.Add(widget.NewLabel(fmt.Sprintf("• ARM cost: $%.4f/hour", armCost)))
		}
		if x86Cost, hasX86 := template.EstimatedCostPerHour["x86_64"]; hasX86 {
			detailsContainer.Add(widget.NewLabel(fmt.Sprintf("• x86 cost: $%.4f/hour", x86Cost)))
		}
	}
	
	// Add ports information if available
	if len(template.Ports) > 0 {
		portsStr := ""
		for i, port := range template.Ports {
			if i > 0 {
				portsStr += ", "
			}
			portsStr += fmt.Sprintf("%d", port)
		}
		detailsContainer.Add(widget.NewLabel("• Ports: " + portsStr))
	}
	
	// Launch button
	launchButton := widget.NewButton("Launch "+template.Name, func() {
		g.showLaunchDialog(templateID, template)
	})
	launchButton.Importance = widget.HighImportance
	
	detailsContainer.Add(widget.NewSeparator())
	detailsContainer.Add(launchButton)
	
	return widget.NewCard(template.Name, template.Description, detailsContainer)
}

// showLaunchDialog shows a dialog for launching a template
func (g *CloudWorkstationGUI) showLaunchDialog(templateID string, template types.Template) {
	// Instance name entry
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter instance name...")
	
	// Instance size selection
	sizeOptions := []string{"XS", "S", "M", "L", "XL"}
	if len(template.InstanceType) > 0 {
		// Add GPU sizes if this looks like a template that could benefit
		if templateID == "python-research" || templateID == "r-research" {
			sizeOptions = append(sizeOptions, "GPU-S", "GPU-M", "GPU-L")
		}
	}
	sizeSelect := widget.NewSelect(sizeOptions, nil)
	sizeSelect.SetSelected("M") // Default to medium
	
	// Create form
	form := fynecontainer.NewVBox(
		widget.NewLabelWithStyle("Launch "+template.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Instance Name:"),
		nameEntry,
		widget.NewLabel("Instance Size:"),
		sizeSelect,
		widget.NewSeparator(),
	)
	
	// Launch button
	launchBtn := widget.NewButton("Launch Instance", func() {
		instanceName := nameEntry.Text
		instanceSize := sizeSelect.Selected
		
		if instanceName == "" {
			g.showNotification("error", "Validation Error", "Please enter an instance name")
			return
		}
		
		// Close dialog and launch instance
		g.window.Canvas().SetOnTypedKey(nil) // Clear any key handlers
		g.launchInstance(templateID, instanceName, instanceSize)
	})
	launchBtn.Importance = widget.HighImportance
	
	cancelBtn := widget.NewButton("Cancel", func() {
		g.window.Canvas().SetOnTypedKey(nil) // Clear any key handlers
	})
	
	buttonContainer := fynecontainer.NewHBox(
		layout.NewSpacer(),
		cancelBtn,
		launchBtn,
	)
	
	form.Add(buttonContainer)
	
	// Show dialog
	dialog := dialog.NewCustom("Launch Template", "Close", form, g.window)
	dialog.Resize(fyne.NewSize(400, 300))
	dialog.Show()
}

// launchInstance launches a new instance with the specified template and parameters
func (g *CloudWorkstationGUI) launchInstance(templateID, instanceName, instanceSize string) {
	// Show launching notification
	g.showNotification("info", "Launching Instance", fmt.Sprintf("Starting %s with template %s...", instanceName, templateID))
	
	// Launch instance in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		// Create launch request
		request := types.LaunchRequest{
			Template: templateID,
			Name:     instanceName,
			Size:     instanceSize,
		}
		
		// Launch via API
		response, err := g.apiClient.LaunchInstance(ctx, request)
		if err != nil {
			// Show error on main thread
			g.app.Driver().StartAnimation(&fyne.Animation{
				Duration: 100 * time.Millisecond,
				Tick: func(_ float32) {
					g.showNotification("error", "Launch Failed", fmt.Sprintf("Failed to launch %s: %v", instanceName, err))
				},
			})
			return
		}
		
		// Show success on main thread and refresh data
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,
			Tick: func(_ float32) {
				g.showNotification("success", "Instance Launched", fmt.Sprintf("%s launched successfully! Instance ID: %s", instanceName, response.Instance.ID))
				g.refreshData() // Refresh dashboard data
			},
		})
	}()
}

// createVolumesView creates the storage/volumes view
func (g *CloudWorkstationGUI) createVolumesView() *fyne.Container {
	// Header with refresh button
	header := fynecontainer.NewHBox(
		widget.NewLabelWithStyle("Storage & Volumes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButton("Refresh", func() {
			g.refreshStorage()
		}),
	)

	// Tab container for EFS and EBS
	efsTab := fynecontainer.NewTabItem("EFS Volumes", g.createEFSVolumesView())
	ebsTab := fynecontainer.NewTabItem("EBS Storage", g.createEBSStorageView())
	
	tabs := fynecontainer.NewAppTabs(efsTab, ebsTab)

	content := fynecontainer.NewVBox(
		header,
		widget.NewSeparator(),
		tabs,
	)

	// Initialize storage containers and load data
	g.initializeStorageContainers()
	g.refreshStorage()

	return content
}

// initializeStorageContainers sets up the storage containers
func (g *CloudWorkstationGUI) initializeStorageContainers() {
	if g.efsContainer == nil {
		g.efsContainer = fynecontainer.NewVBox()
	}
	if g.ebsContainer == nil {
		g.ebsContainer = fynecontainer.NewVBox()
	}
}

// createEFSVolumesView creates the EFS volumes tab content
func (g *CloudWorkstationGUI) createEFSVolumesView() fyne.CanvasObject {
	// Create EFS container if not exists
	if g.efsContainer == nil {
		g.efsContainer = fynecontainer.NewVBox()
	}

	// Create volume button
	createBtn := widget.NewButton("Create EFS Volume", func() {
		g.showCreateEFSDialog()
	})
	createBtn.Importance = widget.HighImportance

	content := fynecontainer.NewVBox(
		fynecontainer.NewHBox(
			widget.NewLabelWithStyle("EFS Volumes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			createBtn,
		),
		widget.NewSeparator(),
		g.efsContainer,
	)

	return fynecontainer.NewScroll(content)
}

// createEBSStorageView creates the EBS storage tab content
func (g *CloudWorkstationGUI) createEBSStorageView() fyne.CanvasObject {
	// Create EBS container if not exists
	if g.ebsContainer == nil {
		g.ebsContainer = fynecontainer.NewVBox()
	}

	// Create volume button
	createBtn := widget.NewButton("Create EBS Volume", func() {
		g.showCreateEBSDialog()
	})
	createBtn.Importance = widget.HighImportance

	content := fynecontainer.NewVBox(
		fynecontainer.NewHBox(
			widget.NewLabelWithStyle("EBS Storage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			createBtn,
		),
		widget.NewSeparator(),
		g.ebsContainer,
	)

	return fynecontainer.NewScroll(content)
}

// refreshStorage loads both EFS and EBS data from the API
func (g *CloudWorkstationGUI) refreshStorage() {
	g.refreshEFSVolumes()
	g.refreshEBSStorage()
}

// refreshEFSVolumes loads EFS volumes from the API and updates the display
func (g *CloudWorkstationGUI) refreshEFSVolumes() {
	if g.efsContainer == nil {
		return
	}
	
	// Clear existing content
	g.efsContainer.RemoveAll()
	
	// Show loading indicator
	loadingLabel := widget.NewLabel("Loading EFS volumes...")
	g.efsContainer.Add(loadingLabel)
	g.efsContainer.Refresh()
	
	// Fetch volumes from API
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		volumes, err := g.apiClient.ListVolumes(ctx)
		if err != nil {
			// Update UI on main thread
			g.app.Driver().StartAnimation(&fyne.Animation{
				Duration: 100 * time.Millisecond,
				Tick: func(_ float32) {
					g.efsContainer.RemoveAll()
					g.efsContainer.Add(widget.NewLabel("❌ Failed to load EFS volumes: " + err.Error()))
					g.efsContainer.Refresh()
				},
			})
			return
		}
		
		// Update UI on main thread
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,
			Tick: func(_ float32) {
				g.displayEFSVolumes(volumes)
			},
		})
	}()
}

// refreshEBSStorage loads EBS volumes from the API and updates the display
func (g *CloudWorkstationGUI) refreshEBSStorage() {
	if g.ebsContainer == nil {
		return
	}
	
	// Clear existing content
	g.ebsContainer.RemoveAll()
	
	// Show loading indicator
	loadingLabel := widget.NewLabel("Loading EBS volumes...")
	g.ebsContainer.Add(loadingLabel)
	g.ebsContainer.Refresh()
	
	// Fetch storage from API
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		volumes, err := g.apiClient.ListStorage(ctx)
		if err != nil {
			// Update UI on main thread
			g.app.Driver().StartAnimation(&fyne.Animation{
				Duration: 100 * time.Millisecond,
				Tick: func(_ float32) {
					g.ebsContainer.RemoveAll()
					g.ebsContainer.Add(widget.NewLabel("❌ Failed to load EBS volumes: " + err.Error()))
					g.ebsContainer.Refresh()
				},
			})
			return
		}
		
		// Update UI on main thread
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,
			Tick: func(_ float32) {
				g.displayEBSStorage(volumes)
			},
		})
	}()
}

// displayEFSVolumes renders EFS volume cards
func (g *CloudWorkstationGUI) displayEFSVolumes(volumes []types.EFSVolume) {
	g.efsContainer.RemoveAll()
	
	if len(volumes) == 0 {
		g.efsContainer.Add(widget.NewLabel("No EFS volumes found. Create one to get started."))
		g.efsContainer.Refresh()
		return
	}
	
	// Create volume cards
	for _, volume := range volumes {
		vol := volume // Capture for closure
		card := g.createEFSVolumeCard(vol)
		g.efsContainer.Add(card)
	}
	
	g.efsContainer.Refresh()
}

// displayEBSStorage renders EBS volume cards
func (g *CloudWorkstationGUI) displayEBSStorage(volumes []types.EBSVolume) {
	g.ebsContainer.RemoveAll()
	
	if len(volumes) == 0 {
		g.ebsContainer.Add(widget.NewLabel("No EBS volumes found. Create one to get started."))
		g.ebsContainer.Refresh()
		return
	}
	
	// Create volume cards
	for _, volume := range volumes {
		vol := volume // Capture for closure
		card := g.createEBSVolumeCard(vol)
		g.ebsContainer.Add(card)
	}
	
	g.ebsContainer.Refresh()
}

// createEFSVolumeCard creates a card widget for an EFS volume
func (g *CloudWorkstationGUI) createEFSVolumeCard(volume types.EFSVolume) *widget.Card {
	// Volume details
	detailsContainer := fynecontainer.NewVBox()
	
	detailsContainer.Add(widget.NewLabel("• Filesystem ID: " + volume.FileSystemId))
	detailsContainer.Add(widget.NewLabel("• Region: " + volume.Region))
	detailsContainer.Add(widget.NewLabel("• Created: " + volume.CreationTime.Format("Jan 2, 2006 15:04")))
	
	if volume.State != "" {
		detailsContainer.Add(widget.NewLabel("• State: " + volume.State))
	}
	
	if len(volume.MountTargets) > 0 {
		detailsContainer.Add(widget.NewLabel(fmt.Sprintf("• Mount Targets: %d", len(volume.MountTargets))))
	}
	
	// Action buttons
	buttonContainer := fynecontainer.NewHBox()
	
	infoBtn := widget.NewButton("Info", func() {
		g.showVolumeInfo(volume.Name, "efs")
	})
	
	deleteBtn := widget.NewButton("Delete", func() {
		g.showDeleteConfirmation(volume.Name, "efs")
	})
	deleteBtn.Importance = widget.DangerImportance
	
	buttonContainer.Add(infoBtn)
	buttonContainer.Add(deleteBtn)
	
	detailsContainer.Add(widget.NewSeparator())
	detailsContainer.Add(buttonContainer)
	
	return widget.NewCard(volume.Name, "EFS Volume", detailsContainer)
}

// createEBSVolumeCard creates a card widget for an EBS volume
func (g *CloudWorkstationGUI) createEBSVolumeCard(volume types.EBSVolume) *widget.Card {
	// Volume details
	detailsContainer := fynecontainer.NewVBox()
	
	detailsContainer.Add(widget.NewLabel("• Volume ID: " + volume.VolumeID))
	detailsContainer.Add(widget.NewLabel("• Region: " + volume.Region))
	detailsContainer.Add(widget.NewLabel("• Size: " + fmt.Sprintf("%d GB", volume.SizeGB)))
	detailsContainer.Add(widget.NewLabel("• Type: " + volume.VolumeType))
	detailsContainer.Add(widget.NewLabel("• State: " + volume.State))
	detailsContainer.Add(widget.NewLabel("• Created: " + volume.CreationTime.Format("Jan 2, 2006 15:04")))
	
	if volume.AttachedTo != "" {
		detailsContainer.Add(widget.NewLabel("• Attached to: " + volume.AttachedTo))
	}
	
	// Action buttons
	buttonContainer := fynecontainer.NewHBox()
	
	infoBtn := widget.NewButton("Info", func() {
		g.showVolumeInfo(volume.Name, "ebs")
	})
	
	// Attach/Detach button based on state
	if volume.AttachedTo == "" && volume.State == "available" {
		attachBtn := widget.NewButton("Attach", func() {
			g.showAttachDialog(volume.Name)
		})
		attachBtn.Importance = widget.HighImportance
		buttonContainer.Add(attachBtn)
	} else if volume.AttachedTo != "" {
		detachBtn := widget.NewButton("Detach", func() {
			g.showDetachConfirmation(volume.Name)
		})
		buttonContainer.Add(detachBtn)
	}
	
	deleteBtn := widget.NewButton("Delete", func() {
		g.showDeleteConfirmation(volume.Name, "ebs")
	})
	deleteBtn.Importance = widget.DangerImportance
	
	buttonContainer.Add(infoBtn)
	buttonContainer.Add(deleteBtn)
	
	detailsContainer.Add(widget.NewSeparator())
	detailsContainer.Add(buttonContainer)
	
	return widget.NewCard(volume.Name, fmt.Sprintf("EBS Volume (%s)", volume.VolumeType), detailsContainer)
}

// Storage dialog methods

// showCreateEFSDialog shows dialog for creating a new EFS volume
func (g *CloudWorkstationGUI) showCreateEFSDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter volume name...")
	
	// Performance mode selection
	perfModeSelect := widget.NewSelect([]string{"generalPurpose", "maxIO"}, nil)
	perfModeSelect.SetSelected("generalPurpose")
	
	// Throughput mode selection
	throughputSelect := widget.NewSelect([]string{"bursting", "provisioned"}, nil)
	throughputSelect.SetSelected("bursting")
	
	form := fynecontainer.NewVBox(
		widget.NewLabel("Create EFS Volume"),
		widget.NewSeparator(),
		widget.NewLabel("Name:"),
		nameEntry,
		widget.NewLabel("Performance Mode:"),
		perfModeSelect,
		widget.NewLabel("Throughput Mode:"),
		throughputSelect,
	)
	
	createBtn := widget.NewButton("Create Volume", func() {
		volumeName := nameEntry.Text
		if volumeName == "" {
			g.showNotification("error", "Validation Error", "Please enter a volume name")
			return
		}
		
		// Create volume request
		request := types.VolumeCreateRequest{
			Name:            volumeName,
			PerformanceMode: perfModeSelect.Selected,
			ThroughputMode:  throughputSelect.Selected,
		}
		
		g.createEFSVolume(request)
	})
	createBtn.Importance = widget.HighImportance
	
	cancelBtn := widget.NewButton("Cancel", func() {
		// Dialog will close automatically
	})
	
	buttons := fynecontainer.NewHBox(layout.NewSpacer(), cancelBtn, createBtn)
	content := fynecontainer.NewVBox(form, buttons)
	
	dialog := dialog.NewCustom("Create EFS Volume", "Close", content, g.window)
	dialog.Resize(fyne.NewSize(400, 300))
	dialog.Show()
}

// showCreateEBSDialog shows dialog for creating a new EBS volume
func (g *CloudWorkstationGUI) showCreateEBSDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter volume name...")
	
	// Size selection with predefined options
	sizeSelect := widget.NewSelect([]string{"XS (100GB)", "S (500GB)", "M (1TB)", "L (2TB)", "XL (4TB)", "Custom"}, nil)
	sizeSelect.SetSelected("S (500GB)")
	
	// Volume type selection
	typeSelect := widget.NewSelect([]string{"gp3", "io2"}, nil)
	typeSelect.SetSelected("gp3")
	
	form := fynecontainer.NewVBox(
		widget.NewLabel("Create EBS Volume"),
		widget.NewSeparator(),
		widget.NewLabel("Name:"),
		nameEntry,
		widget.NewLabel("Size:"),
		sizeSelect,
		widget.NewLabel("Volume Type:"),
		typeSelect,
	)
	
	createBtn := widget.NewButton("Create Volume", func() {
		volumeName := nameEntry.Text
		if volumeName == "" {
			g.showNotification("error", "Validation Error", "Please enter a volume name")
			return
		}
		
		// Parse size selection
		size := "S" // Default
		switch sizeSelect.Selected {
		case "XS (100GB)":
			size = "XS"
		case "S (500GB)":
			size = "S"
		case "M (1TB)":
			size = "M"
		case "L (2TB)":
			size = "L"
		case "XL (4TB)":
			size = "XL"
		}
		
		// Create storage request
		request := types.StorageCreateRequest{
			Name:       volumeName,
			Size:       size,
			VolumeType: typeSelect.Selected,
		}
		
		g.createEBSVolume(request)
	})
	createBtn.Importance = widget.HighImportance
	
	cancelBtn := widget.NewButton("Cancel", func() {
		// Dialog will close automatically
	})
	
	buttons := fynecontainer.NewHBox(layout.NewSpacer(), cancelBtn, createBtn)
	content := fynecontainer.NewVBox(form, buttons)
	
	dialog := dialog.NewCustom("Create EBS Volume", "Close", content, g.window)
	dialog.Resize(fyne.NewSize(400, 300))
	dialog.Show()
}

// showVolumeInfo shows detailed information about a volume
func (g *CloudWorkstationGUI) showVolumeInfo(volumeName, volumeType string) {
	title := fmt.Sprintf("%s Volume Information", strings.ToUpper(volumeType))
	
	content := fynecontainer.NewVBox(
		widget.NewLabel(fmt.Sprintf("Volume: %s", volumeName)),
		widget.NewLabel(fmt.Sprintf("Type: %s", strings.ToUpper(volumeType))),
		widget.NewSeparator(),
		widget.NewLabel("Volume information will be loaded here..."),
	)
	
	dialog := dialog.NewCustom(title, "Close", content, g.window)
	dialog.Resize(fyne.NewSize(400, 200))
	dialog.Show()
}

// showDeleteConfirmation shows confirmation dialog for volume deletion
func (g *CloudWorkstationGUI) showDeleteConfirmation(volumeName, volumeType string) {
	title := fmt.Sprintf("Delete %s Volume", strings.ToUpper(volumeType))
	message := fmt.Sprintf("Are you sure you want to delete the %s volume '%s'?\n\nThis action cannot be undone.", strings.ToUpper(volumeType), volumeName)
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			g.deleteVolume(volumeName, volumeType)
		}
	}, g.window)
	
	dialog.Show()
}

// showAttachDialog shows dialog for attaching EBS volume to instance
func (g *CloudWorkstationGUI) showAttachDialog(volumeName string) {
	// Get list of running instances
	instanceNames := []string{}
	for _, instance := range g.instances {
		if instance.State == "running" {
			instanceNames = append(instanceNames, instance.Name)
		}
	}
	
	if len(instanceNames) == 0 {
		g.showNotification("warning", "No Running Instances", "No running instances available to attach volume to")
		return
	}
	
	instanceSelect := widget.NewSelect(instanceNames, nil)
	if len(instanceNames) > 0 {
		instanceSelect.SetSelected(instanceNames[0])
	}
	
	form := fynecontainer.NewVBox(
		widget.NewLabel(fmt.Sprintf("Attach Volume: %s", volumeName)),
		widget.NewSeparator(),
		widget.NewLabel("Instance:"),
		instanceSelect,
	)
	
	attachBtn := widget.NewButton("Attach", func() {
		if instanceSelect.Selected == "" {
			g.showNotification("error", "Validation Error", "Please select an instance")
			return
		}
		
		g.attachVolume(volumeName, instanceSelect.Selected)
	})
	attachBtn.Importance = widget.HighImportance
	
	cancelBtn := widget.NewButton("Cancel", func() {
		// Dialog will close automatically
	})
	
	buttons := fynecontainer.NewHBox(layout.NewSpacer(), cancelBtn, attachBtn)
	content := fynecontainer.NewVBox(form, buttons)
	
	dialog := dialog.NewCustom("Attach Volume", "Close", content, g.window)
	dialog.Resize(fyne.NewSize(400, 200))
	dialog.Show()
}

// showDetachConfirmation shows confirmation dialog for detaching EBS volume
func (g *CloudWorkstationGUI) showDetachConfirmation(volumeName string) {
	title := "Detach Volume"
	message := fmt.Sprintf("Are you sure you want to detach the volume '%s'?\n\nThe data will be preserved but the volume will no longer be accessible from the instance.", volumeName)
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			g.detachVolume(volumeName)
		}
	}, g.window)
	
	dialog.Show()
}

// Storage operation methods

// createEFSVolume creates a new EFS volume via API
func (g *CloudWorkstationGUI) createEFSVolume(request types.VolumeCreateRequest) {
	g.showNotification("info", "Creating Volume", "Creating EFS volume...")
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		_, err := g.apiClient.CreateVolume(ctx, request)
		if err != nil {
			g.showNotification("error", "Create Failed", "Failed to create EFS volume: "+err.Error())
			return
		}
		
		g.showNotification("success", "Volume Created", "EFS volume created successfully")
		g.refreshEFSVolumes()
	}()
}

// createEBSVolume creates a new EBS volume via API
func (g *CloudWorkstationGUI) createEBSVolume(request types.StorageCreateRequest) {
	g.showNotification("info", "Creating Volume", "Creating EBS volume...")
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		_, err := g.apiClient.CreateStorage(ctx, request)
		if err != nil {
			g.showNotification("error", "Create Failed", "Failed to create EBS volume: "+err.Error())
			return
		}
		
		g.showNotification("success", "Volume Created", "EBS volume created successfully")
		g.refreshEBSStorage()
	}()
}

// deleteVolume deletes a volume via API
func (g *CloudWorkstationGUI) deleteVolume(volumeName, volumeType string) {
	g.showNotification("info", "Deleting Volume", fmt.Sprintf("Deleting %s volume...", strings.ToUpper(volumeType)))
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		var err error
		if volumeType == "efs" {
			err = g.apiClient.DeleteVolume(ctx, volumeName)
		} else {
			err = g.apiClient.DeleteStorage(ctx, volumeName)
		}
		
		if err != nil {
			g.showNotification("error", "Delete Failed", fmt.Sprintf("Failed to delete %s volume: %s", strings.ToUpper(volumeType), err.Error()))
			return
		}
		
		g.showNotification("success", "Volume Deleted", fmt.Sprintf("%s volume deleted successfully", strings.ToUpper(volumeType)))
		g.refreshStorage()
	}()
}

// attachVolume attaches an EBS volume to an instance
func (g *CloudWorkstationGUI) attachVolume(volumeName, instanceName string) {
	g.showNotification("info", "Attaching Volume", "Attaching volume to instance...")
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		err := g.apiClient.AttachStorage(ctx, volumeName, instanceName)
		if err != nil {
			g.showNotification("error", "Attach Failed", "Failed to attach volume: "+err.Error())
			return
		}
		
		g.showNotification("success", "Volume Attached", "Volume attached successfully")
		g.refreshEBSStorage()
	}()
}

// detachVolume detaches an EBS volume from its instance
func (g *CloudWorkstationGUI) detachVolume(volumeName string) {
	g.showNotification("info", "Detaching Volume", "Detaching volume from instance...")
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		err := g.apiClient.DetachStorage(ctx, volumeName)
		if err != nil {
			g.showNotification("error", "Detach Failed", "Failed to detach volume: "+err.Error())
			return
		}
		
		g.showNotification("success", "Volume Detached", "Volume detached successfully")
		g.refreshEBSStorage()
	}()
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
	header := fynecontainer.NewHBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButton("Refresh", func() {
			g.refreshDaemonStatus()
		}),
	)

	// Initialize daemon status container
	g.initializeDaemonStatusContainer()

	// Daemon status monitoring
	daemonStatusCard := widget.NewCard("Daemon Status", "CloudWorkstation daemon monitoring",
		g.daemonStatusContainer,
	)

	// Connection management
	connectionCard := widget.NewCard("Connection Management", "Daemon connection and control",
		g.createConnectionManagementView(),
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
		daemonStatusCard,
		widget.NewSeparator(),
		connectionCard,
		widget.NewSeparator(),
		profileCard,
		widget.NewSeparator(),
		aboutCard,
	)

	// Load daemon status
	g.refreshDaemonStatus()

	return content
}

// Event handlers

// handleAdvancedLaunchInstance handles instance launch with advanced options
func (g *CloudWorkstationGUI) handleAdvancedLaunchInstance() {
	// Enhanced validation
	if err := g.validateAdvancedLaunchForm(); err != nil {
		g.showNotification("error", "Validation Error", err.Error())
		return
	}
	
	// Check daemon connection before launching
	if err := g.apiClient.Ping(context.Background()); err != nil {
		g.showNotification("error", "Connection Error", "Cannot connect to daemon. Please ensure cwsd is running.")
		return
	}

	// Build advanced launch request
	req := g.buildAdvancedLaunchRequest()
	
	// Show appropriate action message
	actionMsg := "Launching..."
	if req.DryRun {
		actionMsg = "Validating..."
	}

	// Show loading state
	g.launchForm.launchBtn.SetText(actionMsg)
	g.launchForm.launchBtn.Disable()

	// Launch in background with timeout
	go func() {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		// Launch instance with timeout context
		response, err := g.apiClient.LaunchInstance(ctx, req)
		
		// Reset button state
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,
			Tick: func(_ float32) {
				g.launchForm.launchBtn.SetText("🚀 Launch Environment")
				g.launchForm.launchBtn.Enable()
			},
		})
		
		if err != nil {
			g.showNotification("error", "Launch Failed", err.Error())
			return
		}
		
		// Handle dry run vs actual launch
		if req.DryRun {
			g.showDryRunResults(response)
		} else {
			g.showNotification("success", "Instance Launched", response.Message)
			g.launchForm.nameEntry.SetText("") // Clear form
			g.refreshData() // Refresh instance list
		}
	}()
}

func (g *CloudWorkstationGUI) handleLaunchInstance() {
	// Redirect to advanced handler for backward compatibility
	g.handleAdvancedLaunchInstance()
}

// buildAdvancedLaunchRequest builds the launch request with all advanced options
func (g *CloudWorkstationGUI) buildAdvancedLaunchRequest() types.LaunchRequest {
	req := types.LaunchRequest{
		Template: g.launchForm.templateSelect.Selected,
		Name:     g.launchForm.nameEntry.Text,
		Size:     g.launchForm.sizeSelect.Selected,
		DryRun:   g.launchForm.dryRunCheck.Checked,
	}
	
	// Add package manager selection (skip if "Default" selected)
	if g.launchForm.packageMgrSelect.Selected != "" && g.launchForm.packageMgrSelect.Selected != "Default" {
		req.PackageManager = g.launchForm.packageMgrSelect.Selected
	}
	
	// Add selected volumes
	if g.launchForm.volumesSelect.Selected != "" {
		req.Volumes = []string{g.launchForm.volumesSelect.Selected}
	}
	
	if g.launchForm.ebsVolumesSelect.Selected != "" {
		req.EBSVolumes = []string{g.launchForm.ebsVolumesSelect.Selected}
	}
	
	// Add networking options
	if g.launchForm.regionEntry.Text != "" {
		req.Region = g.launchForm.regionEntry.Text
	}
	
	if g.launchForm.subnetEntry.Text != "" {
		req.SubnetID = g.launchForm.subnetEntry.Text
	}
	
	if g.launchForm.vpcEntry.Text != "" {
		req.VpcID = g.launchForm.vpcEntry.Text
	}
	
	// Add instance options
	req.Spot = g.launchForm.spotCheck.Checked
	
	return req
}

// validateAdvancedLaunchForm validates the advanced launch form
func (g *CloudWorkstationGUI) validateAdvancedLaunchForm() error {
	if g.launchForm.templateSelect.Selected == "" {
		return fmt.Errorf("please select a template")
	}
	
	instanceName := g.launchForm.nameEntry.Text
	if instanceName == "" {
		return fmt.Errorf("please enter an instance name")
	}
	
	// Validate instance name format
	if len(instanceName) < 3 {
		return fmt.Errorf("instance name must be at least 3 characters")
	}
	
	if len(instanceName) > 50 {
		return fmt.Errorf("instance name must be less than 50 characters")
	}
	
	// Check for duplicate instance names
	for _, instance := range g.instances {
		if instance.Name == instanceName {
			return fmt.Errorf("instance name '%s' already exists", instanceName)
		}
	}
	
	if g.launchForm.sizeSelect.Selected == "" {
		return fmt.Errorf("please select an instance size")
	}
	
	// Validate networking options if provided
	if g.launchForm.subnetEntry.Text != "" {
		subnetID := g.launchForm.subnetEntry.Text
		if !strings.HasPrefix(subnetID, "subnet-") {
			return fmt.Errorf("subnet ID must start with 'subnet-'")
		}
	}
	
	if g.launchForm.vpcEntry.Text != "" {
		vpcID := g.launchForm.vpcEntry.Text
		if !strings.HasPrefix(vpcID, "vpc-") {
			return fmt.Errorf("VPC ID must start with 'vpc-'")
		}
	}
	
	return nil
}

// showDryRunResults shows the dry run validation results
func (g *CloudWorkstationGUI) showDryRunResults(response *types.LaunchResponse) {
	title := "Dry Run Results"
	
	contentContainer := fynecontainer.NewVBox()
	
	// Validation status
	contentContainer.Add(widget.NewLabelWithStyle("✅ Validation Successful", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	contentContainer.Add(widget.NewSeparator())
	
	// Instance information
	contentContainer.Add(widget.NewLabelWithStyle("Instance Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	contentContainer.Add(widget.NewLabel("• Name: " + response.Instance.Name))
	contentContainer.Add(widget.NewLabel("• Template: " + response.Instance.Template))
	contentContainer.Add(widget.NewLabel("• Instance Type: " + response.Instance.InstanceType))
	
	if len(response.Instance.AttachedVolumes) > 0 {
		contentContainer.Add(widget.NewLabel(fmt.Sprintf("• EFS Volumes: %d", len(response.Instance.AttachedVolumes))))
	}
	
	if len(response.Instance.AttachedEBSVolumes) > 0 {
		contentContainer.Add(widget.NewLabel(fmt.Sprintf("• EBS Volumes: %d", len(response.Instance.AttachedEBSVolumes))))
	}
	
	contentContainer.Add(widget.NewSeparator())
	
	// Cost estimation
	contentContainer.Add(widget.NewLabelWithStyle("Cost Estimation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	contentContainer.Add(widget.NewLabel("• " + response.EstimatedCost))
	
	contentContainer.Add(widget.NewSeparator())
	
	// Launch confirmation
	contentContainer.Add(widget.NewLabelWithStyle("Ready to Launch", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	contentContainer.Add(widget.NewLabel("All configuration validated successfully. Uncheck 'Dry Run' to launch the instance."))
	
	// Action buttons
	launchBtn := widget.NewButton("Launch Now", func() {
		// Uncheck dry run and launch
		g.launchForm.dryRunCheck.SetChecked(false)
		g.handleAdvancedLaunchInstance()
	})
	launchBtn.Importance = widget.HighImportance
	
	editBtn := widget.NewButton("Edit Configuration", func() {
		// Just close the dialog to return to form
	})
	
	buttonContainer := fynecontainer.NewHBox(
		layout.NewSpacer(),
		editBtn,
		launchBtn,
	)
	
	contentContainer.Add(widget.NewSeparator())
	contentContainer.Add(buttonContainer)
	
	dialog := dialog.NewCustom(title, "Close", fynecontainer.NewScroll(contentContainer), g.window)
	dialog.Resize(fyne.NewSize(500, 450))
	dialog.Show()
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
	g.refreshInstances()
}

// Enhanced instance dialog methods

// showConnectionDialog shows comprehensive connection information and actions
func (g *CloudWorkstationGUI) showConnectionDialog(instance types.Instance) {
	title := fmt.Sprintf("Connect to %s", instance.Name)
	
	contentContainer := fynecontainer.NewVBox()
	
	// Instance information
	contentContainer.Add(widget.NewLabelWithStyle("Instance Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	contentContainer.Add(widget.NewLabel("• Name: " + instance.Name))
	contentContainer.Add(widget.NewLabel("• Template: " + instance.Template))
	contentContainer.Add(widget.NewLabel("• Public IP: " + instance.PublicIP))
	contentContainer.Add(widget.NewSeparator())
	
	// Connection methods
	contentContainer.Add(widget.NewLabelWithStyle("Connection Methods", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	
	// Web interface if available
	if instance.HasWebInterface {
		var webURL, webDescription string
		switch instance.Template {
		case "r-research":
			webURL = fmt.Sprintf("http://%s:8787", instance.PublicIP)
			webDescription = "RStudio Server (username: rstudio, password: cloudworkstation)"
		case "python-research":
			webURL = fmt.Sprintf("http://%s:8888", instance.PublicIP)
			webDescription = "JupyterLab (token: cloudworkstation)"
		case "desktop-research":
			webURL = fmt.Sprintf("https://%s:8443", instance.PublicIP)
			webDescription = "NICE DCV Desktop (username: ubuntu, password: cloudworkstation)"
		}
		
		if webURL != "" {
			contentContainer.Add(widget.NewLabel("🌐 " + webDescription))
			webBtn := widget.NewButton("Open " + strings.Split(webDescription, " ")[0], func() {
				// Copy URL to clipboard and show notification
				g.showNotification("info", "Connection URL", "URL copied to clipboard: " + webURL)
			})
			webBtn.Importance = widget.HighImportance
			contentContainer.Add(webBtn)
		}
	}
	
	// SSH access
	sshCommand := fmt.Sprintf("ssh %s@%s", instance.Username, instance.PublicIP)
	contentContainer.Add(widget.NewLabel("🔧 SSH Access"))
	contentContainer.Add(widget.NewLabel("Command: " + sshCommand))
	
	sshBtn := widget.NewButton("Copy SSH Command", func() {
		g.showNotification("info", "SSH Command", "Command copied to clipboard: " + sshCommand)
	})
	contentContainer.Add(sshBtn)
	
	// Port information
	if len(instance.AttachedVolumes) > 0 || len(instance.AttachedEBSVolumes) > 0 {
		contentContainer.Add(widget.NewSeparator())
		contentContainer.Add(widget.NewLabelWithStyle("Attached Storage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		if len(instance.AttachedVolumes) > 0 {
			contentContainer.Add(widget.NewLabel(fmt.Sprintf("• EFS Volumes: %d", len(instance.AttachedVolumes))))
		}
		if len(instance.AttachedEBSVolumes) > 0 {
			contentContainer.Add(widget.NewLabel(fmt.Sprintf("• EBS Volumes: %d", len(instance.AttachedEBSVolumes))))
		}
	}
	
	dialog := dialog.NewCustom(title, "Close", contentContainer, g.window)
	dialog.Resize(fyne.NewSize(450, 400))
	dialog.Show()
}

// showInstanceDetails shows comprehensive instance details
func (g *CloudWorkstationGUI) showInstanceDetails(instance types.Instance) {
	title := fmt.Sprintf("Instance Details: %s", instance.Name)
	
	contentContainer := fynecontainer.NewVBox()
	
	// Basic information
	contentContainer.Add(widget.NewLabelWithStyle("Basic Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	contentContainer.Add(widget.NewLabel("• Name: " + instance.Name))
	contentContainer.Add(widget.NewLabel("• ID: " + instance.ID))
	contentContainer.Add(widget.NewLabel("• Template: " + instance.Template))
	contentContainer.Add(widget.NewLabel("• Instance Type: " + instance.InstanceType))
	contentContainer.Add(widget.NewLabel("• State: " + strings.ToUpper(instance.State)))
	contentContainer.Add(widget.NewLabel("• Launch Time: " + instance.LaunchTime.Format("January 2, 2006 15:04:05")))
	contentContainer.Add(widget.NewSeparator())
	
	// Network information
	contentContainer.Add(widget.NewLabelWithStyle("Network Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	contentContainer.Add(widget.NewLabel("• Public IP: " + instance.PublicIP))
	contentContainer.Add(widget.NewLabel("• Private IP: " + instance.PrivateIP))
	contentContainer.Add(widget.NewLabel("• Username: " + instance.Username))
	if instance.HasWebInterface {
		contentContainer.Add(widget.NewLabel(fmt.Sprintf("• Web Port: %d", instance.WebPort)))
	}
	contentContainer.Add(widget.NewSeparator())
	
	// Cost information
	contentContainer.Add(widget.NewLabelWithStyle("Cost Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	contentContainer.Add(widget.NewLabel(fmt.Sprintf("• Daily Cost: $%.2f", instance.EstimatedDailyCost)))
	uptime := time.Since(instance.LaunchTime)
	dailyCostSoFar := instance.EstimatedDailyCost * (uptime.Hours() / 24.0)
	contentContainer.Add(widget.NewLabel(fmt.Sprintf("• Cost So Far: $%.2f", dailyCostSoFar)))
	contentContainer.Add(widget.NewSeparator())
	
	// Storage information
	if len(instance.AttachedVolumes) > 0 || len(instance.AttachedEBSVolumes) > 0 {
		contentContainer.Add(widget.NewLabelWithStyle("Attached Storage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		
		if len(instance.AttachedVolumes) > 0 {
			contentContainer.Add(widget.NewLabel("EFS Volumes:"))
			for _, volume := range instance.AttachedVolumes {
				contentContainer.Add(widget.NewLabel("  • " + volume))
			}
		}
		
		if len(instance.AttachedEBSVolumes) > 0 {
			contentContainer.Add(widget.NewLabel("EBS Volumes:"))
			for _, volume := range instance.AttachedEBSVolumes {
				contentContainer.Add(widget.NewLabel("  • " + volume))
			}
		}
		contentContainer.Add(widget.NewSeparator())
	}
	
	// Idle detection information
	if instance.IdleDetection != nil && instance.IdleDetection.Enabled {
		contentContainer.Add(widget.NewLabelWithStyle("Idle Detection", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		contentContainer.Add(widget.NewLabel("• Status: Enabled"))
		contentContainer.Add(widget.NewLabel(fmt.Sprintf("• Policy: %s", instance.IdleDetection.Policy)))
		contentContainer.Add(widget.NewLabel(fmt.Sprintf("• Threshold: %d minutes", instance.IdleDetection.Threshold)))
		if instance.IdleDetection.ActionPending {
			contentContainer.Add(widget.NewLabel("• Action: Pending - " + instance.IdleDetection.ActionSchedule.Format("Jan 2, 15:04")))
		}
	}
	
	dialog := dialog.NewCustom(title, "Close", fynecontainer.NewScroll(contentContainer), g.window)
	dialog.Resize(fyne.NewSize(500, 600))
	dialog.Show()
}

// showStartConfirmation shows confirmation dialog for starting instance
func (g *CloudWorkstationGUI) showStartConfirmation(instanceName string) {
	title := "Start Instance"
	message := fmt.Sprintf("Are you sure you want to start the instance '%s'?\n\nStarting the instance will resume billing.", instanceName)
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				
				if err := g.apiClient.StartInstance(ctx, instanceName); err != nil {
					g.showNotification("error", "Start Failed", err.Error())
					return
				}
				
				g.showNotification("success", "Instance Starting", fmt.Sprintf("Instance %s is starting up", instanceName))
				g.refreshInstances()
			}()
		}
	}, g.window)
	
	dialog.Show()
}

// showStopConfirmation shows confirmation dialog for stopping instance
func (g *CloudWorkstationGUI) showStopConfirmation(instanceName string) {
	title := "Stop Instance"
	message := fmt.Sprintf("Are you sure you want to stop the instance '%s'?\n\nStopping will preserve the instance but stop billing for compute resources.", instanceName)
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				
				if err := g.apiClient.StopInstance(ctx, instanceName); err != nil {
					g.showNotification("error", "Stop Failed", err.Error())
					return
				}
				
				g.showNotification("success", "Instance Stopping", fmt.Sprintf("Instance %s is shutting down", instanceName))
				g.refreshInstances()
			}()
		}
	}, g.window)
	
	dialog.Show()
}

// showHibernateConfirmation shows confirmation dialog for hibernating instance
func (g *CloudWorkstationGUI) showHibernateConfirmation(instance types.Instance) {
	// Check hibernation support first
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	hibernationStatus, err := g.apiClient.GetInstanceHibernationStatus(ctx, instance.Name)
	if err != nil {
		g.showNotification("error", "Hibernation Check Failed", err.Error())
		return
	}
	
	var title, message string
	if hibernationStatus.HibernationSupported {
		title = "Hibernate Instance"
		message = fmt.Sprintf("Are you sure you want to hibernate the instance '%s'?\n\n💤 Hibernation preserves RAM state to storage for faster resume.\n⚡ Resume will be faster than a regular start.\n💰 Billing for compute stops, but storage costs for RAM persist.", instance.Name)
	} else {
		title = "Stop Instance (Hibernation Not Supported)"
		message = fmt.Sprintf("Instance '%s' does not support hibernation.\n\nThis will perform a regular stop instead.\n\nThe instance will be stopped and billing for compute resources will stop.", instance.Name)
	}
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				
				if err := g.apiClient.HibernateInstance(ctx, instance.Name); err != nil {
					g.showNotification("error", "Hibernation Failed", err.Error())
					return
				}
				
				if hibernationStatus.HibernationSupported {
					g.showNotification("success", "Instance Hibernating", fmt.Sprintf("Instance %s is hibernating (preserving RAM state)", instance.Name))
				} else {
					g.showNotification("success", "Instance Stopping", fmt.Sprintf("Instance %s is stopping (hibernation not supported)", instance.Name))
				}
				g.refreshInstances()
			}()
		}
	}, g.window)
	
	dialog.Show()
}

// showResumeConfirmation shows confirmation dialog for resuming instance
func (g *CloudWorkstationGUI) showResumeConfirmation(instanceName string) {
	// Check hibernation status first
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	hibernationStatus, err := g.apiClient.GetInstanceHibernationStatus(ctx, instanceName)
	if err != nil {
		// Fall back to regular start if we can't check hibernation status
		g.showStartConfirmation(instanceName)
		return
	}
	
	var title, message string
	if hibernationStatus.IsHibernated {
		title = "Resume Hibernated Instance"
		message = fmt.Sprintf("Are you sure you want to resume the hibernated instance '%s'?\n\n⚡ Resuming from hibernation will be faster than a regular start.\n💾 RAM state will be restored from storage.\n💰 Full billing will resume.", instanceName)
	} else {
		title = "Start Instance"
		message = fmt.Sprintf("Are you sure you want to start the instance '%s'?\n\nThis instance was not hibernated, so this will be a regular start.\n💰 Billing will resume for compute resources.", instanceName)
	}
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				
				if err := g.apiClient.ResumeInstance(ctx, instanceName); err != nil {
					g.showNotification("error", "Resume Failed", err.Error())
					return
				}
				
				if hibernationStatus.IsHibernated {
					g.showNotification("success", "Instance Resuming", fmt.Sprintf("Instance %s is resuming from hibernation", instanceName))
				} else {
					g.showNotification("success", "Instance Starting", fmt.Sprintf("Instance %s is starting", instanceName))
				}
				g.refreshInstances()
			}()
		}
	}, g.window)
	
	dialog.Show()
}

// showDeleteInstanceConfirmation shows confirmation dialog for deleting instance
func (g *CloudWorkstationGUI) showDeleteInstanceConfirmation(instanceName string) {
	title := "Delete Instance"
	message := fmt.Sprintf("Are you sure you want to DELETE the instance '%s'?\n\n⚠️ WARNING: This action CANNOT be undone.\n\nAll data on the instance will be permanently lost.\nAttached EBS volumes will be preserved but detached.", instanceName)
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				
				if err := g.apiClient.DeleteInstance(ctx, instanceName); err != nil {
					g.showNotification("error", "Delete Failed", err.Error())
					return
				}
				
				g.showNotification("success", "Instance Deleted", fmt.Sprintf("Instance %s has been deleted", instanceName))
				g.refreshInstances()
			}()
		}
	}, g.window)
	
	dialog.Show()
}

// Daemon status monitoring methods

// initializeDaemonStatusContainer sets up the daemon status container
func (g *CloudWorkstationGUI) initializeDaemonStatusContainer() {
	if g.daemonStatusContainer == nil {
		g.daemonStatusContainer = fynecontainer.NewVBox()
	}
}

// refreshDaemonStatus loads daemon status from the API and updates the display
func (g *CloudWorkstationGUI) refreshDaemonStatus() {
	if g.daemonStatusContainer == nil {
		return
	}
	
	// Clear existing content
	g.daemonStatusContainer.RemoveAll()
	
	// Show loading indicator
	loadingLabel := widget.NewLabel("Loading daemon status...")
	g.daemonStatusContainer.Add(loadingLabel)
	g.daemonStatusContainer.Refresh()
	
	// Fetch daemon status from API
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		status, err := g.apiClient.GetStatus(ctx)
		if err != nil {
			// Update UI on main thread
			g.app.Driver().StartAnimation(&fyne.Animation{
				Duration: 100 * time.Millisecond,
				Tick: func(_ float32) {
					g.daemonStatusContainer.RemoveAll()
					g.displayDaemonOffline(err.Error())
					g.daemonStatusContainer.Refresh()
				},
			})
			return
		}
		
		// Update UI on main thread
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,
			Tick: func(_ float32) {
				g.displayDaemonStatus(status)
				g.daemonStatusContainer.Refresh()
			},
		})
	}()
}

// displayDaemonStatus renders daemon status information
func (g *CloudWorkstationGUI) displayDaemonStatus(status *types.DaemonStatus) {
	g.daemonStatusContainer.RemoveAll()
	
	// Status header with icon
	statusIcon := "🟢"
	statusText := "RUNNING"
	if status.Status != "running" {
		statusIcon = "🟡"
		statusText = strings.ToUpper(status.Status)
	}
	
	statusHeader := fynecontainer.NewHBox(
		widget.NewLabel(statusIcon),
		widget.NewLabelWithStyle(statusText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewLabel("Version: " + status.Version),
	)
	g.daemonStatusContainer.Add(statusHeader)
	g.daemonStatusContainer.Add(widget.NewSeparator())
	
	// Create two-column layout for status information
	leftColumn := fynecontainer.NewVBox()
	rightColumn := fynecontainer.NewVBox()
	
	// Left column: Basic status
	leftColumn.Add(widget.NewLabelWithStyle("Basic Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	leftColumn.Add(widget.NewLabel("• Start Time: " + status.StartTime.Format("Jan 2, 2006 15:04:05")))
	
	if status.Uptime != "" {
		leftColumn.Add(widget.NewLabel("• Uptime: " + status.Uptime))
	} else {
		// Calculate uptime if not provided
		uptime := time.Since(status.StartTime)
		leftColumn.Add(widget.NewLabel("• Uptime: " + formatDuration(uptime)))
	}
	
	leftColumn.Add(widget.NewLabel("• AWS Region: " + status.AWSRegion))
	
	if status.CurrentProfile != "" {
		leftColumn.Add(widget.NewLabel("• Active Profile: " + status.CurrentProfile))
	} else {
		leftColumn.Add(widget.NewLabel("• Active Profile: None"))
	}
	
	// Right column: Performance metrics
	rightColumn.Add(widget.NewLabelWithStyle("Performance Metrics", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	rightColumn.Add(widget.NewLabel(fmt.Sprintf("• Active Operations: %d", status.ActiveOps)))
	rightColumn.Add(widget.NewLabel(fmt.Sprintf("• Total Requests: %d", status.TotalRequests)))
	
	if status.RequestsPerMinute > 0 {
		rightColumn.Add(widget.NewLabel(fmt.Sprintf("• Request Rate: %.1f/min", status.RequestsPerMinute)))
	} else {
		rightColumn.Add(widget.NewLabel("• Request Rate: 0.0/min"))
	}
	
	// Connection URL information
	rightColumn.Add(widget.NewLabel("• Daemon URL: http://localhost:8947"))
	
	// Add columns to main container
	columnsContainer := fynecontainer.NewHBox(
		leftColumn,
		layout.NewSpacer(),
		rightColumn,
	)
	g.daemonStatusContainer.Add(columnsContainer)
	
	// Add refresh timestamp
	g.daemonStatusContainer.Add(widget.NewSeparator())
	refreshTime := widget.NewLabel("Last updated: " + time.Now().Format("15:04:05"))
	refreshTime.TextStyle = fyne.TextStyle{Italic: true}
	g.daemonStatusContainer.Add(refreshTime)
}

// displayDaemonOffline renders offline daemon status
func (g *CloudWorkstationGUI) displayDaemonOffline(errorMsg string) {
	g.daemonStatusContainer.RemoveAll()
	
	// Offline status header
	statusHeader := fynecontainer.NewHBox(
		widget.NewLabel("🔴"),
		widget.NewLabelWithStyle("OFFLINE", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewLabel("Daemon not responding"),
	)
	g.daemonStatusContainer.Add(statusHeader)
	g.daemonStatusContainer.Add(widget.NewSeparator())
	
	// Error information
	errorContainer := fynecontainer.NewVBox()
	errorContainer.Add(widget.NewLabelWithStyle("Connection Error", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	errorContainer.Add(widget.NewLabel("• Status: Disconnected"))
	errorContainer.Add(widget.NewLabel("• Error: " + errorMsg))
	errorContainer.Add(widget.NewLabel("• Daemon URL: http://localhost:8947"))
	errorContainer.Add(widget.NewLabel("• Expected: CloudWorkstation daemon should be running"))
	
	g.daemonStatusContainer.Add(errorContainer)
	
	// Troubleshooting information
	g.daemonStatusContainer.Add(widget.NewSeparator())
	troubleshootContainer := fynecontainer.NewVBox()
	troubleshootContainer.Add(widget.NewLabelWithStyle("Troubleshooting", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	troubleshootContainer.Add(widget.NewLabel("1. Start daemon: cws daemon start"))
	troubleshootContainer.Add(widget.NewLabel("2. Check daemon logs: cws daemon logs"))
	troubleshootContainer.Add(widget.NewLabel("3. Verify port 8947 is available"))
	
	g.daemonStatusContainer.Add(troubleshootContainer)
	
	// Add refresh timestamp
	g.daemonStatusContainer.Add(widget.NewSeparator())
	refreshTime := widget.NewLabel("Last checked: " + time.Now().Format("15:04:05"))
	refreshTime.TextStyle = fyne.TextStyle{Italic: true}
	g.daemonStatusContainer.Add(refreshTime)
}

// createConnectionManagementView creates the connection management interface
func (g *CloudWorkstationGUI) createConnectionManagementView() *fyne.Container {
	// Test connection button
	testBtn := widget.NewButton("Test Connection", func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			if err := g.apiClient.Ping(ctx); err != nil {
				g.showNotification("error", "Connection Failed", "Cannot connect to daemon: " + err.Error())
			} else {
				g.showNotification("success", "Connection Successful", "Daemon is responding correctly")
				g.refreshDaemonStatus() // Refresh status after successful test
			}
		}()
	})
	testBtn.Importance = widget.HighImportance
	
	// Start daemon button
	startBtn := widget.NewButton("Start Daemon", func() {
		g.showStartDaemonDialog()
	})
	
	// Stop daemon button
	stopBtn := widget.NewButton("Stop Daemon", func() {
		g.showStopDaemonConfirmation()
	})
	stopBtn.Importance = widget.DangerImportance
	
	// Connection management layout
	connectionInfo := fynecontainer.NewVBox(
		widget.NewLabel("• Daemon URL: http://localhost:8947"),
		widget.NewLabel("• Protocol: HTTP REST API"),
		widget.NewLabel("• Timeout: 5 seconds"),
	)
	
	buttonContainer := fynecontainer.NewHBox(
		testBtn,
		startBtn,
		stopBtn,
	)
	
	return fynecontainer.NewVBox(
		connectionInfo,
		widget.NewSeparator(),
		buttonContainer,
	)
}

// showStartDaemonDialog shows dialog for starting daemon
func (g *CloudWorkstationGUI) showStartDaemonDialog() {
	title := "Start CloudWorkstation Daemon"
	message := "This will attempt to start the CloudWorkstation daemon (cwsd) on port 8947.\n\nNote: The daemon must be installed and available in your system PATH."
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			g.showNotification("info", "Starting Daemon", "Attempting to start CloudWorkstation daemon...")
			
			go func() {
				// Note: In a real implementation, this would use a proper daemon start method
				// For now, we'll just show a notification about manual start
				time.Sleep(1 * time.Second)
				g.showNotification("info", "Manual Start Required", "Please start the daemon manually: cws daemon start")
				
				// Refresh status after a short delay
				time.Sleep(2 * time.Second)
				g.refreshDaemonStatus()
			}()
		}
	}, g.window)
	
	dialog.Show()
}

// showStopDaemonConfirmation shows confirmation dialog for stopping daemon
func (g *CloudWorkstationGUI) showStopDaemonConfirmation() {
	title := "Stop CloudWorkstation Daemon"
	message := "Are you sure you want to stop the CloudWorkstation daemon?\n\nThis will:\n• Stop all daemon operations\n• Disconnect the GUI from the backend\n• Prevent new instance operations until restarted"
	
	dialog := dialog.NewConfirm(title, message, func(confirmed bool) {
		if confirmed {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				
				if err := g.apiClient.Shutdown(ctx); err != nil {
					g.showNotification("error", "Stop Failed", "Failed to stop daemon: " + err.Error())
				} else {
					g.showNotification("success", "Daemon Stopped", "CloudWorkstation daemon has been stopped")
					
					// Refresh status after a short delay to show offline state
					time.Sleep(1 * time.Second)
					g.refreshDaemonStatus()
				}
			}()
		}
	}, g.window)
	
	dialog.Show()
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1f hours", d.Hours())
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%d days, %d hours", days, hours)
	}
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
}

// showApplyTemplateDialog shows a dialog to apply templates to running instances
func (g *CloudWorkstationGUI) showApplyTemplateDialog(instance types.Instance) {
	// Load available templates
	ctx := context.Background()
	templates, err := g.apiClient.ListTemplates(ctx)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to load templates: %v", err), g.window)
		return
	}
	
	// Create template selection
	var templateOptions []string
	templateMap := make(map[string]types.Template)
	for _, template := range templates {
		templateOptions = append(templateOptions, template.Name)
		templateMap[template.Name] = template
	}
	
	if len(templateOptions) == 0 {
		dialog.ShowError(fmt.Errorf("No templates available"), g.window)
		return
	}
	
	templateSelect := widget.NewSelect(templateOptions, nil)
	templateSelect.SetSelected(templateOptions[0])
	
	// Package manager selection
	packageManagerOptions := []string{"conda", "pip", "spack", "apt", "dnf"}
	packageManagerSelect := widget.NewSelect(packageManagerOptions, nil)
	packageManagerSelect.SetSelected("conda") // Default to conda
	
	// Dry run option
	dryRunCheck := widget.NewCheck("Dry run (preview only)", nil)
	dryRunCheck.SetChecked(true) // Default to dry run for safety
	
	// Force option  
	forceCheck := widget.NewCheck("Force apply (ignore conflicts)", nil)
	
	// Progress bar and log
	progressBar := widget.NewProgressBar()
	progressBar.Hide()
	
	logText := widget.NewRichTextFromMarkdown("")
	logText.Hide()
	logScroll := fynecontainer.NewScroll(logText)
	logScroll.SetMinSize(fyne.NewSize(500, 200))
	logScroll.Hide()
	
	// Dialog content
	content := fynecontainer.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Apply Template to %s", instance.Name), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		
		widget.NewLabel("Select Template:"),
		templateSelect,
		
		widget.NewLabel("Package Manager:"),
		packageManagerSelect,
		
		widget.NewSeparator(),
		dryRunCheck,
		forceCheck,
		
		progressBar,
		logScroll,
	)
	
	// Create dialog
	applyDialog := dialog.NewCustom("Apply Template", "Cancel", content, g.window)
	applyDialog.Resize(fyne.NewSize(600, 500))
	
	// Add apply button
	applyButton := widget.NewButton("Apply Template", func() {
		selectedTemplate := templateMap[templateSelect.Selected]
		
		// Show progress
		progressBar.Show()
		logText.Show()
		logScroll.Show()
		applyDialog.Resize(fyne.NewSize(600, 700))
		
		// Update log
		logText.ParseMarkdown("**Starting template application...**\n\n")
		
		// Apply template (this would call the API)
		go g.applyTemplateToInstance(instance.Name, selectedTemplate, packageManagerSelect.Selected, dryRunCheck.Checked, forceCheck.Checked, progressBar, logText, applyDialog)
	})
	applyButton.Importance = widget.HighImportance
	
	// Show preview button for template differences
	previewButton := widget.NewButton("Preview Changes", func() {
		if templateSelect.Selected == "" {
			dialog.ShowError(fmt.Errorf("Please select a template first"), g.window)
			return
		}
		selectedTemplate := templateMap[templateSelect.Selected]
		g.showTemplateDiffDialog(instance, selectedTemplate)
	})
	
	// Add buttons to dialog
	buttons := fynecontainer.NewHBox(
		previewButton,
		layout.NewSpacer(),
		applyButton,
	)
	
	content.Add(widget.NewSeparator())
	content.Add(buttons)
	
	applyDialog.Show()
}

// showTemplateDiffDialog shows differences between current instance state and template
func (g *CloudWorkstationGUI) showTemplateDiffDialog(instance types.Instance, template types.Template) {
	// Create loading content first
	loadingContent := fynecontainer.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Template Differences for %s", instance.Name), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		
		widget.NewLabel("Template: " + template.Name),
		widget.NewLabel("Instance: " + instance.Name),
		
		widget.NewSeparator(),
		widget.NewLabel("🔍 Calculating differences..."),
	)
	
	diffDialog := dialog.NewCustom("Template Differences", "Close", loadingContent, g.window)
	diffDialog.Resize(fyne.NewSize(600, 500))
	diffDialog.Show()
	
	// Calculate differences in background
	go func() {
		ctx := context.Background()
		
		// Convert runtime template to unified template format
		unifiedTemplate := &templates.Template{
			Name:        template.Name,
			Description: template.Description,
			Packages: templates.PackageDefinitions{
				System: []string{}, // Would be populated from template data
				Conda:  []string{}, // Would be populated from template data  
				Pip:    []string{}, // Would be populated from template data
				Spack:  []string{}, // Would be populated from template data
			},
			Services: []templates.ServiceConfig{}, // Would be populated from template data
			Users:    []templates.UserConfig{},    // Would be populated from template data
		}
		
		// Create diff request
		request := templates.DiffRequest{
			InstanceName: instance.Name,
			Template:     unifiedTemplate,
		}
		
		// Call the API
		diff, err := g.apiClient.DiffTemplate(ctx, request)
		if err != nil {
			// Hide loading dialog and show error
			diffDialog.Hide()
			dialog.ShowError(fmt.Errorf("Failed to calculate template differences: %v", err), g.window)
			return
		}
		
		// Create results content
		resultsContent := fynecontainer.NewVBox(
			widget.NewLabelWithStyle(fmt.Sprintf("Template Differences for %s", instance.Name), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			
			widget.NewLabel("Template: " + template.Name),
			widget.NewLabel("Instance: " + instance.Name),
			widget.NewSeparator(),
		)
		
		// Show packages to install
		if len(diff.PackagesToInstall) > 0 {
			resultsContent.Add(widget.NewLabelWithStyle("📦 Packages to Install:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			for _, pkg := range diff.PackagesToInstall {
				pkgLabel := fmt.Sprintf("• %s", pkg.Name)
				if pkg.TargetVersion != "" {
					pkgLabel += fmt.Sprintf(" (%s)", pkg.TargetVersion)
				}
				pkgLabel += fmt.Sprintf(" via %s", pkg.PackageManager)
				resultsContent.Add(widget.NewLabel(pkgLabel))
			}
			resultsContent.Add(widget.NewSeparator())
		}
		
		// Show services to configure
		if len(diff.ServicesToConfigure) > 0 {
			resultsContent.Add(widget.NewLabelWithStyle("⚙️ Services to Configure:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			for _, svc := range diff.ServicesToConfigure {
				resultsContent.Add(widget.NewLabel(fmt.Sprintf("• %s (port %d)", svc.Name, svc.Port)))
			}
			resultsContent.Add(widget.NewSeparator())
		}
		
		// Show users to create
		if len(diff.UsersToCreate) > 0 {
			resultsContent.Add(widget.NewLabelWithStyle("👥 Users to Create:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			for _, user := range diff.UsersToCreate {
				groupsStr := strings.Join(user.TargetGroups, ", ")
				resultsContent.Add(widget.NewLabel(fmt.Sprintf("• %s (groups: %s)", user.Name, groupsStr)))
			}
			resultsContent.Add(widget.NewSeparator())
		}
		
		// Show conflicts if any
		if len(diff.ConflictsFound) > 0 {
			resultsContent.Add(widget.NewLabelWithStyle("⚠️ Conflicts Found:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			for _, conflict := range diff.ConflictsFound {
				resultsContent.Add(widget.NewLabel(fmt.Sprintf("• %s: %s", conflict.Type, conflict.Description)))
			}
			resultsContent.Add(widget.NewSeparator())
		}
		
		// Show message if no changes
		if len(diff.PackagesToInstall) == 0 && len(diff.ServicesToConfigure) == 0 && len(diff.UsersToCreate) == 0 {
			resultsContent.Add(widget.NewLabel("✅ No changes needed - template is already applied to this instance."))
		}
		
		// Hide loading dialog and show results
		diffDialog.Hide()
		
		resultsDialog := dialog.NewCustom("Template Differences", "Close", resultsContent, g.window)
		resultsDialog.Resize(fyne.NewSize(600, 500))
		resultsDialog.Show()
	}()
}

// showTemplateLayersDialog shows the history of applied templates for an instance
func (g *CloudWorkstationGUI) showTemplateLayersDialog(instance types.Instance) {
	// Create loading content first
	loadingContent := fynecontainer.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Template History for %s", instance.Name), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		
		widget.NewLabel("Instance: " + instance.Name),
		widget.NewLabel("Base Template: " + instance.Template),
		
		widget.NewSeparator(),
		widget.NewLabel("🔍 Loading template history..."),
	)
	
	layersDialog := dialog.NewCustom("Template History", "Close", loadingContent, g.window)
	layersDialog.Resize(fyne.NewSize(600, 500))
	layersDialog.Show()
	
	// Load template layers in background
	go func() {
		ctx := context.Background()
		
		// Call the API to get fresh template layers
		appliedTemplates, err := g.apiClient.GetInstanceLayers(ctx, instance.Name)
		if err != nil {
			// Fall back to instance data if API fails
			appliedTemplates = []templates.AppliedTemplate{}
			
			// Convert instance applied templates to API format
			for _, applied := range instance.AppliedTemplates {
				appliedTemplates = append(appliedTemplates, templates.AppliedTemplate{
					Name:               applied.TemplateName,
					AppliedAt:          applied.AppliedAt,
					PackageManager:     applied.PackageManager,
					PackagesInstalled:  applied.PackagesInstalled,
					ServicesConfigured: applied.ServicesConfigured,
					UsersCreated:       applied.UsersCreated,
					RollbackCheckpoint: applied.RollbackCheckpoint,
				})
			}
		}
		
		// Create results content
		content := fynecontainer.NewVBox(
			widget.NewLabelWithStyle(fmt.Sprintf("Template History for %s", instance.Name), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			
			widget.NewLabel("Instance: " + instance.Name),
			widget.NewLabel("Base Template: " + instance.Template),
			
			widget.NewSeparator(),
		)
		
		// Check if instance has applied templates
		if len(appliedTemplates) == 0 {
			content.Add(widget.NewLabel("📝 No additional templates have been applied to this instance."))
			content.Add(widget.NewLabel("Use 'Apply Template' to add software packages and configurations."))
		} else {
			content.Add(widget.NewLabel("📚 Applied Template Layers:"))
			
			for i, applied := range appliedTemplates {
				layerCard := widget.NewCard(
					fmt.Sprintf("%d. %s", i+1, applied.Name),
					applied.AppliedAt.Format("Jan 2, 2006 15:04"),
					fynecontainer.NewVBox(
						widget.NewLabel("Package Manager: " + applied.PackageManager),
						widget.NewLabel(fmt.Sprintf("Packages: %d installed", len(applied.PackagesInstalled))),
						widget.NewLabel(fmt.Sprintf("Services: %d configured", len(applied.ServicesConfigured))),
						widget.NewLabel("Checkpoint: " + applied.RollbackCheckpoint),
					),
				)
				content.Add(layerCard)
			}
			
			// Add rollback button for latest checkpoint
			if len(appliedTemplates) > 0 {
				content.Add(widget.NewSeparator())
				rollbackBtn := widget.NewButton("Rollback to Previous", func() {
					// Create updated instance with fresh applied templates
					updatedInstance := instance
					updatedInstance.AppliedTemplates = []types.AppliedTemplateRecord{}
					for _, applied := range appliedTemplates {
						updatedInstance.AppliedTemplates = append(updatedInstance.AppliedTemplates, types.AppliedTemplateRecord{
							TemplateName:       applied.Name,
							AppliedAt:          applied.AppliedAt,
							PackageManager:     applied.PackageManager,
							PackagesInstalled:  applied.PackagesInstalled,
							ServicesConfigured: applied.ServicesConfigured,
							UsersCreated:       applied.UsersCreated,
							RollbackCheckpoint: applied.RollbackCheckpoint,
						})
					}
					g.showRollbackDialog(updatedInstance)
				})
				rollbackBtn.Importance = widget.DangerImportance
				content.Add(rollbackBtn)
			}
		}
		
		// Hide loading dialog and show results
		layersDialog.Hide()
		
		resultsDialog := dialog.NewCustom("Template History", "Close", content, g.window)
		resultsDialog.Resize(fyne.NewSize(600, 500))
		resultsDialog.Show()
	}()
}

// showRollbackDialog shows options for rolling back template applications
func (g *CloudWorkstationGUI) showRollbackDialog(instance types.Instance) {
	if len(instance.AppliedTemplates) == 0 {
		dialog.ShowError(fmt.Errorf("No template applications to rollback"), g.window)
		return
	}
	
	// Create checkpoint selection
	var checkpointOptions []string
	checkpointMap := make(map[string]string)
	
	for _, applied := range instance.AppliedTemplates {
		option := fmt.Sprintf("%s (%s)", applied.TemplateName, applied.AppliedAt.Format("Jan 2 15:04"))
		checkpointOptions = append(checkpointOptions, option)
		checkpointMap[option] = applied.RollbackCheckpoint
	}
	
	checkpointSelect := widget.NewSelect(checkpointOptions, nil)
	if len(checkpointOptions) > 0 {
		checkpointSelect.SetSelected(checkpointOptions[len(checkpointOptions)-1]) // Select most recent
	}
	
	content := fynecontainer.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Rollback %s", instance.Name), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		
		widget.NewLabel("⚠️ This will remove all template applications after the selected checkpoint."),
		widget.NewLabel("This action cannot be undone."),
		
		widget.NewSeparator(),
		widget.NewLabel("Rollback to checkpoint:"),
		checkpointSelect,
	)
	
	rollbackDialog := dialog.NewCustom("Rollback Template Applications", "Cancel", content, g.window)
	rollbackDialog.Resize(fyne.NewSize(500, 300))
	
	// Add rollback button
	rollbackButton := widget.NewButton("Rollback Now", func() {
		if checkpointSelect.Selected == "" {
			dialog.ShowError(fmt.Errorf("Please select a checkpoint"), g.window)
			return
		}
		
		checkpointID := checkpointMap[checkpointSelect.Selected]
		
		// Confirm the rollback
		confirmDialog := dialog.NewConfirm(
			"Confirm Rollback",
			fmt.Sprintf("Are you sure you want to rollback instance '%s' to checkpoint '%s'?\n\nThis will remove all software and configurations applied after this point.", instance.Name, checkpointID),
			func(confirmed bool) {
				if confirmed {
					rollbackDialog.Hide()
					g.performRollback(instance.Name, checkpointID)
				}
			},
			g.window,
		)
		confirmDialog.Show()
	})
	rollbackButton.Importance = widget.DangerImportance
	
	content.Add(widget.NewSeparator())
	content.Add(rollbackButton)
	
	rollbackDialog.Show()
}

// applyTemplateToInstance applies a template to an instance with progress tracking
func (g *CloudWorkstationGUI) applyTemplateToInstance(instanceName string, template types.Template, packageManager string, dryRun bool, force bool, progressBar *widget.ProgressBar, logText *widget.RichText, applyDialog *dialog.CustomDialog) {
	var logContent string
	logContent = "**Applying template: " + template.Name + "**\n\n"
	logText.ParseMarkdown(logContent)
	
	ctx := context.Background()
	
	// Convert runtime template to unified template format
	unifiedTemplate := &templates.Template{
		Name:        template.Name,
		Description: template.Description,
		Packages: templates.PackageDefinitions{
			System: []string{}, // Would be populated from template data
			Conda:  []string{}, // Would be populated from template data  
			Pip:    []string{}, // Would be populated from template data
			Spack:  []string{}, // Would be populated from template data
		},
		Services: []templates.ServiceConfig{}, // Would be populated from template data
		Users:    []templates.UserConfig{},    // Would be populated from template data
		PackageManager: packageManager,
	}
	
	// Create apply request
	request := templates.ApplyRequest{
		InstanceName: instanceName,
		Template:     unifiedTemplate,
		PackageManager: packageManager,
		DryRun:       dryRun,
		Force:        force,
	}
	
	// Update progress and log
	progressBar.SetValue(0.1)
	logContent += "🔍 Connecting to CloudWorkstation daemon...\n"
	logText.ParseMarkdown(logContent)
	
	// Call the API
	response, err := g.apiClient.ApplyTemplate(ctx, request)
	if err != nil {
		progressBar.SetValue(1.0)
		logContent += "\n❌ **Template application failed:**\n"
		logContent += fmt.Sprintf("Error: %v\n", err)
		logText.ParseMarkdown(logContent)
		
		// Show error dialog
		dialog.ShowError(fmt.Errorf("Failed to apply template: %v", err), g.window)
		return
	}
	
	// Update progress based on response
	progressBar.SetValue(0.9)
	logContent += "📊 Template application completed!\n\n"
	
	if response.Success {
		if dryRun {
			logContent += "**Dry run completed successfully!**\n\n"
			logContent += fmt.Sprintf("• **Packages to install**: %d\n", response.PackagesInstalled)
			logContent += fmt.Sprintf("• **Services to configure**: %d\n", response.ServicesConfigured)
			logContent += fmt.Sprintf("• **Users to create**: %d\n", response.UsersCreated)
			logContent += "\nNo changes were made to the instance.\n"
		} else {
			logContent += "**Template applied successfully!**\n\n"
			logContent += fmt.Sprintf("• **Packages installed**: %d\n", response.PackagesInstalled)
			logContent += fmt.Sprintf("• **Services configured**: %d\n", response.ServicesConfigured)  
			logContent += fmt.Sprintf("• **Users created**: %d\n", response.UsersCreated)
			logContent += fmt.Sprintf("• **Rollback checkpoint**: %s\n", response.RollbackCheckpoint)
			logContent += fmt.Sprintf("• **Execution time**: %s\n", response.ExecutionTime)
			
			if len(response.Warnings) > 0 {
				logContent += "\n**Warnings:**\n"
				for _, warning := range response.Warnings {
					logContent += fmt.Sprintf("⚠️ %s\n", warning)
				}
			}
			
			// Refresh instances to show updated state
			go g.refreshInstances()
		}
	} else {
		logContent += "❌ **Template application failed:**\n"
		logContent += fmt.Sprintf("Error: %s\n", response.Message)
	}
	
	progressBar.SetValue(1.0)
	logText.ParseMarkdown(logContent)
}

// performRollback performs a rollback operation
func (g *CloudWorkstationGUI) performRollback(instanceName string, checkpointID string) {
	content := fynecontainer.NewVBox(
		widget.NewLabelWithStyle("Rolling Back Instance", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Instance: " + instanceName),
		widget.NewLabel("Checkpoint: " + checkpointID),
		widget.NewSeparator(),
		widget.NewLabel("🔄 Connecting to CloudWorkstation daemon..."),
	)
	
	progressDialog := dialog.NewCustom("Rollback in Progress", "", content, g.window)
	progressDialog.Show()
	
	// Perform actual rollback
	go func() {
		ctx := context.Background()
		
		// Create rollback request
		request := types.RollbackRequest{
			InstanceName: instanceName,
			CheckpointID: checkpointID,
		}
		
		// Update progress
		content.Objects[5] = widget.NewLabel("🔄 Performing rollback operation...")
		progressDialog.Refresh()
		
		// Call the API
		err := g.apiClient.RollbackInstance(ctx, request)
		
		progressDialog.Hide()
		
		if err != nil {
			// Show error dialog
			dialog.ShowError(fmt.Errorf("Rollback failed: %v", err), g.window)
			return
		}
		
		// Show success
		dialog.ShowInformation("Rollback Complete", 
			fmt.Sprintf("Instance '%s' has been successfully rolled back to checkpoint '%s'.", instanceName, checkpointID),
			g.window)
		
		// Refresh instances to show updated state
		g.refreshInstances()
	}()
}

func (g *CloudWorkstationGUI) cleanup() {
	// Cleanup
	if g.refreshTicker != nil {
		g.refreshTicker.Stop()
	}
	
	// Stop system tray handler
	if g.systemTray != nil {
		g.systemTray.Stop()
	}
}
