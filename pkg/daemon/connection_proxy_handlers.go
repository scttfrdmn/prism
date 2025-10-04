package daemon

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
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
		// Allow connections from the CloudWorkstation GUI
		origin := r.Header.Get("Origin")
		return strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1")
	},
}

// handleSSHProxy handles WebSocket connections for embedded SSH terminals
func (s *Server) handleSSHProxy(w http.ResponseWriter, r *http.Request) {
	// Extract instance name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/ssh-proxy/")
	instanceName := strings.Split(path, "/")[0]

	if instanceName == "" {
		s.writeError(w, http.StatusBadRequest, "Instance name required")
		return
	}

	// Upgrade to WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "WebSocket upgrade failed: "+err.Error())
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// TODO: Implement SSH connection multiplexing
	// This is a placeholder - actual implementation will:
	// 1. Get instance connection details from daemon state
	// 2. Establish SSH connection to instance
	// 3. Create bidirectional data flow between WebSocket and SSH
	// 4. Handle terminal resize events
	// 5. Manage connection cleanup

	// For now, send a placeholder message
	err = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("SSH proxy for %s - implementation in progress\r\n", instanceName)))
	if err != nil {
		return
	}

	// Keep connection alive and handle messages
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Echo back for now - replace with actual SSH communication
		if messageType == websocket.TextMessage {
			err = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Echo: %s", string(p))))
			if err != nil {
				break
			}
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

	// TODO: Implement DCV proxy logic
	// This will:
	// 1. Get instance DCV connection details
	// 2. Proxy requests to DCV server on instance
	// 3. Handle DCV authentication
	// 4. Manage CORS headers for iframe embedding

	// Placeholder response
	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, `
		<html>
			<head><title>DCV Proxy - %s</title></head>
			<body>
				<h2>DCV Desktop Proxy</h2>
				<p>Instance: %s</p>
				<p>DCV desktop proxy implementation in progress...</p>
				<p>This will provide embedded desktop access via DCV web client.</p>
			</body>
		</html>
	`, instanceName, instanceName)
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
	token := r.URL.Query().Get("token")

	if region == "" {
		region = "us-west-2" // Default region
	}

	// TODO: Use token for AWS federation - currently placeholder
	_ = token

	// Build target AWS service URL
	var targetURL *url.URL
	var err error

	switch serviceName {
	case "braket":
		targetURL, err = url.Parse(fmt.Sprintf("https://%s.console.aws.amazon.com/braket/home?region=%s", region, region))
	case "sagemaker":
		targetURL, err = url.Parse(fmt.Sprintf("https://%s.console.aws.amazon.com/sagemaker/home?region=%s", region, region))
	case "console":
		targetURL, err = url.Parse(fmt.Sprintf("https://%s.console.aws.amazon.com/ec2/home?region=%s", region, region))
	case "cloudshell":
		targetURL, err = url.Parse(fmt.Sprintf("https://%s.console.aws.amazon.com/cloudshell/home?region=%s", region, region))
	default:
		s.writeError(w, http.StatusBadRequest, "Unsupported AWS service: "+serviceName)
		return
	}

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Invalid service URL: "+err.Error())
		return
	}

	// TODO: Implement AWS federation token injection
	// This will:
	// 1. Decode the federation token
	// 2. Create AWS console federation URL
	// 3. Handle AWS service-specific authentication
	// 4. Manage CORS headers for iframe embedding

	// For now, create a reverse proxy
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

// handleWebProxy handles enhanced web interface proxy (existing functionality enhanced)
func (s *Server) handleWebProxy(w http.ResponseWriter, r *http.Request) {
	// Extract instance name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/web-proxy/")
	instanceName := strings.Split(path, "/")[0]

	if instanceName == "" {
		s.writeError(w, http.StatusBadRequest, "Instance name required")
		return
	}

	// TODO: Use existing proxy logic but with enhanced CORS and embedding support
	// This will build on the existing /proxy/ endpoint implementation

	// Placeholder response
	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, `
		<html>
			<head><title>Web Proxy - %s</title></head>
			<body>
				<h2>Web Interface Proxy</h2>
				<p>Instance: %s</p>
				<p>Enhanced web proxy implementation in progress...</p>
				<p>This will provide embedded web interface access (Jupyter, RStudio, etc.).</p>
			</body>
		</html>
	`, instanceName, instanceName)
}
