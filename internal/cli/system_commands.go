// Package cli implements system and daemon management commands.
//
// This file contains all daemon and system-related functionality extracted from the main App struct
// following the architectural pattern of separating concerns into specialized command modules.
//
// SystemCommands handles:
//   - Daemon lifecycle (start, stop, status, logs)  
//   - Daemon configuration management (show, set, reset)
//   - Version verification and compatibility checking
//   - Configuration file management and persistence
//
// Design Pattern: Command Pattern with delegation from main App struct.
// Each system operation is encapsulated in methods that can be called independently
// while maintaining access to the parent App's context and configuration.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

// SystemCommands handles all system and daemon-related operations
type SystemCommands struct {
	app *App
}

// NewSystemCommands creates a new SystemCommands instance
func NewSystemCommands(app *App) *SystemCommands {
	return &SystemCommands{
		app: app,
	}
}

// Daemon handles daemon management commands
func (s *SystemCommands) Daemon(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws daemon <action>")
	}

	action := args[0]

	switch action {
	case "start":
		return s.daemonStart()
	case "stop":
		return s.daemonStop()
	case "status":
		return s.daemonStatus()
	case "logs":
		return s.daemonLogs()
	case "config":
		return s.daemonConfig(args[1:])
	default:
		return fmt.Errorf("unknown daemon action: %s\nAvailable actions: start, stop, status, logs, config", action)
	}
}

func (s *SystemCommands) daemonStart() error {
	// Check if daemon is already running
	if err := s.app.apiClient.Ping(s.app.ctx); err == nil {
		// Daemon is running, but check if it's the right version
		daemonVersion, err := s.getDaemonVersion()
		if err != nil {
			fmt.Printf("⚠️  Daemon is running but version check failed: %v\n", err)
			fmt.Println("🔄 Restarting daemon to ensure version compatibility...")
			if err := s.daemonStop(); err != nil {
				return fmt.Errorf("failed to stop outdated daemon: %w", err)
			}
			// Continue to start new daemon below
		} else if daemonVersion != version.Version {
			fmt.Printf("🔄 Daemon version mismatch (running: %s, CLI: %s)\n", daemonVersion, version.Version)
			fmt.Println("🔄 Restarting daemon with matching version...")
			if err := s.daemonStop(); err != nil {
				return fmt.Errorf("failed to stop outdated daemon: %w", err)
			}
			// Continue to start new daemon below
		} else {
			fmt.Println("✅ Daemon is already running (version match)")
			return nil
		}
	}

	fmt.Println("🚀 Starting CloudWorkstation daemon...")

	// Start daemon in the background
	cmd := exec.Command("cwsd")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Printf("✅ Daemon started (PID %d)\n", cmd.Process.Pid)
	fmt.Println("⏳ Waiting for daemon to initialize...")

	// Wait for daemon to be ready and verify version matches
	if err := s.waitForDaemonAndVerifyVersion(); err != nil {
		return fmt.Errorf("daemon startup verification failed: %w", err)
	}

	fmt.Println("✅ Daemon is ready and version verified")
	return nil
}

// getDaemonVersion retrieves the version from the running daemon
func (s *SystemCommands) getDaemonVersion() (string, error) {
	// Get daemon status which includes version information
	status, err := s.app.apiClient.GetStatus(s.app.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get daemon status: %w", err)
	}

	return status.Version, nil
}

// waitForDaemonAndVerifyVersion waits for daemon to be ready and verifies version matches
func (s *SystemCommands) waitForDaemonAndVerifyVersion() error {
	// Wait for daemon to be responsive (up to 10 seconds)
	maxAttempts := 20
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Try to ping the daemon
		if err := s.app.apiClient.Ping(s.app.ctx); err == nil {
			// Daemon is responsive, now verify version
			daemonVersion, err := s.getDaemonVersion()
			if err != nil {
				return fmt.Errorf("daemon is running but version check failed: %w", err)
			}

			if daemonVersion != version.Version {
				return fmt.Errorf("daemon version mismatch after restart (expected: %s, got: %s)", version.Version, daemonVersion)
			}

			// Success - daemon is running with correct version
			return nil
		}

		// Daemon not ready yet, wait and retry
		if attempt < maxAttempts {
			fmt.Printf("🔄 Daemon not ready yet, retrying in 0.5s (attempt %d/%d)\n", attempt, maxAttempts)
			time.Sleep(500 * time.Millisecond)
		}
	}

	return fmt.Errorf("daemon failed to start within 10 seconds")
}

func (s *SystemCommands) daemonStop() error {
	fmt.Println("⏹️ Stopping daemon...")

	// Try graceful shutdown via API
	if err := s.app.apiClient.Shutdown(s.app.ctx); err != nil {
		fmt.Println("❌ Failed to stop daemon via API:", err)
		fmt.Println("Find the daemon process and stop it manually:")
		fmt.Println("  ps aux | grep cwsd")
		fmt.Println("  kill <PID>")
		return err
	}

	fmt.Println("✅ Daemon stopped successfully")
	return nil
}

func (s *SystemCommands) daemonStatus() error {
	// Check if daemon is running
	if err := s.app.apiClient.Ping(s.app.ctx); err != nil {
		fmt.Println("❌ Daemon is not running")
		fmt.Println("Start with: cws daemon start")
		return nil
	}

	status, err := s.app.apiClient.GetStatus(s.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to get daemon status: %w", err)
	}

	fmt.Printf("✅ Daemon Status\n")
	fmt.Printf("   Version: %s\n", status.Version)
	fmt.Printf("   Status: %s\n", status.Status)
	fmt.Printf("   Start Time: %s\n", status.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   AWS Region: %s\n", status.AWSRegion)
	if status.AWSProfile != "" {
		fmt.Printf("   AWS Profile: %s\n", status.AWSProfile)
	}
	fmt.Printf("   Active Operations: %d\n", status.ActiveOps)
	fmt.Printf("   Total Requests: %d\n", status.TotalRequests)

	return nil
}

func (s *SystemCommands) daemonLogs() error {
	// TODO: Implement log viewing
	fmt.Println("📋 Daemon logs not implemented yet")
	fmt.Println("Check system logs manually for now")
	return nil
}

func (s *SystemCommands) daemonConfig(args []string) error {
	if len(args) == 0 {
		return s.daemonConfigShow()
	}

	switch args[0] {
	case "show":
		return s.daemonConfigShow()
	case "set":
		return s.daemonConfigSet(args[1:])
	case "reset":
		return s.daemonConfigReset()
	default:
		return fmt.Errorf("unknown daemon config command: %s\nAvailable commands: show, set, reset", args[0])
	}
}

// daemonConfigShow displays current daemon configuration
func (s *SystemCommands) daemonConfigShow() error {
	// Load configuration from daemon config file
	daemonConfig, err := s.loadDaemonConfig()
	if err != nil {
		return fmt.Errorf("failed to load daemon configuration: %w", err)
	}

	fmt.Printf("🔧 CloudWorkstation Daemon Configuration\n\n")
	fmt.Printf("Instance Retention:\n")
	if daemonConfig.InstanceRetentionMinutes == 0 {
		fmt.Printf("  • Retention Period: ♾️  Indefinite (until AWS removes instances)\n")
		fmt.Printf("  • Description: Terminated instances stay visible until AWS cleanup\n")
	} else {
		fmt.Printf("  • Retention Period: %d minutes\n", daemonConfig.InstanceRetentionMinutes)
		fmt.Printf("  • Description: Terminated instances cleaned up after %d minutes\n", daemonConfig.InstanceRetentionMinutes)
	}

	fmt.Printf("\nServer Settings:\n")
	fmt.Printf("  • Port: %s\n", daemonConfig.Port)

	fmt.Printf("\n💡 Configuration Commands:\n")
	fmt.Printf("  cws daemon config set retention <minutes>  # Set retention period (0=indefinite)\n")
	fmt.Printf("  cws daemon config reset                     # Reset to defaults (5 minutes)\n")

	return nil
}

// daemonConfigSet sets daemon configuration values
func (s *SystemCommands) daemonConfigSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws daemon config set <setting> <value>\nAvailable settings: retention")
	}

	setting := args[0]
	value := args[1]

	// Load current configuration
	daemonConfig, err := s.loadDaemonConfig()
	if err != nil {
		return fmt.Errorf("failed to load daemon configuration: %w", err)
	}

	switch setting {
	case "retention":
		var retentionMinutes int
		if value == "indefinite" || value == "infinite" || value == "0" {
			retentionMinutes = 0
		} else {
			_, err := fmt.Sscanf(value, "%d", &retentionMinutes)
			if err != nil || retentionMinutes < 0 {
				return fmt.Errorf("invalid retention value: %s\nUse: 0 (indefinite), or positive integer (minutes)", value)
			}
		}

		daemonConfig.InstanceRetentionMinutes = retentionMinutes

		// Save configuration
		if err := s.saveDaemonConfig(daemonConfig); err != nil {
			return fmt.Errorf("failed to save daemon configuration: %w", err)
		}

		if retentionMinutes == 0 {
			fmt.Printf("✅ Instance retention set to indefinite\n")
			fmt.Printf("   Terminated instances will remain visible until AWS cleanup\n")
		} else {
			fmt.Printf("✅ Instance retention set to %d minutes\n", retentionMinutes)
			fmt.Printf("   Terminated instances will be cleaned up after %d minutes\n", retentionMinutes)
		}

		fmt.Printf("\n⚠️  Changes take effect after daemon restart: cws daemon stop && cws daemon start\n")

	default:
		return fmt.Errorf("unknown setting: %s\nAvailable settings: retention", setting)
	}

	return nil
}

// daemonConfigReset resets daemon configuration to defaults
func (s *SystemCommands) daemonConfigReset() error {
	defaultConfig := s.getDefaultDaemonConfig()

	if err := s.saveDaemonConfig(defaultConfig); err != nil {
		return fmt.Errorf("failed to save daemon configuration: %w", err)
	}

	fmt.Printf("✅ Daemon configuration reset to defaults\n")
	fmt.Printf("   Instance retention: %d minutes\n", defaultConfig.InstanceRetentionMinutes)
	fmt.Printf("   Port: %s\n", defaultConfig.Port)

	fmt.Printf("\n⚠️  Changes take effect after daemon restart: cws daemon stop && cws daemon start\n")

	return nil
}

// Helper functions for daemon configuration
func (s *SystemCommands) loadDaemonConfig() (*DaemonConfig, error) {
	// Load daemon configuration using the same config system the daemon uses
	// We need to import the daemon package to use its config functions
	return s.loadDaemonConfigFromFile()
}

func (s *SystemCommands) saveDaemonConfig(config *DaemonConfig) error {
	return s.saveDaemonConfigToFile(config)
}

func (s *SystemCommands) getDefaultDaemonConfig() *DaemonConfig {
	return &DaemonConfig{
		InstanceRetentionMinutes: 5,
		Port:                     "8947",
	}
}

// DaemonConfig represents daemon configuration for CLI purposes
type DaemonConfig struct {
	InstanceRetentionMinutes int    `json:"instance_retention_minutes"`
	Port                     string `json:"port"`
}

// loadDaemonConfigFromFile loads daemon config from the standard location
func (s *SystemCommands) loadDaemonConfigFromFile() (*DaemonConfig, error) {
	configPath := s.getDaemonConfigPath()

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return s.getDefaultDaemonConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read daemon config: %w", err)
	}

	// Parse config
	config := s.getDefaultDaemonConfig() // Start with defaults
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse daemon config: %w", err)
	}

	return config, nil
}

// saveDaemonConfigToFile saves daemon config to the standard location
func (s *SystemCommands) saveDaemonConfigToFile(config *DaemonConfig) error {
	configPath := s.getDaemonConfigPath()

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write daemon config: %w", err)
	}

	return nil
}

// getDaemonConfigPath returns the standard daemon configuration file path
func (s *SystemCommands) getDaemonConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "daemon_config.json" // Fallback
	}
	return filepath.Join(homeDir, ".cloudworkstation", "daemon_config.json")
}