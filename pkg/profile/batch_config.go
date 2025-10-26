package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BatchInvitationConfig contains configuration settings for batch invitation operations
type BatchInvitationConfig struct {
	// General settings
	DefaultConcurrency int `json:"defaultConcurrency"`
	DefaultValidDays   int `json:"defaultValidDays"`

	// Security settings
	DefaultDeviceBound  bool `json:"defaultDeviceBound"`
	DefaultMaxDevices   int  `json:"defaultMaxDevices"`
	DefaultCanInvite    bool `json:"defaultCanInvite"`
	DefaultTransferable bool `json:"defaultTransferable"`

	// CSV settings
	DefaultHasHeader       bool   `json:"defaultHasHeader"`
	DefaultDelimiter       string `json:"defaultDelimiter"`
	IncludeEncodedData     bool   `json:"includeEncodedData"`
	DefaultOutputDirectory string `json:"defaultOutputDirectory"`

	// Admin settings
	RequireAdminAuth     bool   `json:"requireAdminAuth"`
	AdminInvitationToken string `json:"adminInvitationToken"`
	NotificationWebhook  string `json:"notificationWebhook"`
	AuditLoggingEnabled  bool   `json:"auditLoggingEnabled"`
	LogDirectory         string `json:"logDirectory"`

	// Performance settings
	BatchSizeLimit       int  `json:"batchSizeLimit"`
	EnableRateLimiting   bool `json:"enableRateLimiting"`
	MaxOperationsPerHour int  `json:"maxOperationsPerHour"`

	// Last updated
	LastUpdated time.Time `json:"lastUpdated"`
}

// DefaultBatchInvitationConfig returns the default configuration
func DefaultBatchInvitationConfig() *BatchInvitationConfig {
	return &BatchInvitationConfig{
		// General settings
		DefaultConcurrency: 5,
		DefaultValidDays:   30,

		// Security settings
		DefaultDeviceBound:  true,
		DefaultMaxDevices:   1,
		DefaultCanInvite:    false,
		DefaultTransferable: false,

		// CSV settings
		DefaultHasHeader:       true,
		DefaultDelimiter:       ",",
		IncludeEncodedData:     false,
		DefaultOutputDirectory: "",

		// Admin settings
		RequireAdminAuth:     false,
		AdminInvitationToken: "",
		NotificationWebhook:  "",
		AuditLoggingEnabled:  false,
		LogDirectory:         "",

		// Performance settings
		BatchSizeLimit:       1000,
		EnableRateLimiting:   false,
		MaxOperationsPerHour: 100,

		// Last updated
		LastUpdated: time.Now(),
	}
}

// BatchConfigManager manages batch invitation configuration
type BatchConfigManager struct {
	config     *BatchInvitationConfig
	configPath string
	mutex      sync.RWMutex
}

// NewBatchConfigManager creates a new batch config manager
func NewBatchConfigManager() (*BatchConfigManager, error) {
	// Determine config path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".prism")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "batch_config.json")

	// Create manager
	manager := &BatchConfigManager{
		configPath: configPath,
	}

	// Load config or create default
	if err := manager.Load(); err != nil {
		// If file doesn't exist, create default config
		if os.IsNotExist(err) {
			manager.config = DefaultBatchInvitationConfig()
			if saveErr := manager.Save(); saveErr != nil {
				return nil, fmt.Errorf("failed to save default config: %w", saveErr)
			}
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	return manager, nil
}

// Load loads the configuration from disk
func (m *BatchConfigManager) Load() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Read config file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	// Parse config
	config := &BatchInvitationConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	m.config = config
	return nil
}

// Save saves the configuration to disk
func (m *BatchConfigManager) Save() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Update last updated timestamp
	m.config.LastUpdated = time.Now()

	// Marshal config to JSON
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig returns a copy of the current configuration
func (m *BatchConfigManager) GetConfig() *BatchInvitationConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent race conditions
	configCopy := *m.config
	return &configCopy
}

// UpdateConfig updates the configuration with the provided values
func (m *BatchConfigManager) UpdateConfig(updates *BatchInvitationConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Apply updates
	if updates.DefaultConcurrency > 0 {
		m.config.DefaultConcurrency = updates.DefaultConcurrency
	}

	if updates.DefaultValidDays > 0 {
		m.config.DefaultValidDays = updates.DefaultValidDays
	}

	// Security settings - these are booleans, so we always update them
	m.config.DefaultDeviceBound = updates.DefaultDeviceBound
	m.config.DefaultCanInvite = updates.DefaultCanInvite
	m.config.DefaultTransferable = updates.DefaultTransferable

	if updates.DefaultMaxDevices > 0 {
		m.config.DefaultMaxDevices = updates.DefaultMaxDevices
	}

	// CSV settings
	m.config.DefaultHasHeader = updates.DefaultHasHeader

	if updates.DefaultDelimiter != "" {
		m.config.DefaultDelimiter = updates.DefaultDelimiter
	}

	m.config.IncludeEncodedData = updates.IncludeEncodedData

	if updates.DefaultOutputDirectory != "" {
		m.config.DefaultOutputDirectory = updates.DefaultOutputDirectory
	}

	// Admin settings
	m.config.RequireAdminAuth = updates.RequireAdminAuth

	if updates.AdminInvitationToken != "" {
		m.config.AdminInvitationToken = updates.AdminInvitationToken
	}

	if updates.NotificationWebhook != "" {
		m.config.NotificationWebhook = updates.NotificationWebhook
	}

	m.config.AuditLoggingEnabled = updates.AuditLoggingEnabled

	if updates.LogDirectory != "" {
		m.config.LogDirectory = updates.LogDirectory
	}

	// Performance settings
	if updates.BatchSizeLimit > 0 {
		m.config.BatchSizeLimit = updates.BatchSizeLimit
	}

	m.config.EnableRateLimiting = updates.EnableRateLimiting

	if updates.MaxOperationsPerHour > 0 {
		m.config.MaxOperationsPerHour = updates.MaxOperationsPerHour
	}

	// Save the updated config
	return m.Save()
}

// ResetToDefaults resets the configuration to default values
func (m *BatchConfigManager) ResetToDefaults() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.config = DefaultBatchInvitationConfig()
	return m.Save()
}

// ApplyConfigToBatchManager applies configuration to a BatchInvitationManager
func (m *BatchConfigManager) ApplyConfigToBatchManager(batchManager *BatchInvitationManager) {
	config := m.GetConfig()

	// Apply relevant settings to the batch manager
	batchManager.defaultConcurrency = config.DefaultConcurrency
	batchManager.defaultValidDays = config.DefaultValidDays
	batchManager.defaultDeviceBound = config.DefaultDeviceBound
	batchManager.defaultMaxDevices = config.DefaultMaxDevices
	batchManager.defaultCanInvite = config.DefaultCanInvite
	batchManager.defaultTransferable = config.DefaultTransferable
}

// ApplyConfigToDeviceManager applies configuration to a BatchDeviceManager
func (m *BatchConfigManager) ApplyConfigToDeviceManager(deviceManager *BatchDeviceManager) {
	config := m.GetConfig()

	// Apply relevant settings to the device manager
	deviceManager.defaultConcurrency = config.DefaultConcurrency
}
