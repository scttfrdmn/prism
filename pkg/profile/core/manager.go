package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager provides simplified profile management operations.
// This replaces the complex ManagerEnhanced with a clean, focused implementation.
type Manager struct {
	configPath string
	config     *ProfileConfig
	mutex      sync.RWMutex
}

// NewManager creates a new simplified profile manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	// Create CloudWorkstation directory if it doesn't exist
	cwsDir := filepath.Join(homeDir, ".cloudworkstation")
	if err := os.MkdirAll(cwsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	
	configPath := filepath.Join(cwsDir, ConfigFileName)
	
	manager := &Manager{
		configPath: configPath,
		config: &ProfileConfig{
			Profiles:  make(map[string]*Profile),
			Current:   "",
			Version:   DefaultConfigVersion,
			UpdatedAt: time.Now(),
		},
	}
	
	// Load existing configuration
	if err := manager.load(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load profile configuration: %w", err)
		}
	}
	
	return manager, nil
}

// List returns all configured profiles
func (m *Manager) List() []*Profile {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	profiles := make([]*Profile, 0, len(m.config.Profiles))
	for _, profile := range m.config.Profiles {
		profiles = append(profiles, profile)
	}
	
	return profiles
}

// Get retrieves a profile by name
func (m *Manager) Get(name string) (*Profile, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	profile, exists := m.config.Profiles[name]
	if !exists {
		return nil, &ProfileNotFoundError{Name: name}
	}
	
	return profile, nil
}

// Set creates or updates a profile
func (m *Manager) Set(name string, profile *Profile) error {
	// Validate profile
	if err := m.validateProfile(profile); err != nil {
		return err
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Ensure profile name matches
	profile.Name = name
	
	// Set creation time if new profile
	if _, exists := m.config.Profiles[name]; !exists {
		profile.CreatedAt = time.Now()
	}
	
	// Update last used time
	now := time.Now()
	profile.LastUsed = &now
	
	// Handle default profile logic
	if profile.Default {
		// Remove default flag from other profiles
		for _, p := range m.config.Profiles {
			p.Default = false
		}
		// Set this profile as current
		m.config.Current = name
	}
	
	// Store profile
	m.config.Profiles[name] = profile
	m.config.UpdatedAt = time.Now()
	
	// Save configuration
	return m.save()
}

// Delete removes a profile
func (m *Manager) Delete(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if profile exists
	profile, exists := m.config.Profiles[name]
	if !exists {
		return &ProfileNotFoundError{Name: name}
	}
	
	// Don't allow deleting the current profile without setting a new one
	if m.config.Current == name && len(m.config.Profiles) > 1 {
		return fmt.Errorf("cannot delete current profile '%s' - switch to another profile first", name)
	}
	
	// Delete the profile
	delete(m.config.Profiles, name)
	
	// Clear current if this was the current profile
	if m.config.Current == name {
		m.config.Current = ""
	}
	
	// If this was the default profile, make another profile default
	if profile.Default && len(m.config.Profiles) > 0 {
		for _, p := range m.config.Profiles {
			p.Default = true
			m.config.Current = p.Name
			break
		}
	}
	
	m.config.UpdatedAt = time.Now()
	
	// Save configuration
	return m.save()
}

// SetCurrent sets the current active profile
func (m *Manager) SetCurrent(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if profile exists
	profile, exists := m.config.Profiles[name]
	if !exists {
		return &ProfileNotFoundError{Name: name}
	}
	
	// Update last used time
	now := time.Now()
	profile.LastUsed = &now
	
	// Set as current
	m.config.Current = name
	m.config.UpdatedAt = time.Now()
	
	// Save configuration
	return m.save()
}

// GetCurrent returns the currently active profile
func (m *Manager) GetCurrent() (*Profile, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if m.config.Current == "" {
		return nil, &NoCurrentProfileError{}
	}
	
	profile, exists := m.config.Profiles[m.config.Current]
	if !exists {
		// Current profile was deleted - reset current
		m.mutex.RUnlock()
		m.mutex.Lock()
		m.config.Current = ""
		m.mutex.Unlock()
		m.mutex.RLock()
		return nil, &NoCurrentProfileError{}
	}
	
	return profile, nil
}

// GetCurrentName returns the name of the currently active profile
func (m *Manager) GetCurrentName() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.config.Current
}

// CreateDefault creates a default profile if none exist
func (m *Manager) CreateDefault(awsProfile, region string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Don't create if profiles already exist
	if len(m.config.Profiles) > 0 {
		return nil
	}
	
	// Create default profile
	profile := &Profile{
		Name:       DefaultProfileName,
		AWSProfile: awsProfile,
		Region:     region,
		Default:    true,
		CreatedAt:  time.Now(),
	}
	
	// Validate
	if err := m.validateProfile(profile); err != nil {
		return err
	}
	
	// Store
	m.config.Profiles[DefaultProfileName] = profile
	m.config.Current = DefaultProfileName
	m.config.UpdatedAt = time.Now()
	
	return m.save()
}

// validateProfile validates a profile configuration
func (m *Manager) validateProfile(profile *Profile) error {
	if profile.Name == "" {
		return &ValidationError{Field: "name", Message: "profile name is required"}
	}
	
	if strings.TrimSpace(profile.Name) != profile.Name {
		return &ValidationError{Field: "name", Message: "profile name cannot have leading/trailing whitespace"}
	}
	
	if profile.AWSProfile == "" {
		return &ValidationError{Field: "aws_profile", Message: "AWS profile is required"}
	}
	
	if profile.Region == "" {
		return &ValidationError{Field: "region", Message: "region is required"}
	}
	
	// Basic region format validation
	if len(profile.Region) < 9 || !strings.Contains(profile.Region, "-") {
		return &ValidationError{Field: "region", Message: "invalid region format"}
	}
	
	return nil
}

// load loads configuration from disk
func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}
	
	var config ProfileConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse profile configuration: %w", err)
	}
	
	// Validate config version
	if config.Version > DefaultConfigVersion {
		return fmt.Errorf("unsupported profile configuration version %d (max: %d)", config.Version, DefaultConfigVersion)
	}
	
	// Migrate if needed
	if config.Version < DefaultConfigVersion {
		m.migrateConfig(&config)
	}
	
	m.config = &config
	return nil
}

// save saves configuration to disk
func (m *Manager) save() error {
	m.config.Version = DefaultConfigVersion
	m.config.UpdatedAt = time.Now()
	
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile configuration: %w", err)
	}
	
	// Write atomically - write to temp file first, then rename
	tempPath := m.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile configuration: %w", err)
	}
	
	if err := os.Rename(tempPath, m.configPath); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("failed to save profile configuration: %w", err)
	}
	
	return nil
}

// migrateConfig migrates older config versions to current format
func (m *Manager) migrateConfig(config *ProfileConfig) {
	// For now, no migrations needed since this is the initial version
	// Future migrations would go here
	config.Version = DefaultConfigVersion
}

// GetConfigPath returns the path to the configuration file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// Stats returns statistics about the profile configuration
func (m *Manager) Stats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return map[string]interface{}{
		"total_profiles":  len(m.config.Profiles),
		"current_profile": m.config.Current,
		"config_version":  m.config.Version,
		"last_updated":    m.config.UpdatedAt,
		"config_path":     m.configPath,
	}
}