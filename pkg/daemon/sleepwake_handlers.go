// Package daemon provides sleep/wake monitoring REST API handlers
package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/scttfrdmn/prism/pkg/sleepwake"
)

// RegisterSleepWakeRoutes registers all sleep/wake API routes
func (s *Server) RegisterSleepWakeRoutes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Sleep/wake monitoring endpoints
	mux.HandleFunc("/api/v1/sleep-wake/status", applyMiddleware(s.handleSleepWakeStatus))
	mux.HandleFunc("/api/v1/sleep-wake/configure", applyMiddleware(s.handleSleepWakeConfigure))
}

// handleSleepWakeStatus returns the current status of the sleep/wake monitor
func (s *Server) handleSleepWakeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.sleepWakeMonitor == nil {
		response := map[string]interface{}{
			"enabled": false,
			"error":   "Sleep/wake monitoring not initialized on this platform",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	status := s.sleepWakeMonitor.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode status", http.StatusInternalServerError)
		return
	}
}

// handleSleepWakeConfigure handles configuration updates for sleep/wake monitoring
func (s *Server) handleSleepWakeConfigure(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.sleepWakeMonitor == nil {
		http.Error(w, "Sleep/wake monitoring not initialized on this platform", http.StatusServiceUnavailable)
		return
	}

	var config sleepwake.Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid configuration format", http.StatusBadRequest)
		return
	}

	// Update the configuration
	if err := s.sleepWakeMonitor.UpdateConfig(config); err != nil {
		http.Error(w, "Failed to update configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// If monitoring was just disabled, stop the monitor
	if !config.Enabled && s.sleepWakeMonitor.GetStatus().Running {
		if err := s.sleepWakeMonitor.Stop(); err != nil {
			http.Error(w, "Failed to stop monitor: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// If monitoring was just enabled, start the monitor
	if config.Enabled && !s.sleepWakeMonitor.GetStatus().Running {
		if err := s.sleepWakeMonitor.Start(); err != nil {
			http.Error(w, "Failed to start monitor: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	response := map[string]interface{}{
		"success": true,
		"status":  s.sleepWakeMonitor.GetStatus(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
