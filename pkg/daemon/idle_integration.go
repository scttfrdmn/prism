package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// IdleMetrics represents the metrics data received from instance agents.
type IdleMetrics struct {
	InstanceID  string      `json:"instance_id"`
	InstanceName string     `json:"instance_name"`
	Metrics     idle.UsageMetrics `json:"metrics"`
}

// RegisterIdleHandlers registers the idle detection API handlers.
func (s *Server) RegisterIdleHandlers() {
	// Create idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		fmt.Printf("Failed to initialize idle manager: %v\n", err)
		return
	}

	// Store idle manager in server context
	s.idleManager = idleManager

	// Register API endpoints
	s.mux.HandleFunc("/api/v1/idle/metrics", s.handleIdleMetrics)
	s.mux.HandleFunc("/api/v1/idle/status", s.handleIdleStatus)
	s.mux.HandleFunc("/api/v1/idle/config", s.handleIdleConfig)
	s.mux.HandleFunc("/api/v1/idle/enable", s.handleIdleEnable)
	s.mux.HandleFunc("/api/v1/idle/disable", s.handleIdleDisable)

	// Start idle detection loop
	go s.startIdleDetectionLoop()
}

// handleIdleMetrics handles incoming metrics from instance agents.
func (s *Server) handleIdleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse metrics from request body
	var metrics IdleMetrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse metrics: %v", err), http.StatusBadRequest)
		return
	}

	// Process metrics
	state, err := s.idleManager.ProcessMetrics(metrics.InstanceID, metrics.InstanceName, &metrics.Metrics)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process metrics: %v", err), http.StatusInternalServerError)
		return
	}

	// Return state to agent
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(state); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode state: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleIdleStatus handles requests for idle status.
func (s *Server) handleIdleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get instance ID from query parameters
	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		// Return all instance states
		states := make(map[string]*idle.IdleState)
		for id, state := range s.idleManager.GetAllIdleStates() {
			states[id] = state
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(states); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode states: %v", err), http.StatusInternalServerError)
			return
		}
		return
	}

	// Get state for specific instance
	state := s.idleManager.GetIdleState(instanceID)
	if state == nil {
		http.Error(w, fmt.Sprintf("No idle state for instance %q", instanceID), http.StatusNotFound)
		return
	}

	// Return state
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(state); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode state: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleIdleConfig handles idle detection configuration.
func (s *Server) handleIdleConfig(w http.ResponseWriter, r *http.Request) {
	// Handle configuration requests
	switch r.Method {
	case http.MethodGet:
		// Return current configuration
		config := s.idleManager.GetConfig()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(config); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode config: %v", err), http.StatusInternalServerError)
			return
		}
		return

	case http.MethodPut, http.MethodPost:
		// Update configuration
		var config idle.Config
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse config: %v", err), http.StatusBadRequest)
			return
		}

		// Set configuration
		if err := s.idleManager.SetConfig(&config); err != nil {
			http.Error(w, fmt.Sprintf("Failed to set config: %v", err), http.StatusInternalServerError)
			return
		}

		// Return success
		w.WriteHeader(http.StatusOK)
		return

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// handleIdleEnable handles enabling idle detection.
func (s *Server) handleIdleEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enable idle detection
	if err := s.idleManager.Enable(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to enable idle detection: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}

// handleIdleDisable handles disabling idle detection.
func (s *Server) handleIdleDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Disable idle detection
	if err := s.idleManager.Disable(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to disable idle detection: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}

// startIdleDetectionLoop starts the idle detection loop.
func (s *Server) startIdleDetectionLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkIdleInstances()
		case <-s.ctx.Done():
			return
		}
	}
}

// checkIdleInstances checks for idle instances and takes appropriate actions.
func (s *Server) checkIdleInstances() {
	if s.idleManager == nil {
		return
	}

	// Check if idle detection is enabled
	if !s.idleManager.IsEnabled() {
		return
	}

	// Get pending actions
	pendingStates := s.idleManager.CheckPendingActions()
	for _, state := range pendingStates {
		// Take action based on state.NextAction
		if state.NextAction == nil {
			continue
		}

		switch state.NextAction.Action {
		case idle.Stop:
			s.stopIdleInstance(state)
		case idle.Hibernate:
			s.hibernateIdleInstance(state)
		case idle.Notify:
			s.notifyIdleInstance(state)
		}
	}
}

// stopIdleInstance stops an idle instance.
func (s *Server) stopIdleInstance(state *idle.IdleState) {
	// Get instance
	instance, err := s.stateManager.GetInstance(state.InstanceName)
	if err != nil {
		fmt.Printf("Failed to get instance %q: %v\n", state.InstanceName, err)
		return
	}

	// Verify the instance ID matches
	if instance.InstanceID != state.InstanceID {
		fmt.Printf("Instance ID mismatch for %q: %q != %q\n", state.InstanceName, instance.InstanceID, state.InstanceID)
		return
	}

	// Stop instance
	ctx := context.Background()
	_, err = s.awsManager.StopInstance(ctx, &types.StopInstanceRequest{
		Name: state.InstanceName,
	})
	if err != nil {
		fmt.Printf("Failed to stop instance %q: %v\n", state.InstanceName, err)
		return
	}

	// Create history entry
	entry := idle.HistoryEntry{
		InstanceID:    state.InstanceID,
		InstanceName:  state.InstanceName,
		Action:        idle.Stop,
		Time:          time.Now(),
		IdleDuration:  time.Since(*state.IdleSince),
		Metrics:       state.LastMetrics,
	}

	// Record action in history
	if err := s.idleManager.AddHistoryEntry(entry); err != nil {
		fmt.Printf("Failed to add history entry: %v\n", err)
	}

	// Reset next action
	state.NextAction = nil

	fmt.Printf("Stopped idle instance %q (idle for %s)\n", state.InstanceName, entry.IdleDuration)
}

// hibernateIdleInstance hibernates an idle instance.
func (s *Server) hibernateIdleInstance(state *idle.IdleState) {
	// Get instance
	instance, err := s.stateManager.GetInstance(state.InstanceName)
	if err != nil {
		fmt.Printf("Failed to get instance %q: %v\n", state.InstanceName, err)
		return
	}

	// Verify the instance ID matches
	if instance.InstanceID != state.InstanceID {
		fmt.Printf("Instance ID mismatch for %q: %q != %q\n", state.InstanceName, instance.InstanceID, state.InstanceID)
		return
	}

	// Hibernate instance
	ctx := context.Background()
	_, err = s.awsManager.HibernateInstance(ctx, &types.HibernateInstanceRequest{
		Name: state.InstanceName,
	})
	if err != nil {
		fmt.Printf("Failed to hibernate instance %q: %v\n", state.InstanceName, err)
		return
	}

	// Create history entry
	entry := idle.HistoryEntry{
		InstanceID:    state.InstanceID,
		InstanceName:  state.InstanceName,
		Action:        idle.Hibernate,
		Time:          time.Now(),
		IdleDuration:  time.Since(*state.IdleSince),
		Metrics:       state.LastMetrics,
	}

	// Record action in history
	if err := s.idleManager.AddHistoryEntry(entry); err != nil {
		fmt.Printf("Failed to add history entry: %v\n", err)
	}

	// Reset next action
	state.NextAction = nil

	fmt.Printf("Hibernated idle instance %q (idle for %s)\n", state.InstanceName, entry.IdleDuration)
}

// notifyIdleInstance sends a notification about an idle instance.
func (s *Server) notifyIdleInstance(state *idle.IdleState) {
	// Create history entry
	entry := idle.HistoryEntry{
		InstanceID:    state.InstanceID,
		InstanceName:  state.InstanceName,
		Action:        idle.Notify,
		Time:          time.Now(),
		IdleDuration:  time.Since(*state.IdleSince),
		Metrics:       state.LastMetrics,
	}

	// Record action in history
	if err := s.idleManager.AddHistoryEntry(entry); err != nil {
		fmt.Printf("Failed to add history entry: %v\n", err)
	}

	// Reset next action
	state.NextAction = nil

	fmt.Printf("Notified about idle instance %q (idle for %s)\n", state.InstanceName, entry.IdleDuration)
}