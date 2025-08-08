package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
)

// handleAutonomousIdleStatus handles GET /api/v1/idle/autonomous/status
func (s *Server) handleAutonomousIdleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get status from autonomous idle service if running
	status := map[string]interface{}{
		"available": true,
		"running":   false,
	}

	// TODO: Add autonomous idle service to server and get its status
	// if s.autonomousIdleService != nil {
	//     status = s.autonomousIdleService.GetStatus()
	// }

	json.NewEncoder(w).Encode(status)
}

// handleAutonomousIdleStart handles POST /api/v1/idle/autonomous/start
func (s *Server) handleAutonomousIdleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse configuration from request body
	var config idle.AutonomousConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		// Use default configuration if no config provided
		defaultConfig := idle.DefaultAutonomousConfig()
		config = *defaultConfig
	}

	// TODO: Implement autonomous idle service startup
	// This would create and start the autonomous idle service
	response := map[string]interface{}{
		"message": "Autonomous idle detection would start with this configuration",
		"config":  config,
		"note":    "Implementation pending - this is the framework structure",
	}

	json.NewEncoder(w).Encode(response)
}

// handleAutonomousIdleStop handles POST /api/v1/idle/autonomous/stop
func (s *Server) handleAutonomousIdleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// TODO: Stop autonomous idle service
	response := map[string]interface{}{
		"message": "Autonomous idle detection stopped",
	}

	json.NewEncoder(w).Encode(response)
}

// handleAutonomousIdleConfig handles GET/POST /api/v1/idle/autonomous/config
func (s *Server) handleAutonomousIdleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return current autonomous configuration
		config := idle.DefaultAutonomousConfig()
		json.NewEncoder(w).Encode(config)

	case http.MethodPost:
		// Update autonomous configuration
		var config idle.AutonomousConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid configuration")
			return
		}

		// TODO: Validate and save configuration
		response := map[string]interface{}{
			"message": "Configuration updated",
			"config":  config,
		}
		json.NewEncoder(w).Encode(response)

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}