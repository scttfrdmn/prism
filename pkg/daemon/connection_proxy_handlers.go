package daemon

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/scttfrdmn/prism/pkg/types"
	"golang.org/x/crypto/ssh"
)

// RegisterConnectionProxyRoutes registers all connection proxy endpoints
func (s *Server) RegisterConnectionProxyRoutes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// SSH WebSocket proxy endpoints
	mux.HandleFunc("/ssh-proxy/", applyMiddleware(s.handleSSHProxy))

	// DCV desktop proxy endpoints
	mux.HandleFunc("/dcv-proxy/", applyMiddleware(s.handleDCVProxy))

	// AWS service proxy endpoints
	mux.HandleFunc("/aws-proxy/", applyMiddleware(s.handleAWSServiceProxy))

	// Enhanced web proxy (existing /proxy/ enhanced)
	mux.HandleFunc("/web-proxy/", applyMiddleware(s.handleWebProxy))
}

// WebSocket upgrader for SSH connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from the Prism GUI
		origin := r.Header.Get("Origin")
		return strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1")
	},
}

// handleSSHProxy handles WebSocket connections for embedded SSH terminals with full SSH multiplexing
func (s *Server) handleSSHProxy(w http.ResponseWriter, r *http.Request) {
	// Parse instance name from URL
	instanceName, err := s.parseSSHProxyInstanceName(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get instance details
	instance, err := s.getInstanceForSSHProxy(instanceName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Upgrade to WebSocket
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "WebSocket upgrade failed: "+err.Error())
		return
	}
	defer func() {
		_ = wsConn.Close()
	}()

	// Send welcome message
	if err := s.sendSSHWelcomeMessage(wsConn, instanceName, instance.PublicIP); err != nil {
		return
	}

	// Setup SSH connection
	sshClient, session, err := s.setupSSHConnection(wsConn, &instance)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v\r\n", err)))
		return
	}
	defer sshClient.Close()
	defer session.Close()

	// Setup SSH pipes
	sshStdin, sshStdout, sshStderr, err := s.setupSSHPipes(session)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v\r\n", err)))
		return
	}

	// Start shell
	if err := session.Shell(); err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to start shell: %v\r\n", err)))
		return
	}

	_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Connected to %s\r\n", instanceName)))

	// Run SSH proxy loop
	s.runSSHProxyLoop(wsConn, sshStdin, sshStdout, sshStderr, session)
}

// parseSSHProxyInstanceName extracts instance name from URL
func (s *Server) parseSSHProxyInstanceName(r *http.Request) (string, error) {
	path := strings.TrimPrefix(r.URL.Path, "/ssh-proxy/")
	instanceName := strings.Split(path, "/")[0]

	if instanceName == "" {
		return "", fmt.Errorf("instance name required")
	}
	return instanceName, nil
}

// getInstanceForSSHProxy retrieves instance from state
func (s *Server) getInstanceForSSHProxy(instanceName string) (types.Instance, error) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		return types.Instance{}, fmt.Errorf("failed to load state: %w", err)
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		return types.Instance{}, fmt.Errorf("instance not found: %s", instanceName)
	}
	return instance, nil
}

// sendSSHWelcomeMessage sends initial connection message
func (s *Server) sendSSHWelcomeMessage(wsConn *websocket.Conn, instanceName, publicIP string) error {
	welcomeMsg := fmt.Sprintf("Connecting to %s (%s)...\r\n", instanceName, publicIP)
	return wsConn.WriteMessage(websocket.TextMessage, []byte(welcomeMsg))
}

// setupSSHConnection establishes SSH connection with proper authentication
func (s *Server) setupSSHConnection(wsConn *websocket.Conn, instance *types.Instance) (*ssh.Client, *ssh.Session, error) {
	// Get SSH key and config
	signer, err := s.loadSSHKey(wsConn)
	if err != nil {
		return nil, nil, err
	}

	hostKeyCallback, err := s.setupHostKeyVerification(wsConn, instance.PublicIP)
	if err != nil {
		return nil, nil, err
	}

	// Determine SSH username
	sshUsername := instance.Username
	if sshUsername == "" {
		sshUsername = "ec2-user"
	}

	sshConfig := &ssh.ClientConfig{
		User: sshUsername,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	// Connect to SSH server
	sshAddr := fmt.Sprintf("%s:22", instance.PublicIP)
	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to SSH: %w", err)
	}

	// Create SSH session
	session, err := sshClient.NewSession()
	if err != nil {
		sshClient.Close()
		return nil, nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Request pseudo terminal
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", 80, 40, modes); err != nil {
		session.Close()
		sshClient.Close()
		return nil, nil, fmt.Errorf("failed to request PTY: %w", err)
	}

	return sshClient, session, nil
}

// loadSSHKey loads and parses SSH private key
func (s *Server) loadSSHKey(wsConn *websocket.Conn) (ssh.Signer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	keyPath := filepath.Join(homeDir, ".prism", "ssh_keys", "id_ed25519")
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
	}
	return signer, nil
}

// setupHostKeyVerification configures host key verification
func (s *Server) setupHostKeyVerification(wsConn *websocket.Conn, publicIP string) (ssh.HostKeyCallback, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	knownHostsPath := filepath.Join(homeDir, ".prism", "known_hosts")

	// Try to load known hosts file
	if _, err := os.Stat(knownHostsPath); err == nil {
		hostKeyCallback, err := loadKnownHosts(knownHostsPath)
		if err != nil {
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Warning: Failed to load known hosts, using insecure mode: %v\r\n", err)))
			return ssh.InsecureIgnoreHostKey(), nil
		}
		return hostKeyCallback, nil
	}

	// Create known hosts file if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Warning: Failed to create known hosts directory, using insecure mode: %v\r\n", err)))
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// Use trust-on-first-use
	return trustOnFirstUse(knownHostsPath, publicIP), nil
}

// setupSSHPipes sets up SSH session pipes
func (s *Server) setupSSHPipes(session *ssh.Session) (io.WriteCloser, io.Reader, io.Reader, error) {
	sshStdin, err := session.StdinPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	sshStdout, err := session.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	sshStderr, err := session.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	return sshStdin, sshStdout, sshStderr, nil
}

// runSSHProxyLoop handles bidirectional data forwarding
func (s *Server) runSSHProxyLoop(wsConn *websocket.Conn, sshStdin io.WriteCloser, sshStdout, sshStderr io.Reader, session *ssh.Session) {
	done := make(chan error, 3)

	// WebSocket -> SSH (stdin)
	go s.forwardWebSocketToSSH(wsConn, sshStdin, done)

	// SSH stdout -> WebSocket
	go s.forwardSSHToWebSocket(wsConn, sshStdout, done, "stdout")

	// SSH stderr -> WebSocket
	go s.forwardSSHToWebSocket(wsConn, sshStderr, done, "stderr")

	// Wait for session to end
	<-done
	_ = session.Wait()
}

// forwardWebSocketToSSH forwards WebSocket messages to SSH stdin
func (s *Server) forwardWebSocketToSSH(wsConn *websocket.Conn, sshStdin io.WriteCloser, done chan error) {
	for {
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			done <- err
			return
		}
		if _, err := sshStdin.Write(message); err != nil {
			done <- err
			return
		}
	}
}

// forwardSSHToWebSocket forwards SSH output to WebSocket
func (s *Server) forwardSSHToWebSocket(wsConn *websocket.Conn, sshOutput io.Reader, done chan error, name string) {
	buf := make([]byte, 32*1024)
	for {
		n, err := sshOutput.Read(buf)
		if err != nil {
			if err != io.EOF {
				done <- err
			}
			return
		}
		if err := wsConn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
			done <- err
			return
		}
	}
}

// handleDCVProxy handles DCV desktop connections via iframe
func (s *Server) handleDCVProxy(w http.ResponseWriter, r *http.Request) {
	// Extract instance name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/dcv-proxy/")
	instanceName := strings.Split(path, "/")[0]

	if instanceName == "" {
		s.writeError(w, http.StatusBadRequest, "Instance name required")
		return
	}

	// Get instance connection details from state
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load state: %v", err))
		return
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %s", instanceName))
		return
	}

	// DCV typically runs on port 8443 for HTTPS
	dcvPort := r.URL.Query().Get("port")
	if dcvPort == "" {
		dcvPort = "8443"
	}

	// Build DCV connection URL
	dcvURL := fmt.Sprintf("https://%s:%s", instance.PublicIP, dcvPort)

	// Create reverse proxy to DCV server
	targetURL, err := url.Parse(dcvURL)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Invalid DCV URL: %v", err))
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify headers for iframe embedding
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("X-Frame-Options")
		resp.Header.Del("Content-Security-Policy")
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		return nil
	}

	// Proxy the request to DCV server
	proxy.ServeHTTP(w, r)
}

// handleAWSServiceProxy handles AWS service connections with federation
func (s *Server) handleAWSServiceProxy(w http.ResponseWriter, r *http.Request) {
	// Extract service name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/aws-proxy/")
	serviceName := strings.Split(path, "/")[0]

	if serviceName == "" {
		s.writeError(w, http.StatusBadRequest, "AWS service name required")
		return
	}

	// Get query parameters
	region := r.URL.Query().Get("region")
	federationToken := r.URL.Query().Get("token")

	if region == "" {
		region = "us-west-2" // Default region
	}

	// Build target AWS service URL based on service name
	var destinationURL string

	switch serviceName {
	case "braket":
		destinationURL = fmt.Sprintf("https://%s.console.aws.amazon.com/braket/home?region=%s", region, region)
	case "sagemaker":
		destinationURL = fmt.Sprintf("https://%s.console.aws.amazon.com/sagemaker/home?region=%s", region, region)
	case "console":
		destinationURL = fmt.Sprintf("https://%s.console.aws.amazon.com/ec2/home?region=%s", region, region)
	case "cloudshell":
		destinationURL = fmt.Sprintf("https://%s.console.aws.amazon.com/cloudshell/home?region=%s", region, region)
	default:
		s.writeError(w, http.StatusBadRequest, "Unsupported AWS service: "+serviceName)
		return
	}

	// If federation token provided, generate AWS console federation URL
	var finalURL string
	if federationToken != "" {
		// AWS Console Federation URL format:
		// https://signin.aws.amazon.com/federation?Action=login&Issuer=<issuer>&Destination=<destination>&SigninToken=<token>

		// URL-encode the destination
		encodedDestination := url.QueryEscape(destinationURL)
		issuer := url.QueryEscape("Prism")

		// Build federation signin URL
		finalURL = fmt.Sprintf("https://signin.aws.amazon.com/federation?Action=login&Issuer=%s&Destination=%s&SigninToken=%s",
			issuer, encodedDestination, federationToken)
	} else {
		// No federation token - direct link to AWS console (requires user to be logged in)
		finalURL = destinationURL
	}

	targetURL, err := url.Parse(finalURL)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Invalid service URL: "+err.Error())
		return
	}

	// Create reverse proxy with federation support
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify headers for embedding
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Remove X-Frame-Options to allow embedding
		resp.Header.Del("X-Frame-Options")
		// Add CORS headers
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		resp.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		return nil
	}

	// Handle the proxy request
	proxy.ServeHTTP(w, r)
}

// handleWebProxy handles enhanced web interface proxy with embedding support
func (s *Server) handleWebProxy(w http.ResponseWriter, r *http.Request) {
	// Extract instance name and target path from URL
	path := strings.TrimPrefix(r.URL.Path, "/web-proxy/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		s.writeError(w, http.StatusBadRequest, "Instance name required")
		return
	}

	instanceName := parts[0]
	targetPath := "/"
	if len(parts) > 1 {
		targetPath = "/" + parts[1]
	}

	// Get instance connection info from state
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load state: %v", err))
		return
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %s", instanceName))
		return
	}

	// Determine target port (default to 8888 for Jupyter, can be customized)
	targetPort := r.URL.Query().Get("port")
	if targetPort == "" {
		targetPort = "8888" // Default to Jupyter port
	}

	// Build target URL for the instance
	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%s%s", instance.PublicIP, targetPort, targetPath))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Invalid target URL: %v", err))
		return
	}

	// Create reverse proxy with enhanced CORS headers for iframe embedding
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify response headers to enable embedding
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Remove headers that prevent iframe embedding
		resp.Header.Del("X-Frame-Options")
		resp.Header.Del("Content-Security-Policy")

		// Add CORS headers for cross-origin access
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		resp.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		resp.Header.Set("Access-Control-Allow-Credentials", "true")

		return nil
	}

	// Handle CORS preflight requests
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Proxy the request
	proxy.ServeHTTP(w, r)
}

// loadKnownHosts loads SSH known hosts from file
func loadKnownHosts(path string) (ssh.HostKeyCallback, error) {
	knownHostsBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read known hosts: %w", err)
	}

	// Parse known hosts entries
	knownHosts := make(map[string]ssh.PublicKey)
	lines := strings.Split(string(knownHostsBytes), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse known_hosts line format: "host key-type key-data"
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		host := parts[0]
		keyType := parts[1]
		keyData := parts[2]

		// Decode the public key
		keyBytes := []byte(keyType + " " + keyData)
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey(keyBytes)
		if err != nil {
			continue
		}

		knownHosts[host] = pubKey
	}

	// Return callback that verifies against known hosts
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Check if we have this host's key
		expectedKey, exists := knownHosts[hostname]
		if !exists {
			return fmt.Errorf("host %s not found in known_hosts", hostname)
		}

		// Compare keys
		if !bytes.Equal(key.Marshal(), expectedKey.Marshal()) {
			return fmt.Errorf("host key mismatch for %s", hostname)
		}

		return nil
	}, nil
}

// trustOnFirstUse creates a host key callback that accepts and saves new host keys
func trustOnFirstUse(knownHostsPath, hostname string) ssh.HostKeyCallback {
	return func(host string, remote net.Addr, key ssh.PublicKey) error {
		// Read existing known_hosts
		knownHosts := make(map[string]string)
		if data, err := os.ReadFile(knownHostsPath); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					knownHosts[parts[0]] = line
				}
			}
		}

		// Check if we already have this host
		if existingLine, exists := knownHosts[hostname]; exists {
			// Parse existing key
			parts := strings.Fields(existingLine)
			if len(parts) >= 3 {
				keyBytes := []byte(parts[1] + " " + parts[2])
				existingKey, _, _, _, err := ssh.ParseAuthorizedKey(keyBytes)
				if err == nil {
					// Verify key matches
					if !bytes.Equal(key.Marshal(), existingKey.Marshal()) {
						return fmt.Errorf("WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED for %s", hostname)
					}
					return nil
				}
			}
		}

		// New host - add to known_hosts
		authorizedKey := ssh.MarshalAuthorizedKey(key)
		knownHostEntry := fmt.Sprintf("%s %s", hostname, strings.TrimSpace(string(authorizedKey)))

		// Append to known_hosts file
		f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("failed to open known_hosts for writing: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString(knownHostEntry + "\n"); err != nil {
			return fmt.Errorf("failed to write to known_hosts: %w", err)
		}

		return nil
	}
}
