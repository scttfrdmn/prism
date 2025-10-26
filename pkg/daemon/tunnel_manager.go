package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// TunnelManager manages SSH tunnels for instance services
type TunnelManager struct {
	mu      sync.RWMutex
	tunnels map[string]*SSHTunnel // key: instanceName-serviceName
	state   StateManager
}

// SSHTunnel represents an active SSH tunnel
type SSHTunnel struct {
	InstanceName string
	ServiceName  string
	RemotePort   int
	LocalPort    int
	PublicIP     string
	Username     string
	KeyPath      string
	AuthToken    string // Authentication token (e.g., Jupyter token)
	cmd          *exec.Cmd
	cancel       context.CancelFunc
	status       string // "active", "connecting", "failed"
	startTime    time.Time
	lastCheck    time.Time
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(state StateManager) *TunnelManager {
	return &TunnelManager{
		tunnels: make(map[string]*SSHTunnel),
		state:   state,
	}
}

// CreateTunnel creates an SSH tunnel for a service
func (tm *TunnelManager) CreateTunnel(instance *types.Instance, service types.Service) (*SSHTunnel, error) {
	fmt.Printf("[DEBUG] CreateTunnel: START for %s/%s\n", instance.Name, service.Name)
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key := fmt.Sprintf("%s-%s", instance.Name, service.Name)

	// Check if tunnel already exists
	if existing, ok := tm.tunnels[key]; ok {
		if existing.status == "active" {
			fmt.Printf("[DEBUG] CreateTunnel: Tunnel already exists\n")
			return existing, nil
		}
		// Clean up failed tunnel
		tm.cleanupTunnel(existing)
	}

	// Determine SSH key path from profile
	fmt.Printf("[DEBUG] CreateTunnel: Getting SSH key path\n")
	keyPath, err := tm.getSSHKeyPath(instance)
	if err != nil {
		fmt.Printf("[DEBUG] CreateTunnel: SSH key path error: %v\n", err)
		return nil, fmt.Errorf("failed to get SSH key: %w", err)
	}
	fmt.Printf("[DEBUG] CreateTunnel: SSH key found at %s\n", keyPath)

	// Allocate local port
	fmt.Printf("[DEBUG] CreateTunnel: Allocating local port\n")
	localPort := tm.allocateLocalPort(service.Port)
	fmt.Printf("[DEBUG] CreateTunnel: Allocated port %d\n", localPort)

	// Determine SSH username - use instance username or fallback to "ubuntu"
	username := instance.Username
	if username == "" {
		username = "ubuntu" // Default for Ubuntu AMIs
	}

	tunnel := &SSHTunnel{
		InstanceName: instance.Name,
		ServiceName:  service.Name,
		RemotePort:   service.Port,
		LocalPort:    localPort,
		PublicIP:     instance.PublicIP,
		Username:     username,
		KeyPath:      keyPath,
		status:       "connecting",
		startTime:    time.Now(),
		lastCheck:    time.Now(),
	}

	// Create SSH tunnel command
	ctx, cancel := context.WithCancel(context.Background())
	tunnel.cancel = cancel

	// SSH tunnel: ssh -N -L localPort:localhost:remotePort user@host -i keyfile
	args := []string{
		"-N",                                                          // No command, just forward
		"-L", fmt.Sprintf("%d:localhost:%d", localPort, service.Port), // Local port forwarding
		"-o", "StrictHostKeyChecking=no", // Auto-accept host key
		"-o", "UserKnownHostsFile=/dev/null", // Don't save host key
		"-o", "BatchMode=yes", // Never prompt for password/passphrase
		"-o", "ConnectTimeout=10", // Timeout after 10 seconds
		"-o", "ServerAliveInterval=60", // Keep connection alive
		"-o", "ServerAliveCountMax=3", // Retry 3 times
		"-i", keyPath, // SSH key
		fmt.Sprintf("%s@%s", tunnel.Username, tunnel.PublicIP), // Connection
	}

	fmt.Printf("[DEBUG] Creating SSH tunnel: ssh %s\n", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "ssh", args...)
	// Capture output for debugging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	tunnel.cmd = cmd

	// Start tunnel
	if err := cmd.Start(); err != nil {
		cancel()
		fmt.Printf("[DEBUG] SSH tunnel start error: %v\n", err)
		return nil, fmt.Errorf("failed to start SSH tunnel: %w", err)
	}

	fmt.Printf("[DEBUG] SSH tunnel process started (PID: %d)\n", cmd.Process.Pid)

	// Wait briefly to see if tunnel connects
	time.Sleep(500 * time.Millisecond)

	// Check if process is still running
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		cancel()
		return nil, fmt.Errorf("SSH tunnel failed to start")
	}

	tunnel.status = "active"

	// Extract authentication token for services that need it
	if service.Name == "jupyter" {
		token := tm.extractJupyterToken(instance, tunnel)
		if token != "" {
			tunnel.AuthToken = token
		}
	}

	tm.tunnels[key] = tunnel

	// Monitor tunnel in background
	go tm.monitorTunnel(tunnel)

	return tunnel, nil
}

// allocateLocalPort allocates a consistent local port for a service
func (tm *TunnelManager) allocateLocalPort(remotePort int) int {
	// Try to use same port number locally for consistency
	if tm.isPortAvailable(remotePort) {
		return remotePort
	}

	// If port is in use, find next available port
	for port := remotePort + 1; port < 65535; port++ {
		if tm.isPortAvailable(port) {
			return port
		}
	}

	// Fallback to original port (will likely fail, but let SSH error)
	return remotePort
}

// isPortAvailable checks if a local port is available
// NOTE: Must be called with tm.mu lock already held (either read or write lock)
func (tm *TunnelManager) isPortAvailable(port int) bool {
	// Check if port is already used by another tunnel
	// No locking here - caller must hold the lock
	for _, tunnel := range tm.tunnels {
		if tunnel.LocalPort == port {
			return false
		}
	}

	// Try to bind to the port to check if it's available
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false // Port is in use
	}
	listener.Close()
	return true
}

// getSSHKeyPath gets the SSH key path for an instance using the EC2 KeyName
func (tm *TunnelManager) getSSHKeyPath(instance *types.Instance) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// If instance has KeyName from EC2, use that directly
	if instance.KeyName != "" {
		fmt.Printf("[DEBUG] Using EC2 KeyName: %s\n", instance.KeyName)

		// Try standard SSH locations with EC2 KeyName
		candidatePaths := []string{
			filepath.Join(homeDir, ".ssh", instance.KeyName),
			filepath.Join(homeDir, ".ssh", instance.KeyName+".pem"),
			filepath.Join(homeDir, ".prism", "profiles", "test", "ssh", instance.KeyName),
			filepath.Join(homeDir, ".prism", "profiles", "test", "ssh", instance.KeyName+".pem"),
		}

		for _, keyPath := range candidatePaths {
			fmt.Printf("[DEBUG] Trying SSH key: %s\n", keyPath)
			if _, err := os.Stat(keyPath); err == nil {
				fmt.Printf("[DEBUG] Found SSH key: %s\n", keyPath)
				return keyPath, nil
			}
		}

		return "", fmt.Errorf("SSH key not found for EC2 KeyName '%s'. Tried %d locations in ~/.ssh/", instance.KeyName, len(candidatePaths))
	}

	// LEGACY FALLBACK: If no KeyName, try guessing based on region
	// This maintains backward compatibility but should not be needed
	fmt.Printf("[DEBUG] No EC2 KeyName available, falling back to region-based guessing\n")

	region := instance.Region

	// Try standardized naming with different profile names
	// Priority: test (most common), default, no-profile
	standardizedNames := []string{
		fmt.Sprintf("cws-test-%s-key", region),    // cws-test-us-west-2-key (STANDARD FORMAT)
		fmt.Sprintf("cws-default-%s-key", region), // cws-default-us-west-2-key
		fmt.Sprintf("cws-%s-key", region),         // cws-us-west-2-key
	}

	var candidatePaths []string

	// Try standardized locations first
	for _, keyName := range standardizedNames {
		candidatePaths = append(candidatePaths,
			filepath.Join(homeDir, ".ssh", keyName),
			filepath.Join(homeDir, ".prism", "profiles", "test", "ssh", keyName),
		)
	}

	// For backward compatibility, also try legacy formats
	// Legacy naming: cws-test-aws-{regionshort}-key where regionshort has hyphens removed
	// Example: us-west-2 → west2, so cws-test-aws-west2-key
	regionShort := strings.TrimPrefix(region, "us-")
	regionShort = strings.Replace(regionShort, "-", "", -1) // west-2 → west2

	legacyFormats := []string{
		fmt.Sprintf("cws-test-aws-%s-key", regionShort), // cws-test-aws-west2-key
		fmt.Sprintf("cws-aws-%s-key", regionShort),      // cws-aws-west2-key
	}
	for _, legacyName := range legacyFormats {
		candidatePaths = append(candidatePaths,
			filepath.Join(homeDir, ".ssh", legacyName),
			filepath.Join(homeDir, ".prism", "profiles", "test", "ssh", legacyName),
		)
	}

	for _, keyPath := range candidatePaths {
		fmt.Printf("[DEBUG] Trying SSH key: %s\n", keyPath)
		if _, err := os.Stat(keyPath); err == nil {
			fmt.Printf("[DEBUG] Found SSH key: %s\n", keyPath)
			return keyPath, nil
		}
	}

	// Final fallback: scan ~/.ssh for any cws-* keys
	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), "cws-") && !strings.HasSuffix(entry.Name(), ".pub") {
				keyPath := filepath.Join(sshDir, entry.Name())
				// Verify it's a valid SSH key file
				if _, err := os.Stat(keyPath); err == nil {
					return keyPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("SSH key not found. Expected format: cws-test-%s-key in ~/.ssh/ (tried %d locations + fallback scan)", region, len(candidatePaths))
}

// monitorTunnel monitors tunnel health and restarts if needed
func (tm *TunnelManager) monitorTunnel(tunnel *SSHTunnel) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.mu.Lock()
			tunnel.lastCheck = time.Now()

			// Check if process is still running
			if tunnel.cmd.ProcessState != nil && tunnel.cmd.ProcessState.Exited() {
				tunnel.status = "failed"
				tm.mu.Unlock()
				return
			}
			tm.mu.Unlock()

		case <-context.Background().Done():
			return
		}
	}
}

// GetTunnel retrieves an existing tunnel
func (tm *TunnelManager) GetTunnel(instanceName, serviceName string) (*SSHTunnel, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	key := fmt.Sprintf("%s-%s", instanceName, serviceName)
	tunnel, ok := tm.tunnels[key]
	return tunnel, ok
}

// GetInstanceTunnels returns all tunnels for an instance
func (tm *TunnelManager) GetInstanceTunnels(instanceName string) []*SSHTunnel {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tunnels []*SSHTunnel
	for _, tunnel := range tm.tunnels {
		if tunnel.InstanceName == instanceName {
			tunnels = append(tunnels, tunnel)
		}
	}
	return tunnels
}

// CloseTunnel closes a specific tunnel
func (tm *TunnelManager) CloseTunnel(instanceName, serviceName string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key := fmt.Sprintf("%s-%s", instanceName, serviceName)
	tunnel, ok := tm.tunnels[key]
	if !ok {
		return fmt.Errorf("tunnel not found")
	}

	tm.cleanupTunnel(tunnel)
	delete(tm.tunnels, key)
	return nil
}

// CloseInstanceTunnels closes all tunnels for an instance
func (tm *TunnelManager) CloseInstanceTunnels(instanceName string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for key, tunnel := range tm.tunnels {
		if tunnel.InstanceName == instanceName {
			tm.cleanupTunnel(tunnel)
			delete(tm.tunnels, key)
		}
	}
}

// cleanupTunnel cleans up a tunnel (must be called with lock held)
func (tm *TunnelManager) cleanupTunnel(tunnel *SSHTunnel) {
	if tunnel.cancel != nil {
		tunnel.cancel()
	}
	if tunnel.cmd != nil && tunnel.cmd.Process != nil {
		_ = tunnel.cmd.Process.Kill()
	}
}

// extractJupyterToken extracts the authentication token from a Jupyter instance
func (tm *TunnelManager) extractJupyterToken(instance *types.Instance, tunnel *SSHTunnel) string {
	// Try to extract token from Jupyter runtime files
	// Common locations:
	// - ~/.jupyter/runtime/jpserver-*.json (JupyterLab 3+)
	// - ~/.local/share/jupyter/runtime/jpserver-*.json
	// - jupyter server list output

	// Use SSH to run jupyter server list command
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=5",
		"-i", tunnel.KeyPath,
		fmt.Sprintf("%s@%s", tunnel.Username, tunnel.PublicIP),
		"jupyter", "server", "list", "2>/dev/null", "||", "jupyter", "notebook", "list", "2>/dev/null",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.Output()
	if err != nil {
		// Token extraction is optional - don't fail if it doesn't work
		return ""
	}

	// Parse output to extract token
	// Format: http://localhost:8888/?token=abc123 :: /home/user
	lines := string(output)

	// Look for token= in the output
	if idx := strings.Index(lines, "token="); idx != -1 {
		tokenStart := idx + 6 // len("token=")
		tokenEnd := tokenStart
		for tokenEnd < len(lines) && lines[tokenEnd] != ' ' && lines[tokenEnd] != '\n' && lines[tokenEnd] != '\r' {
			tokenEnd++
		}
		if tokenEnd > tokenStart {
			return lines[tokenStart:tokenEnd]
		}
	}

	return ""
}

// CloseAll closes all tunnels
func (tm *TunnelManager) CloseAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for key, tunnel := range tm.tunnels {
		tm.cleanupTunnel(tunnel)
		delete(tm.tunnels, key)
	}
}
