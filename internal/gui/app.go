package gui

import (
	"embed"
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/internal/gui/pages"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend/dist
var assets embed.FS

// InitializeApp creates and configures the Wails application
func InitializeApp() (*wails.App, error) {
	// Initialize profile managers
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return nil, fmt.Errorf("failed to create profile manager: %w", err)
	}

	secureManager, err := profile.NewSecureInvitationManager(profileManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure invitation manager: %w", err)
	}

	// Create the application with options
	app := wails.NewApp(&options.App{
		Title:  "CloudWorkstation",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup: func(ctx wails.Context) {
			// Initialize pages with context
			dashboardPage := pages.NewDashboardPage(ctx, profileManager)
			profilePage := pages.NewProfilePage(ctx, profileManager)
			invitationPage := pages.NewInvitationPage(ctx, secureManager)
			batchInvitationPage := pages.NewBatchInvitationPage(ctx, secureManager, assets)
			
			// Bind pages to frontend
			ctx.Bind(dashboardPage)
			ctx.Bind(profilePage)
			ctx.Bind(invitationPage)
			ctx.Bind(batchInvitationPage)
		},
	})

	return app, nil
}

// RunGUI starts the GUI application
func RunGUI() error {
	app, err := InitializeApp()
	if err != nil {
		return err
	}
	
	return app.Run()
}