package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

// checkDaemonHealth checks if the daemon is responding to health checks
func checkDaemonHealth() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get("http://localhost:8947/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// findDaemonBinary locates the cwsd daemon binary
func findDaemonBinary() (string, error) {
	// Get the directory where cws-gui is located
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)

	// Try several locations in order of preference
	locations := []string{
		filepath.Join(exeDir, "cwsd"),       // Same directory as GUI (production)
		filepath.Join(exeDir, "..", "cwsd"), // Parent directory
		"./bin/cwsd",                        // Development environment
		"cwsd",                              // In PATH
	}

	// Add platform-specific extension
	if runtime.GOOS == "windows" {
		for i, loc := range locations {
			locations[i] = loc + ".exe"
		}
	}

	// Check each location
	for _, loc := range locations {
		absPath, err := filepath.Abs(loc)
		if err != nil {
			continue
		}

		if _, err := os.Stat(absPath); err == nil {
			// Found it!
			return absPath, nil
		}
	}

	return "", fmt.Errorf("daemon binary (cwsd) not found in expected locations")
}

// startDaemon attempts to start the daemon if it's not running
func startDaemon() error {
	log.Println("🔍 Checking if daemon is running...")

	// Check if daemon is already running
	if checkDaemonHealth() {
		log.Println("✅ Daemon is already running")
		return nil
	}

	log.Println("⚠️  Daemon is not running, attempting to start...")

	// Find the daemon binary
	daemonPath, err := findDaemonBinary()
	if err != nil {
		return fmt.Errorf("cannot start daemon: %w", err)
	}

	log.Printf("📍 Found daemon at: %s", daemonPath)

	// Start the daemon process
	cmd := exec.Command(daemonPath)

	// Redirect output to devnull so daemon runs independently
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Set process group so daemon isn't killed when GUI exits
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}

	// Start daemon in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Detach from daemon process so it continues after GUI exits
	if err := cmd.Process.Release(); err != nil {
		log.Printf("⚠️  Warning: could not release daemon process: %v", err)
	}

	log.Println("⏳ Waiting for daemon to initialize...")

	// Wait for daemon to become ready (up to 10 seconds)
	maxAttempts := 20
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(500 * time.Millisecond)

		if checkDaemonHealth() {
			log.Println("✅ Daemon started successfully!")
			return nil
		}

		if i < maxAttempts-1 {
			log.Printf("🔄 Waiting for daemon to be ready (attempt %d/%d)...", i+1, maxAttempts)
		}
	}

	return fmt.Errorf("daemon started but did not become healthy within 10 seconds")
}

func main() {
	// Parse command line flags
	var (
		minimizeToTray  = flag.Bool("minimize", false, "Start minimized to system tray")
		autoStart       = flag.Bool("autostart", false, "Configure to start automatically at login")
		removeAutoStart = flag.Bool("remove-autostart", false, "Remove automatic startup configuration")
		help            = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	// Handle special flags
	if *help {
		showHelp()
		return
	}

	if *autoStart {
		if err := configureAutoStart(true); err != nil {
			log.Printf("Failed to configure auto-start: %v", err)
			os.Exit(1)
		}
		log.Println("✅ Auto-start configured successfully")
		return
	}

	if *removeAutoStart {
		if err := configureAutoStart(false); err != nil {
			log.Printf("Failed to remove auto-start: %v", err)
			os.Exit(1)
		}
		log.Println("✅ Auto-start removed successfully")
		return
	}

	// Enforce singleton: only one GUI can run at a time
	singleton, err := NewGUISingletonManager()
	if err != nil {
		log.Printf("❌ Failed to create singleton manager: %v", err)
		os.Exit(1)
	}

	if err := singleton.Acquire(); err != nil {
		log.Printf("❌ %v", err)
		os.Exit(0) // Exit gracefully - another GUI is already running
	}
	defer singleton.Release()

	log.Printf("✅ GUI singleton lock acquired (PID: %d)", os.Getpid())

	// Ensure daemon is running before starting GUI
	if err := startDaemon(); err != nil {
		log.Printf("❌ Failed to start daemon: %v", err)
		log.Println("Please start the daemon manually with: cws daemon start")
		// Continue anyway - GUI will show connection error to user
	}

	// Create CloudWorkstation service
	cwsService := NewCloudWorkstationService()

	// Reload API key after daemon is running (daemon may have generated a new key)
	cwsService.ReloadAPIKey()

	// Start WebSocket server for terminal connections (port 8948)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/terminal", cwsService.HandleTerminalWebSocket)
		mux.HandleFunc("/api-key", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			fmt.Fprintf(w, `{"api_key":"%s"}`, cwsService.apiKey)
		})

		log.Println("🔌 Starting WebSocket server on :8948")
		if err := http.ListenAndServe(":8948", mux); err != nil {
			log.Printf("❌ WebSocket server error: %v", err)
		}
	}()

	// Create CloudWorkstation GUI application
	app := application.New(application.Options{
		Name:        "CloudWorkstation",
		Description: "Academic Research Computing Platform - Professional GUI",
		Services: []application.Service{
			application.NewService(cwsService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false, // Keep running in menu bar
		},
	})

	// Create main window with professional styling
	_ = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "CloudWorkstation",
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarDefault,
		},
		BackgroundColour: application.NewRGB(248, 250, 252), // Clean light background
		URL:              "/",
		Width:            1200,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
	})

	// Handle minimize to tray option
	if *minimizeToTray {
		// Hide window on startup (system tray functionality would go here)
		log.Println("⚠️  System tray functionality not yet implemented")
	}

	// Run the application
	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// showHelp displays command line help
func showHelp() {
	log.Printf(`CloudWorkstation GUI v0.5.1

Usage: cws-gui [OPTIONS]

OPTIONS:
  -autostart          Configure to start automatically at login
  -remove-autostart   Remove automatic startup configuration  
  -minimize          Start minimized to system tray (planned)
  -help              Show this help

STARTUP CONFIGURATION:
  # Enable auto-start at login
  cws-gui -autostart

  # Remove auto-start configuration
  cws-gui -remove-autostart

  # Start minimized (when system tray is implemented)
  cws-gui -minimize

EXAMPLES:
  cws-gui                    # Start normally
  cws-gui -autostart        # Configure auto-start
  cws-gui -remove-autostart # Remove auto-start
`)
}
