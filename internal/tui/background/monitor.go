package background

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// MonitorConfig holds configuration options for the instance monitor
type MonitorConfig struct {
	// RefreshInterval is how frequently instance statuses are polled
	RefreshInterval time.Duration
	
	// IdleNotifyThreshold is how close to idle action threshold to notify (in minutes)
	IdleNotifyThreshold int
	
	// CostAlertThreshold is the daily cost threshold for alerts ($)
	CostAlertThreshold float64
}

// DefaultMonitorConfig returns the default monitor configuration
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		RefreshInterval:    1 * time.Minute,
		IdleNotifyThreshold: 5, // Notify 5 minutes before action
		CostAlertThreshold:  10.0, // $10/day
	}
}

// InstanceEvent represents an event from instance monitoring
type InstanceEvent struct {
	// Type is the event type
	Type string
	
	// Instance is the instance name
	Instance string
	
	// Message is the event description
	Message string
	
	// Timestamp is when the event occurred
	Timestamp time.Time
	
	// Level is the event severity (info, warning, error)
	Level string
	
	// Data contains additional event-specific data
	Data map[string]interface{}
}

// EventType constants for instance events
const (
	EventTypeStateChange = "state_change"
	EventTypeIdleWarning = "idle_warning"
	EventTypeCostAlert   = "cost_alert"
	EventTypeError       = "error"
)

// EventLevel constants for event severity
const (
	EventLevelInfo    = "info"
	EventLevelWarning = "warning"
	EventLevelError   = "error"
)

// InstanceMonitor monitors instance statuses and generates events
type InstanceMonitor struct {
	apiClient api.CloudWorkstationAPI
	config    MonitorConfig
	
	// Instance state tracking
	instanceStates map[string]string
	idleStates     map[string]*types.IdleStatus
	
	// Event handling
	eventCh     chan InstanceEvent
	subscribers []chan<- InstanceEvent
	
	// Control
	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	mutex      sync.RWMutex
	running    bool
}

// NewInstanceMonitor creates a new instance monitor
func NewInstanceMonitor(apiClient api.CloudWorkstationAPI, config MonitorConfig) *InstanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &InstanceMonitor{
		apiClient:      apiClient,
		config:         config,
		instanceStates: make(map[string]string),
		idleStates:     make(map[string]*types.IdleStatus),
		eventCh:        make(chan InstanceEvent, 100),
		ctx:            ctx,
		cancelFunc:     cancel,
	}
}

// Start begins instance monitoring
func (m *InstanceMonitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.running {
		return fmt.Errorf("monitor is already running")
	}
	
	// Initialize current state
	if err := m.refreshInstances(); err != nil {
		return fmt.Errorf("failed to initialize instance states: %w", err)
	}
	
	// Start event dispatcher
	m.wg.Add(1)
	go m.dispatchEvents()
	
	// Start monitoring loop
	m.wg.Add(1)
	go m.monitorLoop()
	
	m.running = true
	return nil
}

// Stop halts instance monitoring
func (m *InstanceMonitor) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.running {
		return
	}
	
	m.cancelFunc()
	m.wg.Wait()
	
	// Close event channel after all publishers are done
	close(m.eventCh)
	
	m.running = false
}

// Subscribe adds a subscriber channel to receive instance events
func (m *InstanceMonitor) Subscribe() (<-chan InstanceEvent, func()) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	ch := make(chan InstanceEvent, 10)
	m.subscribers = append(m.subscribers, ch)
	
	unsubscribe := func() {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		
		// Find and remove the subscriber
		for i, sub := range m.subscribers {
			if sub == ch {
				m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
				break
			}
		}
		
		close(ch)
	}
	
	return ch, unsubscribe
}

// refreshInstances fetches the current instance states
func (m *InstanceMonitor) refreshInstances() error {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()
	
	response, err := m.apiClient.ListInstances(ctx)
	if err != nil {
		return err
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Track which instances we've seen
	currentInstances := make(map[string]bool)
	totalCost := 0.0
	
	for _, instance := range response.Instances {
		currentInstances[instance.Name] = true
		totalCost += instance.EstimatedDailyCost
		
		// Check for state changes
		oldState, exists := m.instanceStates[instance.Name]
		if exists && oldState != instance.State {
			// State changed, generate event
			m.queueEvent(InstanceEvent{
				Type:      EventTypeStateChange,
				Instance:  instance.Name,
				Message:   fmt.Sprintf("Instance state changed from %s to %s", oldState, instance.State),
				Timestamp: time.Now(),
				Level:     EventLevelInfo,
				Data: map[string]interface{}{
					"old_state": oldState,
					"new_state": instance.State,
				},
			})
		}
		
		// Update state
		m.instanceStates[instance.Name] = instance.State
		
		// Check idle detection
		if instance.IdleDetection != nil && instance.IdleDetection.Enabled {
			oldIdle, hasOldIdle := m.idleStates[instance.Name]
			
			// Check for approaching idle action
			if instance.IdleDetection.ActionPending {
				timeUntilAction := time.Until(instance.IdleDetection.ActionSchedule)
				minutesUntilAction := int(timeUntilAction.Minutes())
				
				// Only warn if we're within the notification threshold and haven't warned before
				if minutesUntilAction <= m.config.IdleNotifyThreshold &&
					(!hasOldIdle || !oldIdle.ActionPending) {
					
					m.queueEvent(InstanceEvent{
						Type:      EventTypeIdleWarning,
						Instance:  instance.Name,
						Message:   fmt.Sprintf("Instance will %s in %d minutes due to inactivity", 
							instance.IdleDetection.Policy, minutesUntilAction),
						Timestamp: time.Now(),
						Level:     EventLevelWarning,
						Data: map[string]interface{}{
							"idle_time":   instance.IdleDetection.IdleTime,
							"threshold":   instance.IdleDetection.Threshold,
							"action":      instance.IdleDetection.Policy,
							"minutes_left": minutesUntilAction,
						},
					})
				}
			}
			
			// Store current idle state
			m.idleStates[instance.Name] = &types.IdleStatus{
				Instance:       instance.Name,
				Enabled:        instance.IdleDetection.Enabled,
				Policy:         instance.IdleDetection.Policy,
				IdleTime:       instance.IdleDetection.IdleTime,
				ActionSchedule: instance.IdleDetection.ActionSchedule,
				ActionPending:  instance.IdleDetection.ActionPending,
			}
		} else {
			// No idle detection or disabled
			delete(m.idleStates, instance.Name)
		}
	}
	
	// Check for deleted instances
	for name := range m.instanceStates {
		if !currentInstances[name] {
			// Instance no longer exists
			delete(m.instanceStates, name)
			delete(m.idleStates, name)
		}
	}
	
	// Check total cost threshold
	if totalCost > m.config.CostAlertThreshold {
		m.queueEvent(InstanceEvent{
			Type:      EventTypeCostAlert,
			Instance:  "",
			Message:   fmt.Sprintf("Daily cost ($%.2f) exceeds threshold ($%.2f)", 
				totalCost, m.config.CostAlertThreshold),
			Timestamp: time.Now(),
			Level:     EventLevelWarning,
			Data: map[string]interface{}{
				"total_cost":  totalCost,
				"threshold":   m.config.CostAlertThreshold,
				"instance_count": len(currentInstances),
			},
		})
	}
	
	return nil
}

// monitorLoop periodically checks instance statuses
func (m *InstanceMonitor) monitorLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.RefreshInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
			
		case <-ticker.C:
			if err := m.refreshInstances(); err != nil {
				m.queueEvent(InstanceEvent{
					Type:      EventTypeError,
					Instance:  "",
					Message:   fmt.Sprintf("Failed to refresh instances: %v", err),
					Timestamp: time.Now(),
					Level:     EventLevelError,
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
			}
		}
	}
}

// dispatchEvents dispatches events to subscribers
func (m *InstanceMonitor) dispatchEvents() {
	defer m.wg.Done()
	
	for {
		select {
		case <-m.ctx.Done():
			return
			
		case event, ok := <-m.eventCh:
			if !ok {
				return
			}
			
			// Deliver to all subscribers
			m.mutex.RLock()
			for _, subscriber := range m.subscribers {
				select {
				case subscriber <- event:
					// Successfully sent
				default:
					// Subscriber buffer is full, log but don't block
					fmt.Printf("Warning: Subscriber buffer full, dropped event: %s\n", event.Type)
				}
			}
			m.mutex.RUnlock()
		}
	}
}

// queueEvent adds an event to the event queue
func (m *InstanceMonitor) queueEvent(event InstanceEvent) {
	select {
	case m.eventCh <- event:
		// Event queued successfully
	default:
		// Event queue is full, log this but don't block
		fmt.Printf("Warning: Event queue full, dropped event: %s\n", event.Type)
	}
}