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

//go:embed assets/icon.png
var appIcon []byte

// checkDaemonHealth checks if the daemon is responding to health checks
func checkDaemonHealth() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get("http://localhost:8947/api/v1/health")
	if err != nil {
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	return resp.StatusCode == http.StatusOK
}

// findDaemonBinary locates the prismd daemon binary
func findDaemonBinary() (string, error) {
	// Get the directory where prism-gui is located
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)

	// Try several locations in order of preference
	locations := []string{
		filepath.Join(exeDir, "prismd"),       // Same directory as GUI (production)
		filepath.Join(exeDir, "..", "prismd"), // Parent directory
		"./bin/prismd",                        // Development environment
		"prismd",                              // In PATH
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

	return "", fmt.Errorf("daemon binary (prismd) not found in expected locations")
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
	defer func() {
		if err := singleton.Release(); err != nil {
			log.Printf("Warning: failed to release singleton lock: %v", err)
		}
	}()

	log.Printf("✅ GUI singleton lock acquired (PID: %d)", os.Getpid())

	// Ensure daemon is running before starting GUI
	if err := startDaemon(); err != nil {
		log.Printf("❌ Failed to start daemon: %v", err)
		log.Println("Please start the daemon manually with: prism admin daemon start")
		// Continue anyway - GUI will show connection error to user
	}

	// Create Prism service
	cwsService := NewPrismService()

	// Reload API key after daemon is running (daemon may have generated a new key)
	cwsService.ReloadAPIKey()

	// Start WebSocket server for terminal connections (port 8948)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/terminal", cwsService.HandleTerminalWebSocket)
		mux.HandleFunc("/api-key", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			if _, err := fmt.Fprintf(w, `{"api_key":"%s"}`, cwsService.apiKey); err != nil {
				log.Printf("Warning: failed to write API key response: %v", err)
			}
		})

		log.Println("🔌 Starting WebSocket server on 127.0.0.1:8948")
		// Bind loopback only: this is a local IPC bridge between the desktop GUI
		// and its own process, never exposed off-host. TLS is unwarranted for a
		// localhost socket.
		// nosemgrep: go.lang.security.audit.net.use-tls.use-tls -- localhost-only desktop IPC bridge; not network-exposed.
		if err := http.ListenAndServe("127.0.0.1:8948", mux); err != nil {
			log.Printf("❌ WebSocket server error: %v", err)
		}
	}()

	// Create Prism GUI application
	app := application.New(application.Options{
		Name:        "Prism",
		Description: "Academic Research Computing Platform - Professional GUI",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(cwsService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false, // Keep running in menu bar/system tray
		},
	})

	// Create main window with professional styling
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Prism",
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

	// Setup system tray
	trayManager := NewSystemTrayManager(app, window, cwsService)
	if err := trayManager.Setup(); err != nil {
		log.Printf("⚠️  Failed to setup system tray: %v", err)
		log.Println("Continuing without system tray functionality...")
	}

	// Handle minimize to tray option
	if *minimizeToTray {
		// Start minimized to tray
		window.Hide()
		log.Println("✅ Started minimized to system tray")
	}

	// Run the application
	appErr := app.Run()

	// Cleanup runs (deferred singleton.Release() executes here)

	if appErr != nil {
		log.Printf("❌ Application error: %v", appErr)
		os.Exit(1)
	}
}

// showHelp displays command line help
func showHelp() {
	log.Printf(`Prism GUI v0.5.1

Usage: prism-gui [OPTIONS]

OPTIONS:
  -autostart          Configure to start automatically at login
  -remove-autostart   Remove automatic startup configuration  
  -minimize          Start minimized to system tray (planned)
  -help              Show this help

STARTUP CONFIGURATION:
  # Enable auto-start at login
  prism-gui -autostart

  # Remove auto-start configuration
  prism-gui -remove-autostart

  # Start minimized (when system tray is implemented)
  prism-gui -minimize

EXAMPLES:
  prism-gui                    # Start normally
  prism-gui -autostart        # Configure auto-start
  prism-gui -remove-autostart # Remove auto-start
`)
}
