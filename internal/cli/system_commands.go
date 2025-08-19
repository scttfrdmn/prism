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
	"strconv"
	"strings"
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
		return NewUsageError("cws daemon <action>", "cws daemon start")
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
	case "processes":
		return s.daemonProcesses()
	case "cleanup":
		return s.daemonCleanup(args[1:])
	default:
		return NewValidationError("daemon action", action, "start, stop, status, logs, config, processes, cleanup")
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
				return WrapAPIError("stop outdated daemon", err)
			}
			// Continue to start new daemon below
		} else if daemonVersion != version.Version {
			fmt.Printf("🔄 Daemon version mismatch (running: %s, CLI: %s)\n", daemonVersion, version.Version)
			fmt.Println("🔄 Restarting daemon with matching version...")
			if err := s.daemonStop(); err != nil {
				return WrapAPIError("stop outdated daemon", err)
			}
			// Continue to start new daemon below
		} else {
			fmt.Println("✅ Daemon is already running (version match)")
			return nil
		}
	}

	// Message already printed by auto-start caller

	// Start daemon in the background
	cmd := exec.Command("cwsd")
	if err := cmd.Start(); err != nil {
		return WrapAPIError("start daemon process", err)
	}

	fmt.Printf("✅ Daemon started (PID %d)\n", cmd.Process.Pid)
	fmt.Println("⏳ Waiting for daemon to initialize...")

	// Wait for daemon to be ready and verify version matches
	if err := s.waitForDaemonAndVerifyVersion(); err != nil {
		return WrapAPIError("verify daemon startup", err)
	}

	fmt.Println("✅ Daemon is ready and version verified")
	return nil
}

// getDaemonVersion retrieves the version from the running daemon
func (s *SystemCommands) getDaemonVersion() (string, error) {
	// Get daemon status which includes version information
	status, err := s.app.apiClient.GetStatus(s.app.ctx)
	if err != nil {
		return "", WrapAPIError("get daemon status", err)
	}

	return status.Version, nil
}

// waitForDaemonAndVerifyVersion waits for daemon to be ready and verifies version matches
func (s *SystemCommands) waitForDaemonAndVerifyVersion() error {
	// Wait for daemon to be responsive (up to 10 seconds)
	maxAttempts := DaemonStartupMaxAttempts
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
			time.Sleep(DaemonStartupRetryInterval)
		}
	}

	return NewStateError("daemon", "startup", "timeout", "running within timeout")
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
	fmt.Printf("   Start Time: %s\n", status.StartTime.Format(StandardDateFormat))
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
		InstanceRetentionMinutes: DefaultInstanceRetentionMinutes,
		Port:                     DefaultDaemonPort,
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
	if err := os.MkdirAll(configDir, DefaultDirPermissions); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, DefaultFilePermissions); err != nil {
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
	return filepath.Join(homeDir, DefaultConfigDir, DefaultConfigFile)
}

// daemonProcesses lists all daemon processes
func (s *SystemCommands) daemonProcesses() error {
	fmt.Println("🔍 Scanning for CloudWorkstation daemon processes...")

	// Make API call to get daemon processes
	response, err := s.app.apiClient.MakeRequest("GET", "/api/v1/daemon/processes", nil)
	if err != nil {
		// Fallback to direct process detection if daemon is not responding
		fmt.Println("⚠️  Daemon API not responding, performing direct process scan...")
		return s.directProcessScan()
	}

	var processResponse struct {
		Processes []struct {
			PID        int    `json:"pid"`
			Command    string `json:"command"`
			StartTime  string `json:"start_time"`
			Status     string `json:"status"`
			ConfigPath string `json:"config_path,omitempty"`
			LogPath    string `json:"log_path,omitempty"`
		} `json:"processes"`
		Total  int    `json:"total"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(response, &processResponse); err != nil {
		return fmt.Errorf("failed to parse processes response: %w", err)
	}

	if len(processResponse.Processes) == 0 {
		fmt.Println("✅ No daemon processes found")
		return nil
	}

	fmt.Printf("📋 Found %d daemon process(es):\n\n", len(processResponse.Processes))

	for _, proc := range processResponse.Processes {
		fmt.Printf("🔧 Process %d\n", proc.PID)
		fmt.Printf("   Command: %s\n", proc.Command)
		fmt.Printf("   Status: %s\n", proc.Status)
		fmt.Printf("   Start Time: %s\n", proc.StartTime)
		if proc.ConfigPath != "" {
			fmt.Printf("   Config: %s\n", proc.ConfigPath)
		}
		if proc.LogPath != "" {
			fmt.Printf("   Log: %s\n", proc.LogPath)
		}
		fmt.Println()
	}

	fmt.Printf("💡 Management Commands:\n")
	fmt.Printf("  cws daemon stop      # Graceful shutdown\n")
	fmt.Printf("  cws daemon cleanup   # Force cleanup all processes\n")

	return nil
}

// directProcessScan performs direct process scanning when daemon API is unavailable
func (s *SystemCommands) directProcessScan() error {
	// Use system commands to find processes
	cmd := exec.Command("pgrep", "-f", "cwsd")
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 when no processes found
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			fmt.Println("✅ No daemon processes found")
			return nil
		}
		return fmt.Errorf("failed to scan for processes: %w", err)
	}

	pidStrings := strings.Fields(strings.TrimSpace(string(output)))
	if len(pidStrings) == 0 {
		fmt.Println("✅ No daemon processes found")
		return nil
	}

	fmt.Printf("📋 Found %d daemon process(es):\n\n", len(pidStrings))

	for _, pidStr := range pidStrings {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// Get process details
		cmd := exec.Command("ps", "-p", pidStr, "-o", "pid,ppid,command")
		if psOutput, err := cmd.Output(); err == nil {
			lines := strings.Split(string(psOutput), "\n")
			if len(lines) > 1 {
				fmt.Printf("🔧 Process %d\n", pid)
				fmt.Printf("   Details: %s\n", strings.TrimSpace(lines[1]))
				fmt.Println()
			}
		}
	}

	fmt.Printf("💡 Management Commands:\n")
	fmt.Printf("  cws daemon stop      # Graceful shutdown\n")
	fmt.Printf("  cws daemon cleanup   # Force cleanup all processes\n")
	fmt.Printf("  kill -TERM <pid>     # Manual graceful termination\n")
	fmt.Printf("  kill -KILL <pid>     # Manual force termination\n")

	return nil
}

// daemonCleanup performs comprehensive daemon cleanup
func (s *SystemCommands) daemonCleanup(args []string) error {
	var forceKill bool
	var confirmed bool

	// Parse arguments
	for _, arg := range args {
		switch arg {
		case "--force":
			forceKill = true
		case "--yes", "-y":
			confirmed = true
		case "--help", "-h":
			fmt.Printf("Usage: cws daemon cleanup [OPTIONS]\n\n")
			fmt.Printf("Options:\n")
			fmt.Printf("  --force    Force kill processes instead of graceful shutdown\n")
			fmt.Printf("  --yes, -y  Skip confirmation prompts\n")
			fmt.Printf("  --help, -h Show this help message\n\n")
			fmt.Printf("Description:\n")
			fmt.Printf("  Performs comprehensive cleanup of all CloudWorkstation daemon processes\n")
			fmt.Printf("  and related files. This is useful for troubleshooting or uninstallation.\n\n")
			fmt.Printf("Examples:\n")
			fmt.Printf("  cws daemon cleanup           # Interactive cleanup with confirmations\n")
			fmt.Printf("  cws daemon cleanup --yes     # Non-interactive cleanup\n")
			fmt.Printf("  cws daemon cleanup --force   # Force kill all processes\n")
			return nil
		default:
			return fmt.Errorf("unknown cleanup option: %s\nUse --help for usage information", arg)
		}
	}

	fmt.Println("🧹 CloudWorkstation Daemon Cleanup")
	fmt.Println("====================================")

	// Check for running processes first
	cmd := exec.Command("pgrep", "-f", "cwsd")
	output, err := cmd.Output()
	processCount := 0
	if err == nil {
		pids := strings.Fields(strings.TrimSpace(string(output)))
		processCount = len(pids)
	}

	if processCount == 0 {
		fmt.Println("✅ No daemon processes found to cleanup")
		fmt.Println("🧹 Cleaning up daemon files...")
		s.cleanupDaemonFiles()
		fmt.Println("✅ Cleanup completed")
		return nil
	}

	fmt.Printf("⚠️  Found %d daemon process(es) to cleanup\n", processCount)

	if forceKill {
		fmt.Println("🔨 Force kill mode enabled")
	}

	// Confirmation prompt
	if !confirmed {
		fmt.Println("\nThis will:")
		fmt.Printf("  • Stop all CloudWorkstation daemon processes (%d found)\n", processCount)
		fmt.Println("  • Clean up daemon configuration files")
		fmt.Println("  • Remove process ID files and locks")
		fmt.Print("\nContinue with cleanup? [y/N]: ")

		var response string
		_, _ = fmt.Scanln(&response) // Error ignored - user input validation happens below
		if response != "y" && response != "Y" && response != "yes" {
			fmt.Println("❌ Cleanup cancelled")
			return nil
		}
	}

	// Attempt API-based cleanup first
	fmt.Println("\n🔗 Attempting API-based cleanup...")
	apiSuccess := s.performAPICleanup(forceKill)

	if !apiSuccess {
		fmt.Println("⚠️  API cleanup failed, performing direct cleanup...")
		if err := s.performDirectCleanup(forceKill); err != nil {
			return fmt.Errorf("direct cleanup failed: %w", err)
		}
	}

	// Clean up files
	fmt.Println("🧹 Cleaning up daemon files...")
	s.cleanupDaemonFiles()

	// Verify cleanup
	fmt.Println("🔍 Verifying cleanup...")
	cmd = exec.Command("pgrep", "-f", "cwsd")
	if _, err := cmd.Output(); err != nil {
		// No processes found (pgrep returns exit code 1)
		fmt.Println("✅ All daemon processes cleaned up successfully")
	} else {
		fmt.Println("⚠️  Some processes may still be running")
		fmt.Println("💡 You may need to manually kill remaining processes:")
		fmt.Println("   ps aux | grep cwsd")
		fmt.Println("   kill <PID>")
	}

	fmt.Println("✅ Daemon cleanup completed")
	return nil
}

// performAPICleanup attempts cleanup via daemon API
func (s *SystemCommands) performAPICleanup(forceKill bool) bool {
	cleanupRequest := map[string]interface{}{
		"force_kill": forceKill,
		"remove_all": true,
	}

	response, err := s.app.apiClient.MakeRequest("POST", "/api/v1/daemon/cleanup", cleanupRequest)
	if err != nil {
		fmt.Printf("⚠️  API cleanup request failed: %v\n", err)
		return false
	}

	var cleanupResponse struct {
		ProcessesFound   int      `json:"processes_found"`
		ProcessesCleaned int      `json:"processes_cleaned"`
		ProcessesFailed  int      `json:"processes_failed"`
		FailedProcesses  []int    `json:"failed_processes,omitempty"`
		FilesRemoved     []string `json:"files_removed,omitempty"`
		Status           string   `json:"status"`
		Message          string   `json:"message"`
	}

	if err := json.Unmarshal(response, &cleanupResponse); err != nil {
		fmt.Printf("⚠️  Failed to parse cleanup response: %v\n", err)
		return false
	}

	fmt.Printf("✅ API cleanup completed:\n")
	fmt.Printf("   Processes found: %d\n", cleanupResponse.ProcessesFound)
	fmt.Printf("   Processes cleaned: %d\n", cleanupResponse.ProcessesCleaned)
	if cleanupResponse.ProcessesFailed > 0 {
		fmt.Printf("   Processes failed: %d\n", cleanupResponse.ProcessesFailed)
	}
	if len(cleanupResponse.FilesRemoved) > 0 {
		fmt.Printf("   Files removed: %d\n", len(cleanupResponse.FilesRemoved))
	}

	return cleanupResponse.Status == "success"
}

// performDirectCleanup performs direct process cleanup when API is unavailable
func (s *SystemCommands) performDirectCleanup(forceKill bool) error {
	// Find all cwsd processes
	cmd := exec.Command("pgrep", "-f", "cwsd")
	output, err := cmd.Output()
	if err != nil {
		// No processes found
		return nil
	}

	pidStrings := strings.Fields(strings.TrimSpace(string(output)))
	if len(pidStrings) == 0 {
		return nil
	}

	fmt.Printf("🔧 Found %d process(es) to terminate\n", len(pidStrings))

	for _, pidStr := range pidStrings {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		fmt.Printf("   Stopping PID %d...", pid)

		var signal string
		if forceKill {
			signal = "KILL"
		} else {
			signal = "TERM"
		}

		cmd := exec.Command("kill", "-"+signal, pidStr)
		if err := cmd.Run(); err != nil {
			fmt.Printf(" failed (%v)\n", err)
		} else {
			fmt.Printf(" done\n")
		}
	}

	if !forceKill {
		// Wait for graceful shutdown
		fmt.Println("⏳ Waiting for processes to stop...")
		time.Sleep(5 * time.Second)

		// Check if any processes remain and force kill them
		cmd = exec.Command("pgrep", "-f", "cwsd")
		if output, err := cmd.Output(); err == nil {
			remainingPids := strings.Fields(strings.TrimSpace(string(output)))
			if len(remainingPids) > 0 {
				fmt.Printf("🔨 Force killing %d remaining process(es)\n", len(remainingPids))
				for _, pidStr := range remainingPids {
					cmd = exec.Command("kill", "-KILL", pidStr)
					_ = cmd.Run() // Error ignored - force kill may fail for already dead processes
				}
			}
		}
	}

	return nil
}

// cleanupDaemonFiles removes daemon-related files
func (s *SystemCommands) cleanupDaemonFiles() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("⚠️  Cannot determine home directory: %v\n", err)
		return
	}

	configDir := filepath.Join(homeDir, ".cloudworkstation")
	filesToRemove := []string{
		filepath.Join(configDir, "daemon.pid"),
		filepath.Join(configDir, "daemon_registry.json"),
		filepath.Join(configDir, "daemon_process.lock"),
	}

	removedCount := 0
	for _, file := range filesToRemove {
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				fmt.Printf("⚠️  Failed to remove %s: %v\n", file, err)
			} else {
				fmt.Printf("   Removed: %s\n", filepath.Base(file))
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		fmt.Printf("✅ Cleaned up %d daemon file(s)\n", removedCount)
	} else {
		fmt.Println("   No daemon files found to remove")
	}
}
