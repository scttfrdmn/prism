package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/scttfrdmn/cloudworkstation/internal/tui"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
	"github.com/spf13/cobra"
)

// NewTUICommand creates a new tui command
func NewTUICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive terminal UI",
		Long: `Launch the CloudWorkstation Terminal User Interface (TUI).
		
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
	// Check if daemon is running
	if err := checkDaemonForTUI(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Println("Attempting to start daemon...")

		if err := startDaemonForTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start daemon: %v\n", err)
			fmt.Println("Please start the daemon manually with: cws daemon start")
			os.Exit(1)
		}

		fmt.Println("Daemon started successfully.")
	}

	// Print TUI initialization message
	fmt.Printf("Starting CloudWorkstation TUI v%s...\n", version.GetVersion())

	// Create and run the TUI application
	app := tui.NewApp()
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\\n", err)
		os.Exit(1)
	}
}

// checkDaemonForTUI verifies if the daemon is running by checking the API endpoint
func checkDaemonForTUI() error {
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

// startDaemonForTUI attempts to start the daemon if not already running
func startDaemonForTUI() error {
	// Double-check daemon isn't running (avoid port conflicts)
	if checkDaemonForTUI() == nil {
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
	if err := checkDaemonForTUI(); err != nil {
		return fmt.Errorf("daemon started but not responding: %v", err)
	}

	return nil
}
