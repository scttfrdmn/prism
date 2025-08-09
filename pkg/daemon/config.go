package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents daemon configuration options
type Config struct {
	// Instance retention settings
	InstanceRetentionMinutes int `json:"instance_retention_minutes"` // 0 = indefinite, >0 = minutes to retain terminated instances
	
	// Server settings
	Port string `json:"port,omitempty"` // Server port (default: 8947)
	
	// Monitoring settings (future expansion)
	MonitoringIntervalSeconds int `json:"monitoring_interval_seconds,omitempty"` // Future: monitoring frequency
}

// DefaultConfig returns the default daemon configuration
func DefaultConfig() *Config {
	return &Config{
		InstanceRetentionMinutes:  5,    // Default: 5 minutes retention
		Port:                      "8947", // Default port
		MonitoringIntervalSeconds: 60,   // Future: default monitoring interval
	}
}

// LoadConfig loads daemon configuration from the standard location
func LoadConfig() (*Config, error) {
	configPath := getConfigPath()
	
	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read daemon config: %w", err)
	}
	
	// Parse config
	config := DefaultConfig() // Start with defaults
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse daemon config: %w", err)
	}
	
	return config, nil
}

// SaveConfig saves daemon configuration to the standard location
func SaveConfig(config *Config) error {
	configPath := getConfigPath()
	
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon config: %w", err)
	}
	
	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write daemon config: %w", err)
	}
	
	return nil
}

// GetRetentionDuration returns the retention duration from config
func (c *Config) GetRetentionDuration() time.Duration {
	if c.InstanceRetentionMinutes == 0 {
		// Return a very large duration for "indefinite" retention
		// This means terminated instances stay visible until AWS actually removes them
		return time.Hour * 24 * 365 * 10 // 10 years - effectively indefinite
	}
	return time.Duration(c.InstanceRetentionMinutes) * time.Minute
}

// getConfigPath returns the standard daemon configuration file path
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "daemon_config.json" // Fallback
	}
	return filepath.Join(homeDir, ".cloudworkstation", "daemon_config.json")
}