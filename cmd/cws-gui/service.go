package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CloudWorkstationService provides the API bridge between frontend and daemon
type CloudWorkstationService struct {
	daemonURL         string
	client            *http.Client
	connectionManager *ConnectionManager
	apiKey            string // API key for daemon authentication
}

// Template represents a CloudWorkstation template
type Template struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Category       string `json:"category,omitempty"`
	Icon           string `json:"icon,omitempty"`
	ConnectionType string `json:"connection_type,omitempty"` // "dcv", "ssh", "auto"
}

// Instance represents a running CloudWorkstation instance
type Instance struct {
	Name     string  `json:"name"`
	State    string  `json:"state"`
	IP       string  `json:"ip,omitempty"`
	Cost     float64 `json:"hourly_rate,omitempty"`
	Region   string  `json:"region,omitempty"`
	Template string  `json:"template,omitempty"` // Template used to launch instance
	Ports    []int   `json:"ports,omitempty"`    // Open ports
}

// LaunchRequest represents a simple launch configuration
type LaunchRequest struct {
	Template string `json:"template"`
	Name     string `json:"name"`
	Size     string `json:"size,omitempty"`
}

// ConnectionInfo represents connection information for an instance
type ConnectionInfo struct {
	InstanceName string    `json:"instanceName"`
	HasDesktop   bool      `json:"hasDesktop"`
	HasDisplay   bool      `json:"hasDisplay"`
	TemplateType string    `json:"templateType"`
	Services     []string  `json:"services"`
	Ports        []int     `json:"ports"`
	Template     *Template `json:"template,omitempty"`
}

// SSHConnectionInfo represents SSH connection details
type SSHConnectionInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	KeyPath  string `json:"keyPath,omitempty"`
}

// ResearchUser represents a research user with persistent identity
type ResearchUser struct {
	Username      string    `json:"username"`
	FullName      string    `json:"full_name"`
	Email         string    `json:"email"`
	UID           uint32    `json:"uid"`
	GID           uint32    `json:"gid"`
	HomeDirectory string    `json:"home_directory"`
	Shell         string    `json:"shell"`
	SudoAccess    bool      `json:"sudo_access"`
	DockerAccess  bool      `json:"docker_access"`
	SSHPublicKeys []string  `json:"ssh_public_keys"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateResearchUserRequest represents a request to create a new research user
type CreateResearchUserRequest struct {
	Username string `json:"username"`
}

// ResearchUserSSHKeyRequest represents a request to manage SSH keys
type ResearchUserSSHKeyRequest struct {
	Username string `json:"username"`
	KeyType  string `json:"key_type,omitempty"` // "ed25519" or "rsa"
}

func NewCloudWorkstationService() *CloudWorkstationService {
	service := &CloudWorkstationService{
		daemonURL: "http://localhost:8947",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: loadAPIKeyFromState(), // Load API key for authentication
	}

	// Initialize connection manager
	service.connectionManager = NewConnectionManager(service)

	return service
}

// GetTemplates fetches available templates from daemon
func (s *CloudWorkstationService) GetTemplates(ctx context.Context) ([]Template, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.daemonURL+"/api/v1/templates", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key authentication
	s.addAPIKeyHeader(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch templates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var templates []Template
	if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to decode templates: %w", err)
	}

	// Add icons and categories based on template type for better UX
	for i := range templates {
		templates[i].Icon = getTemplateIcon(templates[i].Name)
		templates[i].Category = getTemplateCategory(templates[i].Name)
	}

	return templates, nil
}

// GetInstances fetches running instances from daemon
func (s *CloudWorkstationService) GetInstances(ctx context.Context) ([]Instance, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.daemonURL+"/api/v1/instances", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key authentication
	s.addAPIKeyHeader(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instances: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var instances []Instance
	if err := json.NewDecoder(resp.Body).Decode(&instances); err != nil {
		return nil, fmt.Errorf("failed to decode instances: %w", err)
	}

	return instances, nil
}

// LaunchInstance creates a new instance with simple configuration
func (s *CloudWorkstationService) LaunchInstance(ctx context.Context, req LaunchRequest) error {
	// Validate input
	if req.Template == "" || req.Name == "" {
		return fmt.Errorf("template and name are required")
	}

	// Set default size if not specified
	if req.Size == "" {
		req.Size = "M" // Default to medium size
	}

	// Call daemon API to launch instance
	launchURL := fmt.Sprintf("%s/api/v1/instances", s.daemonURL)

	reqData := map[string]any{
		"template": req.Template,
		"name":     req.Name,
		"size":     req.Size,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to marshal launch request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", launchURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Add API key authentication
	s.addAPIKeyHeader(httpReq)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call daemon API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}

	return nil
}

// StopInstance stops a running instance
func (s *CloudWorkstationService) StopInstance(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	// Call daemon API to stop instance
	stopURL := fmt.Sprintf("%s/api/v1/instances/%s/stop", s.daemonURL, name)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", stopURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call daemon API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}

	return nil
}

// ConnectToInstance gets connection information for an instance
func (s *CloudWorkstationService) ConnectToInstance(ctx context.Context, name string) (map[string]any, error) {
	if name == "" {
		return nil, fmt.Errorf("instance name is required")
	}

	// Call daemon API to get connection info
	connURL := fmt.Sprintf("%s/api/v1/instances/%s/connection", s.daemonURL, name)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", connURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call daemon API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}

	var connectionInfo map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&connectionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode connection info: %w", err)
	}

	return connectionInfo, nil
}

// Helper functions for better UX
func getTemplateIcon(name string) string {
	nameLower := strings.ToLower(name)
	if isPythonML(nameLower) {
		return "🐍"
	}
	if isDataScience(nameLower) {
		return "📊"
	}
	if isLinux(nameLower) {
		return "🖥️"
	}
	if strings.Contains(nameLower, "ubuntu") {
		return "🐧"
	}
	if isWeb(nameLower) {
		return "🌐"
	}
	return "⚙️"
}

// Helper functions for template classification
func isPythonML(name string) bool {
	return strings.Contains(name, "python") || strings.Contains(name, "ml")
}

func isDataScience(name string) bool {
	return strings.Contains(name, "r-") || strings.Contains(name, "r ")
}

func isLinux(name string) bool {
	return strings.Contains(name, "rocky") || strings.Contains(name, "linux")
}

func isWeb(name string) bool {
	return strings.Contains(name, "web") || strings.Contains(name, "node")
}

func getTemplateCategory(name string) string {
	nameLower := strings.ToLower(name)
	if isPythonML(nameLower) {
		return "Machine Learning"
	}
	if isDataScience(nameLower) {
		return "Data Science"
	}
	if isWeb(nameLower) {
		return "Web Development"
	}
	if isBaseSystem(nameLower) {
		return "Base Systems"
	}
	return "General"
}

func isBaseSystem(name string) bool {
	return isLinux(name) || strings.Contains(name, "ubuntu")
}

// GetInstanceConnectionInfo gets connection information for intelligent detection
func (s *CloudWorkstationService) GetInstanceConnectionInfo(ctx context.Context, instanceName string) (*ConnectionInfo, error) {
	// Try to get detailed instance info from daemon first
	if connectionInfo, err := s.getConnectionInfoFromDaemon(instanceName); err == nil {
		return connectionInfo, nil
	}

	// Fallback: analyze based on available instance and template information
	return s.buildConnectionInfoFromTemplate(ctx, instanceName)
}

// getConnectionInfoFromDaemon attempts to get connection info directly from daemon
func (s *CloudWorkstationService) getConnectionInfoFromDaemon(instanceName string) (*ConnectionInfo, error) {
	url := fmt.Sprintf("%s/api/v1/instances/%s/connection-info", s.daemonURL, instanceName)
	resp, err := s.client.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return nil, fmt.Errorf("daemon connection info not available")
	}
	defer func() { _ = resp.Body.Close() }()

	var connectionInfo ConnectionInfo
	if err := json.NewDecoder(resp.Body).Decode(&connectionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode connection info: %w", err)
	}

	return &connectionInfo, nil
}

// buildConnectionInfoFromTemplate builds connection info by analyzing instance and template
func (s *CloudWorkstationService) buildConnectionInfoFromTemplate(ctx context.Context, instanceName string) (*ConnectionInfo, error) {
	targetInstance, err := s.findInstanceByName(ctx, instanceName)
	if err != nil {
		return nil, err
	}

	template, err := s.findTemplateByName(ctx, targetInstance.Template)
	if err != nil {
		return nil, err
	}

	return s.createConnectionInfoFromTemplate(instanceName, template), nil
}

// findInstanceByName finds an instance by name
func (s *CloudWorkstationService) findInstanceByName(ctx context.Context, instanceName string) (*Instance, error) {
	instances, err := s.GetInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}

	for i := range instances {
		if instances[i].Name == instanceName {
			return &instances[i], nil
		}
	}

	return nil, fmt.Errorf("instance %s not found", instanceName)
}

// findTemplateByName finds a template by name
func (s *CloudWorkstationService) findTemplateByName(ctx context.Context, templateName string) (*Template, error) {
	templates, err := s.GetTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}

	for i := range templates {
		if templates[i].Name == templateName {
			return &templates[i], nil
		}
	}

	return nil, nil // Template not found is not an error
}

// createConnectionInfoFromTemplate creates connection info based on template analysis
func (s *CloudWorkstationService) createConnectionInfoFromTemplate(instanceName string, template *Template) *ConnectionInfo {
	hasDesktop := s.templateHasDesktop(template)
	hasDisplay := s.templateHasDisplay(template)
	templateType := ""
	if template != nil {
		templateType = template.Category
	}

	return &ConnectionInfo{
		InstanceName: instanceName,
		HasDesktop:   hasDesktop,
		HasDisplay:   hasDisplay,
		TemplateType: templateType,
		Services:     []string{}, // Would be populated from daemon in full implementation
		Ports:        []int{22},  // SSH always available
		Template:     template,
	}
}

// GetSSHConnectionInfo gets SSH connection details for an instance
func (s *CloudWorkstationService) GetSSHConnectionInfo(ctx context.Context, instanceName string) (*SSHConnectionInfo, error) {
	// Try to get SSH info from daemon first
	url := fmt.Sprintf("%s/api/v1/instances/%s/ssh-info", s.daemonURL, instanceName)
	resp, err := s.client.Get(url)
	if err == nil && resp.StatusCode == 200 {
		defer func() { _ = resp.Body.Close() }()

		var sshInfo SSHConnectionInfo
		if err := json.NewDecoder(resp.Body).Decode(&sshInfo); err == nil {
			return &sshInfo, nil
		}
	}

	// Fallback: use instance IP if available
	instances, err := s.GetInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}

	for _, instance := range instances {
		if instance.Name == instanceName {
			if instance.IP == "" {
				return nil, fmt.Errorf("instance %s has no IP address", instanceName)
			}

			return &SSHConnectionInfo{
				Host:     instance.IP,
				Port:     22,
				Username: s.getDefaultUsername(instanceName),
				KeyPath:  "", // Would use SSH agent or prompt for password
			}, nil
		}
	}

	return nil, fmt.Errorf("instance %s not found", instanceName)
}

// Helper functions for connection detection

func (s *CloudWorkstationService) templateHasDesktop(template *Template) bool {
	if template == nil {
		return false
	}

	text := strings.ToLower(template.Name + " " + template.Description)
	desktopKeywords := []string{"desktop", "workstation", "gui", "gnome", "kde", "xfce", "mate"}

	for _, keyword := range desktopKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

func (s *CloudWorkstationService) templateHasDisplay(template *Template) bool {
	if template == nil {
		return false
	}

	text := strings.ToLower(template.Name + " " + template.Description)
	displayKeywords := []string{"vnc", "x11", "display", "rdp", "visualization", "viz"}

	for _, keyword := range displayKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

func (s *CloudWorkstationService) getDefaultUsername(_ string) string {
	// In a full implementation, this would be based on the template or AMI
	// For now, return common defaults
	return "ubuntu" // Most common default for AWS instances
}

// ConfigureAutoStart configures automatic startup for the GUI application
func (s *CloudWorkstationService) ConfigureAutoStart(_ context.Context, enable bool) error { //nolint:unparam // Error return reserved for future validation
	// This calls the same auto-start configuration that the CLI uses
	// The actual implementation is handled by the autostart.go file

	// For now, we'll simulate success and let the JavaScript handle the message
	// In a full implementation, this would call the configureAutoStart function
	// from autostart.go or execute the cws-gui binary with the appropriate flags

	// Both branches return nil as this is a placeholder implementation
	// that would execute different commands in a complete version
	if enable {
		// Would execute: cws-gui -autostart
		return nil
	}
	// Would execute: cws-gui -remove-autostart
	return nil
}

// RestartDaemon restarts the CloudWorkstation daemon
func (s *CloudWorkstationService) RestartDaemon(_ context.Context) error {
	// This would restart the daemon service
	// For now, return a not implemented error
	return fmt.Errorf("daemon restart functionality not yet implemented in GUI service")
}

// GetResearchUsers fetches all research users from daemon
func (s *CloudWorkstationService) GetResearchUsers(_ context.Context) ([]ResearchUser, error) {
	resp, err := s.client.Get(s.daemonURL + "/api/v1/research-users")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch research users: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}

	var users []ResearchUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode research users: %w", err)
	}

	return users, nil
}

// CreateResearchUser creates a new research user
func (s *CloudWorkstationService) CreateResearchUser(ctx context.Context, req CreateResearchUserRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal create request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.daemonURL+"/api/v1/research-users", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call daemon API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}

	return nil
}

// DeleteResearchUser deletes a research user
func (s *CloudWorkstationService) DeleteResearchUser(ctx context.Context, username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", s.daemonURL+"/api/v1/research-users/"+username, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call daemon API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}

	return nil
}

// GenerateResearchUserSSHKey generates SSH key pair for research user
func (s *CloudWorkstationService) GenerateResearchUserSSHKey(ctx context.Context, req ResearchUserSSHKeyRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}

	// Set default key type if not specified
	if req.KeyType == "" {
		req.KeyType = "ed25519"
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal SSH key request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/research-users/%s/ssh-key", s.daemonURL, req.Username)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call daemon API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}

	return nil
}

// GetResearchUserStatus gets detailed status for a research user
func (s *CloudWorkstationService) GetResearchUserStatus(_ context.Context, username string) (map[string]any, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	url := fmt.Sprintf("%s/api/v1/research-users/%s/status", s.daemonURL, username)
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch research user status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}

	var status map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode research user status: %w", err)
	}

	return status, nil
}

// Connection Management Methods (Phase 2: Tab Management System)

// CreateConnection creates a new embedded connection
func (s *CloudWorkstationService) CreateConnection(ctx context.Context, connectionType string, target string, options map[string]string) (*ConnectionConfig, error) {
	if s.connectionManager == nil {
		return nil, fmt.Errorf("connection manager not initialized")
	}

	var connType ConnectionType
	switch connectionType {
	case "ssh":
		connType = ConnectionTypeSSH
	case "desktop":
		connType = ConnectionTypeDesktop
	case "web":
		connType = ConnectionTypeWeb
	case "aws":
		connType = ConnectionTypeAWS
	default:
		return nil, fmt.Errorf("unsupported connection type: %s", connectionType)
	}

	return s.connectionManager.CreateConnection(ctx, connType, target, options)
}

// GetActiveConnections returns all active connections
func (s *CloudWorkstationService) GetActiveConnections() []*ConnectionConfig {
	if s.connectionManager == nil {
		return []*ConnectionConfig{}
	}

	return s.connectionManager.GetAllConnections()
}

// GetConnection retrieves a specific connection by ID
func (s *CloudWorkstationService) GetConnection(id string) (*ConnectionConfig, error) {
	if s.connectionManager == nil {
		return nil, fmt.Errorf("connection manager not initialized")
	}

	config, exists := s.connectionManager.GetConnection(id)
	if !exists {
		return nil, fmt.Errorf("connection %s not found", id)
	}

	return config, nil
}

// UpdateConnectionStatus updates a connection's status
func (s *CloudWorkstationService) UpdateConnectionStatus(id string, status string, message string) error {
	if s.connectionManager == nil {
		return fmt.Errorf("connection manager not initialized")
	}

	return s.connectionManager.UpdateConnection(id, status, message)
}

// CloseConnection closes a connection and cleans up resources
func (s *CloudWorkstationService) CloseConnection(id string) error {
	if s.connectionManager == nil {
		return fmt.Errorf("connection manager not initialized")
	}

	return s.connectionManager.CloseConnection(id)
}

// RegisterConnectionCallback registers a callback for connection status changes
func (s *CloudWorkstationService) RegisterConnectionCallback(id string, callback func(*ConnectionConfig)) {
	if s.connectionManager != nil {
		s.connectionManager.RegisterCallback(id, callback)
	}
}

// addAPIKeyHeader adds API key authentication header if available
func (s *CloudWorkstationService) addAPIKeyHeader(req *http.Request) {
	if s.apiKey != "" {
		req.Header.Set("X-API-Key", s.apiKey)
	}
}

// loadAPIKeyFromState attempts to load the API key from daemon state
func loadAPIKeyFromState() string {
	// Try to load daemon state to get API key
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "" // No API key available
	}

	stateFile := filepath.Join(homeDir, ".cloudworkstation", "state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "" // No state file or can't read it
	}

	// Parse state to extract API key
	var state struct {
		Config struct {
			APIKey string `json:"api_key"`
		} `json:"config"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return "" // Invalid state format
	}

	return state.Config.APIKey
}
