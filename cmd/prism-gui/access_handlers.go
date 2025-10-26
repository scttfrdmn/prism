// Package main provides GUI application for Prism with embedded access handlers
package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
)

// AccessType represents the type of instance access
type AccessType string

const (
	AccessTypeDesktop  AccessType = "desktop"  // RDP/VNC
	AccessTypeWeb      AccessType = "web"      // Jupyter, RStudio, etc.
	AccessTypeTerminal AccessType = "terminal" // SSH

	// Default username fallback
	defaultUsername = "ubuntu"
)

// InstanceAccess represents access methods for an instance
type InstanceAccess struct {
	InstanceID   string       `json:"instance_id"`
	InstanceName string       `json:"instance_name"`
	PublicIP     string       `json:"public_ip"`
	AccessTypes  []AccessType `json:"access_types"`
	WebURL       string       `json:"web_url,omitempty"`
	WebPort      int          `json:"web_port,omitempty"`
	RDPPort      int          `json:"rdp_port,omitempty"`
	VNCPort      int          `json:"vnc_port,omitempty"`
	SSHPort      int          `json:"ssh_port,omitempty"`
	Username     string       `json:"username"`
}

// OpenRemoteDesktop opens a remote desktop connection to an instance
func (s *PrismService) OpenRemoteDesktop(ctx context.Context, instanceName string) error {
	access, err := s.GetInstanceAccess(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to get instance access: %w", err)
	}

	if access.RDPPort > 0 {
		return s.openRDP(access)
	} else if access.VNCPort > 0 {
		return s.openVNC(access)
	}

	return fmt.Errorf("no remote desktop available for instance %s", instanceName)
}

// openRDP opens an RDP connection
func (s *PrismService) openRDP(access *InstanceAccess) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case osDarwin:
		// macOS: Use Microsoft Remote Desktop or open RDP URL
		rdpURL := fmt.Sprintf("rdp://full%%20address=s:%s:%d&username=s:%s",
			access.PublicIP, access.RDPPort, access.Username)
		cmd = exec.Command("open", rdpURL) //nolint:gosec // Validated RDP URL for desktop access
	case osWindows:
		// Windows: Use mstsc
		cmd = exec.Command("mstsc", "/v:"+fmt.Sprintf("%s:%d", access.PublicIP, access.RDPPort)) //nolint:gosec // Validated instance access
	case osLinux:
		// Linux: Use remmina or xfreerdp
		cmd = exec.Command("xfreerdp", "/v:"+access.PublicIP, "/port:"+fmt.Sprintf("%d", access.RDPPort), //nolint:gosec // Validated instance access
			"/u:"+access.Username, "/dynamic-resolution", "/clipboard")
	default:
		return fmt.Errorf("RDP not supported on %s", runtime.GOOS)
	}

	return cmd.Start()
}

// openVNC opens a VNC connection
func (s *PrismService) openVNC(access *InstanceAccess) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case osDarwin:
		// macOS: Use built-in VNC viewer
		vncURL := fmt.Sprintf("vnc://%s:%d", access.PublicIP, access.VNCPort)
		cmd = exec.Command("open", vncURL) //nolint:gosec // Validated VNC URL for desktop access
	case osWindows:
		// Windows: Use RealVNC or TightVNC if installed
		cmd = exec.Command("vncviewer", fmt.Sprintf("%s:%d", access.PublicIP, access.VNCPort)) //nolint:gosec // Validated instance access
	case osLinux:
		// Linux: Use vncviewer
		cmd = exec.Command("vncviewer", fmt.Sprintf("%s:%d", access.PublicIP, access.VNCPort)) //nolint:gosec // Validated instance access
	default:
		return fmt.Errorf("VNC not supported on %s", runtime.GOOS)
	}

	return cmd.Start()
}

// OpenWebInterface opens the web interface for an instance in an embedded browser
func (s *PrismService) OpenWebInterface(ctx context.Context, instanceName string) (string, error) {
	access, err := s.GetInstanceAccess(ctx, instanceName)
	if err != nil {
		return "", fmt.Errorf("failed to get instance access: %w", err)
	}

	if access.WebPort == 0 {
		return "", fmt.Errorf("no web interface available for instance %s", instanceName)
	}

	// Return the URL for the embedded WebView to navigate to
	// The GUI will use a proxied connection through the daemon for security
	proxyURL := fmt.Sprintf("%s/proxy/%s/", s.daemonURL, instanceName)
	return proxyURL, nil
}

// OpenTerminal opens a terminal connection to an instance
func (s *PrismService) OpenTerminal(ctx context.Context, instanceName string) error {
	access, err := s.GetInstanceAccess(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to get instance access: %w", err)
	}

	if access.SSHPort == 0 {
		return fmt.Errorf("no SSH access available for instance %s", instanceName)
	}

	sshCommand := fmt.Sprintf("ssh %s@%s -p %d", access.Username, access.PublicIP, access.SSHPort)
	cmd, err := s.createTerminalCommand(sshCommand)
	if err != nil {
		return err
	}

	return cmd.Start()
}

// createTerminalCommand creates the appropriate terminal command based on the OS
func (s *PrismService) createTerminalCommand(sshCommand string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case osDarwin:
		return s.createMacOSTerminalCommand(sshCommand), nil
	case osWindows:
		return s.createWindowsTerminalCommand(sshCommand), nil
	case osLinux:
		return s.createLinuxTerminalCommand(sshCommand)
	default:
		return nil, fmt.Errorf("terminal not supported on %s", runtime.GOOS)
	}
}

// createMacOSTerminalCommand creates a macOS Terminal.app command
func (s *PrismService) createMacOSTerminalCommand(sshCommand string) *exec.Cmd {
	script := fmt.Sprintf(`tell application "Terminal"
		do script "%s"
		activate
	end tell`, sshCommand)
	return exec.Command("osascript", "-e", script) //nolint:gosec // Generated AppleScript for terminal access
}

// createWindowsTerminalCommand creates a Windows terminal command
func (s *PrismService) createWindowsTerminalCommand(sshCommand string) *exec.Cmd {
	return exec.Command("cmd", "/c", "start", "cmd", "/k", sshCommand) //nolint:gosec // Validated SSH command for terminal access
}

// createLinuxTerminalCommand creates a Linux terminal command with automatic emulator detection
func (s *PrismService) createLinuxTerminalCommand(sshCommand string) (*exec.Cmd, error) {
	terminals := []string{"gnome-terminal", "konsole", "xterm", "xfce4-terminal"}

	for _, term := range terminals {
		if _, err := exec.LookPath(term); err == nil {
			return s.createLinuxTerminalCommandForEmulator(term, sshCommand), nil
		}
	}

	return nil, fmt.Errorf("no terminal emulator found")
}

// createLinuxTerminalCommandForEmulator creates the command for a specific Linux terminal emulator
func (s *PrismService) createLinuxTerminalCommandForEmulator(term, sshCommand string) *exec.Cmd {
	switch term {
	case "gnome-terminal":
		return exec.Command(term, "--", "bash", "-c", sshCommand) //nolint:gosec // Validated SSH command for terminal access
	case "konsole":
		return exec.Command(term, "-e", sshCommand) //nolint:gosec // Validated SSH command for terminal access
	default:
		return exec.Command(term, "-e", sshCommand) //nolint:gosec // Validated SSH command for terminal access
	}
}

// GetInstanceAccess retrieves access information for an instance
func (s *PrismService) GetInstanceAccess(ctx context.Context, instanceName string) (*InstanceAccess, error) {
	// Use the API client (same method CLI uses) - it handles all the complexity
	instance, err := s.apiClient.GetInstance(ctx, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Use the username from the instance (API client ensures this is populated correctly)
	username := instance.Username
	if username == "" {
		username = defaultUsername // Fallback if somehow empty
	}

	log.Printf("[DEBUG] GetInstanceAccess: instance=%s, template=%s, username=%q (from API client)",
		instance.Name, instance.Template, username)

	access := &InstanceAccess{
		InstanceID:   instance.ID,
		InstanceName: instance.Name,
		PublicIP:     instance.PublicIP,
		Username:     username,
		SSHPort:      22,                               // Default SSH port
		AccessTypes:  []AccessType{AccessTypeTerminal}, // SSH is always available
	}

	// Check for web interface from services
	for _, service := range instance.Services {
		if service.Port > 0 {
			access.WebPort = service.Port
			access.AccessTypes = append(access.AccessTypes, AccessTypeWeb)
			access.WebURL = fmt.Sprintf("http://%s:%d", access.PublicIP, service.Port)
			break // Use first service port
		}
	}

	// Check ports for RDP/VNC (if they exist in Ports field)
	// Note: Instance type may not have Ports field in all cases
	// For now, we rely on services for web access

	return access, nil
}

// EmbeddedWebView represents an embedded web view configuration
type EmbeddedWebView struct {
	URL      string            `json:"url"`
	Title    string            `json:"title"`
	Width    int               `json:"width"`
	Height   int               `json:"height"`
	Headers  map[string]string `json:"headers,omitempty"`
	DevTools bool              `json:"devtools,omitempty"`
}

// CreateEmbeddedWebView creates configuration for an embedded web view
func (s *PrismService) CreateEmbeddedWebView(ctx context.Context, instanceName string) (*EmbeddedWebView, error) {
	proxyURL, err := s.OpenWebInterface(ctx, instanceName)
	if err != nil {
		return nil, err
	}

	return &EmbeddedWebView{
		URL:    proxyURL,
		Title:  fmt.Sprintf("%s - Web Interface", instanceName),
		Width:  1200,
		Height: 800,
		Headers: map[string]string{
			"X-Prism-Instance": instanceName,
		},
		DevTools: false, // Enable for debugging
	}, nil
}
