package cli

import (
	"fmt"
	"os/exec"
	"runtime"
)

// WebCommands handles web service commands
type WebCommands struct {
	app *App
}

// NewWebCommands creates a new WebCommands instance
func NewWebCommands(app *App) *WebCommands {
	return &WebCommands{app: app}
}

// List lists all web services for an instance
func (wc *WebCommands) List(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws web list <instance-name>", "cws web list my-jupyter")
	}

	instanceName := args[0]

	// Ensure daemon is running
	if err := wc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Get instance details
	instance, err := wc.app.apiClient.GetInstance(wc.app.ctx, instanceName)
	if err != nil {
		return WrapAPIError("get instance "+instanceName, err)
	}

	if len(instance.Services) == 0 {
		fmt.Printf("No web services configured for instance %s\n", instanceName)
		return nil
	}

	// Get tunnel status
	tunnels, err := wc.app.apiClient.ListTunnels(wc.app.ctx, instanceName)
	if err != nil {
		// Don't fail - just show services without tunnel status
		tunnels = nil
	}

	fmt.Printf("Web services for %s:\n\n", instanceName)

	for _, service := range instance.Services {
		tunnelActive := false
		var localURL string

		// Check if service has active tunnel
		if tunnels != nil {
			for _, tunnel := range tunnels.Tunnels {
				if tunnel.ServiceName == service.Name {
					tunnelActive = true
					localURL = tunnel.LocalURL
					if tunnel.AuthToken != "" {
						localURL = fmt.Sprintf("%s?token=%s", tunnel.LocalURL, tunnel.AuthToken)
					}
					break
				}
			}
		}

		status := "❌"
		if tunnelActive {
			status = "✅"
		}

		fmt.Printf("%s %s", status, service.Description)
		if service.Description == "" {
			fmt.Printf("%s %s (port %d)", status, service.Name, service.Port)
		} else {
			fmt.Printf(" (port %d)", service.Port)
		}

		if tunnelActive {
			fmt.Printf("\n   URL: %s", localURL)
		} else {
			fmt.Printf("\n   Not tunneled - use 'cws web open %s %s' to access", instanceName, service.Name)
		}
		fmt.Printf("\n\n")
	}

	return nil
}

// Open opens a web service in the default browser
func (wc *WebCommands) Open(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws web open <instance-name> <service-name>", "cws web open my-jupyter jupyter")
	}

	instanceName := args[0]
	serviceName := args[1]

	// Ensure daemon is running
	if err := wc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Create tunnel
	fmt.Printf("🌐 Creating tunnel for %s...\n", serviceName)
	tunnelResp, err := wc.app.apiClient.CreateTunnels(wc.app.ctx, instanceName, []string{serviceName})
	if err != nil {
		return WrapAPIError("create tunnel", err)
	}

	if len(tunnelResp.Tunnels) == 0 {
		return fmt.Errorf("no tunnel created for service %s", serviceName)
	}

	tunnel := tunnelResp.Tunnels[0]
	url := tunnel.LocalURL
	if tunnel.AuthToken != "" {
		url = fmt.Sprintf("%s?token=%s", tunnel.LocalURL, tunnel.AuthToken)
	}

	fmt.Printf("✅ Tunnel created: %s\n", url)
	fmt.Printf("🌐 Opening in browser...\n")

	// Open URL in default browser
	if err := openBrowser(url); err != nil {
		fmt.Printf("⚠️  Could not open browser automatically: %v\n", err)
		fmt.Printf("Please open manually: %s\n", url)
		return nil
	}

	fmt.Printf("✅ Browser opened\n")
	return nil
}

// Close closes web service tunnels
func (wc *WebCommands) Close(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws web close <instance-name> [service-name]", "cws web close my-jupyter jupyter")
	}

	instanceName := args[0]
	var serviceName string
	if len(args) >= 2 {
		serviceName = args[1]
	}

	// Ensure daemon is running
	if err := wc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	if serviceName != "" {
		// Close specific tunnel
		fmt.Printf("🔒 Closing tunnel for %s/%s...\n", instanceName, serviceName)
		if err := wc.app.apiClient.CloseTunnel(wc.app.ctx, instanceName, serviceName); err != nil {
			return WrapAPIError("close tunnel", err)
		}
		fmt.Printf("✅ Tunnel closed\n")
	} else {
		// Close all tunnels for instance
		fmt.Printf("🔒 Closing all tunnels for %s...\n", instanceName)
		if err := wc.app.apiClient.CloseInstanceTunnels(wc.app.ctx, instanceName); err != nil {
			return WrapAPIError("close tunnels", err)
		}
		fmt.Printf("✅ All tunnels closed\n")
	}

	return nil
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
