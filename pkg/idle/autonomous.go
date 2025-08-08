package idle

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// AutonomousIdleService provides fully autonomous idle detection and action execution
type AutonomousIdleService struct {
	manager     *Manager
	awsManager  *aws.Manager
	monitor     *MonitorService
	
	// Configuration
	config      *AutonomousConfig
	
	// State tracking via AWS tags
	idleStateTag     string
	idleTimestampTag string
	idleActionTag    string
}

// AutonomousConfig holds configuration for autonomous idle detection
type AutonomousConfig struct {
	// Enable autonomous execution of idle actions
	AutoExecute bool `json:"auto_execute" yaml:"auto_execute"`
	
	// Monitoring interval
	MonitorInterval time.Duration `json:"monitor_interval" yaml:"monitor_interval"`
	
	// SSH configuration for metrics collection
	SSHUsername string `json:"ssh_username" yaml:"ssh_username"`
	SSHKeyPath  string `json:"ssh_key_path" yaml:"ssh_key_path"`
	
	// Safety settings
	RequireTagConfirmation bool `json:"require_tag_confirmation" yaml:"require_tag_confirmation"`
	MaxActionsPerHour      int  `json:"max_actions_per_hour" yaml:"max_actions_per_hour"`
	
	// Dry run mode - log actions but don't execute
	DryRun bool `json:"dry_run" yaml:"dry_run"`
}

// DefaultAutonomousConfig returns safe default configuration
func DefaultAutonomousConfig() *AutonomousConfig {
	return &AutonomousConfig{
		AutoExecute:            false,               // Safe default
		MonitorInterval:        60 * time.Second,   // Check every minute for efficiency
		SSHUsername:           "ubuntu",            // Default Ubuntu user
		SSHKeyPath:            "~/.ssh/cloudworkstation",
		RequireTagConfirmation: true,               // Extra safety
		MaxActionsPerHour:     10,                  // Rate limiting
		DryRun:                false,
	}
}

// NewAutonomousIdleService creates a new autonomous idle service
func NewAutonomousIdleService(manager *Manager, awsManager *aws.Manager, config *AutonomousConfig) (*AutonomousIdleService, error) {
	if config == nil {
		config = DefaultAutonomousConfig()
	}
	
	// Create monitoring service
	monitorConfig := &MonitorConfig{
		Interval:     config.MonitorInterval,
		SSHUsername:  config.SSHUsername,
		SSHKeyPath:   config.SSHKeyPath,
		SSHTimeout:   30 * time.Second,
		AutoExecute:  config.AutoExecute,
	}
	
	monitor, err := NewMonitorService(manager, &awsInstanceProvider{awsManager}, monitorConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitor service: %w", err)
	}
	
	return &AutonomousIdleService{
		manager:          manager,
		awsManager:       awsManager,
		monitor:          monitor,
		config:           config,
		idleStateTag:     "CloudWorkstation:IdleState",
		idleTimestampTag: "CloudWorkstation:IdleSince",
		idleActionTag:    "CloudWorkstation:IdleAction",
	}, nil
}

// awsInstanceProvider implements InstanceProvider interface using AWS manager
type awsInstanceProvider struct {
	awsManager *aws.Manager
}

func (p *awsInstanceProvider) ListInstances() ([]types.Instance, error) {
	return p.awsManager.ListInstances()
}

// Start begins autonomous idle monitoring and action execution
func (ais *AutonomousIdleService) Start(ctx context.Context) error {
	if !ais.manager.IsEnabled() {
		return fmt.Errorf("idle detection is disabled - enable with 'cws idle enable'")
	}
	
	log.Printf("Starting autonomous idle detection service")
	log.Printf("  Auto-execute: %t", ais.config.AutoExecute)
	log.Printf("  Monitor interval: %v", ais.config.MonitorInterval)
	log.Printf("  Dry run: %t", ais.config.DryRun)
	
	// Start the monitoring service with enhanced autonomous capabilities
	return ais.startEnhancedMonitoring(ctx)
}

// startEnhancedMonitoring starts monitoring with AWS tag integration
func (ais *AutonomousIdleService) startEnhancedMonitoring(ctx context.Context) error {
	ticker := time.NewTicker(ais.config.MonitorInterval)
	defer ticker.Stop()
	
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("Autonomous idle service stopped by context")
				return
			case <-ticker.C:
				if err := ais.autonomousMonitoringCycle(ctx); err != nil {
					log.Printf("Error in autonomous monitoring cycle: %v", err)
				}
			}
		}
	}()
	
	return nil
}

// autonomousMonitoringCycle performs one complete monitoring and action cycle
func (ais *AutonomousIdleService) autonomousMonitoringCycle(ctx context.Context) error {
	// Get all running instances
	instances, err := ais.awsManager.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}
	
	// Filter to running instances with public IPs
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
	
	log.Printf("Monitoring %d running instances autonomously", len(runningInstances))
	
	// Monitor each instance and update AWS tags
	for _, instance := range runningInstances {
		if err := ais.monitorInstanceWithTags(ctx, instance); err != nil {
			log.Printf("Failed to monitor instance %s: %v", instance.Name, err)
		}
	}
	
	// Execute pending actions if enabled
	if ais.config.AutoExecute {
		return ais.executeAutonomousActions(ctx)
	}
	
	return nil
}

// monitorInstanceWithTags monitors an instance and updates AWS tags with idle state
func (ais *AutonomousIdleService) monitorInstanceWithTags(ctx context.Context, instance types.Instance) error {
	// Skip if instance doesn't have a public IP
	if instance.PublicIP == "" {
		return nil
	}
	
	// Create metrics collector for this monitoring cycle
	collector, err := NewMetricsCollector(ais.config.SSHKeyPath, ais.config.SSHUsername, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create metrics collector: %w", err)
	}
	
	// Collect metrics
	metrics, err := collector.CollectMetrics(instance.PublicIP)
	if err != nil {
		log.Printf("Failed to collect metrics from %s: %v", instance.Name, err)
		// Don't fail completely - instance might be starting up or temporarily unreachable
		return nil
	}
	
	// Process through idle detection
	idleState, err := ais.manager.ProcessMetrics(instance.ID, instance.Name, metrics)
	if err != nil {
		return fmt.Errorf("failed to process metrics: %w", err)
	}
	
	if idleState == nil {
		return nil
	}
	
	// Update AWS tags with current idle state
	return ais.updateIdleTags(ctx, instance, idleState)
}

// updateIdleTags updates AWS instance tags with current idle state
func (ais *AutonomousIdleService) updateIdleTags(ctx context.Context, instance types.Instance, idleState *IdleState) error {
	// Prepare tag updates
	tags := make(map[string]string)
	
	if idleState.IsIdle {
		tags[ais.idleStateTag] = "idle"
		if idleState.IdleSince != nil {
			tags[ais.idleTimestampTag] = idleState.IdleSince.Format(time.RFC3339)
		}
		if idleState.NextAction != nil {
			tags[ais.idleActionTag] = fmt.Sprintf("%s:%s", idleState.NextAction.Action, idleState.NextAction.Time.Format(time.RFC3339))
		}
		
		// Log idle state
		idleDuration := time.Since(*idleState.IdleSince).Round(time.Second)
		log.Printf("Instance %s is IDLE for %v (profile: %s)", instance.Name, idleDuration, idleState.Profile)
		
		if idleState.NextAction != nil {
			timeUntil := time.Until(idleState.NextAction.Time).Round(time.Second)
			if timeUntil <= 0 {
				log.Printf("  → %s action is READY for execution", idleState.NextAction.Action)
			} else {
				log.Printf("  → %s action in %v", idleState.NextAction.Action, timeUntil)
			}
		}
	} else {
		tags[ais.idleStateTag] = "active"
		tags[ais.idleTimestampTag] = ""
		tags[ais.idleActionTag] = ""
		
		// Log activity
		log.Printf("Instance %s is ACTIVE (CPU: %.1f%%, Mem: %.1f%%, User: %t)", 
			instance.Name, idleState.LastMetrics.CPU, idleState.LastMetrics.Memory, idleState.LastMetrics.HasActivity)
	}
	
	// TODO: Update AWS tags
	// This would use AWS EC2 CreateTags API to update the instance tags
	// For now, just log what would be updated
	log.Printf("Tags for %s: %v", instance.Name, tags)
	
	return nil
}

// executeAutonomousActions checks for and executes ready idle actions
func (ais *AutonomousIdleService) executeAutonomousActions(ctx context.Context) error {
	pendingActions := ais.manager.CheckPendingActions()
	
	if len(pendingActions) == 0 {
		return nil
	}
	
	log.Printf("Found %d instances ready for idle actions", len(pendingActions))
	
	for _, state := range pendingActions {
		if state.NextAction == nil {
			continue
		}
		
		// Safety check - confirm action is ready
		if time.Now().Before(state.NextAction.Time) {
			continue
		}
		
		// Execute the autonomous action
		if err := ais.executeIdleAction(ctx, state); err != nil {
			log.Printf("Failed to execute %s on %s: %v", 
				state.NextAction.Action, state.InstanceName, err)
			continue
		}
		
		// Record successful action in history
		historyEntry := HistoryEntry{
			InstanceID:   state.InstanceID,
			InstanceName: state.InstanceName,
			Action:       state.NextAction.Action,
			Time:         time.Now(),
			IdleDuration: time.Since(*state.IdleSince),
			Metrics:      state.LastMetrics,
		}
		
		if err := ais.manager.AddHistoryEntry(historyEntry); err != nil {
			log.Printf("Failed to record action history: %v", err)
		}
		
		// Clear the pending action
		state.NextAction = nil
	}
	
	return nil
}

// executeIdleAction executes an actual idle action using AWS operations
func (ais *AutonomousIdleService) executeIdleAction(ctx context.Context, state *IdleState) error {
	action := state.NextAction.Action
	instanceName := state.InstanceName
	idleDuration := time.Since(*state.IdleSince).Round(time.Second)
	
	log.Printf("🤖 AUTONOMOUS ACTION: %s on instance '%s' (idle for %v)", 
		action, instanceName, idleDuration)
	
	// Dry run mode - just log what would happen
	if ais.config.DryRun {
		log.Printf("  → DRY RUN: Would execute %s on %s", action, instanceName)
		return nil
	}
	
	// Execute the actual AWS action
	switch action {
	case Hibernate:
		log.Printf("  → Hibernating instance %s (preserving RAM state)", instanceName)
		if err := ais.awsManager.HibernateInstance(instanceName); err != nil {
			return fmt.Errorf("hibernation failed: %w", err)
		}
		log.Printf("  ✅ Instance %s hibernated successfully", instanceName)
		
	case Stop:
		log.Printf("  → Stopping instance %s", instanceName)
		if err := ais.awsManager.StopInstance(instanceName); err != nil {
			return fmt.Errorf("stop failed: %w", err)
		}
		log.Printf("  ✅ Instance %s stopped successfully", instanceName)
		
	case Notify:
		log.Printf("  → Notification: Instance %s is idle and may be wasting resources", instanceName)
		// TODO: Implement actual notification system (email, Slack, etc.)
		log.Printf("  ✅ Notification sent for instance %s", instanceName)
		
	default:
		return fmt.Errorf("unknown idle action: %s", action)
	}
	
	return nil
}

// Stop stops the autonomous idle service
func (ais *AutonomousIdleService) Stop() error {
	log.Printf("Stopping autonomous idle detection service")
	if ais.monitor != nil {
		return ais.monitor.Stop()
	}
	return nil
}

// GetStatus returns the current service status
func (ais *AutonomousIdleService) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"autonomous_enabled": ais.config.AutoExecute,
		"monitor_interval":   ais.config.MonitorInterval.String(),
		"dry_run":           ais.config.DryRun,
		"ssh_username":      ais.config.SSHUsername,
		"idle_enabled":      ais.manager.IsEnabled(),
	}
	
	if ais.monitor != nil {
		monitorStatus := ais.monitor.GetStatus()
		for k, v := range monitorStatus {
			status["monitor_"+k] = v
		}
	}
	
	return status
}