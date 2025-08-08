package idle

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// MonitorService provides autonomous background monitoring of instance idle states
type MonitorService struct {
	manager   *Manager
	collector *MetricsCollector
	
	// Configuration
	interval        time.Duration
	sshUsername     string
	sshKeyPath      string
	sshTimeout      time.Duration
	
	// State management
	running         bool
	stopChan        chan struct{}
	mutex           sync.RWMutex
	
	// Instance provider interface
	instanceProvider InstanceProvider
}

// InstanceProvider interface allows the monitor to get running instances
type InstanceProvider interface {
	ListInstances() ([]types.Instance, error)
}

// MonitorConfig holds configuration for the monitoring service
type MonitorConfig struct {
	// Monitoring interval (how often to check instances)
	Interval time.Duration `json:"interval" yaml:"interval"`
	
	// SSH configuration for connecting to instances
	SSHUsername string `json:"ssh_username" yaml:"ssh_username"`
	SSHKeyPath  string `json:"ssh_key_path" yaml:"ssh_key_path"`
	SSHTimeout  time.Duration `json:"ssh_timeout" yaml:"ssh_timeout"`
	
	// Auto-execution of idle actions
	AutoExecute bool `json:"auto_execute" yaml:"auto_execute"`
	
	// Maximum concurrent instance monitoring
	MaxConcurrent int `json:"max_concurrent" yaml:"max_concurrent"`
}

// DefaultMonitorConfig returns default monitoring configuration
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		Interval:      2 * time.Minute,  // Check every 2 minutes
		SSHUsername:   "ubuntu",         // Default for Ubuntu instances
		SSHKeyPath:    "~/.ssh/cloudworkstation", // Default SSH key
		SSHTimeout:    30 * time.Second,
		AutoExecute:   false,            // Safe default - require manual approval
		MaxConcurrent: 10,               // Monitor up to 10 instances simultaneously
	}
}

// NewMonitorService creates a new monitoring service
func NewMonitorService(manager *Manager, instanceProvider InstanceProvider, config *MonitorConfig) (*MonitorService, error) {
	if config == nil {
		config = DefaultMonitorConfig()
	}
	
	// Create metrics collector
	collector, err := NewMetricsCollector(config.SSHKeyPath, config.SSHUsername, config.SSHTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics collector: %w", err)
	}
	
	return &MonitorService{
		manager:          manager,
		collector:        collector,
		interval:         config.Interval,
		sshUsername:      config.SSHUsername,
		sshKeyPath:       config.SSHKeyPath,
		sshTimeout:       config.SSHTimeout,
		stopChan:         make(chan struct{}),
		instanceProvider: instanceProvider,
	}, nil
}

// Start begins the background monitoring service
func (ms *MonitorService) Start(ctx context.Context) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	if ms.running {
		return fmt.Errorf("monitoring service is already running")
	}
	
	if !ms.manager.IsEnabled() {
		return fmt.Errorf("idle detection is disabled")
	}
	
	ms.running = true
	
	log.Printf("Starting idle monitoring service (interval: %v)", ms.interval)
	
	// Start the monitoring loop
	go ms.monitorLoop(ctx)
	
	return nil
}

// Stop stops the background monitoring service
func (ms *MonitorService) Stop() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	if !ms.running {
		return fmt.Errorf("monitoring service is not running")
	}
	
	log.Printf("Stopping idle monitoring service...")
	
	close(ms.stopChan)
	ms.running = false
	
	return nil
}

// IsRunning returns whether the monitoring service is running
func (ms *MonitorService) IsRunning() bool {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	return ms.running
}

// monitorLoop is the main monitoring loop
func (ms *MonitorService) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(ms.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("Idle monitoring service stopped by context")
			return
		case <-ms.stopChan:
			log.Printf("Idle monitoring service stopped")
			return
		case <-ticker.C:
			if err := ms.checkAllInstances(ctx); err != nil {
				log.Printf("Error during instance monitoring: %v", err)
			}
		}
	}
}

// checkAllInstances monitors all running instances for idle state
func (ms *MonitorService) checkAllInstances(ctx context.Context) error {
	// Get all running instances
	instances, err := ms.instanceProvider.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}
	
	// Filter to only running instances
	var runningInstances []types.Instance
	for _, instance := range instances {
		if instance.State == "running" && instance.PublicIP != "" {
			runningInstances = append(runningInstances, instance)
		}
	}
	
	if len(runningInstances) == 0 {
		log.Printf("No running instances to monitor")
		return nil
	}
	
	log.Printf("Monitoring %d running instances for idle state", len(runningInstances))
	
	// Monitor instances concurrently (but limit concurrency)
	semaphore := make(chan struct{}, 5) // Max 5 concurrent
	var wg sync.WaitGroup
	
	for _, instance := range runningInstances {
		wg.Add(1)
		go func(inst types.Instance) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			if err := ms.monitorInstance(ctx, inst); err != nil {
				log.Printf("Failed to monitor instance %s: %v", inst.Name, err)
			}
		}(instance)
	}
	
	wg.Wait()
	
	// Check for pending actions and execute if enabled
	return ms.processPendingActions(ctx)
}

// monitorInstance monitors a single instance for idle state
func (ms *MonitorService) monitorInstance(ctx context.Context, instance types.Instance) error {
	log.Printf("Collecting metrics for instance %s (%s)", instance.Name, instance.PublicIP)
	
	// Collect metrics from the instance
	metrics, err := ms.collector.CollectMetrics(instance.PublicIP)
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}
	
	// Process metrics through idle detection system
	idleState, err := ms.manager.ProcessMetrics(instance.ID, instance.Name, metrics)
	if err != nil {
		return fmt.Errorf("failed to process metrics: %w", err)
	}
	
	if idleState == nil {
		return nil
	}
	
	// Log idle state changes
	if idleState.IsIdle {
		if idleState.IdleSince != nil {
			idleDuration := time.Since(*idleState.IdleSince)
			log.Printf("Instance %s is idle for %v (profile: %s)", 
				instance.Name, idleDuration.Round(time.Second), idleState.Profile)
			
			if idleState.NextAction != nil {
				timeUntilAction := time.Until(idleState.NextAction.Time)
				log.Printf("Next action: %s in %v", 
					idleState.NextAction.Action, timeUntilAction.Round(time.Second))
			}
		}
	} else {
		log.Printf("Instance %s is active (CPU: %.1f%%, Mem: %.1f%%, Activity: %t)", 
			instance.Name, metrics.CPU, metrics.Memory, metrics.HasActivity)
	}
	
	return nil
}

// processPendingActions checks for and executes pending idle actions
func (ms *MonitorService) processPendingActions(ctx context.Context) error {
	pendingActions := ms.manager.CheckPendingActions()
	
	if len(pendingActions) == 0 {
		return nil
	}
	
	log.Printf("Found %d pending idle actions", len(pendingActions))
	
	for _, state := range pendingActions {
		if state.NextAction == nil {
			continue
		}
		
		log.Printf("Executing %s action on instance %s (idle for %v)", 
			state.NextAction.Action,
			state.InstanceName,
			time.Since(*state.IdleSince).Round(time.Second))
		
		// Execute the action (this would integrate with the actual AWS operations)
		if err := ms.executeIdleAction(ctx, state); err != nil {
			log.Printf("Failed to execute %s on %s: %v", 
				state.NextAction.Action, state.InstanceName, err)
			continue
		}
		
		// Record history entry
		historyEntry := HistoryEntry{
			InstanceID:   state.InstanceID,
			InstanceName: state.InstanceName,
			Action:       state.NextAction.Action,
			Time:         time.Now(),
			IdleDuration: time.Since(*state.IdleSince),
			Metrics:      state.LastMetrics,
		}
		
		if err := ms.manager.AddHistoryEntry(historyEntry); err != nil {
			log.Printf("Failed to record history entry: %v", err)
		}
		
		// Clear the action from state
		state.NextAction = nil
	}
	
	return nil
}

// executeIdleAction executes an idle action (integration point for AWS operations)
func (ms *MonitorService) executeIdleAction(ctx context.Context, state *IdleState) error {
	// This would integrate with the AWS manager to actually execute actions
	// For now, this is a placeholder that would be implemented with:
	// - awsManager.HibernateInstance(state.InstanceName) for hibernate actions
	// - awsManager.StopInstance(state.InstanceName) for stop actions
	// - notification system for notify actions
	
	log.Printf("Executing %s action on instance %s", state.NextAction.Action, state.InstanceName)
	
	// TODO: Integrate with AWS manager
	switch state.NextAction.Action {
	case Hibernate:
		log.Printf("HIBERNATE: Instance %s would be hibernated", state.InstanceName)
		return nil // awsManager.HibernateInstance(state.InstanceName)
	case Stop:
		log.Printf("STOP: Instance %s would be stopped", state.InstanceName)
		return nil // awsManager.StopInstance(state.InstanceName)
	case Notify:
		log.Printf("NOTIFY: Notification sent for instance %s", state.InstanceName)
		return nil // Send notification
	default:
		return fmt.Errorf("unknown action: %s", state.NextAction.Action)
	}
}

// GetStatus returns the current monitoring service status
func (ms *MonitorService) GetStatus() map[string]interface{} {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	return map[string]interface{}{
		"running":         ms.running,
		"interval":        ms.interval.String(),
		"ssh_username":    ms.sshUsername,
		"ssh_timeout":     ms.sshTimeout.String(),
		"idle_enabled":    ms.manager.IsEnabled(),
	}
}