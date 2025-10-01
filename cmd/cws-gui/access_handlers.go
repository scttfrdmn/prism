package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
)

// AccessType represents the type of instance access
type AccessType string

const (
	AccessTypeDesktop  AccessType = "desktop"  // RDP/VNC
	AccessTypeWeb      AccessType = "web"      // Jupyter, RStudio, etc.
	AccessTypeTerminal AccessType = "terminal" // SSH
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
func (s *CloudWorkstationService) OpenRemoteDesktop(ctx context.Context, instanceName string) error {
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
func (s *CloudWorkstationService) openRDP(access *InstanceAccess) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS: Use Microsoft Remote Desktop or open RDP URL
		rdpURL := fmt.Sprintf("rdp://full%%20address=s:%s:%d&username=s:%s",
			access.PublicIP, access.RDPPort, access.Username)
		cmd = exec.Command("open", rdpURL)
	case "windows":
		// Windows: Use mstsc
		cmd = exec.Command("mstsc", "/v:"+fmt.Sprintf("%s:%d", access.PublicIP, access.RDPPort))
	case "linux":
		// Linux: Use remmina or xfreerdp
		cmd = exec.Command("xfreerdp", "/v:"+access.PublicIP, "/port:"+fmt.Sprintf("%d", access.RDPPort),
			"/u:"+access.Username, "/dynamic-resolution", "/clipboard")
	default:
		return fmt.Errorf("RDP not supported on %s", runtime.GOOS)
	}

	return cmd.Start()
}

// openVNC opens a VNC connection
func (s *CloudWorkstationService) openVNC(access *InstanceAccess) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS: Use built-in VNC viewer
		vncURL := fmt.Sprintf("vnc://%s:%d", access.PublicIP, access.VNCPort)
		cmd = exec.Command("open", vncURL)
	case "windows":
		// Windows: Use RealVNC or TightVNC if installed
		cmd = exec.Command("vncviewer", fmt.Sprintf("%s:%d", access.PublicIP, access.VNCPort))
	case "linux":
		// Linux: Use vncviewer
		cmd = exec.Command("vncviewer", fmt.Sprintf("%s:%d", access.PublicIP, access.VNCPort))
	default:
		return fmt.Errorf("VNC not supported on %s", runtime.GOOS)
	}

	return cmd.Start()
}

// OpenWebInterface opens the web interface for an instance in an embedded browser
func (s *CloudWorkstationService) OpenWebInterface(ctx context.Context, instanceName string) (string, error) {
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
func (s *CloudWorkstationService) OpenTerminal(ctx context.Context, instanceName string) error {
	access, err := s.GetInstanceAccess(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to get instance access: %w", err)
	}

	if access.SSHPort == 0 {
		return fmt.Errorf("no SSH access available for instance %s", instanceName)
	}

	// Open terminal with SSH command
	var cmd *exec.Cmd
	sshCommand := fmt.Sprintf("ssh %s@%s -p %d", access.Username, access.PublicIP, access.SSHPort)

	switch runtime.GOOS {
	case "darwin":
		// macOS: Use Terminal.app
		script := fmt.Sprintf(`tell application "Terminal"
			do script "%s"
			activate
		end tell`, sshCommand)
		cmd = exec.Command("osascript", "-e", script)
	case "windows":
		// Windows: Use Windows Terminal or cmd
		cmd = exec.Command("cmd", "/c", "start", "cmd", "/k", sshCommand)
	case "linux":
		// Linux: Try various terminal emulators
		terminals := []string{"gnome-terminal", "konsole", "xterm", "xfce4-terminal"}
		for _, term := range terminals {
			if _, err := exec.LookPath(term); err == nil {
				switch term {
				case "gnome-terminal":
					cmd = exec.Command(term, "--", "bash", "-c", sshCommand)
				case "konsole":
					cmd = exec.Command(term, "-e", sshCommand)
				default:
					cmd = exec.Command(term, "-e", sshCommand)
				}
				break
			}
		}
		if cmd == nil {
			return fmt.Errorf("no terminal emulator found")
		}
	default:
		return fmt.Errorf("terminal not supported on %s", runtime.GOOS)
	}

	return cmd.Start()
}

// GetInstanceAccess retrieves access information for an instance
func (s *CloudWorkstationService) GetInstanceAccess(ctx context.Context, instanceName string) (*InstanceAccess, error) {
	// Get instance details from daemon
	resp, err := s.client.Get(fmt.Sprintf("%s/api/v1/instances/%s", s.daemonURL, instanceName))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instance not found: %s", instanceName)
	}

	var instance map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("failed to decode instance: %w", err)
	}

	// Build access information
	access := &InstanceAccess{
		InstanceID:   getString(instance, "id"),
		InstanceName: getString(instance, "name"),
		PublicIP:     getString(instance, "public_ip"),
		Username:     getString(instance, "username", "ubuntu"),
		SSHPort:      22,                               // Default SSH port
		AccessTypes:  []AccessType{AccessTypeTerminal}, // SSH is always available
	}

	// Check for web interface
	if getBool(instance, "has_web_interface") {
		access.WebPort = getInt(instance, "web_port")
		if access.WebPort > 0 {
			access.AccessTypes = append(access.AccessTypes, AccessTypeWeb)
			access.WebURL = fmt.Sprintf("http://%s:%d", access.PublicIP, access.WebPort)
		}
	}

	// Check ports for RDP/VNC
	ports := getIntSlice(instance, "ports")
	for _, port := range ports {
		switch port {
		case 3389:
			access.RDPPort = 3389
			access.AccessTypes = append(access.AccessTypes, AccessTypeDesktop)
		case 5900, 5901:
			access.VNCPort = port
			access.AccessTypes = append(access.AccessTypes, AccessTypeDesktop)
		}
	}

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
func (s *CloudWorkstationService) CreateEmbeddedWebView(ctx context.Context, instanceName string) (*EmbeddedWebView, error) {
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
			"X-CloudWorkstation-Instance": instanceName,
		},
		DevTools: false, // Enable for debugging
	}, nil
}

// Helper functions to safely extract values from map
func getString(m map[string]interface{}, key string, defaultVal ...string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func getInt(m map[string]interface{}, key string, defaultVal ...int) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func getBool(m map[string]interface{}, key string, defaultVal ...bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return false
}

func getIntSlice(m map[string]interface{}, key string) []int {
	if val, ok := m[key].([]interface{}); ok {
		result := make([]int, 0, len(val))
		for _, v := range val {
			if num, ok := v.(float64); ok {
				result = append(result, int(num))
			}
		}
		return result
	}
	return []int{}
}
