// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	serviceName = "CloudWorkstationDaemon"
	displayName = "CloudWorkstation Daemon"
	description = "Enterprise research management platform daemon for launching cloud research environments"
)

type cloudWorkstationService struct{}

func (m *cloudWorkstationService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("%s service starting", displayName))

	// Find cwsd binary
	executableDir, err := os.Executable()
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to get executable path: %v", err))
		return
	}
	daemonPath := filepath.Join(filepath.Dir(executableDir), "cwsd.exe")

	// Start the daemon process
	cmd := exec.Command(daemonPath, "--service")
	cmd.Dir = filepath.Dir(executableDir)
	
	// Set up environment
	cmd.Env = append(os.Environ(),
		"CWS_SERVICE_MODE=true",
		fmt.Sprintf("CWS_LOG_PATH=%s", getLogPath()),
		fmt.Sprintf("CWS_CONFIG_PATH=%s", getConfigPath()),
	)

	err = cmd.Start()
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to start daemon: %v", err))
		return
	}

	elog.Info(1, fmt.Sprintf("%s daemon started with PID %d", displayName, cmd.Process.Pid))
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
				
				// Terminate daemon process
				if cmd.Process != nil {
					err := cmd.Process.Kill()
					if err != nil {
						elog.Warning(1, fmt.Sprintf("Failed to kill daemon process: %v", err))
					}
				}
				
				// Wait for process to exit
				done := make(chan error, 1)
				go func() {
					done <- cmd.Wait()
				}()
				
				select {
				case <-done:
					elog.Info(1, fmt.Sprintf("%s daemon stopped", displayName))
				case <-time.After(30 * time.Second):
					elog.Warning(1, fmt.Sprintf("%s daemon did not stop within 30 seconds", displayName))
				}
				
				return
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				elog.Error(1, fmt.Sprintf("Unexpected service command: %d", c.Cmd))
			}
		default:
			// Check if daemon process is still running
			if cmd.Process != nil {
				// Non-blocking check using FindProcess
				process, err := os.FindProcess(cmd.Process.Pid)
				if err != nil || process == nil {
					elog.Error(1, fmt.Sprintf("%s daemon process died unexpectedly", displayName))
					changes <- svc.Status{State: svc.Stopped}
					return
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func getConfigPath() string {
	appData := os.Getenv("PROGRAMDATA")
	if appData == "" {
		appData = "C:\\ProgramData"
	}
	return filepath.Join(appData, "CloudWorkstation")
}

func getLogPath() string {
	appData := os.Getenv("PROGRAMDATA")
	if appData == "" {
		appData = "C:\\ProgramData"
	}
	return filepath.Join(appData, "CloudWorkstation", "Logs")
}

func runService() {
	// Create necessary directories
	configPath := getConfigPath()
	logPath := getLogPath()
	
	os.MkdirAll(configPath, 0755)
	os.MkdirAll(logPath, 0755)

	run := svc.Run
	if debug.Enabled() {
		run = debug.Run
	}
	
	err := run(serviceName, &cloudWorkstationService{})
	if err != nil {
		log.Fatalf("Service run failed: %v", err)
	}
}

func installService() error {
	exepath, err := os.Executable()
	if err != nil {
		return err
	}
	
	m, err := svc.NewService(serviceName, svc.ServiceConfig{
		DisplayName:      displayName,
		Description:      description,
		StartType:        svc.StartAutomatic,
		ServiceStartName: "",
		BinaryPathName:   exepath,
		Dependencies:     []string{"Tcpip", "Dhcp"},
	})
	if err != nil {
		return err
	}
	defer m.Close()
	
	err = m.Install()
	if err != nil {
		return err
	}
	
	// Set up event log
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		return err
	}
	
	fmt.Printf("CloudWorkstation service installed successfully\n")
	fmt.Printf("Service will start automatically on system boot\n")
	return nil
}

func removeService() error {
	m, err := svc.NewService(serviceName, svc.ServiceConfig{})
	if err != nil {
		return err
	}
	defer m.Close()
	
	err = m.Remove()
	if err != nil {
		return err
	}
	
	err = eventlog.Remove(serviceName)
	if err != nil {
		return err
	}
	
	fmt.Printf("CloudWorkstation service removed successfully\n")
	return nil
}

func startService() error {
	m, err := svc.NewService(serviceName, svc.ServiceConfig{})
	if err != nil {
		return err
	}
	defer m.Close()
	
	err = m.Start()
	if err != nil {
		return err
	}
	
	fmt.Printf("CloudWorkstation service started successfully\n")
	return nil
}

func stopService() error {
	m, err := svc.NewService(serviceName, svc.ServiceConfig{})
	if err != nil {
		return err
	}
	defer m.Close()
	
	status, err := m.Control(svc.Stop)
	if err != nil {
		return err
	}
	
	timeout := time.Now().Add(30 * time.Second)
	for status.State != svc.Stopped {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		status, err = m.Query()
		if err != nil {
			return err
		}
	}
	
	fmt.Printf("CloudWorkstation service stopped successfully\n")
	return nil
}

func serviceStatus() error {
	m, err := svc.NewService(serviceName, svc.ServiceConfig{})
	if err != nil {
		return err
	}
	defer m.Close()
	
	status, err := m.Query()
	if err != nil {
		return err
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
	
	fmt.Printf("CloudWorkstation Service Status:\n")
	fmt.Printf("  Service Name: %s\n", serviceName)
	fmt.Printf("  Display Name: %s\n", displayName)
	fmt.Printf("  State: %s\n", stateStr)
	fmt.Printf("  Process ID: %d\n", status.ProcessId)
	fmt.Printf("  Config Path: %s\n", getConfigPath())
	fmt.Printf("  Log Path: %s\n", getLogPath())
	
	return nil
}

func main() {
	if len(os.Args) < 2 {
		runService()
		return
	}

	switch os.Args[1] {
	case "install":
		err := installService()
		if err != nil {
			log.Fatalf("Failed to install service: %v", err)
		}
	case "remove":
		err := removeService()
		if err != nil {
			log.Fatalf("Failed to remove service: %v", err)
		}
	case "start":
		err := startService()
		if err != nil {
			log.Fatalf("Failed to start service: %v", err)
		}
	case "stop":
		err := stopService()
		if err != nil {
			log.Fatalf("Failed to stop service: %v", err)
		}
	case "status":
		err := serviceStatus()
		if err != nil {
			log.Fatalf("Failed to get service status: %v", err)
		}
	default:
		fmt.Printf("Usage: %s {install|remove|start|stop|status}\n", os.Args[0])
		os.Exit(1)
	}
}