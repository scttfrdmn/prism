package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// PIDFileName is the name of the PID file for GUI
	GUIPIDFileName = "cws-gui.pid"

	// ShutdownTimeout is how long to wait for graceful shutdown
	GUIShutdownTimeout = 5 * time.Second
)

// GUISingletonManager manages GUI singleton enforcement
type GUISingletonManager struct {
	pidFile string
}

// NewGUISingletonManager creates a new GUI singleton manager
func NewGUISingletonManager() (*GUISingletonManager, error) {
	// Get Prism state directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".prism")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	pidFile := filepath.Join(stateDir, GUIPIDFileName)

	return &GUISingletonManager{
		pidFile: pidFile,
	}, nil
}

// Acquire attempts to acquire the singleton lock
// If another GUI is running, it will bring it to the foreground and return an error
func (s *GUISingletonManager) Acquire() error {
	// Check if PID file exists
	if _, err := os.Stat(s.pidFile); err == nil {
		// PID file exists, check if process is running
		oldPID, err := s.readPIDFile()
		switch {
		case err != nil:
			// PID file is corrupted, remove it
			_ = os.Remove(s.pidFile)
		case s.isProcessRunning(oldPID):
			// Another GUI is already running
			return fmt.Errorf("another Prism GUI is already running (PID: %d)\n\n"+
				"💡 Only one GUI can run at a time\n"+
				"   The other GUI has been brought to the foreground", oldPID)
		default:
			// Process not running, remove stale PID file
			_ = os.Remove(s.pidFile)
		}
	}

	// Write our PID
	pid := os.Getpid()
	if err := s.writePIDFile(pid); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// Release releases the singleton lock
func (s *GUISingletonManager) Release() error {
	// Remove PID file
	if err := os.Remove(s.pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// readPIDFile reads the PID from the PID file
func (s *GUISingletonManager) readPIDFile() (int, error) {
	data, err := os.ReadFile(s.pidFile)
	if err != nil {
		return 0, err
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %s", pidStr)
	}

	return pid, nil
}

// writePIDFile writes the PID to the PID file
func (s *GUISingletonManager) writePIDFile(pid int) error {
	data := []byte(fmt.Sprintf("%d\n", pid))
	return os.WriteFile(s.pidFile, data, 0644)
}

// isProcessRunning checks if a process with the given PID is running
func (s *GUISingletonManager) isProcessRunning(pid int) bool {
	// Use kill -0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 (null signal) to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}

	// Additional check: verify it's actually cws-gui
	return s.isCWSGUIProcess(pid)
}

// isCWSGUIProcess checks if the process is actually cws-gui
func (s *GUISingletonManager) isCWSGUIProcess(pid int) bool {
	// On macOS/Linux, read process name from ps
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	procName := strings.TrimSpace(string(output))
	return strings.Contains(procName, "cws-gui")
}

// GetPID returns the PID from the PID file, or 0 if not running
func (s *GUISingletonManager) GetPID() (int, error) {
	if _, err := os.Stat(s.pidFile); os.IsNotExist(err) {
		return 0, nil
	}

	pid, err := s.readPIDFile()
	if err != nil {
		return 0, err
	}

	// Verify process is still running
	if !s.isProcessRunning(pid) {
		return 0, nil
	}

	return pid, nil
}
