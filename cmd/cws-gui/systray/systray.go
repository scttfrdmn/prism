package systray

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// SystemTrayHandler manages the system tray functionality
type SystemTrayHandler struct {
	app            desktop.App
	window         fyne.Window
	apiClient      api.CloudWorkstationAPI
	menu           *fyne.Menu
	statusIcon     fyne.Resource
	refreshTicker  *time.Ticker
	instances      []types.Instance
	totalCost      float64
	lastUpdate     time.Time
	isRunning      bool
	onStatusChange func(connected bool)
}

// NewSystemTrayHandler creates a new system tray handler
func NewSystemTrayHandler(app desktop.App, window fyne.Window, apiClient api.CloudWorkstationAPI) *SystemTrayHandler {
	return &SystemTrayHandler{
		app:        app,
		window:     window,
		apiClient:  apiClient,
		statusIcon: theme.ComputerIcon(),
		isRunning:  false,
	}
}

// Setup initializes the system tray with menus and actions
func (h *SystemTrayHandler) Setup() {
	// Create status menu items
	statusMenuItem := fyne.NewMenuItem("Status: Unknown", nil)
	statusMenuItem.Disabled = true
	
	costMenuItem := fyne.NewMenuItem("Cost: $0.00/day", nil)
	costMenuItem.Disabled = true
	
	instanceCountMenuItem := fyne.NewMenuItem("Instances: 0", nil)
	instanceCountMenuItem.Disabled = true
	
	// Instance submenu will be populated dynamically
	
	// Create main menu
	menu := fyne.NewMenu("CloudWorkstation",
		statusMenuItem,
		costMenuItem,
		instanceCountMenuItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Show Dashboard", func() {
			h.window.Show()
			h.window.RequestFocus()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Launch Instances", nil),
		fyne.NewMenuItem("R Research", func() {
			h.window.Show()
			h.window.RequestFocus()
			// TODO: Trigger quick launch for R Research
		}),
		fyne.NewMenuItem("Python ML", func() {
			h.window.Show()
			h.window.RequestFocus()
			// TODO: Trigger quick launch for Python ML
		}),
		fyne.NewMenuItem("Basic Ubuntu", func() {
			h.window.Show()
			h.window.RequestFocus()
			// TODO: Trigger quick launch for Basic Ubuntu
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Refresh Status", func() {
			h.refreshStatus()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			h.window.Close()
		}),
	)
	
	// Store references for updating later
	h.menu = menu
	
	// Set the menu
	h.app.SetSystemTrayMenu(menu)
	h.app.SetSystemTrayIcon(h.statusIcon)
}

// Start begins background refresh for the system tray
func (h *SystemTrayHandler) Start() {
	if h.isRunning {
		return
	}
	
	h.isRunning = true
	
	// Initial refresh
	h.refreshStatus()
	
	// Start ticker for periodic refresh
	h.refreshTicker = time.NewTicker(30 * time.Second)
	go func() {
		for range h.refreshTicker.C {
			if !h.isRunning {
				return
			}
			h.refreshStatus()
		}
	}()
}

// Stop stops the background refresh
func (h *SystemTrayHandler) Stop() {
	if h.refreshTicker != nil {
		h.refreshTicker.Stop()
	}
	h.isRunning = false
}

// SetOnStatusChange sets a callback for status changes
func (h *SystemTrayHandler) SetOnStatusChange(callback func(connected bool)) {
	h.onStatusChange = callback
}

// refreshStatus updates the system tray with current status
func (h *SystemTrayHandler) refreshStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Try to fetch instance data
	response, err := h.apiClient.ListInstances(ctx)
	if err != nil {
		h.updateTrayForDisconnected()
		return
	}
	
	// Update state
	h.instances = response.Instances
	h.totalCost = response.TotalCost
	h.lastUpdate = time.Now()
	
	// Update status display
	h.updateTrayForConnected()
}

// updateTrayForConnected updates the tray for connected state
func (h *SystemTrayHandler) updateTrayForConnected() {
	if h.menu == nil || len(h.menu.Items) < 3 {
		return // Menu not initialized yet
	}
	
	// Count running instances
	runningCount := 0
	for _, instance := range h.instances {
		if instance.State == "running" {
			runningCount++
		}
	}
	
	// Update status menu items
	h.menu.Items[0].Label = "Status: Connected"
	h.menu.Items[1].Label = fmt.Sprintf("Cost: $%.2f/day", h.totalCost)
	h.menu.Items[2].Label = fmt.Sprintf("Instances: %d (%d running)", len(h.instances), runningCount)
	
	// Update icon based on status
	if runningCount > 0 {
		h.statusIcon = theme.ConfirmIcon() // Green check for running instances
	} else {
		h.statusIcon = theme.ComputerIcon() // Default icon for no running instances
	}
	
	// Update icon
	h.app.SetSystemTrayIcon(h.statusIcon)
	
	// Call status change callback if defined
	if h.onStatusChange != nil {
		h.onStatusChange(true)
	}
	
	// Update instances submenu
	h.updateInstancesSubmenu()
}

// updateTrayForDisconnected updates the tray for disconnected state
func (h *SystemTrayHandler) updateTrayForDisconnected() {
	if h.menu == nil || len(h.menu.Items) < 3 {
		return // Menu not initialized yet
	}
	
	// Update status menu items
	h.menu.Items[0].Label = "Status: Disconnected"
	h.menu.Items[1].Label = "Cost: Unknown"
	h.menu.Items[2].Label = "Instances: Unknown"
	
	// Use warning icon for disconnected state
	h.statusIcon = theme.WarningIcon()
	h.app.SetSystemTrayIcon(h.statusIcon)
	
	// Call status change callback if defined
	if h.onStatusChange != nil {
		h.onStatusChange(false)
	}
}

// updateInstancesSubmenu updates the instances submenu
func (h *SystemTrayHandler) updateInstancesSubmenu() {
	// Find the Launch Instances menu item
	for i, item := range h.menu.Items {
		if item.Label == "Launch Instances" && i+1 < len(h.menu.Items) {
			// Create instance action items based on current instances
			var instanceItems []*fyne.MenuItem
			
			// First add running instances
			for _, instance := range h.instances {
				if instance.State == "running" {
					// Create a stable copy of instance for closure
					inst := instance
					
					// Create menu item for this instance
					instanceItem := fyne.NewMenuItem(
						fmt.Sprintf("🟢 %s (%s)", inst.Name, inst.Template),
						func() {
							h.window.Show()
							h.window.RequestFocus()
							// TODO: Navigate to this instance in the UI
						},
					)
					
					// Add connect/stop options
					instanceSubMenu := fyne.NewMenu("",
						fyne.NewMenuItem("Connect", func() {
							h.connectToInstance(inst.Name)
						}),
						fyne.NewMenuItem("Stop", func() {
							h.stopInstance(inst.Name)
						}),
					)
					instanceItem.ChildMenu = instanceSubMenu
					
					instanceItems = append(instanceItems, instanceItem)
				}
			}
			
			// Then add stopped instances
			for _, instance := range h.instances {
				if instance.State == "stopped" {
					// Create a stable copy of instance for closure
					inst := instance
					
					// Create menu item for this instance
					instanceItem := fyne.NewMenuItem(
						fmt.Sprintf("🟡 %s (%s)", inst.Name, inst.Template),
						func() {
							h.window.Show()
							h.window.RequestFocus()
							// TODO: Navigate to this instance in the UI
						},
					)
					
					// Add start option
					instanceSubMenu := fyne.NewMenu("",
						fyne.NewMenuItem("Start", func() {
							h.startInstance(inst.Name)
						}),
					)
					instanceItem.ChildMenu = instanceSubMenu
					
					instanceItems = append(instanceItems, instanceItem)
				}
			}
			
			// If no instances, add a placeholder
			if len(instanceItems) == 0 {
				noInstanceItem := fyne.NewMenuItem("No instances available", nil)
				noInstanceItem.Disabled = true
				instanceItems = append(instanceItems, noInstanceItem)
			}
			
			// Create or update the Instances menu
			if h.menu.Items[i+1].ChildMenu == nil {
				// No existing child menu, create new submenu
				h.menu.Items[i].ChildMenu = fyne.NewMenu("Instances", instanceItems...)
			} else {
				// Update existing child menu
				h.menu.Items[i].ChildMenu.Items = instanceItems
			}
			
			// No need to continue searching
			break
		}
	}
}

// Helper methods for instance management from system tray

func (h *SystemTrayHandler) connectToInstance(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get instance details
	instance, err := h.apiClient.GetInstance(ctx, name)
	if err != nil {
		// TODO: Show error notification
		return
	}
	
	// TODO: Show connection information
	fmt.Printf("Would connect to instance: %s at %s\n", instance.Name, instance.PublicIP)
}

func (h *SystemTrayHandler) startInstance(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := h.apiClient.StartInstance(ctx, name); err != nil {
		// TODO: Show error notification
		return
	}
	
	// Refresh status after action
	h.refreshStatus()
}

func (h *SystemTrayHandler) stopInstance(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := h.apiClient.StopInstance(ctx, name); err != nil {
		// TODO: Show error notification
		return
	}
	
	// Refresh status after action
	h.refreshStatus()
}