package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// Test constants
const (
	testEmbeddingModeIframe = "iframe"
	testServiceBraket       = "braket"
	testServiceConsole      = "console"
)

// MockPrismService for testing ConnectionManager
type MockPrismService struct {
	_ string // unused field for mock compatibility
}

func NewMockPrismService() *PrismService {
	return &PrismService{
		daemonURL: "http://localhost:8947",
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

func TestNewConnectionManager(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)

	if cm == nil {
		t.Fatal("NewConnectionManager returned nil")
	}

	if cm.connections == nil {
		t.Error("connections map not initialized")
	}

	if cm.callbacks == nil {
		t.Error("callbacks map not initialized")
	}

	if cm.service == nil {
		t.Error("service not properly assigned")
	}
}

func TestCreateConnection_SSH(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	config, err := cm.CreateConnection(ctx, ConnectionTypeSSH, "test-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	if config == nil {
		t.Fatal("CreateConnection returned nil config")
	}

	if config.Type != ConnectionTypeSSH {
		t.Errorf("Expected type %s, got %s", ConnectionTypeSSH, config.Type)
	}

	if config.InstanceName != "test-instance" {
		t.Errorf("Expected instance name 'test-instance', got %s", config.InstanceName)
	}

	if config.EmbeddingMode != "websocket" {
		t.Errorf("Expected embedding mode 'websocket', got %s", config.EmbeddingMode)
	}

	if config.Status != "connecting" {
		t.Errorf("Expected status 'connecting', got %s", config.Status)
	}

	expectedURL := "http://localhost:8947/ssh-proxy/test-instance"
	if config.ProxyURL != expectedURL {
		t.Errorf("Expected proxy URL %s, got %s", expectedURL, config.ProxyURL)
	}
}

func TestCreateConnection_Desktop(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	config, err := cm.CreateConnection(ctx, ConnectionTypeDesktop, "desktop-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	if config.Type != ConnectionTypeDesktop {
		t.Errorf("Expected type %s, got %s", ConnectionTypeDesktop, config.Type)
	}

	if config.EmbeddingMode != testEmbeddingModeIframe {
		t.Errorf("Expected embedding mode 'iframe', got %s", config.EmbeddingMode)
	}

	expectedURL := "http://localhost:8947/dcv-proxy/desktop-instance"
	if config.ProxyURL != expectedURL {
		t.Errorf("Expected proxy URL %s, got %s", expectedURL, config.ProxyURL)
	}
}

func TestCreateConnection_AWS_Braket(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	options := map[string]string{
		"service": testServiceBraket,
		"region":  "us-west-2",
	}

	config, err := cm.CreateConnection(ctx, ConnectionTypeAWS, testServiceBraket, options)
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	if config.Type != ConnectionTypeAWS {
		t.Errorf("Expected type %s, got %s", ConnectionTypeAWS, config.Type)
	}

	if config.AWSService != testServiceBraket {
		t.Errorf("Expected AWS service 'braket', got %s", config.AWSService)
	}

	if config.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got %s", config.Region)
	}

	expectedURL := "http://localhost:8947/aws-proxy/braket?region=us-west-2"
	if config.ProxyURL != expectedURL {
		t.Errorf("Expected proxy URL %s, got %s", expectedURL, config.ProxyURL)
	}

	expectedTitle := "⚛️ Braket (us-west-2)"
	if config.Title != expectedTitle {
		t.Errorf("Expected title %s, got %s", expectedTitle, config.Title)
	}
}

func TestCreateConnection_Web(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	options := map[string]string{
		"service": "jupyter",
	}

	config, err := cm.CreateConnection(ctx, ConnectionTypeWeb, "web-instance", options)
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	if config.Type != ConnectionTypeWeb {
		t.Errorf("Expected type %s, got %s", ConnectionTypeWeb, config.Type)
	}

	if config.EmbeddingMode != testEmbeddingModeIframe {
		t.Errorf("Expected embedding mode 'iframe', got %s", config.EmbeddingMode)
	}

	expectedURL := "http://localhost:8947/web-proxy/web-instance"
	if config.ProxyURL != expectedURL {
		t.Errorf("Expected proxy URL %s, got %s", expectedURL, config.ProxyURL)
	}

	expectedTitle := "🌐 jupyter: web-instance"
	if config.Title != expectedTitle {
		t.Errorf("Expected title %s, got %s", expectedTitle, config.Title)
	}
}

func TestGetConnection(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	// Create a connection first
	config, err := cm.CreateConnection(ctx, ConnectionTypeSSH, "test-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	// Test getting the connection
	retrievedConfig, exists := cm.GetConnection(config.ID)
	if !exists {
		t.Error("GetConnection returned false for existing connection")
	}

	if retrievedConfig == nil {
		t.Fatal("GetConnection returned nil config")
	}

	if retrievedConfig.ID != config.ID {
		t.Errorf("Expected ID %s, got %s", config.ID, retrievedConfig.ID)
	}

	// Test getting non-existent connection
	_, exists = cm.GetConnection("non-existent-id")
	if exists {
		t.Error("GetConnection returned true for non-existent connection")
	}
}

func TestGetAllConnections(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	// Initially should be empty
	connections := cm.GetAllConnections()
	if len(connections) != 0 {
		t.Errorf("Expected 0 connections, got %d", len(connections))
	}

	// Create some connections
	_, err := cm.CreateConnection(ctx, ConnectionTypeSSH, "ssh-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	_, err = cm.CreateConnection(ctx, ConnectionTypeDesktop, "desktop-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	// Should now have 2 connections
	connections = cm.GetAllConnections()
	if len(connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(connections))
	}
}

func TestUpdateConnection(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	// Create a connection
	config, err := cm.CreateConnection(ctx, ConnectionTypeSSH, "test-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	// Update the connection status
	err = cm.UpdateConnection(config.ID, "connected", "Successfully connected")
	if err != nil {
		t.Errorf("UpdateConnection failed: %v", err)
	}

	// Verify the update
	updatedConfig, exists := cm.GetConnection(config.ID)
	if !exists {
		t.Fatal("Connection not found after update")
	}

	if updatedConfig.Status != "connected" {
		t.Errorf("Expected status 'connected', got %s", updatedConfig.Status)
	}

	if updatedConfig.Metadata == nil {
		t.Error("Metadata should not be nil after update")
	}

	if updatedConfig.Metadata["status_message"] != "Successfully connected" {
		t.Errorf("Expected status message 'Successfully connected', got %v", updatedConfig.Metadata["status_message"])
	}

	// Test updating non-existent connection
	err = cm.UpdateConnection("non-existent-id", "connected", "")
	if err == nil {
		t.Error("Expected error when updating non-existent connection")
	}
}

func TestCloseConnection(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	// Create a connection
	config, err := cm.CreateConnection(ctx, ConnectionTypeSSH, "test-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	// Verify connection exists
	_, exists := cm.GetConnection(config.ID)
	if !exists {
		t.Fatal("Connection should exist before closing")
	}

	// Close the connection
	err = cm.CloseConnection(config.ID)
	if err != nil {
		t.Errorf("CloseConnection failed: %v", err)
	}

	// Verify connection is removed
	_, exists = cm.GetConnection(config.ID)
	if exists {
		t.Error("Connection should not exist after closing")
	}

	// Test closing non-existent connection
	err = cm.CloseConnection("non-existent-id")
	if err == nil {
		t.Error("Expected error when closing non-existent connection")
	}
}

func TestRegisterCallback(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	// Create a connection
	config, err := cm.CreateConnection(ctx, ConnectionTypeSSH, "test-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	// Register callback
	callbackCalled := false
	var callbackConfig *ConnectionConfig

	callback := func(cfg *ConnectionConfig) {
		callbackCalled = true
		callbackConfig = cfg
	}

	cm.RegisterCallback(config.ID, callback)

	// Update connection to trigger callback
	err = cm.UpdateConnection(config.ID, "connected", "Test message")
	if err != nil {
		t.Errorf("UpdateConnection failed: %v", err)
	}

	// Give callback a moment to be called (it's synchronous in this implementation)
	time.Sleep(10 * time.Millisecond)

	if !callbackCalled {
		t.Error("Callback was not called")
	}

	if callbackConfig == nil {
		t.Error("Callback received nil config")
	} else if callbackConfig.ID != config.ID {
		t.Errorf("Callback received wrong config ID: expected %s, got %s", config.ID, callbackConfig.ID)
	}
}

func TestCreateConnection_InvalidType(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	// Try to create connection with invalid type
	_, err := cm.CreateConnection(ctx, "invalid-type", "test-instance", map[string]string{})
	if err == nil {
		t.Error("Expected error for invalid connection type")
	}

	expectedError := "unsupported connection type: invalid-type"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAWSConnectionDefaults(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	// Test AWS connection with minimal options
	config, err := cm.CreateConnection(ctx, ConnectionTypeAWS, "aws", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	// Should default to console service
	if config.AWSService != testServiceConsole {
		t.Errorf("Expected default AWS service 'console', got %s", config.AWSService)
	}

	// Should default to us-west-2 region
	if config.Region != "us-west-2" {
		t.Errorf("Expected default region 'us-west-2', got %s", config.Region)
	}

	expectedURL := "http://localhost:8947/aws-proxy/console?region=us-west-2"
	if config.ProxyURL != expectedURL {
		t.Errorf("Expected proxy URL %s, got %s", expectedURL, config.ProxyURL)
	}
}

func TestWebConnectionDefaults(t *testing.T) {
	service := NewMockPrismService()
	cm := NewConnectionManager(service)
	ctx := context.Background()

	// Test web connection with minimal options
	config, err := cm.CreateConnection(ctx, ConnectionTypeWeb, "web-instance", map[string]string{})
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	// Should default to jupyter service
	expectedTitle := "🌐 jupyter: web-instance"
	if config.Title != expectedTitle {
		t.Errorf("Expected title %s, got %s", expectedTitle, config.Title)
	}
}
