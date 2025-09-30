package main

import (
	"context"
	"fmt"
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
	// TODO: Implement SSH connection health check
	// This could ping the WebSocket endpoint or check if the terminal is responsive
	return config.Status // Return current status for now
}

func (cm *ConnectionManager) checkDesktopStatus(config *ConnectionConfig) string {
	// TODO: Implement DCV desktop connection health check
	// This could check if the DCV session is active
	return config.Status // Return current status for now
}

func (cm *ConnectionManager) checkWebStatus(config *ConnectionConfig) string {
	// TODO: Implement web interface health check
	// This could ping the proxied service endpoint
	return config.Status // Return current status for now
}

func (cm *ConnectionManager) checkAWSStatus(config *ConnectionConfig) string {
	// TODO: Implement AWS service connection health check
	// This could verify the federation token is still valid
	return config.Status // Return current status for now
}
