package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Operating system constants
const (
	osDarwin  = "darwin"
	osLinux   = "linux"
	osWindows = "windows"
)

// configureAutoStart configures or removes automatic startup at login
//
//nolint:unused // Platform-specific function called conditionally
func configureAutoStart(enable bool) error {
	switch runtime.GOOS {
	case osDarwin:
		return configureMacOSAutoStart(enable)
	case osLinux:
		return configureLinuxAutoStart(enable)
	case osWindows:
		return configureWindowsAutoStart(enable)
	default:
		return fmt.Errorf("auto-start configuration not supported on %s", runtime.GOOS)
	}
}

// configureMacOSAutoStart configures macOS Login Items
//
//nolint:unused // Platform-specific function
func configureMacOSAutoStart(enable bool) error {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve any symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	appName := "Prism GUI"

	if enable {
		// Add to Login Items using osascript
		script := fmt.Sprintf(`
		tell application "System Events"
			make login item at end with properties {path:"%s", hidden:false}
		end tell
		`, execPath)

		cmd := exec.Command("osascript", "-e", script) //nolint:gosec // Generated AppleScript for login item management
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add login item: %w", err)
		}

		fmt.Printf("✅ Added '%s' to macOS Login Items\n", appName)
		fmt.Printf("   Path: %s\n", execPath)
		fmt.Println("   You can also manage this in System Preferences > Users & Groups > Login Items")
	} else {
		// Remove from Login Items
		script := fmt.Sprintf(`
		tell application "System Events"
			delete login item "%s"
		end tell
		`, appName)

		cmd := exec.Command("osascript", "-e", script) //nolint:gosec // Generated AppleScript for login item removal
		if err := cmd.Run(); err != nil {
			// Try alternative removal by path
			script = fmt.Sprintf(`
			tell application "System Events"
				delete (login items whose path is "%s")
			end tell
			`, execPath)
			cmd = exec.Command("osascript", "-e", script) //nolint:gosec // Alternative AppleScript for login item removal
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to remove login item: %w", err)
			}
		}

		fmt.Printf("✅ Removed '%s' from macOS Login Items\n", appName)
	}

	return nil
}

// configureLinuxAutoStart configures XDG autostart
//
//nolint:unused // Platform-specific function
func configureLinuxAutoStart(enable bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	autostartDir := filepath.Join(homeDir, ".config", "autostart")
	desktopFile := filepath.Join(autostartDir, "cloudworkstation-gui.desktop")

	if enable {
		// Create autostart directory if it doesn't exist
		if err := os.MkdirAll(autostartDir, 0750); err != nil {
			return fmt.Errorf("failed to create autostart directory: %w", err)
		}

		// Get the path to the current executable
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		// Resolve any symlinks
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return fmt.Errorf("failed to resolve executable path: %w", err)
		}

		// Create desktop entry
		desktopEntry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=Prism GUI
Comment=Academic Research Computing Platform - Professional GUI
Exec=%s -minimize
Icon=cloudworkstation
Terminal=false
Hidden=false
Categories=Development;Science;Education;
StartupNotify=true
X-GNOME-Autostart-enabled=true
`, execPath)

		if err := os.WriteFile(desktopFile, []byte(desktopEntry), 0600); err != nil {
			return fmt.Errorf("failed to create desktop file: %w", err)
		}

		fmt.Printf("✅ Created autostart entry: %s\n", desktopFile)
		fmt.Printf("   Path: %s\n", execPath)
	} else {
		// Remove autostart file
		if err := os.Remove(desktopFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove desktop file: %w", err)
		}

		fmt.Printf("✅ Removed autostart entry: %s\n", desktopFile)
	}

	return nil
}

// configureWindowsAutoStart configures Windows startup
//
//nolint:unused // Platform-specific function
func configureWindowsAutoStart(enable bool) error {
	keyPath := `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`
	appName := "PrismGUI"

	if enable {
		// Get the path to the current executable
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		// Add to Windows registry using reg command
		cmd := exec.Command("reg", "add", keyPath, "/v", appName, "/d", fmt.Sprintf("\"%s\" -minimize", execPath), "/f") //nolint:gosec // Registry modification with validated executable path
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to add registry entry: %w\nOutput: %s", err, string(output))
		}

		fmt.Printf("✅ Added '%s' to Windows startup\n", appName)
		fmt.Printf("   Path: %s\n", execPath)
		fmt.Println("   You can also manage this in Task Manager > Startup tab")
	} else {
		// Remove from Windows registry
		cmd := exec.Command("reg", "delete", keyPath, "/v", appName, "/f")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Check if the error is because the key doesn't exist
			if strings.Contains(string(output), "unable to find") {
				fmt.Printf("✅ '%s' was not in Windows startup (already removed)\n", appName)
				return nil
			}
			return fmt.Errorf("failed to remove registry entry: %w\nOutput: %s", err, string(output))
		}

		fmt.Printf("✅ Removed '%s' from Windows startup\n", appName)
	}

	return nil
}
