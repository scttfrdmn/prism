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
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
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

	// UI Components
	refreshTicker *time.Ticker

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
		app:       app.NewWithID("com.cloudworkstation.gui"),
		apiClient: api.NewContextClient("http://localhost:8080"),
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

	// Check daemon connectivity with retry logic
	if err := g.checkDaemonConnection(context.Background()); err != nil {
		g.showNotification("error", "Cannot connect to CloudWorkstation daemon", 
			"Make sure the daemon is running with 'cwsd'. GUI will retry automatically.")
		// Continue anyway - daemon might start later
	}

	// Initialize data
	g.refreshData()

	// Setup UI
	g.setupMainLayout()

	// Setup system tray if supported
	if desk, ok := g.app.(desktop.App); ok {
		g.setupSystemTray(desk)
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
	mainLayout := container.NewHSplit(
		g.sidebar,
		container.NewVBox(
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
	titleCard := container.NewVBox(
		widget.NewCard("", "",
			container.NewVBox(
				container.NewHBox(
					widget.NewIcon(theme.ComputerIcon()),
					widget.NewLabelWithStyle("CloudWorkstation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				),
				widget.NewLabelWithStyle(fmt.Sprintf("v%s", version.GetVersion()), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
				widget.NewSeparator(),
				container.NewHBox(
					widget.NewIcon(theme.InfoIcon()),
					widget.NewLabel(fmt.Sprintf("$%.2f/day", g.totalCost)),
				),
			),
		),
	)

	// Navigation buttons
	navButtons := container.NewVBox(
		g.createNavButton("🏠 Dashboard", SectionDashboard),
		g.createNavButton("💻 Instances", SectionInstances),
		g.createNavButton("📋 Templates", SectionTemplates),
		g.createNavButton("💾 Storage", SectionVolumes),
		g.createNavButton("💰 Billing", SectionBilling),
		g.createNavButton("⚙️ Settings", SectionSettings),
	)

	// Quick actions
	quickActions := widget.NewCard("Quick Actions", "",
		container.NewVBox(
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
		container.NewHBox(
			widget.NewIcon(theme.ConfirmIcon()),
			widget.NewRichTextFromMarkdown(fmt.Sprintf("**%s**", statusText)),
		),
	)

	// Combine sidebar elements
	g.sidebar = container.NewVBox(
		titleCard,
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
	g.content = container.NewStack()
}

// setupNotification creates the notification area
func (g *CloudWorkstationGUI) setupNotification() {
	g.notification = container.NewVBox()
	g.notification.Hide() // Hidden by default
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

// createDashboardView creates the main dashboard view
func (g *CloudWorkstationGUI) createDashboardView() fyne.CanvasObject {
	// Header
	header := container.NewHBox(
		widget.NewLabelWithStyle("Dashboard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButton("Refresh", func() {
			g.refreshData()
			g.showNotification("success", "Data refreshed", "")
		}),
	)

	// Overview cards
	overviewCards := container.NewGridWithColumns(3,
		widget.NewCard("Active Instances", "",
			container.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("%d", len(g.getRunningInstances())), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Currently running"),
			),
		),
		widget.NewCard("Daily Cost", "",
			container.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("$%.2f", g.totalCost), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Estimated per day"),
			),
		),
		widget.NewCard("Total Instances", "",
			container.NewVBox(
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
	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		overviewCards,
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			quickLaunchCard,
			recentInstancesCard,
		),
	)

	return container.NewScroll(content)
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

	form := container.NewVBox(
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
		return container.NewVBox(
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

		instanceItem := container.NewHBox(
			widget.NewLabel(statusIcon),
			container.NewVBox(
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

	return container.NewVBox(items...)
}

// createInstancesView creates the instances management view
func (g *CloudWorkstationGUI) createInstancesView() fyne.CanvasObject {
	header := container.NewHBox(
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
	instanceCards := container.NewVBox()

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

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		instanceCards,
	)

	return container.NewScroll(content)
}

// createInstanceCard creates a detailed card for an instance
func (g *CloudWorkstationGUI) createInstanceCard(instance types.Instance) *widget.Card {
	statusIcon := g.getStatusIcon(instance.State)

	// Instance details
	details := container.NewVBox(
		widget.NewLabelWithStyle(instance.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Template: %s", instance.Template)),
		widget.NewLabel(fmt.Sprintf("Cost: $%.2f/day", instance.EstimatedDailyCost)),
		widget.NewLabel(fmt.Sprintf("Launched: %s", instance.LaunchTime.Format("Jan 2, 2006 15:04"))),
	)

	// Status
	status := container.NewVBox(
		container.NewHBox(
			widget.NewLabel(statusIcon),
			widget.NewLabelWithStyle(instance.State, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
	)

	// Actions
	actions := container.NewVBox()

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
	cardContent := container.NewHBox(
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
	templateCards := container.NewGridWithColumns(2,
		widget.NewCard("R Research Environment", "RStudio Server + R packages for data science",
			container.NewVBox(
				widget.NewLabel("• RStudio Server"),
				widget.NewLabel("• Common R packages"),
				widget.NewLabel("• Jupyter Lab"),
				widget.NewButton("Launch R Environment", func() {
					g.quickLaunch("r-research")
				}),
			),
		),
		widget.NewCard("Python ML Environment", "Python + Jupyter + ML libraries",
			container.NewVBox(
				widget.NewLabel("• Jupyter Notebook"),
				widget.NewLabel("• TensorFlow & PyTorch"),
				widget.NewLabel("• Data science libraries"),
				widget.NewButton("Launch Python Environment", func() {
					g.quickLaunch("python-research")
				}),
			),
		),
		widget.NewCard("Basic Ubuntu Server", "Clean Ubuntu server for general use",
			container.NewVBox(
				widget.NewLabel("• Ubuntu 22.04 LTS"),
				widget.NewLabel("• Basic development tools"),
				widget.NewLabel("• Docker pre-installed"),
				widget.NewButton("Launch Ubuntu Server", func() {
					g.quickLaunch("basic-ubuntu")
				}),
			),
		),
		widget.NewCard("Custom Template", "Create your own environment",
			container.NewVBox(
				widget.NewLabel("• Custom AMI"),
				widget.NewLabel("• Custom instance type"),
				widget.NewLabel("• Custom configuration"),
				widget.NewButton("Coming Soon", nil),
			),
		),
	)

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		templateCards,
	)

	return container.NewScroll(content)
}

// createVolumesView creates the storage/volumes view
func (g *CloudWorkstationGUI) createVolumesView() *fyne.Container {
	header := widget.NewLabelWithStyle("Storage & Volumes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	content := container.NewVBox(
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
	costCards := container.NewGridWithColumns(2,
		widget.NewCard("Current Costs", "",
			container.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("$%.2f", g.totalCost), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Daily cost estimate"),
				widget.NewLabel(fmt.Sprintf("Monthly: ~$%.2f", g.totalCost*30)),
			),
		),
		widget.NewCard("Cost Breakdown", "",
			container.NewVBox(
				widget.NewLabel(fmt.Sprintf("Running instances: %d", len(g.getRunningInstances()))),
				widget.NewLabel(fmt.Sprintf("Total instances: %d", len(g.instances))),
				widget.NewLabel("Storage costs: $0.00"),
			),
		),
	)

	content := container.NewVBox(
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
		container.NewVBox(
			widget.NewLabel("Daemon URL: http://localhost:8080"),
			widget.NewLabel(fmt.Sprintf("Status: %s", func() string {
				if g.lastUpdate.IsZero() {
					return "Disconnected"
				}
				return "Connected"
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

	// About
	aboutCard := widget.NewCard("About", "CloudWorkstation information",
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Version: %s", version.GetVersion())),
			widget.NewLabel("A tool for managing cloud research environments"),
			widget.NewHyperlink("Documentation", nil),
			widget.NewHyperlink("GitHub Repository", nil),
		),
	)

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		connectionCard,
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
		content = container.NewHBox(
			widget.NewIcon(icon),
			container.NewVBox(
				widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(message),
			),
			layout.NewSpacer(),
			widget.NewButton("×", func() {
				g.notification.Hide()
			}),
		)
	} else {
		content = container.NewHBox(
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
		}
	}()
}

func (g *CloudWorkstationGUI) setupSystemTray(desk desktop.App) {
	// Create minimal system tray menu
	menu := fyne.NewMenu("CloudWorkstation",
		fyne.NewMenuItem("Open CloudWorkstation", func() {
			g.window.Show()
			g.window.RequestFocus()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			g.app.Quit()
		}),
	)

	desk.SetSystemTrayMenu(menu)
}

func (g *CloudWorkstationGUI) run() {
	// Show window and run
	g.window.ShowAndRun()

	// Cleanup
	if g.refreshTicker != nil {
		g.refreshTicker.Stop()
	}
}
