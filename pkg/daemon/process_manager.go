// Package daemon provides process management for CloudWorkstation daemon lifecycle.
//
// This module implements comprehensive daemon process detection, management,
// and cleanup functionality for CloudWorkstation uninstallation scenarios.
//
// Key Features:
//   - Cross-platform daemon process detection
//   - PID file management with locking
//   - Graceful shutdown with fallback to force kill
//   - Registry-based daemon tracking
//   - Comprehensive cleanup for uninstallation
//
// Usage:
//   manager := daemon.NewProcessManager()
//   processes, err := manager.FindDaemonProcesses()
//   err = manager.GracefulShutdown(pid)
//   err = manager.CleanupProcesses()

package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ProcessInfo represents information about a daemon process
type ProcessInfo struct {
	PID        int       `json:"pid"`
	Command    string    `json:"command"`
	StartTime  time.Time `json:"start_time"`
	ConfigPath string    `json:"config_path,omitempty"`
	LogPath    string    `json:"log_path,omitempty"`
	WorkingDir string    `json:"working_dir,omitempty"`
	User       string    `json:"user,omitempty"`
	Status     string    `json:"status"`
}

// DaemonRegistry represents the daemon registry for tracking instances
type DaemonRegistry struct {
	Processes   []ProcessInfo `json:"processes"`
	LastUpdated time.Time     `json:"last_updated"`
	Version     string        `json:"version"`
}

// ProcessManager interface defines daemon process management operations
type ProcessManager interface {
	FindDaemonProcesses() ([]ProcessInfo, error)
	GracefulShutdown(pid int) error
	ForceKill(pid int) error
	CleanupProcesses() error
	RegisterDaemon(pid int, configPath, logPath string) error
	UnregisterDaemon(pid int) error
	GetPIDFilePath() string
	GetRegistryPath() string
	IsProcessRunning(pid int) bool
	WaitForShutdown(pid int, timeout time.Duration) error
}

// DefaultProcessManager implements ProcessManager interface
type DefaultProcessManager struct {
	configDir    string
	pidFile      string
	registryFile string
	lockFile     string
}

// NewProcessManager creates a new process manager instance
func NewProcessManager() ProcessManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Cannot determine home directory, using current directory")
		homeDir = "."
	}

	configDir := filepath.Join(homeDir, ".cloudworkstation")

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("Warning: Cannot create config directory: %v", err)
	}

	return &DefaultProcessManager{
		configDir:    configDir,
		pidFile:      filepath.Join(configDir, "daemon.pid"),
		registryFile: filepath.Join(configDir, "daemon_registry.json"),
		lockFile:     filepath.Join(configDir, "daemon_process.lock"),
	}
}

// FindDaemonProcesses discovers all running CloudWorkstation daemon processes
func (pm *DefaultProcessManager) FindDaemonProcesses() ([]ProcessInfo, error) {
	var processes []ProcessInfo

	// Method 1: Check PID file
	if pidInfo, err := pm.findProcessFromPIDFile(); err == nil {
		processes = append(processes, pidInfo)
	}

	// Method 2: Check registry
	if registryProcesses, err := pm.findProcessesFromRegistry(); err == nil {
		processes = append(processes, registryProcesses...)
	}

	// Method 3: System-wide process scan
	systemProcesses, err := pm.findProcessesSystemWide()
	if err != nil {
		log.Printf("Warning: System-wide process scan failed: %v", err)
	} else {
		processes = append(processes, systemProcesses...)
	}

	// Deduplicate processes by PID
	processMap := make(map[int]ProcessInfo)
	for _, proc := range processes {
		if pm.IsProcessRunning(proc.PID) {
			processMap[proc.PID] = proc
		}
	}

	// Convert back to slice
	var result []ProcessInfo
	for _, proc := range processMap {
		result = append(result, proc)
	}

	return result, nil
}

// findProcessFromPIDFile reads daemon PID from PID file
func (pm *DefaultProcessManager) findProcessFromPIDFile() (ProcessInfo, error) {
	var info ProcessInfo

	if _, err := os.Stat(pm.pidFile); os.IsNotExist(err) {
		return info, fmt.Errorf("PID file does not exist")
	}

	data, err := os.ReadFile(pm.pidFile)
	if err != nil {
		return info, fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return info, fmt.Errorf("invalid PID in file: %s", pidStr)
	}

	if !pm.IsProcessRunning(pid) {
		return info, fmt.Errorf("process %d not running", pid)
	}

	info = ProcessInfo{
		PID:        pid,
		Command:    pm.getProcessCommand(pid),
		StartTime:  pm.getProcessStartTime(pid),
		ConfigPath: pm.configDir,
		Status:     "running",
	}

	return info, nil
}

// findProcessesFromRegistry reads daemon processes from registry
func (pm *DefaultProcessManager) findProcessesFromRegistry() ([]ProcessInfo, error) {
	if _, err := os.Stat(pm.registryFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("registry file does not exist")
	}

	data, err := os.ReadFile(pm.registryFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var registry DaemonRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	var activeProcesses []ProcessInfo
	for _, proc := range registry.Processes {
		if pm.IsProcessRunning(proc.PID) {
			proc.Status = "running"
			activeProcesses = append(activeProcesses, proc)
		}
	}

	return activeProcesses, nil
}

// findProcessesSystemWide performs system-wide process scan for daemon processes
func (pm *DefaultProcessManager) findProcessesSystemWide() ([]ProcessInfo, error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		return pm.findProcessesUnix()
	case "windows":
		return pm.findProcessesWindows()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// findProcessesUnix finds processes on Unix-like systems
func (pm *DefaultProcessManager) findProcessesUnix() ([]ProcessInfo, error) {
	var processes []ProcessInfo

	// Use pgrep to find cwsd processes
	cmd := exec.Command("pgrep", "-f", "cwsd")
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 when no processes found, which is normal
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return processes, nil
		}
		return nil, fmt.Errorf("pgrep failed: %w", err)
	}

	pidStrings := strings.Fields(strings.TrimSpace(string(output)))
	for _, pidStr := range pidStrings {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		if pm.IsProcessRunning(pid) {
			info := ProcessInfo{
				PID:       pid,
				Command:   pm.getProcessCommand(pid),
				StartTime: pm.getProcessStartTime(pid),
				Status:    "running",
			}
			processes = append(processes, info)
		}
	}

	return processes, nil
}

// findProcessesWindows finds processes on Windows systems
func (pm *DefaultProcessManager) findProcessesWindows() ([]ProcessInfo, error) {
	var processes []ProcessInfo

	// Use tasklist to find cwsd processes
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq cwsd*", "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tasklist failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines[1:] { // Skip header
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			continue
		}

		pidStr := strings.Trim(fields[1], "\"")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		if pm.IsProcessRunning(pid) {
			info := ProcessInfo{
				PID:       pid,
				Command:   pm.getProcessCommand(pid),
				StartTime: pm.getProcessStartTime(pid),
				Status:    "running",
			}
			processes = append(processes, info)
		}
	}

	return processes, nil
}

// GracefulShutdown attempts graceful shutdown of a daemon process
func (pm *DefaultProcessManager) GracefulShutdown(pid int) error {
	if !pm.IsProcessRunning(pid) {
		return fmt.Errorf("process %d is not running", pid)
	}

	log.Printf("Attempting graceful shutdown of daemon PID %d", pid)

	// Send SIGTERM for graceful shutdown
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to process %d: %w", pid, err)
	}

	// Wait for process to terminate gracefully
	timeout := 30 * time.Second
	if err := pm.WaitForShutdown(pid, timeout); err != nil {
		log.Printf("Graceful shutdown timeout for PID %d, will force kill", pid)
		return pm.ForceKill(pid)
	}

	log.Printf("Process %d shut down gracefully", pid)
	return nil
}

// ForceKill forcefully terminates a daemon process
func (pm *DefaultProcessManager) ForceKill(pid int) error {
	if !pm.IsProcessRunning(pid) {
		return nil // Already stopped
	}

	log.Printf("Force killing daemon PID %d", pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process %d: %w", pid, err)
	}

	// Wait a moment for process to die
	time.Sleep(2 * time.Second)

	if pm.IsProcessRunning(pid) {
		return fmt.Errorf("process %d still running after kill", pid)
	}

	log.Printf("Process %d force killed successfully", pid)
	return nil
}

// CleanupProcesses performs comprehensive cleanup of all daemon processes
func (pm *DefaultProcessManager) CleanupProcesses() error {
	log.Printf("Starting comprehensive daemon process cleanup")

	processes, err := pm.FindDaemonProcesses()
	if err != nil {
		return fmt.Errorf("failed to find daemon processes: %w", err)
	}

	if len(processes) == 0 {
		log.Printf("No daemon processes found to cleanup")
		pm.cleanupFiles()
		return nil
	}

	log.Printf("Found %d daemon processes to cleanup", len(processes))

	// Attempt graceful shutdown of all processes
	var failedProcesses []int
	for _, proc := range processes {
		log.Printf("Cleaning up daemon PID %d", proc.PID)
		if err := pm.GracefulShutdown(proc.PID); err != nil {
			log.Printf("Failed to gracefully shutdown PID %d: %v", proc.PID, err)
			failedProcesses = append(failedProcesses, proc.PID)
		}
	}

	// Force kill any remaining processes
	if len(failedProcesses) > 0 {
		log.Printf("Force killing %d remaining processes", len(failedProcesses))
		for _, pid := range failedProcesses {
			if err := pm.ForceKill(pid); err != nil {
				log.Printf("Failed to force kill PID %d: %v", pid, err)
			}
		}
	}

	// Clean up files
	pm.cleanupFiles()

	log.Printf("Daemon process cleanup completed")
	return nil
}

// cleanupFiles removes daemon-related files
func (pm *DefaultProcessManager) cleanupFiles() {
	files := []string{pm.pidFile, pm.registryFile, pm.lockFile}

	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to remove %s: %v", file, err)
		} else {
			log.Printf("Cleaned up file: %s", file)
		}
	}
}

// RegisterDaemon registers a daemon process in the registry
func (pm *DefaultProcessManager) RegisterDaemon(pid int, configPath, logPath string) error {
	registry, err := pm.loadRegistry()
	if err != nil {
		registry = &DaemonRegistry{
			Processes: []ProcessInfo{},
			Version:   "1.0",
		}
	}

	// Check if process already registered
	for i, proc := range registry.Processes {
		if proc.PID == pid {
			// Update existing entry
			registry.Processes[i] = ProcessInfo{
				PID:        pid,
				Command:    pm.getProcessCommand(pid),
				StartTime:  time.Now(),
				ConfigPath: configPath,
				LogPath:    logPath,
				WorkingDir: pm.getProcessWorkingDir(pid),
				User:       pm.getProcessUser(pid),
				Status:     "running",
			}
			registry.LastUpdated = time.Now()
			return pm.saveRegistry(registry)
		}
	}

	// Add new entry
	newProcess := ProcessInfo{
		PID:        pid,
		Command:    pm.getProcessCommand(pid),
		StartTime:  time.Now(),
		ConfigPath: configPath,
		LogPath:    logPath,
		WorkingDir: pm.getProcessWorkingDir(pid),
		User:       pm.getProcessUser(pid),
		Status:     "running",
	}

	registry.Processes = append(registry.Processes, newProcess)
	registry.LastUpdated = time.Now()

	// Also write PID file
	if err := pm.writePIDFile(pid); err != nil {
		log.Printf("Warning: Failed to write PID file: %v", err)
	}

	return pm.saveRegistry(registry)
}

// UnregisterDaemon removes a daemon process from the registry
func (pm *DefaultProcessManager) UnregisterDaemon(pid int) error {
	registry, err := pm.loadRegistry()
	if err != nil {
		return nil // No registry to update
	}

	// Remove process from registry
	var updatedProcesses []ProcessInfo
	for _, proc := range registry.Processes {
		if proc.PID != pid {
			updatedProcesses = append(updatedProcesses, proc)
		}
	}

	registry.Processes = updatedProcesses
	registry.LastUpdated = time.Now()

	return pm.saveRegistry(registry)
}

// loadRegistry loads the daemon registry from disk
func (pm *DefaultProcessManager) loadRegistry() (*DaemonRegistry, error) {
	if _, err := os.Stat(pm.registryFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("registry file does not exist")
	}

	data, err := os.ReadFile(pm.registryFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var registry DaemonRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return &registry, nil
}

// saveRegistry saves the daemon registry to disk
func (pm *DefaultProcessManager) saveRegistry(registry *DaemonRegistry) error {
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(pm.registryFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// writePIDFile writes the daemon PID to the PID file
func (pm *DefaultProcessManager) writePIDFile(pid int) error {
	pidStr := fmt.Sprintf("%d\n", pid)
	return os.WriteFile(pm.pidFile, []byte(pidStr), 0644)
}

// GetPIDFilePath returns the path to the PID file
func (pm *DefaultProcessManager) GetPIDFilePath() string {
	return pm.pidFile
}

// GetRegistryPath returns the path to the registry file
func (pm *DefaultProcessManager) GetRegistryPath() string {
	return pm.registryFile
}

// IsProcessRunning checks if a process is currently running
func (pm *DefaultProcessManager) IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, Signal(0) can be used to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// WaitForShutdown waits for a process to shut down within timeout
func (pm *DefaultProcessManager) WaitForShutdown(pid int, timeout time.Duration) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			if !pm.IsProcessRunning(pid) {
				return nil
			}
		case <-timeoutChan:
			return fmt.Errorf("timeout waiting for process %d to shutdown", pid)
		}
	}
}

// Helper methods for process information

// getProcessCommand gets the command line for a process
func (pm *DefaultProcessManager) getProcessCommand(pid int) string {
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=")
		output, err := cmd.Output()
		if err != nil {
			return "unknown"
		}
		return strings.TrimSpace(string(output))
	case "windows":
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV")
		output, err := cmd.Output()
		if err != nil {
			return "unknown"
		}
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Split(lines[1], ",")
			if len(fields) > 0 {
				return strings.Trim(fields[0], "\"")
			}
		}
	}
	return "unknown"
}

// getProcessStartTime gets the start time for a process
func (pm *DefaultProcessManager) getProcessStartTime(pid int) time.Time {
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart=")
		output, err := cmd.Output()
		if err != nil {
			return time.Now()
		}
		startStr := strings.TrimSpace(string(output))
		// Parse various time formats (this is a simplified version)
		if t, err := time.Parse("Mon Jan 2 15:04:05 2006", startStr); err == nil {
			return t
		}
	}
	return time.Now()
}

// getProcessWorkingDir gets the working directory for a process
func (pm *DefaultProcessManager) getProcessWorkingDir(pid int) string {
	switch runtime.GOOS {
	case "linux":
		linkPath := fmt.Sprintf("/proc/%d/cwd", pid)
		if wd, err := os.Readlink(linkPath); err == nil {
			return wd
		}
	case "darwin":
		cmd := exec.Command("lsof", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "n") {
					return line[1:]
				}
			}
		}
	}
	return ""
}

// getProcessUser gets the user running the process
func (pm *DefaultProcessManager) getProcessUser(pid int) string {
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "user=")
		output, err := cmd.Output()
		if err != nil {
			return "unknown"
		}
		return strings.TrimSpace(string(output))
	}
	return "unknown"
}
