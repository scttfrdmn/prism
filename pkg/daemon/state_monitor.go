package daemon

import (
	"log"
	"sync"
	"time"

	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/scttfrdmn/prism/pkg/state"
	"github.com/scttfrdmn/prism/pkg/types"
)

// StateMonitor monitors instance state changes in the background
// It polls AWS for instances in transitional states and updates local state
type StateMonitor struct {
	awsManager   *aws.Manager
	stateManager *state.Manager
	ticker       *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	running      bool
	// onTick is an optional hook invoked each tick after the transitional-state check. Phase 2
	// (#644) uses it to run the spend observer's accrual pass. Nil = no-op.
	onTick func()
}

// SetTickHook registers a callback run at the end of every monitor tick (e.g. the spend observer).
func (sm *StateMonitor) SetTickHook(fn func()) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onTick = fn
}

// NewStateMonitor creates a new state monitor
func NewStateMonitor(awsManager *aws.Manager, stateManager *state.Manager) *StateMonitor {
	return &StateMonitor{
		awsManager:   awsManager,
		stateManager: stateManager,
		stopCh:       make(chan struct{}),
	}
}

// Start begins background state monitoring
func (sm *StateMonitor) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return nil // Already running
	}

	sm.ticker = time.NewTicker(10 * time.Second)
	sm.running = true

	sm.wg.Add(1)
	go sm.monitorLoop()

	log.Printf("✅ State monitor started (10s polling interval)")
	return nil
}

// Stop gracefully stops the state monitor
func (sm *StateMonitor) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return // Not running
	}

	close(sm.stopCh)
	sm.ticker.Stop()
	sm.running = false

	// Wait for monitor loop to finish
	sm.wg.Wait()

	log.Printf("✅ State monitor stopped")
}

// monitorLoop runs the background monitoring loop
func (sm *StateMonitor) monitorLoop() {
	defer sm.wg.Done()

	for {
		select {
		case <-sm.ticker.C:
			sm.checkTransitionalInstances()
			sm.mu.Lock()
			hook := sm.onTick
			sm.mu.Unlock()
			if hook != nil {
				hook()
			}
		case <-sm.stopCh:
			return
		}
	}
}

// checkTransitionalInstances checks AWS for instances in transitional states
func (sm *StateMonitor) checkTransitionalInstances() {
	// Load current state
	state, err := sm.stateManager.LoadState()
	if err != nil {
		log.Printf("Warning: State monitor failed to load state: %v", err)
		return
	}

	// TTL safety valve (#588): if spored failed to enforce TTL, stop expired instances
	for name, inst := range state.Instances {
		if inst.ExpiresAt != nil && time.Now().After(*inst.ExpiresAt) && inst.State == "running" {
			log.Printf("⏰ TTL safety valve: instance %s expired at %s, stopping", name, inst.ExpiresAt.Format(time.RFC3339))
			go func(n string) {
				if err := sm.awsManager.StopInstance(n); err != nil {
					log.Printf("TTL safety valve: failed to stop %s: %v", n, err)
				}
			}(name)
		}
	}

	// Find instances in transitional states
	var transitionalInstances []types.Instance
	for _, inst := range state.Instances {
		if isTransitionalState(inst.State) {
			transitionalInstances = append(transitionalInstances, inst)
		}
	}

	if len(transitionalInstances) == 0 {
		return // No instances to monitor
	}

	log.Printf("🔍 State monitor checking %d instance(s) in transitional states", len(transitionalInstances))

	// Check each transitional instance
	for _, inst := range transitionalInstances {
		sm.checkInstance(inst)
	}
}

// checkInstance checks a single instance's state from AWS
func (sm *StateMonitor) checkInstance(inst types.Instance) {
	// Get current state from AWS
	awsInstance, err := sm.awsManager.GetInstance(inst.ID)
	if err != nil {
		// Instance might be terminated and gone from AWS
		if inst.State == "shutting-down" {
			// If it was shutting-down and now not found, it's terminated
			sm.handleTerminatedInstance(inst)
		}
		return
	}

	// Check if state changed
	if awsInstance.State != inst.State {
		log.Printf("✅ State changed: %s (%s → %s)", inst.Name, inst.State, awsInstance.State)

		// Preserve and extend state history. GetInstance returns a fresh
		// instance object built from AWS data, so StateHistory must be
		// copied from the cached record before we append the transition.
		awsInstance.StateHistory = append(inst.StateHistory, types.StateTransition{
			FromState: inst.State,
			ToState:   awsInstance.State,
			Timestamp: time.Now(),
			Reason:    "aws_observed",
			Initiator: "system",
		})

		// Update state
		if err := sm.stateManager.SaveInstance(*awsInstance); err != nil {
			log.Printf("Warning: Failed to update instance state: %v", err)
		}

		// Handle terminated instances
		if awsInstance.State == "terminated" {
			sm.handleTerminatedInstance(*awsInstance)
		}
	}
}

// handleTerminatedInstance removes a terminated instance after AWS confirms it's gone
func (sm *StateMonitor) handleTerminatedInstance(inst types.Instance) {
	// Wait for instance to disappear from AWS (eventual consistency)
	// Poll for up to 5 minutes with 10-second intervals
	maxAttempts := 30 // 5 minutes / 10 seconds
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := sm.awsManager.GetInstance(inst.ID)
		if err != nil {
			// Instance not found - it's gone from AWS
			log.Printf("✅ Terminated instance %s confirmed gone from AWS, removing from state", inst.Name)

			// Remove from local state
			if err := sm.stateManager.RemoveInstance(inst.Name); err != nil {
				log.Printf("Warning: Failed to remove terminated instance: %v", err)
			}
			return
		}

		// Still visible in AWS, wait before retrying
		if attempt < maxAttempts {
			time.Sleep(10 * time.Second)
		}
	}

	log.Printf("Warning: Terminated instance %s still visible in AWS after 5 minutes", inst.Name)
}

// isTransitionalState returns true if the instance state is transitional
func isTransitionalState(state string) bool {
	switch state {
	case "pending", "stopping", "shutting-down":
		return true
	default:
		return false
	}
}
