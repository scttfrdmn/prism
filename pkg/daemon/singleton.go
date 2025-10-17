// Package daemon provides singleton enforcement for the CloudWorkstation daemon.
//
// This ensures only one daemon process runs at a time and handles graceful
// shutdown of old processes when a new daemon starts.
package daemon

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
	// PIDFileName is the name of the PID file
	PIDFileName = "cwsd.pid"

	// ShutdownTimeout is how long to wait for graceful shutdown
	ShutdownTimeout = 10 * time.Second

	// ForceKillTimeout is how long to wait before force kill
	ForceKillTimeout = 15 * time.Second
)

// SingletonManager manages daemon singleton enforcement
type SingletonManager struct {
	pidFile string
}

// NewSingletonManager creates a new singleton manager
func NewSingletonManager() (*SingletonManager, error) {
	// Get CloudWorkstation state directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".cloudworkstation")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	pidFile := filepath.Join(stateDir, PIDFileName)

	return &SingletonManager{
		pidFile: pidFile,
	}, nil
}

// Acquire attempts to acquire the singleton lock
// If another daemon is running, it will attempt to gracefully shut it down
func (s *SingletonManager) Acquire() error {
	// Check if PID file exists
	if _, err := os.Stat(s.pidFile); err == nil {
		// PID file exists, check if process is running
		oldPID, err := s.readPIDFile()
		if err != nil {
			// PID file is corrupted, remove it
			_ = os.Remove(s.pidFile)
		} else if s.isProcessRunning(oldPID) {
			// Old daemon is running, attempt graceful shutdown
			if err := s.shutdownOldDaemon(oldPID); err != nil {
				return fmt.Errorf("failed to shutdown old daemon (PID %d): %w\n\n"+
					"💡 Try manually stopping the old daemon:\n"+
					"   kill %d\n"+
					"   # Or force kill if needed:\n"+
					"   kill -9 %d",
					oldPID, err, oldPID, oldPID)
			}
		} else {
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
func (s *SingletonManager) Release() error {
	// Remove PID file
	if err := os.Remove(s.pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// readPIDFile reads the PID from the PID file
func (s *SingletonManager) readPIDFile() (int, error) {
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
func (s *SingletonManager) writePIDFile(pid int) error {
	data := []byte(fmt.Sprintf("%d\n", pid))
	return os.WriteFile(s.pidFile, data, 0644)
}

// isProcessRunning checks if a process with the given PID is running
func (s *SingletonManager) isProcessRunning(pid int) bool {
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

	// Additional check: verify it's actually cwsd
	// This prevents false positives if PID is reused
	return s.isCWSDProcess(pid)
}

// isCWSDProcess checks if the process is actually cwsd
func (s *SingletonManager) isCWSDProcess(pid int) bool {
	// On macOS/Linux, read process name from /proc or ps
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	procName := strings.TrimSpace(string(output))
	return strings.Contains(procName, "cwsd")
}

// shutdownOldDaemon attempts to gracefully shutdown the old daemon
func (s *SingletonManager) shutdownOldDaemon(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Try graceful shutdown with SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for graceful shutdown
	deadline := time.Now().Add(ShutdownTimeout)
	for time.Now().Before(deadline) {
		if !s.isProcessRunning(pid) {
			// Successfully shut down
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Graceful shutdown timed out, try SIGINT
	if err := process.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("failed to send SIGINT: %w", err)
	}

	// Wait again
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !s.isProcessRunning(pid) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Still running, try force kill
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to force kill (SIGKILL): %w", err)
	}

	// Final wait
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !s.isProcessRunning(pid) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("process did not terminate after force kill")
}

// GetPID returns the PID from the PID file, or 0 if not running
func (s *SingletonManager) GetPID() (int, error) {
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
