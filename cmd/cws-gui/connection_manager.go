package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ConnectionManager manages the lifecycle of embedded connections
type ConnectionManager struct {
	connections map[string]*ConnectionConfig
	mutex       sync.RWMutex
	service     *CloudWorkstationService
	callbacks   map[string]func(*ConnectionConfig)
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(service *CloudWorkstationService) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*ConnectionConfig),
		service:     service,
		callbacks:   make(map[string]func(*ConnectionConfig)),
	}
}

// CreateConnection creates a new connection and starts monitoring it
func (cm *ConnectionManager) CreateConnection(ctx context.Context, connectionType ConnectionType, target string, options map[string]string) (*ConnectionConfig, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	var config *ConnectionConfig
	var err error

	// Create connection configuration based on type
	switch connectionType {
	case ConnectionTypeSSH:
		config = &ConnectionConfig{
			ID:            fmt.Sprintf("ssh-%s-%d", target, time.Now().Unix()),
			Type:          ConnectionTypeSSH,
			InstanceName:  target,
			ProxyURL:      fmt.Sprintf("http://localhost:8947/ssh-proxy/%s", target),
			EmbeddingMode: "websocket",
			Title:         fmt.Sprintf("🖥️ SSH: %s", target),
			Status:        "connecting",
			Metadata: map[string]interface{}{
				"connection_type": "ssh",
				"launch_time":     time.Now().Format(time.RFC3339),
			},
		}
	case ConnectionTypeDesktop:
		config = &ConnectionConfig{
			ID:            fmt.Sprintf("desktop-%s-%d", target, time.Now().Unix()),
			Type:          ConnectionTypeDesktop,
			InstanceName:  target,
			ProxyURL:      fmt.Sprintf("http://localhost:8947/dcv-proxy/%s", target),
			EmbeddingMode: "iframe",
			Title:         fmt.Sprintf("🖥️ Desktop: %s", target),
			Status:        "connecting",
			Metadata: map[string]interface{}{
				"connection_type": "desktop",
				"launch_time":     time.Now().Format(time.RFC3339),
			},
		}
	case ConnectionTypeWeb:
		service := options["service"]
		if service == "" {
			service = "jupyter"
		}
		config = &ConnectionConfig{
			ID:            fmt.Sprintf("web-%s-%s-%d", target, service, time.Now().Unix()),
			Type:          ConnectionTypeWeb,
			InstanceName:  target,
			ProxyURL:      fmt.Sprintf("http://localhost:8947/web-proxy/%s", target),
			EmbeddingMode: "iframe",
			Title:         fmt.Sprintf("🌐 %s: %s", service, target),
			Status:        "connecting",
			Metadata: map[string]interface{}{
				"connection_type": "web",
				"service":         service,
				"launch_time":     time.Now().Format(time.RFC3339),
			},
		}
	case ConnectionTypeAWS:
		region := options["region"]
		if region == "" {
			region = "us-west-2"
		}
		service := options["service"]
		if service == "" {
			service = "console"
		}

		var title string
		switch service {
		case "braket":
			title = fmt.Sprintf("⚛️ Braket (%s)", region)
		case "sagemaker":
			title = fmt.Sprintf("🤖 SageMaker (%s)", region)
		case "console":
			title = fmt.Sprintf("🎛️ Console (%s)", region)
		case "cloudshell":
			title = fmt.Sprintf("🖥️ CloudShell (%s)", region)
		default:
			title = fmt.Sprintf("☁️ %s (%s)", service, region)
		}

		config = &ConnectionConfig{
			ID:            fmt.Sprintf("aws-%s-%s-%d", service, region, time.Now().Unix()),
			Type:          ConnectionTypeAWS,
			AWSService:    service,
			Region:        region,
			ProxyURL:      fmt.Sprintf("http://localhost:8947/aws-proxy/%s?region=%s", service, region),
			EmbeddingMode: "iframe",
			Title:         title,
			Status:        "connecting",
			Metadata: map[string]interface{}{
				"connection_type": "aws",
				"service":         service,
				"region":          region,
				"launch_time":     time.Now().Format(time.RFC3339),
			},
		}
	default:
		return nil, fmt.Errorf("unsupported connection type: %s", connectionType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create %s connection: %w", connectionType, err)
	}

	// Store connection
	cm.connections[config.ID] = config

	// Start monitoring connection status
	go cm.monitorConnection(config.ID)

	return config, nil
}

// GetConnection retrieves a connection by ID
func (cm *ConnectionManager) GetConnection(id string) (*ConnectionConfig, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	config, exists := cm.connections[id]
	return config, exists
}

// GetAllConnections returns all active connections
func (cm *ConnectionManager) GetAllConnections() []*ConnectionConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	connections := make([]*ConnectionConfig, 0, len(cm.connections))
	for _, config := range cm.connections {
		connections = append(connections, config)
	}

	return connections
}

// UpdateConnection updates a connection's status
func (cm *ConnectionManager) UpdateConnection(id string, status string, message string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	config, exists := cm.connections[id]
	if !exists {
		return fmt.Errorf("connection %s not found", id)
	}

	config.Status = status
	if config.Metadata == nil {
		config.Metadata = make(map[string]interface{})
	}
	config.Metadata["last_update"] = time.Now().Format(time.RFC3339)

	if message != "" {
		config.Metadata["status_message"] = message
	}

	// Notify callback if registered
	if callback, exists := cm.callbacks[id]; exists {
		callback(config)
	}

	return nil
}

// CloseConnection closes a connection and cleans up resources
func (cm *ConnectionManager) CloseConnection(id string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	config, exists := cm.connections[id]
	if !exists {
		return fmt.Errorf("connection %s not found", id)
	}

	// Update status to disconnected
	config.Status = "disconnected"
	config.Metadata["closed_at"] = time.Now().Format(time.RFC3339)

	// Remove from active connections
	delete(cm.connections, id)
	delete(cm.callbacks, id)

	return nil
}

// RegisterCallback registers a callback for connection status changes
func (cm *ConnectionManager) RegisterCallback(id string, callback func(*ConnectionConfig)) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.callbacks[id] = callback
}

// monitorConnection monitors a connection's status in the background
func (cm *ConnectionManager) monitorConnection(id string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			config, exists := cm.GetConnection(id)
			if !exists {
				// Connection was closed
				return
			}

			// Check connection health based on type
			var newStatus string
			switch config.Type {
			case ConnectionTypeSSH:
				newStatus = cm.checkSSHStatus(config)
			case ConnectionTypeDesktop:
				newStatus = cm.checkDesktopStatus(config)
			case ConnectionTypeWeb:
				newStatus = cm.checkWebStatus(config)
			case ConnectionTypeAWS:
				newStatus = cm.checkAWSStatus(config)
			default:
				newStatus = "unknown"
			}

			if newStatus != config.Status {
				cm.UpdateConnection(id, newStatus, "")
			}
		}
	}
}

// Status check methods for different connection types
func (cm *ConnectionManager) checkSSHStatus(config *ConnectionConfig) string {
	// Health check SSH connection via WebSocket proxy endpoint
	if config.ProxyURL == "" {
		return "error"
	}

	// Check if the WebSocket endpoint is reachable
	// Convert WebSocket URL to HTTP for health check
	healthURL := strings.Replace(config.ProxyURL, "ws://", "http://", 1)
	healthURL = strings.Replace(healthURL, "wss://", "https://", 1)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		return "disconnected"
	}
	defer resp.Body.Close()

	// Check if we get a reasonable response (could be upgrade required for WebSocket)
	if resp.StatusCode < 500 {
		return "connected"
	}

	return "disconnected"
}

func (cm *ConnectionManager) checkDesktopStatus(config *ConnectionConfig) string {
	// Health check DCV desktop connection
	if config.ProxyURL == "" {
		return "error"
	}

	// For DCV connections, check if the session endpoint is responding
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects for DCV authentication flows
			return nil
		},
	}

	resp, err := client.Get(config.ProxyURL)
	if err != nil {
		return "disconnected"
	}
	defer resp.Body.Close()

	// DCV sessions typically respond with 200 OK or redirect to login
	if resp.StatusCode == 200 || resp.StatusCode == 302 || resp.StatusCode == 401 {
		return "connected"
	}

	return "disconnected"
}

func (cm *ConnectionManager) checkWebStatus(config *ConnectionConfig) string {
	// Health check web interface connection
	if config.ProxyURL == "" {
		return "error"
	}

	// For web interfaces (Jupyter, RStudio, etc.), check if the service is responding
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects for authentication flows
			return nil
		},
	}

	resp, err := client.Get(config.ProxyURL)
	if err != nil {
		return "disconnected"
	}
	defer resp.Body.Close()

	// Web services typically respond with 200 OK, or redirects for login/auth
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return "connected"
	}

	return "disconnected"
}

func (cm *ConnectionManager) checkAWSStatus(config *ConnectionConfig) string {
	// Health check AWS service connection
	if config.ProxyURL == "" && config.AuthToken == "" {
		return "error"
	}

	// For AWS service connections, we can check a few things:
	// 1. If there's a ProxyURL, check if it's accessible
	// 2. If there's an AuthToken, we assume it's a federation token and check basic AWS access

	if config.ProxyURL != "" {
		// Check proxied AWS service endpoint
		client := &http.Client{
			Timeout: 10 * time.Second, // AWS services might take longer to respond
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// AWS console redirects are common
				return nil
			},
		}

		resp, err := client.Get(config.ProxyURL)
		if err != nil {
			return "disconnected"
		}
		defer resp.Body.Close()

		// AWS services typically respond with 200 OK or redirects for authentication
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return "connected"
		}

		return "disconnected"
	}

	// If no ProxyURL but has AuthToken, assume it's a direct federation connection
	if config.AuthToken != "" {
		// For federation tokens, we assume they're valid if they exist
		// A more sophisticated check would validate the token with AWS STS
		return "connected"
	}

	// No way to verify connection
	return "error"
}
