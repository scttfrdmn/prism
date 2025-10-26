package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the CLI configuration
type Config struct {
	Daemon struct {
		URL string `json:"url"`
	} `json:"daemon"`
	AWS struct {
		Profile string `json:"profile"`
		Region  string `json:"region"`
	} `json:"aws"`
}

// LoadConfig loads the configuration from disk
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Set defaults
	config.Daemon.URL = "http://localhost:8947" // Default daemon URL (CWS on phone keypad)

	// Get config path
	configPath := getConfigPath()
	configFile := filepath.Join(configPath, "config.json")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create empty config file
		if err := saveConfig(config); err != nil {
			return nil, fmt.Errorf("failed to create config file: %w", err)
		}
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	if url := os.Getenv("CWSD_URL"); url != "" {
		config.Daemon.URL = url
	}
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		config.AWS.Profile = profile
	}
	if region := os.Getenv("AWS_REGION"); region != "" {
		config.AWS.Region = region
	} else if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		config.AWS.Region = region
	}

	return config, nil
}

// saveConfig saves the configuration to disk
func saveConfig(config *Config) error {
	configPath := getConfigPath()
	configFile := filepath.Join(configPath, "config.json")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigPath returns the path to the config directory
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fall back to current directory
		return ".prism"
	}
	return filepath.Join(homeDir, ".prism")
}
