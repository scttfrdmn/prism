package types

import "time"

// Config manages application configuration
type Config struct {
	DefaultProfile string `json:"default_profile"`
	DefaultRegion  string `json:"default_region"`
	APIKey        string `json:"api_key,omitempty"`
	APIKeyCreated time.Time `json:"api_key_created,omitempty"`
}

// State manages the application state
type State struct {
	Instances  map[string]Instance  `json:"instances"`
	Volumes    map[string]EFSVolume `json:"volumes"`
	EBSVolumes map[string]EBSVolume `json:"ebs_volumes"`
	Config     Config               `json:"config"`
}

// DaemonStatus represents the status of the CloudWorkstation daemon
type DaemonStatus struct {
	// Version of the daemon
	Version string `json:"version"`
	
	// Status of the daemon (running, starting, stopping)
	Status string `json:"status"`
	
	// StartTime is when the daemon was started
	StartTime time.Time `json:"start_time"`
	
	// Uptime is the duration the daemon has been running
	Uptime string `json:"uptime,omitempty"`
	
	// ActiveOps is the number of currently active operations
	ActiveOps int `json:"active_ops"`
	
	// TotalRequests is the total number of requests processed
	TotalRequests int64 `json:"total_requests"`
	
	// RequestsPerMinute is the current request rate
	RequestsPerMinute float64 `json:"requests_per_minute,omitempty"`
	
	// AWSRegion is the current AWS region being used
	AWSRegion string `json:"aws_region"`
	
	// CurrentProfile is the active profile ID (if applicable)
	CurrentProfile string `json:"current_profile,omitempty"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	APIKey       string    `json:"api_key"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Message      string    `json:"message"`
}