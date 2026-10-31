//go:build windows

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	serviceName = "PrismDaemon"
	displayName = "Prism Daemon"
	description = "Enterprise research management platform daemon for launching cloud research environments"
)

// PrismService implements the Windows service interface
type PrismService struct {
	elog   debug.Log // the Windows Event Log, or the console if it can't be opened
	ctx    context.Context
	cancel context.CancelFunc
}

// Execute implements the main service execution loop
func (ps *PrismService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	// Initialize event log. Its source is registered by "prismd-service install" but not by
	// the MSI's ServiceInstall, so register it if needed, and fall back to the console rather
	// than exit before reporting Running (which failed the MSI install with error 1920, #757).
	elog := openServiceLog()
	defer func() { _ = elog.Close() }() // nothing to do if closing the log fails
	ps.elog = elog

	elog.Info(1, fmt.Sprintf("%s service starting", displayName))

	// Set up context for graceful shutdown
	ps.ctx, ps.cancel = context.WithCancel(context.Background())

	// Find and start the daemon process
	daemonCmd, err := ps.startDaemon()
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to start daemon: %v", err))
		return
	}

	elog.Info(1, fmt.Sprintf("%s daemon started with PID %d", displayName, daemonCmd.Process.Pid))
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// Service main loop
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				elog.Info(1, fmt.Sprintf("%s service stopping", displayName))
				changes <- svc.Status{State: svc.StopPending}

				// Cancel context to signal graceful shutdown
				ps.cancel()

				// Give daemon time to shutdown gracefully
				shutdownComplete := make(chan error, 1)
				go func() {
					shutdownComplete <- daemonCmd.Wait()
				}()

				select {
				case <-shutdownComplete:
					elog.Info(1, fmt.Sprintf("%s daemon stopped gracefully", displayName))
				case <-time.After(30 * time.Second):
					elog.Warning(1, fmt.Sprintf("%s daemon did not stop within 30 seconds, force killing", displayName))
					if daemonCmd.Process != nil {
						daemonCmd.Process.Kill()
					}
				}

				return

			case svc.Pause:
				elog.Info(1, fmt.Sprintf("%s service paused", displayName))
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}

			case svc.Continue:
				elog.Info(1, fmt.Sprintf("%s service resumed", displayName))
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

			default:
				elog.Error(1, fmt.Sprintf("Unexpected service command: %d", c.Cmd))
			}

		case <-ps.ctx.Done():
			// Context cancelled, shutting down
			return

		default:
			// Health check - verify daemon is still running
			if daemonCmd.ProcessState != nil && daemonCmd.ProcessState.Exited() {
				elog.Error(1, fmt.Sprintf("%s daemon process exited unexpectedly with code %d",
					displayName, daemonCmd.ProcessState.ExitCode()))
				changes <- svc.Status{State: svc.Stopped}
				return
			}
			time.Sleep(5 * time.Second) // Check every 5 seconds
		}
	}
}

// openServiceLog opens the service's Event Log source, registering it first if it doesn't exist
// yet, and falls back to a console logger if the Event Log can't be used.
func openServiceLog() debug.Log {
	if elog, err := eventlog.Open(serviceName); err == nil {
		return elog
	}
	if err := eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		log.Printf("Failed to register event log source: %v", err)
	}
	if elog, err := eventlog.Open(serviceName); err == nil {
		return elog
	}
	log.Printf("Event log unavailable; logging to the console")
	return debug.New(serviceName)
}

// startDaemon locates and starts the prismd daemon process
func (ps *PrismService) startDaemon() (*exec.Cmd, error) {
	// Get current executable directory
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	binDir := filepath.Dir(execPath)
	daemonPath := filepath.Join(binDir, "prismd.exe")

	// Verify daemon executable exists
	if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("daemon executable not found at %s", daemonPath)
	}

	// Create daemon command
	// No "--service" argument: prismd defines no such flag (flag.Parse exited on it); the
	// environment below (PRISM_SERVICE_MODE) tells it it's running under the service.
	cmd := exec.CommandContext(ps.ctx, daemonPath)
	cmd.Dir = binDir

	// Set up service environment variables
	cmd.Env = append(os.Environ(),
		"PRISM_SERVICE_MODE=true",
		fmt.Sprintf("PRISM_LOG_PATH=%s", getLogPath()),
		fmt.Sprintf("PRISM_CONFIG_PATH=%s", getConfigPath()),
		"PRISM_DAEMON_PORT=8947",
	)

	// Start daemon process
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Wait a moment to ensure daemon started successfully
	time.Sleep(2 * time.Second)

	// Verify process is still running
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return nil, fmt.Errorf("daemon process exited immediately with code %d",
			cmd.ProcessState.ExitCode())
	}

	return cmd, nil
}

// getConfigPath returns the Windows configuration directory path
func getConfigPath() string {
	appData := os.Getenv("PROGRAMDATA")
	if appData == "" {
		appData = "C:\\ProgramData"
	}
	return filepath.Join(appData, "Prism")
}

// getLogPath returns the Windows log directory path
func getLogPath() string {
	return filepath.Join(getConfigPath(), "Logs")
}

// runService is the main service entry point
func runService() error {
	// Create necessary directories
	configPath := getConfigPath()
	logPath := getLogPath()

	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.MkdirAll(logPath, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Determine if running under the Service Control Manager. The deprecated
	// IsAnInteractiveSession can misreport on current Windows, which ran the service
	// through debug.Run, so it never reported Running to the SCM (#757).
	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("failed to determine session type: %w", err)
	}

	var run svc.Handler = &PrismService{}

	if !isService {
		// Running in debug/console mode
		log.Println("Prism Service running in console mode (for debugging)")
		return debug.Run(serviceName, run)
	}

	// Running as Windows service
	return svc.Run(serviceName, run)
}

// installService installs the Windows service
func installService() error {
	exepath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer func() { _ = m.Disconnect() }()

	// Check if service already exists
	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}

	// Create the service
	s, err = m.CreateService(serviceName, exepath, mgr.Config{
		DisplayName:      displayName,
		Description:      description,
		StartType:        mgr.StartAutomatic,
		ServiceStartName: "",
		Dependencies:     []string{"Tcpip", "Dhcp"},
	})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Set up event log
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("failed to setup event log: %w", err)
	}

	// Configure service recovery options
	recoveryActions := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
	}

	if err := s.SetRecoveryActions(recoveryActions, uint32((24 * time.Hour).Seconds())); err != nil {
		// Log warning but don't fail installation
		log.Printf("Warning: Failed to set service recovery options: %v", err)
	}

	fmt.Printf("Prism service installed successfully\n")
	fmt.Printf("Service will start automatically on system boot\n")
	fmt.Printf("Start the service with: net start %s\n", serviceName)
	return nil
}

// removeService removes the Windows service
func removeService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer func() { _ = m.Disconnect() }()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", serviceName)
	}
	defer s.Close()

	// Stop service if running
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %w", err)
	}

	if status.State != svc.Stopped {
		fmt.Println("Stopping service...")
		if _, err := s.Control(svc.Stop); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}

		// Wait for service to stop
		timeout := time.Now().Add(30 * time.Second)
		for {
			status, err := s.Query()
			if err != nil {
				return fmt.Errorf("failed to query service status: %w", err)
			}
			if status.State == svc.Stopped {
				break
			}
			if time.Now().After(timeout) {
				return fmt.Errorf("timeout waiting for service to stop")
			}
			time.Sleep(300 * time.Millisecond)
		}
	}

	// Delete service
	err = s.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Remove event log
	err = eventlog.Remove(serviceName)
	if err != nil {
		// Log warning but don't fail removal
		log.Printf("Warning: Failed to remove event log: %v", err)
	}

	fmt.Printf("Prism service removed successfully\n")
	return nil
}

// startService starts the Windows service
func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer func() { _ = m.Disconnect() }()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", serviceName, err)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Printf("Prism service started successfully\n")
	return nil
}

// stopService stops the Windows service
func stopService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer func() { _ = m.Disconnect() }()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait for service to stop
	timeout := time.Now().Add(30 * time.Second)
	for status.State != svc.Stopped {
		if time.Now().After(timeout) {
			return fmt.Errorf("timeout waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("failed to query service status: %w", err)
		}
	}

	fmt.Printf("Prism service stopped successfully\n")
	return nil
}

// serviceStatus shows the current service status
func serviceStatus() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer func() { _ = m.Disconnect() }()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", serviceName)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %w", err)
	}

	config, err := s.Config()
	if err != nil {
		log.Printf("Warning: Failed to get service config: %v", err)
	}

	var stateStr string
	switch status.State {
	case svc.Stopped:
		stateStr = "Stopped"
	case svc.StartPending:
		stateStr = "Start Pending"
	case svc.StopPending:
		stateStr = "Stop Pending"
	case svc.Running:
		stateStr = "Running"
	case svc.ContinuePending:
		stateStr = "Continue Pending"
	case svc.PausePending:
		stateStr = "Pause Pending"
	case svc.Paused:
		stateStr = "Paused"
	default:
		stateStr = "Unknown"
	}

	fmt.Printf("Prism Service Status:\n")
	fmt.Printf("  Service Name: %s\n", serviceName)
	fmt.Printf("  Display Name: %s\n", displayName)
	fmt.Printf("  State: %s\n", stateStr)

	if config.StartType == mgr.StartAutomatic {
		fmt.Printf("  Start Type: Automatic\n")
	} else {
		fmt.Printf("  Start Type: %d\n", config.StartType)
	}

	fmt.Printf("  Process ID: %d\n", status.ProcessId)
	fmt.Printf("  Config Path: %s\n", getConfigPath())
	fmt.Printf("  Log Path: %s\n", getLogPath())

	return nil
}

// showUsage displays command usage information
func showUsage() {
	fmt.Printf("Prism Windows Service Wrapper\n\n")
	fmt.Printf("USAGE:\n")
	fmt.Printf("    %s [command]\n\n", os.Args[0])
	fmt.Printf("COMMANDS:\n")
	fmt.Printf("    install    Install the Windows service\n")
	fmt.Printf("    remove     Remove the Windows service\n")
	fmt.Printf("    start      Start the service\n")
	fmt.Printf("    stop       Stop the service\n")
	fmt.Printf("    restart    Restart the service\n")
	fmt.Printf("    status     Show service status\n")
	fmt.Printf("    debug      Run in console mode for debugging\n")
	fmt.Printf("    help       Show this help message\n\n")
	fmt.Printf("NOTES:\n")
	fmt.Printf("    - Service commands require administrator privileges\n")
	fmt.Printf("    - Running without arguments starts the service\n")
	fmt.Printf("    - Use 'debug' to run in console mode for troubleshooting\n")
}

func main() {
	// No arguments, or "--service" (how the MSI's ServiceInstall starts it): run as the
	// service. "--service" used to fall through to an unknown-command exit (#757).
	if len(os.Args) < 2 || os.Args[1] == "--service" {
		if err := runService(); err != nil {
			log.Fatalf("Service run failed: %v", err)
		}
		return
	}

	command := os.Args[1]
	if err := executeCommand(command); err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

func executeCommand(command string) error {
	commandHandlers := map[string]func() error{
		"install":   installService,
		"remove":    removeService,
		"uninstall": removeService,
		"start":     startService,
		"stop":      stopService,
		"restart":   handleRestart,
		"status":    serviceStatus,
		"debug":     handleDebugMode,
	}

	if handler, exists := commandHandlers[command]; exists {
		return handler()
	}

	switch command {
	case "help", "--help", "-h":
		showUsage()
		return nil
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		showUsage()
		os.Exit(1)
		return nil
	}
}

func handleRestart() error {
	fmt.Println("Stopping service...")
	if err := stopService(); err != nil {
		log.Printf("Warning: Failed to stop service: %v", err)
	}

	time.Sleep(2 * time.Second)

	fmt.Println("Starting service...")
	return startService()
}

func handleDebugMode() error {
	fmt.Println("Running Prism service in console mode...")
	fmt.Println("Press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	service := &PrismService{}
	service.ctx, service.cancel = context.WithCancel(context.Background())

	go func() {
		<-sigCh
		fmt.Println("\nShutdown signal received, stopping...")
		service.cancel()
	}()

	return debug.Run(serviceName, service)
}
