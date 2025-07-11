package idle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const (
	// ConfigDirName is the name of the CloudWorkstation configuration directory.
	ConfigDirName = ".cloudworkstation"

	// IdleConfigFileName is the name of the idle configuration file.
	IdleConfigFileName = "idle.json"

	// IdleHistoryFileName is the name of the idle history file.
	IdleHistoryFileName = "idle_history.json"

	// IdleLogDirName is the name of the idle log directory.
	IdleLogDirName = "logs"

	// IdleActionsLogName is the name of the idle actions log file.
	IdleActionsLogName = "idle-actions.log"
)

// DefaultProfiles contains the default idle detection profiles.
var DefaultProfiles = map[string]Profile{
	"standard": {
		Name:            "standard",
		CPUThreshold:    10.0,
		MemoryThreshold: 30.0,
		NetworkThreshold: 50.0,
		DiskThreshold:    100.0,
		GPUThreshold:     5.0,
		IdleMinutes:      30,
		Action:           Stop,
		Notification:     true,
	},
	"batch": {
		Name:            "batch",
		CPUThreshold:    5.0,
		MemoryThreshold: 20.0,
		NetworkThreshold: 25.0,
		DiskThreshold:    50.0,
		GPUThreshold:     3.0,
		IdleMinutes:      60,
		Action:           Hibernate,
		Notification:     true,
	},
	"gpu": {
		Name:            "gpu",
		CPUThreshold:    5.0,
		MemoryThreshold: 20.0,
		NetworkThreshold: 50.0,
		DiskThreshold:    100.0,
		GPUThreshold:     3.0,
		IdleMinutes:      15,
		Action:           Stop,
		Notification:     true,
	},
	"data-intensive": {
		Name:            "data-intensive",
		CPUThreshold:    8.0,
		MemoryThreshold: 40.0,
		NetworkThreshold: 100.0,
		DiskThreshold:    200.0,
		GPUThreshold:     5.0,
		IdleMinutes:      45,
		Action:           Stop,
		Notification:     true,
	},
}

// DefaultDomainMappings contains the default domain-to-profile mappings.
var DefaultDomainMappings = map[string]string{
	"machine-learning": "gpu",
	"genomics":         "batch",
	"data-science":     "standard",
	"climate-science":  "batch",
	"visualization":    "gpu",
	"neuroimaging":     "gpu",
	"hpc":              "batch",
}

// Manager handles idle detection operations.
type Manager struct {
	// configPath is the path to the configuration file
	configPath string

	// historyPath is the path to the history file
	historyPath string

	// logDirPath is the path to the log directory
	logDirPath string

	// logPath is the path to the actions log file
	logPath string

	// config contains the idle detection configuration
	config *Config

	// history contains the idle detection history
	history *History

	// states maps instance IDs to idle states
	states map[string]*IdleState
}

// NewManager creates a new idle detection manager.
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ConfigDirName)
	if err := ensureDir(configDir); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	logDirPath := filepath.Join(configDir, IdleLogDirName)
	if err := ensureDir(logDirPath); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	configPath := filepath.Join(configDir, IdleConfigFileName)
	historyPath := filepath.Join(configDir, IdleHistoryFileName)
	logPath := filepath.Join(logDirPath, IdleActionsLogName)

	manager := &Manager{
		configPath:  configPath,
		historyPath: historyPath,
		logDirPath:  logDirPath,
		logPath:     logPath,
		config:      &Config{},
		history:     &History{},
		states:      make(map[string]*IdleState),
	}

	// Load existing configuration or create default
	if err := manager.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load idle configuration: %w", err)
	}

	// Load history if it exists
	if err := manager.loadHistory(); err != nil {
		return nil, fmt.Errorf("failed to load idle history: %w", err)
	}

	return manager, nil
}

// loadConfig loads the idle configuration from disk.
func (m *Manager) loadConfig() error {
	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Create default configuration
		m.config = &Config{
			Enabled:        true,
			DefaultProfile: "standard",
			Profiles:       DefaultProfiles,
			DomainMappings: DefaultDomainMappings,
			InstanceOverrides: make(map[string]InstanceOverride),
		}
		return m.saveConfig()
	}

	// Read config file
	data, err := ioutil.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// saveConfig saves the idle configuration to disk.
func (m *Manager) saveConfig() error {
	// Marshal JSON
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := ioutil.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadHistory loads the idle history from disk.
func (m *Manager) loadHistory() error {
	// Check if history file exists
	if _, err := os.Stat(m.historyPath); os.IsNotExist(err) {
		// Create empty history
		m.history = &History{
			Entries: []HistoryEntry{},
		}
		return nil
	}

	// Read history file
	data, err := ioutil.ReadFile(m.historyPath)
	if err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, m.history); err != nil {
		return fmt.Errorf("failed to parse history file: %w", err)
	}

	return nil
}

// saveHistory saves the idle history to disk.
func (m *Manager) saveHistory() error {
	// Marshal JSON
	data, err := json.MarshalIndent(m.history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Write history file
	if err := ioutil.WriteFile(m.historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// logAction logs an idle action to the actions log file.
func (m *Manager) logAction(entry *HistoryEntry) error {
	// Create log message
	logMsg := fmt.Sprintf("[%s] %s: Instance %s (%s) %s after being idle for %s\n",
		entry.Time.Format(time.RFC3339),
		entry.Action,
		entry.InstanceName,
		entry.InstanceID,
		entry.Action,
		entry.IdleDuration,
	)

	// Open log file in append mode
	f, err := os.OpenFile(m.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Write log message
	if _, err := f.WriteString(logMsg); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// Enable enables idle detection.
func (m *Manager) Enable() error {
	m.config.Enabled = true
	return m.saveConfig()
}

// Disable disables idle detection.
func (m *Manager) Disable() error {
	m.config.Enabled = false
	return m.saveConfig()
}

// IsEnabled returns whether idle detection is enabled.
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled
}

// GetProfiles returns the idle detection profiles.
func (m *Manager) GetProfiles() map[string]Profile {
	return m.config.Profiles
}

// GetProfile returns the profile with the given name.
func (m *Manager) GetProfile(name string) (Profile, error) {
	profile, ok := m.config.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found", name)
	}
	return profile, nil
}

// AddProfile adds a new profile.
func (m *Manager) AddProfile(profile Profile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	m.config.Profiles[profile.Name] = profile
	return m.saveConfig()
}

// RemoveProfile removes a profile.
func (m *Manager) RemoveProfile(name string) error {
	if name == "standard" || name == "batch" || name == "gpu" || name == "data-intensive" {
		return fmt.Errorf("cannot remove built-in profile %q", name)
	}

	if _, ok := m.config.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	delete(m.config.Profiles, name)
	return m.saveConfig()
}

// GetDefaultProfile returns the default profile.
func (m *Manager) GetDefaultProfile() (Profile, error) {
	return m.GetProfile(m.config.DefaultProfile)
}

// SetDefaultProfile sets the default profile.
func (m *Manager) SetDefaultProfile(name string) error {
	if _, ok := m.config.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	m.config.DefaultProfile = name
	return m.saveConfig()
}

// GetDomainMappings returns the domain-to-profile mappings.
func (m *Manager) GetDomainMappings() map[string]string {
	return m.config.DomainMappings
}

// SetDomainMapping sets a domain-to-profile mapping.
func (m *Manager) SetDomainMapping(domain, profile string) error {
	if _, ok := m.config.Profiles[profile]; !ok {
		return fmt.Errorf("profile %q not found", profile)
	}

	m.config.DomainMappings[domain] = profile
	return m.saveConfig()
}

// RemoveDomainMapping removes a domain-to-profile mapping.
func (m *Manager) RemoveDomainMapping(domain string) error {
	delete(m.config.DomainMappings, domain)
	return m.saveConfig()
}

// GetInstanceOverrides returns the instance overrides.
func (m *Manager) GetInstanceOverrides() map[string]InstanceOverride {
	return m.config.InstanceOverrides
}

// GetInstanceOverride returns the override for the given instance.
func (m *Manager) GetInstanceOverride(instance string) (InstanceOverride, bool) {
	override, ok := m.config.InstanceOverrides[instance]
	return override, ok
}

// SetInstanceOverride sets an instance override.
func (m *Manager) SetInstanceOverride(instance string, override InstanceOverride) error {
	if override.Profile != "" {
		if _, ok := m.config.Profiles[override.Profile]; !ok {
			return fmt.Errorf("profile %q not found", override.Profile)
		}
	}

	m.config.InstanceOverrides[instance] = override
	return m.saveConfig()
}

// RemoveInstanceOverride removes an instance override.
func (m *Manager) RemoveInstanceOverride(instance string) error {
	delete(m.config.InstanceOverrides, instance)
	return m.saveConfig()
}

// GetHistory returns the idle history.
func (m *Manager) GetHistory() []HistoryEntry {
	return m.history.Entries
}

// GetInstanceHistory returns the idle history for the given instance.
func (m *Manager) GetInstanceHistory(instanceID string) []HistoryEntry {
	var entries []HistoryEntry
	for _, entry := range m.history.Entries {
		if entry.InstanceID == instanceID {
			entries = append(entries, entry)
		}
	}
	return entries
}

// AddHistoryEntry adds a history entry.
func (m *Manager) AddHistoryEntry(entry HistoryEntry) error {
	m.history.Entries = append(m.history.Entries, entry)
	
	// Log the action
	if err := m.logAction(&entry); err != nil {
		return err
	}
	
	return m.saveHistory()
}

// ClearHistory clears the idle history.
func (m *Manager) ClearHistory() error {
	m.history.Entries = []HistoryEntry{}
	return m.saveHistory()
}

// GetIdleState returns the idle state for the given instance.
func (m *Manager) GetIdleState(instanceID string) *IdleState {
	state, ok := m.states[instanceID]
	if !ok {
		return nil
	}
	return state
}

// SetIdleState sets the idle state for the given instance.
func (m *Manager) SetIdleState(state *IdleState) {
	m.states[state.InstanceID] = state
}

// RemoveIdleState removes the idle state for the given instance.
func (m *Manager) RemoveIdleState(instanceID string) {
	delete(m.states, instanceID)
}

// ProcessMetrics processes the usage metrics for an instance.
func (m *Manager) ProcessMetrics(instanceID, instanceName string, metrics *UsageMetrics) (*IdleState, error) {
	if !m.config.Enabled {
		return nil, nil
	}

	// Get or create idle state
	state := m.GetIdleState(instanceID)
	if state == nil {
		state = &IdleState{
			InstanceID:   instanceID,
			InstanceName: instanceName,
			Profile:      m.config.DefaultProfile,
			IsIdle:       false,
			LastActivity: metrics.Timestamp,
			LastMetrics:  metrics,
		}
		m.SetIdleState(state)
	}

	// Update last metrics
	state.LastMetrics = metrics

	// Get profile for the instance
	var profile Profile
	var err error

	// Check for instance override
	if override, ok := m.GetInstanceOverride(instanceName); ok {
		// Use override profile
		profile, err = m.GetProfile(override.Profile)
		if err != nil {
			// Fall back to default profile
			profile, _ = m.GetDefaultProfile()
		}

		// Apply overrides
		if override.CPUThreshold != nil {
			profile.CPUThreshold = *override.CPUThreshold
		}
		if override.MemoryThreshold != nil {
			profile.MemoryThreshold = *override.MemoryThreshold
		}
		if override.NetworkThreshold != nil {
			profile.NetworkThreshold = *override.NetworkThreshold
		}
		if override.DiskThreshold != nil {
			profile.DiskThreshold = *override.DiskThreshold
		}
		if override.GPUThreshold != nil {
			profile.GPUThreshold = *override.GPUThreshold
		}
		if override.IdleMinutes != nil {
			profile.IdleMinutes = *override.IdleMinutes
		}
		if override.Action != nil {
			profile.Action = *override.Action
		}
		if override.Notification != nil {
			profile.Notification = *override.Notification
		}
	} else {
		// Use profile from state
		profile, err = m.GetProfile(state.Profile)
		if err != nil {
			// Fall back to default profile
			profile, _ = m.GetDefaultProfile()
		}
	}

	// Update profile in state
	state.Profile = profile.Name

	// Check if there's user activity
	if metrics.HasActivity {
		// Update last activity time
		state.LastActivity = metrics.Timestamp
		state.IsIdle = false
		state.IdleSince = nil
		state.NextAction = nil
		return state, nil
	}

	// Check if any metric is above threshold
	isIdle := true
	isIdle = isIdle && metrics.CPU < profile.CPUThreshold
	isIdle = isIdle && metrics.Memory < profile.MemoryThreshold
	isIdle = isIdle && metrics.Network < profile.NetworkThreshold
	isIdle = isIdle && metrics.Disk < profile.DiskThreshold
	if metrics.GPU != nil {
		isIdle = isIdle && *metrics.GPU < profile.GPUThreshold
	}

	// Check idle state changes
	if isIdle && !state.IsIdle {
		// Transition to idle
		state.IsIdle = true
		idleSince := metrics.Timestamp
		state.IdleSince = &idleSince
		
		// Schedule action
		actionTime := idleSince.Add(time.Duration(profile.IdleMinutes) * time.Minute)
		state.NextAction = &ScheduledAction{
			Action: profile.Action,
			Time:   actionTime,
		}
	} else if !isIdle && state.IsIdle {
		// Transition to active
		state.IsIdle = false
		state.IdleSince = nil
		state.NextAction = nil
		state.LastActivity = metrics.Timestamp
	}

	return state, nil
}

// CheckPendingActions checks for pending idle actions.
func (m *Manager) CheckPendingActions() []*IdleState {
	if !m.config.Enabled {
		return nil
	}

	now := time.Now()
	var pendingActions []*IdleState

	for _, state := range m.states {
		if state.NextAction != nil && now.After(state.NextAction.Time) {
			pendingActions = append(pendingActions, state)
		}
	}

	return pendingActions
}

// ensureDir ensures a directory exists, creating it if necessary.
func ensureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}