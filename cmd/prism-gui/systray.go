package main

import (
	_ "embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Embedded icon assets for system tray
//
//go:embed assets/tray-icon.png
var trayIconData []byte

//go:embed assets/tray-icon-dark.png
var trayIconDarkData []byte

// SystemTrayManager manages the system tray icon and menu
type SystemTrayManager struct {
	app     *application.App
	tray    *application.SystemTray
	window  *application.WebviewWindow
	service *PrismService
}

// NewSystemTrayManager creates a new system tray manager
func NewSystemTrayManager(app *application.App, window *application.WebviewWindow, service *PrismService) *SystemTrayManager {
	return &SystemTrayManager{
		app:     app,
		window:  window,
		service: service,
	}
}

// Setup initializes the system tray with icon and menu
func (stm *SystemTrayManager) Setup() error {
	log.Println("🎨 Setting up system tray...")

	// Create system tray (API changed in Wails v3 alpha.60)
	stm.tray = stm.app.SystemTray.New()

	// Set the tray icons. SetIcon/SetDarkModeIcon return the tray for chaining,
	// not an error: treating that non-nil pointer as one logged a bogus
	// "Failed to set tray icon" with the whole icon dumped on every launch.
	stm.tray.SetIcon(trayIconData).SetDarkModeIcon(trayIconDarkData)

	// Set tooltip
	stm.tray.SetTooltip("Prism - Cloud Workstations")

	// Attach window to tray (clicking tray toggles window)
	stm.tray.AttachWindow(stm.window)

	// Create and set menu
	menu := stm.createMenu()
	stm.tray.SetMenu(menu)

	log.Println("✅ System tray setup complete")
	return nil
}

// createMenu creates the system tray context menu
func (stm *SystemTrayManager) createMenu() *application.Menu {
	menu := stm.app.NewMenu()

	// Show/Hide Window
	menu.Add("Show Window").
		SetAccelerator("CmdOrCtrl+Shift+P").
		OnClick(func(ctx *application.Context) {
			if stm.window.IsVisible() {
				stm.window.Hide()
			} else {
				stm.window.Show()
				stm.window.Focus()
			}
		})

	menu.AddSeparator()

	// Quick Launch submenu
	quickLaunch := menu.AddSubmenu("Quick Launch")
	stm.addQuickLaunchItems(quickLaunch)

	menu.AddSeparator()

	// My Workspaces
	menu.Add("My Workspaces").
		OnClick(func(ctx *application.Context) {
			stm.window.Show()
			stm.window.Focus()
			stm.window.ExecJS("window.dispatchEvent(new CustomEvent('prism-navigate',{detail:'workspaces'}))")
		})

	// Cost Summary
	menu.Add("Cost Summary").
		OnClick(func(ctx *application.Context) {
			stm.window.Show()
			stm.window.Focus()
			stm.window.ExecJS("window.dispatchEvent(new CustomEvent('prism-navigate',{detail:'dashboard'}))")
		})

	menu.AddSeparator()

	// Settings
	menu.Add("Settings").
		SetAccelerator("CmdOrCtrl+,").
		OnClick(func(ctx *application.Context) {
			stm.window.Show()
			stm.window.Focus()
			stm.window.ExecJS("window.dispatchEvent(new CustomEvent('prism-navigate',{detail:'settings'}))")
		})

	menu.AddSeparator()

	// Quit
	menu.Add("Quit Prism").
		SetAccelerator("CmdOrCtrl+Q").
		OnClick(func(ctx *application.Context) {
			log.Println("User requested quit from system tray")
			stm.app.Quit()
		})

	return menu
}

// addQuickLaunchItems adds template quick launch items to submenu
func (stm *SystemTrayManager) addQuickLaunchItems(submenu *application.Menu) {
	// Add most common templates as quick launch options
	templates := []string{
		"Python Machine Learning",
		"R Research Environment",
		"Rocky Linux 9 + Conda Stack",
		"Basic Ubuntu",
	}

	for _, template := range templates {
		templateName := template // Capture for closure
		submenu.Add(templateName).
			OnClick(func(ctx *application.Context) {
				stm.window.Show()
				stm.window.Focus()
				stm.window.ExecJS("window.dispatchEvent(new CustomEvent('prism-navigate',{detail:'templates'}))")
				log.Printf("Quick launch requested: %s", templateName)
			})
	}

	submenu.AddSeparator()

	submenu.Add("Browse All Templates...").
		OnClick(func(ctx *application.Context) {
			stm.window.Show()
			stm.window.Focus()
			stm.window.ExecJS("window.dispatchEvent(new CustomEvent('prism-navigate',{detail:'templates'}))")
		})
}

// Show shows the main window
func (stm *SystemTrayManager) Show() {
	if stm.window != nil {
		stm.window.Show()
		stm.window.Focus()
	}
}

// Hide hides the main window
func (stm *SystemTrayManager) Hide() {
	if stm.window != nil {
		stm.window.Hide()
	}
}

// IsVisible returns whether the main window is visible
func (stm *SystemTrayManager) IsVisible() bool {
	if stm.window != nil {
		return stm.window.IsVisible()
	}
	return false
}

// UpdateMenu updates the tray menu (useful for dynamic content)
func (stm *SystemTrayManager) UpdateMenu() {
	if stm.tray != nil {
		menu := stm.createMenu()
		stm.tray.SetMenu(menu)
	}
}
