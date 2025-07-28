package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/cloudworkstation/internal/tui"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
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

// checkDaemonForTUI verifies if the daemon is running
func checkDaemonForTUI() error {
	// Use the daemon status command to check if daemon is running
	cmd := exec.Command("cwsd", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try to find cwsd binary
		cwsdPath, _ := exec.LookPath("cwsd")
		if cwsdPath == "" {
			// Try in bin directory
			cwsdPath = "./bin/cwsd"
			if _, err := os.Stat(cwsdPath); os.IsNotExist(err) {
				cwsdPath = "../bin/cwsd"
			}
			
			if _, err := os.Stat(cwsdPath); os.IsNotExist(err) {
				return fmt.Errorf("daemon executable not found in PATH or bin directory")
			}
			
			cmd = exec.Command(cwsdPath, "status")
			output, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("daemon not running: %v", err)
			}
		} else {
			return fmt.Errorf("daemon not running: %v", err)
		}
	}

	// Check output for running status (simplified check)
	if string(output) == "" {
		return fmt.Errorf("daemon is not running")
	}

	return nil
}

// startDaemonForTUI attempts to start the daemon
func startDaemonForTUI() error {
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
	cmd.Process.Release()
	
	// Wait a moment for daemon to initialize
	waitCmd := exec.Command("sleep", "2")
	waitCmd.Run()

	return nil
}