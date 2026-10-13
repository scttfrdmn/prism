package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// Manager handles state persistence
type Manager struct {
	statePath string
	userPath  string
	mutex     sync.RWMutex
	userMutex sync.RWMutex
}

// NewManager creates a new state manager
func NewManager() (*Manager, error) {
	var stateDir string

	// Check for custom state directory via environment variable
	// This allows users to specify a custom location and enables test isolation
	if customDir := os.Getenv("PRISM_STATE_DIR"); customDir != "" {
		stateDir = customDir
	} else {
		// Default to ~/.prism for backward compatibility
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		stateDir = filepath.Join(homeDir, ".prism")
	}

	if err := os.MkdirAll(stateDir, 0700); err != nil { //nolint:gosec // G703: state path under PRISM_STATE_DIR or ~/.prism (local user config), not external input
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	statePath := filepath.Join(stateDir, "state.json")
	userPath := filepath.Join(stateDir, "users.json")

	return &Manager{
		statePath: statePath,
		userPath:  userPath,
	}, nil
}

// LoadState loads the current state from disk with corruption recovery
func (m *Manager) LoadState() (*types.State, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check if state file exists
	if _, err := os.Stat(m.statePath); os.IsNotExist(err) {
		// Return empty state if file doesn't exist
		return m.newEmptyState(), nil
	}

	// Try to load primary state file
	state, err := m.loadStateFile(m.statePath)
	if err == nil {
		return state, nil
	}

	// Primary state file is corrupted, try backups
	backups := []string{
		m.statePath + ".bak",
		m.statePath + ".bak1",
		m.statePath + ".bak2",
		m.statePath + ".bak3",
	}

	for _, backup := range backups {
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			continue
		}

		state, err := m.loadStateFile(backup)
		if err == nil {
			// Successfully recovered from backup
			// Restore backup to primary state file
			if err := m.restoreBackup(backup); err != nil {
				// Log error but continue with recovered state
				fmt.Fprintf(os.Stderr, "Warning: failed to restore backup: %v\n", err)
			}
			return state, nil
		}
	}

	// All backups failed, return empty state
	fmt.Fprintf(os.Stderr, "Warning: all state files corrupted, initializing empty state\n")
	return m.newEmptyState(), nil
}

// loadStateFile loads and validates a state file
func (m *Manager) loadStateFile(path string) (*types.State, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: state path under PRISM_STATE_DIR or ~/.prism (local user config), not external input
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state types.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Validate state structure
	if err := m.validateState(&state); err != nil {
		return nil, fmt.Errorf("invalid state structure: %w", err)
	}

	// Ensure maps are initialized
	if state.Instances == nil {
		state.Instances = make(map[string]types.Instance)
	}
	if state.StorageVolumes == nil {
		state.StorageVolumes = make(map[string]types.StorageVolume)
	}

	return &state, nil
}

// validateState performs basic validation on loaded state
func (m *Manager) validateState(state *types.State) error {
	// Check version (for future compatibility)
	if state.Version == "" {
		// Old state file without version, auto-upgrade
		state.Version = types.CurrentStateVersion
	}

	// Add more validation as needed
	return nil
}

// newEmptyState creates a new empty state with defaults
func (m *Manager) newEmptyState() *types.State {
	return &types.State{
		Version:        types.CurrentStateVersion,
		Instances:      make(map[string]types.Instance),
		StorageVolumes: make(map[string]types.StorageVolume),
		Config: types.Config{
			DefaultRegion: "us-east-1",
		},
	}
}

// restoreBackup atomically restores a backup to the primary state file
func (m *Manager) restoreBackup(backupPath string) error {
	data, err := os.ReadFile(backupPath) //nolint:gosec // G304: state path under PRISM_STATE_DIR or ~/.prism (local user config), not external input
	if err != nil {
		return err
	}

	tempPath := m.statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil { //nolint:gosec // G703: state path under PRISM_STATE_DIR or ~/.prism (local user config), not external input
		return err
	}

	return os.Rename(tempPath, m.statePath)
}

// SaveState saves the current state to disk with atomic writes and backup rotation
func (m *Manager) SaveState(state *types.State) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Set version
	state.Version = types.CurrentStateVersion

	// Rotate backups before writing new state
	if err := m.rotateBackups(); err != nil {
		// Log error but continue - backup rotation failure shouldn't prevent save
		fmt.Fprintf(os.Stderr, "Warning: backup rotation failed: %v\n", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first, then rename for atomicity
	tempPath := m.statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	if err := os.Rename(tempPath, m.statePath); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// rotateBackups rotates backup files before writing new state
// state.json -> state.json.bak
// state.json.bak -> state.json.bak1
// state.json.bak1 -> state.json.bak2
// state.json.bak2 -> state.json.bak3
// state.json.bak3 -> deleted
func (m *Manager) rotateBackups() error {
	// Check if primary state file exists
	if _, err := os.Stat(m.statePath); os.IsNotExist(err) {
		return nil // No state file to backup yet
	}

	// Rotate existing backups (from oldest to newest)
	backups := []struct {
		from string
		to   string
	}{
		{m.statePath + ".bak2", m.statePath + ".bak3"},
		{m.statePath + ".bak1", m.statePath + ".bak2"},
		{m.statePath + ".bak", m.statePath + ".bak1"},
		{m.statePath, m.statePath + ".bak"},
	}

	for _, backup := range backups {
		if _, err := os.Stat(backup.from); os.IsNotExist(err) {
			continue
		}

		// Remove destination if it exists (best effort: the WriteFile/Rename
		// below overwrites it and reports any real failure)
		_ = os.Remove(backup.to)

		// Copy file (preserve original for atomic write)
		if backup.from == m.statePath {
			data, err := os.ReadFile(backup.from)
			if err != nil {
				return fmt.Errorf("failed to read state for backup: %w", err)
			}
			if err := os.WriteFile(backup.to, data, 0600); err != nil { //nolint:gosec // G703: state path under PRISM_STATE_DIR or ~/.prism (local user config), not external input
				return fmt.Errorf("failed to write backup: %w", err)
			}
		} else {
			// Rename backup files
			if err := os.Rename(backup.from, backup.to); err != nil {
				return fmt.Errorf("failed to rotate backup: %w", err)
			}
		}
	}

	return nil
}

// SaveInstance saves a single instance to state
func (m *Manager) SaveInstance(instance types.Instance) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	state.Instances[instance.Name] = instance
	return m.SaveState(state)
}

// RemoveInstance removes an instance from state
func (m *Manager) RemoveInstance(name string) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	delete(state.Instances, name)
	return m.SaveState(state)
}

// SaveStorageVolume saves a single storage volume to state
func (m *Manager) SaveStorageVolume(volume types.StorageVolume) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	state.StorageVolumes[volume.Name] = volume
	return m.SaveState(state)
}

// RemoveStorageVolume removes a storage volume from state
func (m *Manager) RemoveStorageVolume(name string) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	delete(state.StorageVolumes, name)
	return m.SaveState(state)
}

// UpdateConfig updates the configuration
func (m *Manager) UpdateConfig(config types.Config) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	state.Config = config
	return m.SaveState(state)
}

// SaveAPIKey saves a new API key to the configuration
func (m *Manager) SaveAPIKey(apiKey string) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	state.Config.APIKey = apiKey
	state.Config.APIKeyCreated = time.Now()
	return m.SaveState(state)
}

// GetAPIKey retrieves the current API key
func (m *Manager) GetAPIKey() (string, time.Time, error) {
	state, err := m.LoadState()
	if err != nil {
		return "", time.Time{}, err
	}

	return state.Config.APIKey, state.Config.APIKeyCreated, nil
}

// ClearAPIKey removes the API key from the configuration
func (m *Manager) ClearAPIKey() error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	state.Config.APIKey = ""
	state.Config.APIKeyCreated = time.Time{}
	return m.SaveState(state)
}
