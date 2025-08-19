package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
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
	app.Window.NewWithOptions(application.WebviewWindowOptions{
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

	// Run the application
	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}