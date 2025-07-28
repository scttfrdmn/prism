package daemon

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
)

// handleIdleStatus handles GET /api/v1/idle/status
func (s *Server) handleIdleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	status := map[string]interface{}{
		"enabled":         s.idleManager.IsEnabled(),
		"default_profile": s.idleManager.GetProfiles()[s.getDefaultProfileName()].Name,
		"profiles":        s.idleManager.GetProfiles(),
		"domain_mappings": s.idleManager.GetDomainMappings(),
	}

	json.NewEncoder(w).Encode(status)
}

// getDefaultProfileName gets the default profile name
func (s *Server) getDefaultProfileName() string {
	profile, err := s.idleManager.GetDefaultProfile()
	if err != nil {
		return "standard" // fallback
	}
	return profile.Name
}

// handleIdleEnable handles POST /api/v1/idle/enable
func (s *Server) handleIdleEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := s.idleManager.Enable(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to enable idle detection")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleIdleDisable handles POST /api/v1/idle/disable
func (s *Server) handleIdleDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := s.idleManager.Disable(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to disable idle detection")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleIdleProfiles handles /api/v1/idle/profiles
func (s *Server) handleIdleProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profiles := s.idleManager.GetProfiles()
		json.NewEncoder(w).Encode(profiles)
	case http.MethodPost:
		var profile idle.Profile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := s.idleManager.AddProfile(profile); err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		w.WriteHeader(http.StatusCreated)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleIdlePendingActions handles GET /api/v1/idle/pending-actions
func (s *Server) handleIdlePendingActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	pendingActions := s.idleManager.CheckPendingActions()
	json.NewEncoder(w).Encode(pendingActions)
}

// handleIdleExecuteActions handles POST /api/v1/idle/execute-actions
func (s *Server) handleIdleExecuteActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check for pending actions that need to be executed
	pendingActions := s.idleManager.CheckPendingActions()
	if len(pendingActions) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "No pending actions",
			"count":   0,
		})
		return
	}

	executed := 0
	errors := []string{}

	// Execute each pending action
	for _, state := range pendingActions {
		if state.NextAction == nil {
			continue
		}

		var err error
		switch state.NextAction.Action {
		case idle.Hibernate:
			err = s.executeHibernationAction(state)
		case idle.Stop:
			err = s.executeStopAction(state)
		case idle.Notify:
			err = s.executeNotifyAction(state)
		}

		if err != nil {
			errors = append(errors, err.Error())
		} else {
			executed++
			// Record history entry
			historyEntry := idle.HistoryEntry{
				InstanceID:   state.InstanceID,
				InstanceName: state.InstanceName,
				Action:       state.NextAction.Action,
				Time:         time.Now(),
				IdleDuration: time.Since(*state.IdleSince),
				Metrics:      state.LastMetrics,
			}
			s.idleManager.AddHistoryEntry(historyEntry)

			// Clear the action from state
			state.NextAction = nil
		}
	}

	response := map[string]interface{}{
		"executed": executed,
		"errors":   errors,
		"total":    len(pendingActions),
	}

	json.NewEncoder(w).Encode(response)
}

// executeHibernationAction executes hibernation action using the hibernation API
func (s *Server) executeHibernationAction(state *idle.IdleState) error {
	// Create AWS manager for the hibernation operation
	awsManager, err := aws.NewManager() // Use default configuration
	if err != nil {
		return err
	}

	// Execute hibernation using our hibernation infrastructure
	return awsManager.HibernateInstance(state.InstanceName)
}

// executeStopAction executes stop action
func (s *Server) executeStopAction(state *idle.IdleState) error {
	// Create AWS manager for the stop operation
	awsManager, err := aws.NewManager() // Use default configuration
	if err != nil {
		return err
	}

	return awsManager.StopInstance(state.InstanceName)
}

// executeNotifyAction executes notification action
func (s *Server) executeNotifyAction(state *idle.IdleState) error {
	// For now, just log the notification
	// In the future, this could send actual notifications (email, slack, etc.)
	notification := idle.Notification{
		InstanceID:   state.InstanceID,
		InstanceName: state.InstanceName,
		Message:      "Instance has been idle and may be costing unnecessary resources",
		Action:       state.NextAction.Action,
		Time:         state.NextAction.Time,
		IsWarning:    true,
	}

	// Log the notification (in a real implementation, send via email/slack/etc)
	_ = notification
	return nil
}

// handleIdleHistory handles GET /api/v1/idle/history
func (s *Server) handleIdleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	history := s.idleManager.GetHistory()
	json.NewEncoder(w).Encode(history)
}