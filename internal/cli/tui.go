package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/scttfrdmn/prism/internal/tui"
	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/version"
	"github.com/spf13/cobra"
)

// NewTUICommand creates a new tui command
func NewTUICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive terminal UI",
		Long: `Launch the Prism Terminal User Interface (TUI).

This provides an interactive terminal interface for managing your cloud workstations.
Press 'q' or 'Esc' at any time to exit the TUI.`,
		Run: func(cmd *cobra.Command, args []string) {
			runTUI()
		},
	}

	return cmd
}

// runTUI launches the terminal UI
func runTUI() {
	// Check if auto-start is disabled via environment variable
	if os.Getenv(AutoStartDisableEnvVar) != "" {
		// Auto-start disabled, just check if daemon is running
		apiClient := client.NewClient(DefaultDaemonURL)
		if err := apiClient.Ping(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n\n💡 Tip: Auto-start is disabled via %s environment variable\n",
				DaemonNotRunningMessage, AutoStartDisableEnvVar)
			os.Exit(1)
		}
	} else {
		// Ensure daemon is running with auto-start
		if err := ensureDaemonForTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "ensure daemon for TUI"))
			os.Exit(1)
		}
	}

	// Print TUI initialization message
	fmt.Printf("Starting Prism TUI v%s...\n", version.GetVersion())

	// Create and run the TUI application
	app := tui.NewApp()
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

// ensureDaemonForTUI ensures the daemon is running, with auto-start if needed
func ensureDaemonForTUI() error {
	// Create a temporary App instance to use the ensureDaemonRunning logic
	// This reuses all the singleton enforcement, binary discovery, and version checking
	config := &Config{}
	config.Daemon.URL = DefaultDaemonURL
	if envURL := os.Getenv(DaemonURLEnvVar); envURL != "" {
		config.Daemon.URL = envURL
	}

	// Load API key from daemon state if available
	apiKey := loadAPIKeyFromState()

	apiClient := client.NewClientWithOptions(config.Daemon.URL, client.Options{
		APIKey: apiKey,
	})
	ctx := context.Background()

	// Check if daemon is already running
	if err := apiClient.Ping(ctx); err == nil {
		// Daemon is running, check version compatibility
		if err := apiClient.CheckVersionCompatibility(ctx, version.GetVersion()); err != nil {
			return fmt.Errorf("version compatibility check failed: %w", err)
		}
		return nil // Already running and compatible
	}

	// Auto-start daemon with user feedback
	fmt.Println(DaemonAutoStartMessage)
	fmt.Printf("⏳ Please wait while the daemon initializes (typically 2-3 seconds)...\n")

	// Use SystemCommands to start the daemon (reuses all the sophisticated logic)
	app := &App{
		version:   version.GetVersion(),
		apiClient: apiClient,
		ctx:       ctx,
		config:    config,
	}
	app.systemCommands = NewSystemCommands(app)

	if err := app.systemCommands.Daemon([]string{"start"}); err != nil {
		fmt.Println(DaemonAutoStartFailedMessage)
		fmt.Printf("\n💡 Troubleshooting:\n")
		fmt.Printf("   • Check if 'cwsd' binary is in your PATH\n")
		fmt.Printf("   • Try manual start: cws daemon start\n")
		fmt.Printf("   • Check daemon logs for errors\n")
		return WrapAPIError("auto-start daemon", err)
	}

	fmt.Println(DaemonAutoStartSuccessMessage)

	// Reload API key after daemon start (daemon generates new key on first start)
	apiKey = loadAPIKeyFromState()
	apiClient = client.NewClientWithOptions(config.Daemon.URL, client.Options{
		APIKey: apiKey,
	})

	// Check version compatibility after successful start
	if err := apiClient.CheckVersionCompatibility(ctx, version.GetVersion()); err != nil {
		return fmt.Errorf("version compatibility check failed after daemon auto-start: %w", err)
	}

	return nil
}
