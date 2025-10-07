// Package main provides connection management for the CloudWorkstation GUI
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Connection status constants
const (
	statusDisconnected = "disconnected"
	statusConnected    = "connected"
	statusError        = "error"
	defaultRegion      = "us-west-2"
)

// AWS service constants
const (
	serviceConsole    = "console"
	serviceSageMaker  = "sagemaker"
	serviceBraket     = "braket"
	serviceCloudShell = "cloudshell"
)

// Connection embedding constants (kept for future use)
const (
	_ = "iframe" // embeddingModeIframe - reserved for future iframe embedding features
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
func (cm *ConnectionManager) CreateConnection(_ context.Context, connectionType ConnectionType, target string, options map[string]string) (*ConnectionConfig, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	var config *ConnectionConfig

	// Create connection configuration based on type
	switch connectionType {
	case ConnectionTypeSSH:
		config = cm.createSSHConnection(target)
	case ConnectionTypeDesktop:
		config = cm.createDesktopConnection(target)
	case ConnectionTypeWeb:
		config = cm.createWebConnection(target, options)
	case ConnectionTypeAWS:
		config = cm.createAWSConnection(target, options)
	default:
		return nil, fmt.Errorf("unsupported connection type: %s", connectionType)
	}

	// Store connection and start monitoring
	cm.connections[config.ID] = config
	go cm.monitorConnection(config.ID)

	return config, nil
}

// Helper functions for creating different connection types
func (cm *ConnectionManager) createSSHConnection(target string) *ConnectionConfig {
	return &ConnectionConfig{
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
}

func (cm *ConnectionManager) createDesktopConnection(target string) *ConnectionConfig {
	return &ConnectionConfig{
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
}

func (cm *ConnectionManager) createWebConnection(target string, options map[string]string) *ConnectionConfig {
	service := options["service"]
	if service == "" {
		service = "jupyter"
	}
	return &ConnectionConfig{
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
}

func (cm *ConnectionManager) createAWSConnection(_ string, options map[string]string) *ConnectionConfig {
	region := options["region"]
	if region == "" {
		region = defaultRegion
	}
	service := options["service"]
	if service == "" {
		service = serviceConsole
	}

	title := cm.getAWSConnectionTitle(service, region)

	return &ConnectionConfig{
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
}

func (cm *ConnectionManager) getAWSConnectionTitle(service, region string) string {
	switch service {
	case serviceBraket:
		return fmt.Sprintf("⚛️ Braket (%s)", region)
	case serviceSageMaker:
		return fmt.Sprintf("🤖 SageMaker (%s)", region)
	case serviceConsole:
		return fmt.Sprintf("🎛️ Console (%s)", region)
	case serviceCloudShell:
		return fmt.Sprintf("🖥️ CloudShell (%s)", region)
	default:
		return fmt.Sprintf("☁️ %s (%s)", service, region)
	}
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
	config.Status = statusDisconnected
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

	for range ticker.C {
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
			_ = cm.UpdateConnection(id, newStatus, "")
		}
	}
}

// Status check methods for different connection types
func (cm *ConnectionManager) checkSSHStatus(config *ConnectionConfig) string {
	// Health check SSH connection via WebSocket proxy endpoint
	if config.ProxyURL == "" {
		return statusError
	}

	// Check if the WebSocket endpoint is reachable
	// Convert WebSocket URL to HTTP for health check
	healthURL := strings.Replace(config.ProxyURL, "ws://", "http://", 1)
	healthURL = strings.Replace(healthURL, "wss://", "https://", 1)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		return statusDisconnected
	}
	defer func() { _ = resp.Body.Close() }()

	// Check if we get a reasonable response (could be upgrade required for WebSocket)
	if resp.StatusCode < 500 {
		return statusConnected
	}

	return statusDisconnected
}

func (cm *ConnectionManager) checkDesktopStatus(config *ConnectionConfig) string {
	// Health check DCV desktop connection
	if config.ProxyURL == "" {
		return statusError
	}

	// For DCV connections, check if the session endpoint is responding
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Allow redirects for DCV authentication flows
			return nil
		},
	}

	resp, err := client.Get(config.ProxyURL)
	if err != nil {
		return statusDisconnected
	}
	defer func() { _ = resp.Body.Close() }()

	// DCV sessions typically respond with 200 OK or redirect to login
	if resp.StatusCode == 200 || resp.StatusCode == 302 || resp.StatusCode == 401 {
		return statusConnected
	}

	return statusDisconnected
}

func (cm *ConnectionManager) checkWebStatus(config *ConnectionConfig) string {
	// Health check web interface connection
	if config.ProxyURL == "" {
		return statusError
	}

	// For web interfaces (Jupyter, RStudio, etc.), check if the service is responding
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Allow redirects for authentication flows
			return nil
		},
	}

	resp, err := client.Get(config.ProxyURL)
	if err != nil {
		return statusDisconnected
	}
	defer func() { _ = resp.Body.Close() }()

	// Web services typically respond with 200 OK, or redirects for login/auth
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return statusConnected
	}

	return statusDisconnected
}

func (cm *ConnectionManager) checkAWSStatus(config *ConnectionConfig) string {
	// Health check AWS service connection
	if config.ProxyURL == "" && config.AuthToken == "" {
		return statusError
	}

	// For AWS service connections, we can check a few things:
	// 1. If there's a ProxyURL, check if it's accessible
	// 2. If there's an AuthToken, we assume it's a federation token and check basic AWS access

	if config.ProxyURL != "" {
		// Check proxied AWS service endpoint
		client := &http.Client{
			Timeout: 10 * time.Second, // AWS services might take longer to respond
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				// AWS console redirects are common
				return nil
			},
		}

		resp, err := client.Get(config.ProxyURL)
		if err != nil {
			return statusDisconnected
		}
		defer func() { _ = resp.Body.Close() }()

		// AWS services typically respond with 200 OK or redirects for authentication
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return statusConnected
		}

		return statusDisconnected
	}

	// If no ProxyURL but has AuthToken, assume it's a direct federation connection
	if config.AuthToken != "" {
		// For federation tokens, we assume they're valid if they exist
		// A more sophisticated check would validate the token with AWS STS
		return statusConnected
	}

	// No way to verify connection
	return statusError
}
