// Package sleepwake provides automatic instance hibernation when the system sleeps
//
// This package monitors system power events (sleep/wake) and automatically hibernates
// running Prism instances to prevent cost waste when users forget to stop instances
// before closing their laptop or shutting down.
//
// Key Features:
// - Automatic hibernation on system sleep
// - Optional resume on wake
// - Configurable grace period and exclusions
// - Cross-platform support (macOS primary, Linux/Windows stubs)
package sleepwake

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Errors
var (
	// ErrPlatformNotSupported indicates the current platform doesn't support sleep/wake monitoring
	ErrPlatformNotSupported = errors.New("sleep/wake monitoring not supported on this platform")
)

// EventType represents the type of power event
type EventType int

const (
	// EventSleep indicates the system is about to sleep
	EventSleep EventType = iota
	// EventWake indicates the system has woken up
	EventWake
	// EventShutdown indicates the system is shutting down (future)
	EventShutdown
)

func (e EventType) String() string {
	switch e {
	case EventSleep:
		return "sleep"
	case EventWake:
		return "wake"
	case EventShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}

// Event represents a power management event
type Event struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // "iokit", "systemd", "win32", etc.
}

// HibernationMode determines which instances are hibernated on sleep
type HibernationMode string

const (
	// HibernationModeIdleOnly only hibernates instances detected as idle (RECOMMENDED DEFAULT)
	// Integrates with idle detection system to check CPU/memory/network/disk/GPU usage
	// Safe for long-running jobs - they won't be interrupted
	HibernationModeIdleOnly HibernationMode = "idle_only"

	// HibernationModeAll hibernates all instances except those in exclusion list
	// User must manually maintain exclusion list for active workloads
	// Higher risk of interrupting work
	HibernationModeAll HibernationMode = "all"

	// HibernationModeManualOnly disables automatic hibernation on sleep
	// User must manually hibernate instances via CLI
	// Most conservative option
	HibernationModeManualOnly HibernationMode = "manual_only"
)

// Config represents sleep/wake monitor configuration
type Config struct {
	// Enabled controls whether sleep/wake monitoring is active
	Enabled bool `json:"enabled" yaml:"enabled"`

	// HibernateOnSleep controls whether instances are hibernated when system sleeps
	HibernateOnSleep bool `json:"hibernate_on_sleep" yaml:"hibernate_on_sleep"`

	// HibernationMode determines which instances get hibernated
	// - "idle_only": Only hibernate idle instances (default, safest)
	// - "all": Hibernate all except excluded (manual exclusion list)
	// - "manual_only": No automatic hibernation
	HibernationMode HibernationMode `json:"hibernation_mode" yaml:"hibernation_mode"`

	// IdleCheckTimeout is how long to wait when checking if an instance is idle
	// Default: 10 seconds
	IdleCheckTimeout time.Duration `json:"idle_check_timeout" yaml:"idle_check_timeout"`

	// ResumeOnWake controls whether instances are resumed when system wakes
	// Default: false (user must manually resume for control)
	ResumeOnWake bool `json:"resume_on_wake" yaml:"resume_on_wake"`

	// GracePeriod is the time to wait after sleep event before hibernating
	// This allows cancellation if system wakes quickly (e.g., lid closed briefly)
	// Default: 30 seconds
	GracePeriod time.Duration `json:"grace_period" yaml:"grace_period"`

	// ExcludedInstances is a list of instance names that should not be auto-hibernated
	// Useful for critical instances that should never auto-hibernate regardless of idle status
	ExcludedInstances []string `json:"excluded_instances" yaml:"excluded_instances"`

	// StateFilePath is where the sleep/wake state is persisted
	// Default: ~/.prism/sleep_wake_state.json
	StateFilePath string `json:"state_file_path" yaml:"state_file_path"`
}

// DefaultConfig returns the default sleep/wake configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	stateFile := filepath.Join(homeDir, ".prism", "sleep_wake_state.json")

	return Config{
		Enabled:           true,
		HibernateOnSleep:  true,
		HibernationMode:   HibernationModeIdleOnly, // Safe default: only hibernate idle instances
		IdleCheckTimeout:  10 * time.Second,
		ResumeOnWake:      false, // Default: manual resume for safety
		GracePeriod:       30 * time.Second,
		ExcludedInstances: []string{},
		StateFilePath:     stateFile,
	}
}

// State represents the persisted sleep/wake state
type State struct {
	mu sync.RWMutex

	// LastSleepTime is when the system last went to sleep
	LastSleepTime time.Time `json:"last_sleep_time"`

	// LastWakeTime is when the system last woke up
	LastWakeTime time.Time `json:"last_wake_time"`

	// HibernatedInstances tracks which instances were auto-hibernated during sleep
	// Map of instance name -> hibernation time
	HibernatedInstances map[string]time.Time `json:"hibernated_instances"`

	// TotalSleepEvents is the count of sleep events processed
	TotalSleepEvents int64 `json:"total_sleep_events"`

	// TotalWakeEvents is the count of wake events processed
	TotalWakeEvents int64 `json:"total_wake_events"`

	// TotalHibernated is the count of instances hibernated due to sleep
	TotalHibernated int64 `json:"total_hibernated"`

	// TotalResumed is the count of instances resumed due to wake
	TotalResumed int64 `json:"total_resumed"`

	// StateFilePath is where this state is persisted
	StateFilePath string `json:"-"`
}

// NewState creates a new State with the given file path
func NewState(filePath string) *State {
	return &State{
		HibernatedInstances: make(map[string]time.Time),
		StateFilePath:       filePath,
	}
}

// Load loads the state from disk
func (s *State) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.StateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// State file doesn't exist yet, use defaults
			return nil
		}
		return err
	}

	return json.Unmarshal(data, s)
}

// Save persists the state to disk
func (s *State) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(s.StateFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.StateFilePath, data, 0600)
}

// AddHibernatedInstance records that an instance was hibernated
func (s *State) AddHibernatedInstance(instanceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.HibernatedInstances[instanceName] = time.Now()
	s.TotalHibernated++
}

// RemoveHibernatedInstance removes an instance from the hibernated list
func (s *State) RemoveHibernatedInstance(instanceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.HibernatedInstances, instanceName)
}

// GetHibernatedInstances returns a copy of the hibernated instances map
func (s *State) GetHibernatedInstances() map[string]time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]time.Time)
	for k, v := range s.HibernatedInstances {
		result[k] = v
	}
	return result
}

// ClearHibernatedInstances removes all hibernated instances from tracking
func (s *State) ClearHibernatedInstances() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.HibernatedInstances = make(map[string]time.Time)
}

// RecordSleepEvent records a sleep event
func (s *State) RecordSleepEvent() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastSleepTime = time.Now()
	s.TotalSleepEvents++
}

// RecordWakeEvent records a wake event
func (s *State) RecordWakeEvent() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastWakeTime = time.Now()
	s.TotalWakeEvents++
}

// RecordResumed increments the resumed counter
func (s *State) RecordResumed() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalResumed++
}

// GetStats returns a snapshot of the statistics
func (s *State) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Stats{
		LastSleepTime:    s.LastSleepTime,
		LastWakeTime:     s.LastWakeTime,
		TotalSleepEvents: s.TotalSleepEvents,
		TotalWakeEvents:  s.TotalWakeEvents,
		TotalHibernated:  s.TotalHibernated,
		TotalResumed:     s.TotalResumed,
		ActivelyTracked:  len(s.HibernatedInstances),
	}
}

// Stats represents sleep/wake monitoring statistics
type Stats struct {
	LastSleepTime    time.Time `json:"last_sleep_time"`
	LastWakeTime     time.Time `json:"last_wake_time"`
	TotalSleepEvents int64     `json:"total_sleep_events"`
	TotalWakeEvents  int64     `json:"total_wake_events"`
	TotalHibernated  int64     `json:"total_hibernated"`
	TotalResumed     int64     `json:"total_resumed"`
	ActivelyTracked  int       `json:"actively_tracked"` // Currently hibernated instances
}

// Status represents the current status of the sleep/wake monitor
type Status struct {
	Enabled             bool              `json:"enabled"`
	Running             bool              `json:"running"`
	Platform            string            `json:"platform"` // "darwin", "linux", "windows"
	HibernateOnSleep    bool              `json:"hibernate_on_sleep"`
	HibernationMode     HibernationMode   `json:"hibernation_mode"`   // "idle_only", "all", "manual_only"
	IdleCheckTimeout    string            `json:"idle_check_timeout"` // e.g., "10s"
	ResumeOnWake        bool              `json:"resume_on_wake"`
	GracePeriod         string            `json:"grace_period"`
	ExcludedInstances   []string          `json:"excluded_instances"`
	Stats               Stats             `json:"stats"`
	HibernatedInstances map[string]string `json:"hibernated_instances"` // name -> time string
}
