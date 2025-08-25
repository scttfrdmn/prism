package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/scttfrdmn/cloudworkstation/pkg/version"
	"github.com/spf13/cobra"
)

// NewGUICommand creates a new gui command
func NewGUICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gui",
		Short: "Launch the graphical user interface",
		Long: `Launch the CloudWorkstation Graphical User Interface (GUI).

This provides a professional desktop interface for managing your cloud workstations.
The GUI includes template browsing, instance management, remote desktop connections,
and comprehensive settings configuration.`,
		Run: func(cmd *cobra.Command, args []string) {
			runGUI()
		},
	}

	cmd.Flags().Bool("minimize", false, "Start minimized to system tray")
	cmd.Flags().Bool("autostart", false, "Configure to start automatically at login")
	cmd.Flags().Bool("remove-autostart", false, "Remove automatic startup configuration")

	return cmd
}

// runGUI launches the graphical UI
func runGUI() {
	// Check if daemon is running
	if err := checkDaemonForGUI(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "check daemon for GUI"))
		fmt.Println("Attempting to start daemon...")

		if err := startDaemonForGUI(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", FormatErrorForCLI(err, "start daemon for GUI"))
			fmt.Println("Please start the daemon manually with: cws daemon start")
			os.Exit(1)
		}

		fmt.Println("Daemon started successfully.")
	}

	// Print GUI initialization message
	fmt.Printf("Starting CloudWorkstation GUI v%s...\n", version.GetVersion())

	// Find and execute the GUI binary
	guiPath, err := findGUIBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GUI Error: %s\n", FormatErrorForCLI(err, "find GUI binary"))
		fmt.Println("\nTo install GUI support:")
		fmt.Println("  1. Install Wails CLI: go install github.com/wailsapp/wails/v3/cmd/wails@latest")
		fmt.Println("  2. Build GUI: make build-gui")
		fmt.Println("  3. Or use TUI instead: cws tui")
		os.Exit(1)
	}

	// Execute the GUI with any passed flags
	args := os.Args[2:] // Skip "cws gui"
	cmd := exec.Command(guiPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "GUI error: %v\n", err)
		os.Exit(1)
	}
}

// checkDaemonForGUI verifies if the daemon is running by checking the API endpoint
func checkDaemonForGUI() error {
	// Check if daemon is responding on port 8947 using HTTP ping
	cmd := exec.Command("curl", "-s", "-f", "http://localhost:8947/api/v1/ping")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("daemon not responding on port 8947")
	}

	// If we get any response, daemon is running
	if len(output) > 0 {
		return nil
	}

	return fmt.Errorf("daemon API not responding")
}

// startDaemonForGUI attempts to start the daemon if not already running
func startDaemonForGUI() error {
	// Double-check daemon isn't running (avoid port conflicts)
	if checkDaemonForGUI() == nil {
		return nil // Already running
	}

	// First, try to find cwsd binary
	cwsdPath, _ := exec.LookPath("cwsd")
	if cwsdPath == "" {
		// Try in bin directory
		cwsdPath = "./bin/cwsd"
		if _, err := os.Stat(cwsdPath); os.IsNotExist(err) {
			cwsdPath = "../bin/cwsd"
			if _, err := os.Stat(cwsdPath); os.IsNotExist(err) {
				return fmt.Errorf("daemon executable not found in PATH or bin directory")
			}
		}
	}

	// Start daemon
	cmd := exec.Command(cwsdPath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Start in background
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start daemon: %v", err)
	}

	// If successful, detach
	_ = cmd.Process.Release()

	// Wait a moment for daemon to initialize
	waitCmd := exec.Command("sleep", "3")
	_ = waitCmd.Run()

	// Final check that daemon started successfully
	if err := checkDaemonForGUI(); err != nil {
		return fmt.Errorf("daemon started but not responding: %v", err)
	}

	return nil
}

// findGUIBinary locates the cws-gui binary
func findGUIBinary() (string, error) {
	// Try in PATH first
	if guiPath, err := exec.LookPath("cws-gui"); err == nil {
		return guiPath, nil
	}

	// Try in bin directory (relative to current working directory)
	paths := []string{
		"./bin/cws-gui",
		"../bin/cws-gui",
		"./cmd/cws-gui/cws-gui", // Development location
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf(`GUI binary 'cws-gui' not found

The CloudWorkstation GUI requires the cws-gui binary to be built and available.

Installation options:
  • Homebrew (includes GUI): brew install cloudworkstation
  • Manual build: make build-gui (requires Wails CLI)
  • Alternative: Use TUI instead with 'cws tui'

For GUI development:
  • Install Wails: go install github.com/wailsapp/wails/v3/cmd/wails@latest
  • Build GUI: make build-gui`)
}
