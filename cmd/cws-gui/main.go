package main

import (
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
)

const (
	version = "0.1.0"
)

// NavigationSection represents different sections of the app
type NavigationSection int

const (
	SectionDashboard NavigationSection = iota
	SectionInstances
	SectionTemplates
	SectionVolumes
	SectionBilling
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
	instances     []types.Instance
	templates     map[string]types.Template
	totalCost     float64
	lastUpdate    time.Time
	
	// UI Components
	refreshTicker *time.Ticker
	
	// Form state
	launchForm struct {
		templateSelect *widget.Select
		nameEntry     *widget.Entry
		sizeSelect    *widget.Select
		launchBtn     *widget.Button
	}
}

func main() {
	log.Printf("CloudWorkstation GUI v%s starting...", version)
	
	// Create the application
	gui := &CloudWorkstationGUI{
		app:       app.NewWithID("com.cloudworkstation.gui"),
		apiClient: api.NewClient("http://localhost:8080"),
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
	metadata.Version = version
	
	// Create main window
	g.window = g.app.NewWindow("CloudWorkstation")
	g.window.Resize(fyne.NewSize(1200, 800))
	g.window.SetMaster()
	
	// Check daemon connectivity
	if err := g.apiClient.Ping(); err != nil {
		g.showNotification("error", "Cannot connect to CloudWorkstation daemon", "Make sure it's running with 'cwsd'")
		// Continue anyway for demo purposes
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
				widget.NewLabelWithStyle(fmt.Sprintf("v%s", version), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
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
				if err := g.apiClient.Ping(); err != nil {
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
			widget.NewLabel(fmt.Sprintf("Version: %s", version)),
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
	if g.launchForm.templateSelect.Selected == "" || g.launchForm.nameEntry.Text == "" {
		g.showNotification("error", "Validation Error", "Please select a template and enter an instance name")
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
	
	// Launch in background
	go func() {
		response, err := g.apiClient.LaunchInstance(req)
		
		// Update UI on main thread
		g.app.Driver().StartAnimation(&fyne.Animation{
			Duration: 100 * time.Millisecond,
			Tick: func(f float32) {
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

func (g *CloudWorkstationGUI) handleConnectInstance(name string) {
	connectionInfo, err := g.apiClient.ConnectInstance(name)
	if err != nil {
		g.showNotification("error", "Connection Failed", err.Error())
		return
	}
	
	g.showNotification("info", "Connection Information", connectionInfo)
}

func (g *CloudWorkstationGUI) handleStartInstance(name string) {
	if err := g.apiClient.StartInstance(name); err != nil {
		g.showNotification("error", "Start Failed", err.Error())
		return
	}
	
	g.showNotification("success", "Instance Starting", fmt.Sprintf("Instance %s is starting up", name))
	g.refreshData()
}

func (g *CloudWorkstationGUI) handleStopInstance(name string) {
	if err := g.apiClient.StopInstance(name); err != nil {
		g.showNotification("error", "Stop Failed", err.Error())
		return
	}
	
	g.showNotification("success", "Instance Stopping", fmt.Sprintf("Instance %s is shutting down", name))
	g.refreshData()
}

func (g *CloudWorkstationGUI) handleDeleteInstance(name string) {
	if err := g.apiClient.DeleteInstance(name); err != nil {
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

func (g *CloudWorkstationGUI) refreshData() {
	// Fetch instances
	response, err := g.apiClient.ListInstances()
	if err != nil {
		log.Printf("Failed to refresh instance data: %v", err)
		return
	}
	
	g.instances = response.Instances
	g.totalCost = response.TotalCost
	g.lastUpdate = time.Now()
	
	// Refresh current view
	g.navigateToSection(g.currentSection)
}

func (g *CloudWorkstationGUI) startBackgroundRefresh() {
	// Initial refresh
	g.refreshData()
	
	// Start ticker for periodic refresh
	g.refreshTicker = time.NewTicker(30 * time.Second)
	go func() {
		for range g.refreshTicker.C {
			g.refreshData()
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