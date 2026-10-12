package sleepwake

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// InstanceManager defines the interface for hibernating/resuming instances
// This will be implemented by the daemon's API client or daemon directly
type InstanceManager interface {
	// ListInstances returns the names of all running instances
	ListInstances(ctx context.Context) ([]string, error)

	// IsInstanceIdle checks if an instance is currently idle (low resource usage)
	// This integrates with the existing idle detection system
	IsInstanceIdle(ctx context.Context, instanceName string) (bool, error)

	// HibernateInstance hibernates a single instance
	HibernateInstance(ctx context.Context, instanceName string) error

	// ResumeInstance resumes a hibernated instance
	ResumeInstance(ctx context.Context, instanceName string) error
}

// Monitor monitors system power events and manages instance hibernation
type Monitor struct {
	config Config
	state  *State
	mgr    InstanceManager

	// Platform-specific implementation
	platformMonitor platformMonitor

	// Control channels
	stopCh chan struct{}
	doneCh chan struct{}

	// Event notification
	eventCh chan Event
	mu      sync.RWMutex
	running bool
}

// platformMonitor is the platform-specific interface for sleep/wake detection
type platformMonitor interface {
	// Start begins monitoring for sleep/wake events
	Start(eventCh chan<- Event) error

	// Stop stops monitoring
	Stop() error

	// Platform returns the platform name
	Platform() string
}

// NewMonitor creates a new sleep/wake monitor
func NewMonitor(config Config, mgr InstanceManager) (*Monitor, error) {
	// Load or create state
	state := NewState(config.StateFilePath)
	if err := state.Load(); err != nil {
		log.Printf("Warning: Failed to load sleep/wake state: %v (using defaults)", err)
	}

	// Create platform-specific monitor
	pm, err := newPlatformMonitor()
	if err != nil {
		return nil, fmt.Errorf("failed to create platform monitor: %w", err)
	}

	return &Monitor{
		config:          config,
		state:           state,
		mgr:             mgr,
		platformMonitor: pm,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
		eventCh:         make(chan Event, 10),
	}, nil
}

// Start begins monitoring for sleep/wake events
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("monitor already running")
	}

	if !m.config.Enabled {
		log.Println("Sleep/wake monitoring is disabled")
		return nil
	}

	// Start platform-specific monitoring
	if err := m.platformMonitor.Start(m.eventCh); err != nil {
		return fmt.Errorf("failed to start platform monitor: %w", err)
	}

	// Start event processing goroutine
	go m.eventLoop()

	m.running = true
	log.Printf("Sleep/wake monitor started (platform: %s)", m.platformMonitor.Platform())
	return nil
}

// Stop stops the monitor
func (m *Monitor) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	// Signal stop
	close(m.stopCh)

	// Stop platform monitor
	if err := m.platformMonitor.Stop(); err != nil {
		log.Printf("Warning: Failed to stop platform monitor: %v", err)
	}

	// Wait for event loop to finish
	<-m.doneCh

	// Save final state
	if err := m.state.Save(); err != nil {
		log.Printf("Warning: Failed to save sleep/wake state: %v", err)
	}

	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	log.Println("Sleep/wake monitor stopped")
	return nil
}

// eventLoop processes power events
func (m *Monitor) eventLoop() {
	defer close(m.doneCh)

	for {
		select {
		case <-m.stopCh:
			return

		case event := <-m.eventCh:
			m.handleEvent(event)
		}
	}
}

// handleEvent processes a single power event
func (m *Monitor) handleEvent(event Event) {
	log.Printf("Power event: %s at %s", event.Type, event.Timestamp.Format(time.RFC3339))

	switch event.Type {
	case EventSleep:
		m.handleSleep(event)
	case EventWake:
		m.handleWake(event)
	case EventShutdown:
		m.handleShutdown(event)
	}

	// Save state after handling event
	if err := m.state.Save(); err != nil {
		log.Printf("Warning: Failed to save state after %s event: %v", event.Type, err)
	}
}

// handleSleep handles system sleep events
func (m *Monitor) handleSleep(event Event) {
	m.state.RecordSleepEvent()

	if !m.config.HibernateOnSleep {
		log.Println("Hibernate on sleep is disabled, skipping instance hibernation")
		return
	}

	// Wait for grace period
	if m.config.GracePeriod > 0 {
		log.Printf("Waiting %s grace period before hibernating instances...", m.config.GracePeriod)
		select {
		case <-time.After(m.config.GracePeriod):
			// Grace period elapsed, proceed with hibernation
		case <-m.stopCh:
			// Monitor stopped, abort hibernation
			log.Println("Monitor stopped during grace period, aborting hibernation")
			return
		}
	}

	// Get list of running instances
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	instances, err := m.mgr.ListInstances(ctx)
	if err != nil {
		log.Printf("Error: Failed to list instances: %v", err)
		return
	}

	if len(instances) == 0 {
		log.Println("No running instances to hibernate")
		return
	}

	// Determine which instances to hibernate based on mode
	toHibernate, skipped := m.selectInstancesForHibernation(ctx, instances)

	if len(toHibernate) == 0 {
		log.Printf("No instances to hibernate (mode: %s, total: %d, skipped: %d)",
			m.config.HibernationMode, len(instances), len(skipped))
		return
	}

	log.Printf("Hibernating %d instance(s) due to system sleep (mode: %s)...",
		len(toHibernate), m.config.HibernationMode)

	// Hibernate each selected instance
	hibernated := 0
	for _, instanceName := range toHibernate {
		if err := m.hibernateInstance(instanceName); err != nil {
			log.Printf("✗ Failed to hibernate %s: %v", instanceName, err)
			continue
		}

		m.state.AddHibernatedInstance(instanceName)
		log.Printf("✓ Hibernated: %s", instanceName)
		hibernated++
	}

	// Log summary
	log.Printf("Hibernation complete: %d hibernated, %d skipped", hibernated, len(skipped))
	for reason, count := range skipped {
		log.Printf("  Skipped %d instance(s): %s", count, reason)
	}
}

// handleWake handles system wake events
func (m *Monitor) handleWake(event Event) {
	m.state.RecordWakeEvent()

	log.Println("System woke up")

	// Get instances that were hibernated during sleep
	hibernated := m.state.GetHibernatedInstances()
	if len(hibernated) == 0 {
		log.Println("No instances were hibernated during sleep")
		return
	}

	log.Printf("Found %d instance(s) that were hibernated during sleep", len(hibernated))

	if !m.config.ResumeOnWake {
		log.Println("Resume on wake is disabled, instances remain hibernated")
		log.Println("Use 'prism workspace start <instance>' to manually resume instances")
		return
	}

	log.Printf("Resuming %d instance(s)...", len(hibernated))

	// Resume each hibernated instance
	for instanceName := range hibernated {
		if err := m.resumeInstance(instanceName); err != nil {
			log.Printf("Error: Failed to resume instance %s: %v", instanceName, err)
			continue
		}

		m.state.RemoveHibernatedInstance(instanceName)
		m.state.RecordResumed()
		log.Printf("✓ Resumed instance: %s", instanceName)
	}

	log.Printf("Successfully resumed %d instance(s)", len(hibernated))
}

// handleShutdown handles system shutdown events (future)
func (m *Monitor) handleShutdown(event Event) {
	log.Println("System shutdown detected")
	// Same as sleep for now
	m.handleSleep(event)
}

// hibernateInstance hibernates a single instance
func (m *Monitor) hibernateInstance(instanceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return m.mgr.HibernateInstance(ctx, instanceName)
}

// resumeInstance resumes a single instance
func (m *Monitor) resumeInstance(instanceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return m.mgr.ResumeInstance(ctx, instanceName)
}

// selectInstancesForHibernation determines which instances should be hibernated
// based on hibernation mode, exclusion list, and idle status
func (m *Monitor) selectInstancesForHibernation(ctx context.Context, instances []string) ([]string, map[string]int) {
	toHibernate := make([]string, 0, len(instances))
	skipped := make(map[string]int) // reason -> count

	// Build exclusion map
	excluded := make(map[string]bool)
	for _, name := range m.config.ExcludedInstances {
		excluded[name] = true
	}

	for _, instanceName := range instances {
		// Check manual exclusion first
		if excluded[instanceName] {
			skipped["manually excluded"]++
			log.Printf("⊘ %s: Skipped (manually excluded)", instanceName)
			continue
		}

		// Check hibernation mode
		switch m.config.HibernationMode {
		case HibernationModeManualOnly:
			// Never auto-hibernate in manual mode
			skipped["manual mode active"]++
			log.Printf("⊘ %s: Skipped (manual hibernation mode)", instanceName)
			continue

		case HibernationModeAll:
			// Hibernate all non-excluded instances
			toHibernate = append(toHibernate, instanceName)

		case HibernationModeIdleOnly:
			// Check if instance is idle
			idleCtx, cancel := context.WithTimeout(ctx, m.config.IdleCheckTimeout)
			isIdle, err := m.mgr.IsInstanceIdle(idleCtx, instanceName)
			cancel()

			if err != nil {
				// Error checking idle status - skip to be safe
				skipped["idle check failed"]++
				log.Printf("⊘ %s: Skipped (idle check error: %v)", instanceName, err)
				continue
			}

			if isIdle {
				// Instance is idle, safe to hibernate
				toHibernate = append(toHibernate, instanceName)
			} else {
				// Instance is active, don't interrupt
				skipped["active workload"]++
				log.Printf("⊘ %s: Skipped (active workload detected)", instanceName)
			}

		default:
			// Unknown mode - skip to be safe
			skipped["unknown mode"]++
			log.Printf("⊘ %s: Skipped (unknown hibernation mode: %s)", instanceName, m.config.HibernationMode)
		}
	}

	return toHibernate, skipped
}

// GetStatus returns the current monitor status
func (m *Monitor) GetStatus() Status {
	m.mu.RLock()
	running := m.running
	m.mu.RUnlock()

	stats := m.state.GetStats()
	hibernatedMap := m.state.GetHibernatedInstances()

	// Convert hibernated instances to string map
	hibernatedStr := make(map[string]string)
	for name, t := range hibernatedMap {
		hibernatedStr[name] = t.Format(time.RFC3339)
	}

	return Status{
		Enabled:             m.config.Enabled,
		Running:             running,
		Platform:            m.platformMonitor.Platform(),
		HibernateOnSleep:    m.config.HibernateOnSleep,
		HibernationMode:     m.config.HibernationMode,
		IdleCheckTimeout:    m.config.IdleCheckTimeout.String(),
		ResumeOnWake:        m.config.ResumeOnWake,
		GracePeriod:         m.config.GracePeriod.String(),
		ExcludedInstances:   m.config.ExcludedInstances,
		Stats:               stats,
		HibernatedInstances: hibernatedStr,
	}
}

// UpdateConfig updates the monitor configuration
func (m *Monitor) UpdateConfig(config Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config

	// If monitor is running and config was disabled, stop it
	if m.running && !config.Enabled {
		log.Println("Sleep/wake monitoring disabled via config update")
		// Stop will be called externally
	}

	return nil
}

// newPlatformMonitor creates a platform-specific monitor
// This function is implemented in platform-specific files (monitor_darwin.go, monitor_linux.go, monitor_windows.go)
