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
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// Instance represents a running CloudWorkstation instance
type Instance struct {
	Name   string  `json:"name"`
	State  string  `json:"state"`
	IP     string  `json:"ip,omitempty"`
	Cost   float64 `json:"hourly_rate,omitempty"`
	Region string  `json:"region,omitempty"`
}

// LaunchRequest represents a simple launch configuration
type LaunchRequest struct {
	Template string `json:"template"`
	Name     string `json:"name"`
	Size     string `json:"size,omitempty"`
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

	// TODO: Call daemon API to launch instance
	// For now, return success for UI development
	return nil
}

// StopInstance stops a running instance
func (s *CloudWorkstationService) StopInstance(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	// TODO: Call daemon API to stop instance
	return nil
}

// ConnectToInstance gets connection information for an instance
func (s *CloudWorkstationService) ConnectToInstance(ctx context.Context, name string) (map[string]interface{}, error) {
	if name == "" {
		return nil, fmt.Errorf("instance name is required")
	}

	// TODO: Call daemon API to get connection info
	// For now, return mock data for UI development
	return map[string]interface{}{
		"ssh":     fmt.Sprintf("ssh ec2-user@%s.compute.amazonaws.com", name),
		"jupyter": fmt.Sprintf("http://%s.compute.amazonaws.com:8888", name),
		"rstudio": fmt.Sprintf("http://%s.compute.amazonaws.com:8787", name),
	}, nil
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