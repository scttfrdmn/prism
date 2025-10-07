package main

import (
	"embed"
	"flag"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Parse command line flags
	var (
		minimizeToTray  = flag.Bool("minimize", false, "Start minimized to system tray")
		autoStart       = flag.Bool("autostart", false, "Configure to start automatically at login")
		removeAutoStart = flag.Bool("remove-autostart", false, "Remove automatic startup configuration")
		help            = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	// Handle special flags
	if *help {
		showHelp()
		return
	}

	if *autoStart {
		if err := configureAutoStart(true); err != nil {
			log.Printf("Failed to configure auto-start: %v", err)
			os.Exit(1)
		}
		log.Println("✅ Auto-start configured successfully")
		return
	}

	if *removeAutoStart {
		if err := configureAutoStart(false); err != nil {
			log.Printf("Failed to remove auto-start: %v", err)
			os.Exit(1)
		}
		log.Println("✅ Auto-start removed successfully")
		return
	}

	// Create CloudWorkstation GUI application
	app := application.New(application.Options{
		Name:        "CloudWorkstation",
		Description: "Academic Research Computing Platform - Professional GUI",
		Services: []application.Service{
			application.NewService(NewCloudWorkstationService()),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create main window with professional styling
	_ = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "CloudWorkstation",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(248, 250, 252), // Clean light background
		URL:              "/",
		Width:            1200,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
	})

	// Handle minimize to tray option
	if *minimizeToTray {
		// Hide window on startup (system tray functionality would go here)
		log.Println("⚠️  System tray functionality not yet implemented")
	}

	// Run the application
	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// showHelp displays command line help
func showHelp() {
	log.Printf(`CloudWorkstation GUI v0.5.1

Usage: cws-gui [OPTIONS]

OPTIONS:
  -autostart          Configure to start automatically at login
  -remove-autostart   Remove automatic startup configuration  
  -minimize          Start minimized to system tray (planned)
  -help              Show this help

STARTUP CONFIGURATION:
  # Enable auto-start at login
  cws-gui -autostart

  # Remove auto-start configuration
  cws-gui -remove-autostart

  # Start minimized (when system tray is implemented)
  cws-gui -minimize

EXAMPLES:
  cws-gui                    # Start normally
  cws-gui -autostart        # Configure auto-start
  cws-gui -remove-autostart # Remove auto-start
`)
}
