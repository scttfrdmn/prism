package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/usermgmt"
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".cloudworkstation")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	statePath := filepath.Join(stateDir, "state.json")
	userPath := filepath.Join(stateDir, "users.json")

	return &Manager{
		statePath: statePath,
		userPath:  userPath,
	}, nil
}

// LoadState loads the current state from disk
func (m *Manager) LoadState() (*types.State, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check if state file exists
	if _, err := os.Stat(m.statePath); os.IsNotExist(err) {
		// Return empty state if file doesn't exist
		return &types.State{
			Instances:  make(map[string]types.Instance),
			Volumes:    make(map[string]types.EFSVolume),
			EBSVolumes: make(map[string]types.EBSVolume),
			Config: types.Config{
				DefaultRegion: "us-east-1",
			},
		}, nil
	}

	data, err := os.ReadFile(m.statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state types.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Ensure maps are initialized (backward compatibility)
	if state.Instances == nil {
		state.Instances = make(map[string]types.Instance)
	}
	if state.Volumes == nil {
		state.Volumes = make(map[string]types.EFSVolume)
	}
	if state.EBSVolumes == nil {
		state.EBSVolumes = make(map[string]types.EBSVolume)
	}

	return &state, nil
}

// SaveState saves the current state to disk
func (m *Manager) SaveState(state *types.State) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first, then rename for atomicity
	tempPath := m.statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	if err := os.Rename(tempPath, m.statePath); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
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

// SaveVolume saves a single EFS volume to state
func (m *Manager) SaveVolume(volume types.EFSVolume) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	state.Volumes[volume.Name] = volume
	return m.SaveState(state)
}

// RemoveVolume removes an EFS volume from state
func (m *Manager) RemoveVolume(name string) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	delete(state.Volumes, name)
	return m.SaveState(state)
}

// SaveEBSVolume saves a single EBS volume to state
func (m *Manager) SaveEBSVolume(volume types.EBSVolume) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	state.EBSVolumes[volume.Name] = volume
	return m.SaveState(state)
}

// RemoveEBSVolume removes an EBS volume from state
func (m *Manager) RemoveEBSVolume(name string) error {
	state, err := m.LoadState()
	if err != nil {
		return err
	}

	delete(state.EBSVolumes, name)
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