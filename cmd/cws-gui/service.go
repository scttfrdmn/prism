package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CloudWorkstationService provides the API bridge between frontend and daemon
type CloudWorkstationService struct {
	daemonURL string
	client    *http.Client
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
	Ports    []int   `json:"ports,omitempty"`   // Open ports
}

// LaunchRequest represents a simple launch configuration
type LaunchRequest struct {
	Template string `json:"template"`
	Name     string `json:"name"`
	Size     string `json:"size,omitempty"`
}

// ConnectionInfo represents connection information for an instance
type ConnectionInfo struct {
	InstanceName string   `json:"instanceName"`
	HasDesktop   bool     `json:"hasDesktop"`
	HasDisplay   bool     `json:"hasDisplay"`
	TemplateType string   `json:"templateType"`
	Services     []string `json:"services"`
	Ports        []int    `json:"ports"`
	Template     *Template `json:"template,omitempty"`
}

// SSHConnectionInfo represents SSH connection details
type SSHConnectionInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	KeyPath  string `json:"keyPath,omitempty"`
}

func NewCloudWorkstationService() *CloudWorkstationService {
	return &CloudWorkstationService{
		daemonURL: "http://localhost:8947",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetTemplates fetches available templates from daemon
func (s *CloudWorkstationService) GetTemplates(ctx context.Context) ([]Template, error) {
	resp, err := s.client.Get(s.daemonURL + "/api/v1/templates")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch templates: %w", err)
	}
	defer resp.Body.Close()

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
	resp, err := s.client.Get(s.daemonURL + "/api/v1/instances")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instances: %w", err)
	}
	defer resp.Body.Close()

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
	
	reqData := map[string]interface{}{
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
	
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call daemon API: %w", err)
	}
	defer resp.Body.Close()
	
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
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}
	
	return nil
}

// ConnectToInstance gets connection information for an instance
func (s *CloudWorkstationService) ConnectToInstance(ctx context.Context, name string) (map[string]interface{}, error) {
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
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned error status: %d", resp.StatusCode)
	}
	
	var connectionInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&connectionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode connection info: %w", err)
	}
	
	return connectionInfo, nil
}

// Helper functions for better UX
func getTemplateIcon(name string) string {
	nameLower := strings.ToLower(name)
	switch {
	case strings.Contains(nameLower, "python") || strings.Contains(nameLower, "ml"):
		return "🐍"
	case strings.Contains(nameLower, "r-") || strings.Contains(nameLower, "r "):
		return "📊"
	case strings.Contains(nameLower, "rocky") || strings.Contains(nameLower, "linux"):
		return "🖥️"
	case strings.Contains(nameLower, "ubuntu"):
		return "🐧"
	case strings.Contains(nameLower, "web") || strings.Contains(nameLower, "node"):
		return "🌐"
	default:
		return "⚙️"
	}
}

func getTemplateCategory(name string) string {
	nameLower := strings.ToLower(name)
	switch {
	case strings.Contains(nameLower, "python") || strings.Contains(nameLower, "ml"):
		return "Machine Learning"
	case strings.Contains(nameLower, "r-") || strings.Contains(nameLower, "r "):
		return "Data Science"
	case strings.Contains(nameLower, "web") || strings.Contains(nameLower, "node"):
		return "Web Development"
	case strings.Contains(nameLower, "rocky") || strings.Contains(nameLower, "linux") || strings.Contains(nameLower, "ubuntu"):
		return "Base Systems"
	default:
		return "General"
	}
}

// GetInstanceConnectionInfo gets connection information for intelligent detection
func (s *CloudWorkstationService) GetInstanceConnectionInfo(ctx context.Context, instanceName string) (*ConnectionInfo, error) {
	// Try to get detailed instance info from daemon first
	url := fmt.Sprintf("%s/api/v1/instances/%s/connection-info", s.daemonURL, instanceName)
	resp, err := s.client.Get(url)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		
		var connectionInfo ConnectionInfo
		if err := json.NewDecoder(resp.Body).Decode(&connectionInfo); err == nil {
			return &connectionInfo, nil
		}
	}
	
	// Fallback: analyze based on available instance and template information
	instances, err := s.GetInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}
	
	var targetInstance *Instance
	for i := range instances {
		if instances[i].Name == instanceName {
			targetInstance = &instances[i]
			break
		}
	}
	
	if targetInstance == nil {
		return nil, fmt.Errorf("instance %s not found", instanceName)
	}
	
	// Get templates for analysis
	templates, err := s.GetTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}
	
	var template *Template
	for i := range templates {
		if templates[i].Name == targetInstance.Template {
			template = &templates[i]
			break
		}
	}
	
	// Analyze template to determine connection characteristics
	hasDesktop := s.templateHasDesktop(template)
	hasDisplay := s.templateHasDisplay(template)
	templateType := ""
	if template != nil {
		templateType = template.Category
	}
	
	connectionInfo := &ConnectionInfo{
		InstanceName: instanceName,
		HasDesktop:   hasDesktop,
		HasDisplay:   hasDisplay,
		TemplateType: templateType,
		Services:     []string{}, // Would be populated from daemon in full implementation
		Ports:        []int{22},  // SSH always available
		Template:     template,
	}
	
	return connectionInfo, nil
}

// GetSSHConnectionInfo gets SSH connection details for an instance
func (s *CloudWorkstationService) GetSSHConnectionInfo(ctx context.Context, instanceName string) (*SSHConnectionInfo, error) {
	// Try to get SSH info from daemon first
	url := fmt.Sprintf("%s/api/v1/instances/%s/ssh-info", s.daemonURL, instanceName)
	resp, err := s.client.Get(url)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		
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

func (s *CloudWorkstationService) getDefaultUsername(instanceName string) string {
	// In a full implementation, this would be based on the template or AMI
	// For now, return common defaults
	return "ubuntu" // Most common default for AWS instances
}